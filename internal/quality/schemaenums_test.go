// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"encoding/json"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sprimault/filature/internal/core"
)

// cheminSchemaManifeste est le contrat que lisent les auteurs de plugins.
var cheminSchemaManifeste = filepath.Join(racine, "schemas", "plugin-manifest.schema.json")

// enumDuSchema lit une énumération du schéma de manifeste, désignée par le
// chemin de ses clés.
//
// Le chemin est donné clé par clé plutôt qu'en pointeur JSON : une clé absente
// nomme alors l'endroit exact où le schéma a bougé, au lieu d'un « introuvable »
// sur la chaîne entière.
func enumDuSchema(t *testing.T, chemin ...string) []string {
	t.Helper()

	brut, err := os.ReadFile(cheminSchemaManifeste)
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
			t.Fatalf("%s : la clé %q est absente sous %v", cheminSchemaManifeste, cle, chemin[:i])
		}
		if i == len(chemin)-1 {
			liste, ok := valeur.([]any)
			if !ok {
				t.Fatalf("%s : %v n'est pas une énumération", cheminSchemaManifeste, chemin)
			}
			var valeurs []string
			for _, v := range liste {
				valeurs = append(valeurs, v.(string))
			}
			return valeurs
		}
		suivant, ok := valeur.(map[string]any)
		if !ok {
			t.Fatalf("%s : %v ne mène nulle part", cheminSchemaManifeste, chemin[:i+1])
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
// Les deux énumérations de déclenchement diffèrent d'un terme, et c'est voulu :
// strangling est réservé aux modes. Le test le vérifie plutôt que de l'admettre,
// parce que c'est la seule asymétrie du contrat et qu'elle se perdrait dans un
// rapprochement à gros grain.
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
	comparerListes(t, "les déclenchements d'un mode",
		declencheurs, enumDuSchema(t, "$defs", "mode", "properties", "trigger", "enum"))

	// Une capacité ne peut pas se déclarer sur l'étranglement : le jeu le
	// déclenche de lui-même, et un pion qui s'y accrocherait agirait sans que
	// son camp l'ait joué.
	capacite := enumDuSchema(t, "$defs", "ability", "properties", "trigger", "enum")
	if slices.Contains(capacite, string(core.OnStrangling)) {
		t.Errorf("le schéma ouvre %q aux capacités, il est réservé aux modes", core.OnStrangling)
	}
	attendus := slices.DeleteFunc(slices.Clone(declencheurs), func(d string) bool {
		return d == string(core.OnStrangling)
	})
	comparerListes(t, "les déclenchements d'une capacité", attendus, capacite)
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
