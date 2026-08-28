// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"encoding/json"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/sprimault/filature/internal/ai"
	"github.com/sprimault/filature/internal/core"
	"github.com/sprimault/filature/internal/render"
)

// Les trois contrats publics que ce fichier rapproche du code.
var (
	cheminSchemaManifeste = filepath.Join(racine, "schemas", "plugin-manifest.schema.json")
	cheminSchemaFormes    = filepath.Join(racine, "schemas", "shapes.schema.json")
	cheminSchemaBot       = filepath.Join(racine, "schemas", "bot-protocol.schema.json")
)

// enumDuSchema lit une énumération du schéma de manifeste, désignée par le
// chemin de ses clés.
func enumDuSchema(t *testing.T, chemin ...string) []string {
	t.Helper()
	return enumDuFichier(t, cheminSchemaManifeste, chemin...)
}

// enumDuFichier lit une énumération d'un schéma, désignée par le chemin de ses
// clés.
//
// Le chemin est donné clé par clé plutôt qu'en pointeur JSON : une clé absente
// nomme alors l'endroit exact où le schéma a bougé, au lieu d'un « introuvable »
// sur la chaîne entière.
func enumDuFichier(t *testing.T, cheminSchema string, chemin ...string) []string {
	t.Helper()

	brut, err := os.ReadFile(cheminSchema)
	if err != nil {
		t.Fatal(err)
	}
	var courant map[string]any
	if err := json.Unmarshal(brut, &courant); err != nil {
		t.Fatal(err)
	}

	for i, cle := range chemin {
		valeur, present := courant[cle]
		if !present {
			t.Fatalf("%s : la clé %q est absente sous %v", cheminSchema, cle, chemin[:i])
		}
		if i == len(chemin)-1 {
			liste, ok := valeur.([]any)
			if !ok {
				t.Fatalf("%s : %v n'est pas une énumération", cheminSchema, chemin)
			}
			// Les énumérations de version portent des entiers, celles du
			// vocabulaire des chaînes : les deux se comparent comme du texte,
			// et un nombre JSON arrive en float64.
			var valeurs []string
			for _, v := range liste {
				switch valeur := v.(type) {
				case string:
					valeurs = append(valeurs, valeur)
				case float64:
					valeurs = append(valeurs, strconv.FormatFloat(valeur, 'f', -1, 64))
				default:
					t.Fatalf("%s : %v porte une valeur de type %T", cheminSchema, chemin, v)
				}
			}
			return valeurs
		}
		suivant, ok := valeur.(map[string]any)
		if !ok {
			t.Fatalf("%s : %v ne mène nulle part", cheminSchema, chemin[:i+1])
		}
		courant = suivant
	}
	return nil
}

