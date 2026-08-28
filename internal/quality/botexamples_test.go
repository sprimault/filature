// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/core"
)

// TestBotExamplesFollowTheTypes rapproche les exemples JSON du protocole de bot
// des structures que le jeu sérialise.
//
// Ce sont des exemples au sens fort : docs/protocole-bot.md §4 exige qu'un coup
// figure « tel quel » dans legal_moves, donc un bot écrit d'après l'exemple
// envoie ce que le document montre. Il a montré « pion » et « Colonne » pendant
// que le jeu envoyait « piece » et « Column » — un coup construit dessus
// n'égalait aucune entrée, et son auteur n'avait aucun moyen de comprendre
// pourquoi.
//
// Le contrôle porte sur les clés et non sur les valeurs : un exemple a le droit
// de choisir sa graine et son tour, pas ses noms de champs.
func TestBotExamplesFollowTheTypes(t *testing.T) {
	exemples := exemplesJSON(t, filepath.Join(racine, "docs", "protocole-bot.md"))

	t.Run("hello porte tous les réglages", func(t *testing.T) {
		hello := exemples["hello"]
		if hello == nil {
			t.Fatal("aucun exemple de hello dans le document")
		}
		reglages, ok := hello["settings"].(map[string]any)
		if !ok {
			t.Fatal("l'exemple de hello ne porte pas de settings")
		}
		comparerListes(t, "les réglages annoncés à un bot",
			clesDe(t, core.DefaultSettings()), clesTriees(reglages))
	})

	t.Run("move porte les champs d'un coup", func(t *testing.T) {
		message := exemples["move"]
		if message == nil {
			t.Fatal("aucun exemple de move dans le document")
		}
		coup, ok := message["move"].(map[string]any)
		if !ok {
			t.Fatal("l'exemple de move ne porte pas de coup")
		}

		// Le coup de l'exemple est un déplacement : il sérialise donc ce que
		// Move émet moins ses champs omitempty laissés vides.
		modele := core.Move{
			Turn: 7, Side: core.SideInspectors, Type: core.MoveStep, Piece: 2,
			From: core.Position{Column: 18, Row: 9},
			To:   core.Position{Column: 19, Row: 9},
		}
		comparerListes(t, "les champs d'un coup", clesDe(t, modele), clesTriees(coup))

		for _, champ := range []string{"from", "to"} {
			position, ok := coup[champ].(map[string]any)
			if !ok {
				t.Fatalf("le champ %s de l'exemple n'est pas une position", champ)
			}
			comparerListes(t, "les champs d'une position "+champ,
				clesDe(t, core.Position{}), clesTriees(position))
		}
	})
}

// exemplesJSON extrait les blocs JSON d'un document, indexés par leur champ
// « type ».
//
// Les blocs qui portent une ellipse à la place d'un objet sont sautés : ils
// abrègent une charge utile décrite ailleurs, et les décoder échouerait sur une
// syntaxe que le document assume.
func exemplesJSON(t *testing.T, chemin string) map[string]map[string]any {
	t.Helper()

	contenu, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}

	exemples := map[string]map[string]any{}
	blocs := strings.Split(string(contenu), "```json")
	for _, bloc := range blocs[1:] {
		fin := strings.Index(bloc, "```")
		if fin < 0 {
			continue
		}
		brut := bloc[:fin]
		if strings.Contains(brut, "{ …") {
			continue
		}

		var message map[string]any
		if err := json.Unmarshal([]byte(brut), &message); err != nil {
			t.Errorf("exemple JSON du document illisible : %v\n%s", err, brut)
			continue
		}
		if typ, ok := message["type"].(string); ok {
			exemples[typ] = message
		}
	}

	if len(exemples) == 0 {
		t.Fatalf("aucun exemple JSON trouvé dans %s : le contrôle ne vérifie plus rien", chemin)
	}
	return exemples
}

// clesDe rend les clés qu'une valeur produit une fois sérialisée, triées.
func clesDe(t *testing.T, v any) []string {
	t.Helper()

	brut, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var champs map[string]any
	if err := json.Unmarshal(brut, &champs); err != nil {
		t.Fatal(err)
	}
	return clesTriees(champs)
}

// clesTriees rend les clés d'une table dans un ordre stable.
func clesTriees(table map[string]any) []string {
	cles := make([]string, 0, len(table))
	for cle := range table {
		cles = append(cles, cle)
	}
	slices.Sort(cles)
	return cles
}
