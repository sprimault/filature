// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sprimault/evasion/internal/core"
	"github.com/sprimault/evasion/internal/schema"
)

// majSchemas réécrit les schémas générés au lieu de les compare.
//
// Jamais automatique : un attendu régénéré sans être relu ne teste plus rien.
// La régénération se demande, et le diff se relit en revue comme le reste.
var majSchemas = flag.Bool("maj-schemas", false, "réécrire les schémas générés")

// cheminSchemaVue est le contrat que lisent les bots et le mode réseau.
var cheminSchemaVue = filepath.Join(racine, "schemas", "view.schema.json")

// TestViewSchemaFollowsStruct vérifie que le contrat publié correspond au type
// Go dont il est tiré.
//
// Un champ ajouté à la vue, un tag modifié, un secret rendu obligatoire : tout
// cela change le schéma, et le test échoue tant que le fichier n'a pas été
// régénéré puis relu. TestViewSchemaMarksSecretsOptional garde le troisième cas
// de plus près.
//
// Un bot conforme au schéma doit pouvoir lire ce que le jeu envoie. Si les deux
// divergent, c'est le bot qui casse, chez son auteur, sans que rien ici ne
// l'annonce.
func TestViewSchemaFollowsStruct(t *testing.T) {
	attendu, err := schema.Generate(
		reflect.TypeOf(core.View{}),
		"https://github.com/sprimault/evasion/schemas/view.schema.json",
		"Vue Evasion",
		"Ce qu'un camp a le droit de savoir. Le jeu n'expose rien d'autre, ni à l'interface, ni au reseau, ni a un bot. Genere depuis core.View : ne pas modifier a la main.",
		"Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>. SPDX-License-Identifier: Apache-2.0",
	)
	if err != nil {
		t.Fatalf("génération : %v", err)
	}

	if *majSchemas {
		if err := os.WriteFile(cheminSchemaVue, attendu, 0o600); err != nil {
			t.Fatal(err)
		}
		t.Log("schéma réécrit : relire le diff avant de le livrer")
		return
	}

	publie, err := os.ReadFile(cheminSchemaVue)
	if err != nil {
		t.Fatalf("%v — lancer « go test ./internal/quality -maj-schemas » pour l'écrire", err)
	}

	if string(publie) != string(attendu) {
		t.Error("schemas/view.schema.json ne correspond plus à core.View — " +
			"relancer « go test ./internal/quality -maj-schemas » et relire le diff")
	}
}

// TestViewSchemaMarksSecretsOptional vérifie que le contrat dit ce
// que la vue fait.
//
// Position, zone scellée et résistance manquent aux inspecteurs : les déclarer
// obligatoires ferait écrire des bots qui les attendent toujours, et qui
// tomberaient sur la première vue où ils manquent.
func TestViewSchemaMarksSecretsOptional(t *testing.T) {
	brut, err := os.ReadFile(cheminSchemaVue)
	if err != nil {
		t.Skipf("schéma absent : %v", err)
	}

	requis := requiredFields(t, brut, "View")
	for _, secret := range []string{"fugitive_position", "sealed_zone", "stamina"} {
		if requis[secret] {
			t.Errorf("%s est déclaré obligatoire alors qu'il manque aux inspecteurs", secret)
		}
	}

	// Et l'inverse : ce qui est toujours là doit être annoncé comme tel.
	for _, public := range []string{"side", "turn", "phase", "known_trails"} {
		if !requis[public] {
			t.Errorf("%s n'est pas déclaré obligatoire alors qu'il est toujours présent", public)
		}
	}
}

// requiredFields lit la liste des champs obligatoires d'une définition du schéma.
func requiredFields(t *testing.T, brut []byte, definition string) map[string]bool {
	t.Helper()

	var doc struct {
		Defs map[string]struct {
			Requis []string `json:"required"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(brut, &doc); err != nil {
		t.Fatalf("schéma illisible : %v", err)
	}

	def, connue := doc.Defs[definition]
	if !connue {
		t.Fatalf("le schéma n'a pas de définition %s", definition)
	}

	requis := make(map[string]bool, len(def.Requis))
	for _, champ := range def.Requis {
		requis[champ] = true
	}
	return requis
}