// TestSchemaEnumsMatchTheVocabulary rapproche les énumérations du schéma
// publié des constantes du noyau.
//
// Les deux décrivent le même contrat et personne ne les tenait ensemble :
// TestVocabularyFullyEnumerated compare EffectTypes et Targets aux constantes
// du même fichier, ce qui laisse le schéma libre de dériver. Une primitive
// ajoutée au noyau et oubliée là ferait refuser au chargement un manifeste que
// son auteur aurait validé contre le contrat publié — et il n'aurait aucun
// moyen de comprendre pourquoi.
//
// Les déclenchements se partagent entre les deux énumérations sans se recouvrir,
// et c'est ce que le test vérifie : chacun est ouvert là où le noyau le
// consulte, et leur réunion couvre les constantes. Un déclenchement ouvert des
// deux côtés serait inerte d'un des deux, un déclenchement ouvert nulle part
// serait une constante morte — les deux cas ont existé.
func TestSchemaEnumsMatchTheVocabulary(t *testing.T) {
	var primitives []string
	for _, e := range core.EffectTypes() {
		primitives = append(primitives, string(e))
	}
	comparerListes(t, "les primitives",
		primitives, enumDuSchema(t, "$defs", "effect", "properties", "type", "enum"))

	var cibles []string
	for _, c := range core.Targets() {
		cibles = append(cibles, string(c))
	}
	comparerListes(t, "les cibles",
		cibles, enumDuSchema(t, "$defs", "effect", "properties", "target", "enum"))

	fset := token.NewFileSet()
	chemin := filepath.Join(racine, "internal", "core", "registry.go")
	var declencheurs []string
	for _, c := range constantesDuType(t, fset, chemin, "Trigger") {
		declencheurs = append(declencheurs, c.valeur)
	}
	// Un mode ne se déclare que sur l'étranglement, seul moment que le jeu
	// déclenche de lui-même ; une capacité ne s'y déclare jamais, un pion qui
	// s'y accrocherait agissant sans que son camp l'ait joué.
	mode := enumDuSchema(t, "$defs", "mode", "properties", "trigger", "enum")
	comparerListes(t, "les déclenchements d'un mode", []string{string(core.OnStrangling)}, mode)

	capacite := enumDuSchema(t, "$defs", "ability", "properties", "trigger", "enum")
	if slices.Contains(capacite, string(core.OnStrangling)) {
		t.Errorf("le schéma ouvre %q aux capacités, il est réservé aux modes", core.OnStrangling)
	}

	// La réunion des deux couvre les constantes, sans recouvrement : c'est ce
	// qui interdit qu'une valeur reste déclarée sans que rien ne la déclenche.
	comparerListes(t, "les déclenchements", declencheurs, append(slices.Clone(capacite), mode...))
}

// TestContractVersionsMatchTheirSchemas rapproche les trois numéros de contrat
// des schémas qui les publient.
//
// Les trois avaient dérivé : le schéma de manifeste figeait effects_version à 1
// quand le noyau appliquait la 2, et celui du protocole restait à 1 dans ses
// deux messages. Un auteur qui validait son manifeste contre le contrat publié
// obtenait un fichier que le chargeur refuse, et un bot conforme au schéma
// aurait été écarté à la poignée de main.
//
// Chaque numéro est une énumération à une seule valeur : c'est ce qui permet à
// un auteur de savoir contre quoi il écrit, et à ce test de le vérifier.
func TestContractVersionsMatchTheirSchemas(t *testing.T) {
	cas := []struct {
		quoi   string
		noyau  int
		schema []string
	}{
		{"effects_version", core.EffectsVersion,
			enumDuSchema(t, "properties", "effects_version", "enum")},
		{"shapes_version", render.ShapesVersion,
			enumDuFichier(t, cheminSchemaFormes, "properties", "shapes_version", "enum")},
		{"protocol dans hello", ai.BotProtocol,
			enumDuFichier(t, cheminSchemaBot, "$defs", "hello", "properties", "protocol", "enum")},
		{"protocol dans ready", ai.BotProtocol,
			enumDuFichier(t, cheminSchemaBot, "$defs", "ready", "properties", "protocol", "enum")},
	}

	for _, c := range cas {
		if len(c.schema) != 1 {
			t.Errorf("%s : le schéma publie %d versions, attendu une seule", c.quoi, len(c.schema))
			continue
		}
		if c.schema[0] != strconv.Itoa(c.noyau) {
			t.Errorf("%s : le schéma publie %s, ce binaire applique %d",
				c.quoi, c.schema[0], c.noyau)
		}
	}
}

// comparerListes signale ce qu'une énumération oublie et ce qu'elle invente,
// sans tenir compte de l'ordre.
func comparerListes(t *testing.T, quoi string, noyau, schema []string) {
	t.Helper()

	publiees := map[string]bool{}
	for _, v := range schema {
		publiees[v] = true
	}
	for _, v := range noyau {
		if !publiees[v] {
			t.Errorf("le schéma oublie %q parmi %s", v, quoi)
		}
		delete(publiees, v)
	}
	for reste := range publiees {
		t.Errorf("le schéma publie %q parmi %s, que le noyau ne connaît pas", reste, quoi)
	}
}
