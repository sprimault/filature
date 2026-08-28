// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/core"
)

// TestReadmePresetsAreAccepted vérifie que les préréglages cités par les README
// sont ceux que le binaire reconnaît.
//
// Les deux README publiaient « --preset ville », en français jusque dans la
// version anglaise, et listaient quartier, faubourg et ville comme valeurs
// possibles. Les trois sont refusées depuis que les clés sont passées à
// l'anglais : la première ligne de commande de la page d'entrée du dépôt
// échouait, sur un message qui ne dit pas où chercher la bonne valeur.
//
// Le contrôle prend la ligne telle qu'elle est écrite et la confronte à
// PresetByKey, qui est ce que le binaire appelle. Un README ne se relit pas à
// chaque renommage ; ce test, si.
func TestReadmePresetsAreAccepted(t *testing.T) {
	for _, nom := range []string{"README.md", "README.fr.md"} {
		t.Run(nom, func(t *testing.T) {
			contenu, err := os.ReadFile(filepath.Join(racine, nom))
			if err != nil {
				t.Fatal(err)
			}

			ligne := ligneDuDrapeau(t, string(contenu), "--preset ")

			// L'argument de l'exemple : c'est lui qu'un lecteur recopie.
			exemple := strings.Fields(strings.TrimPrefix(ligne, "--preset "))
			if len(exemple) == 0 {
				t.Fatalf("%s montre --preset sans argument", nom)
			}
			if _, connu := core.PresetByKey(exemple[0]); !connu {
				t.Errorf("%s donne en exemple « --preset %s », que le binaire refuse",
					nom, exemple[0])
			}

			// Et les valeurs annoncées à côté, qui sont ce qu'on cherche quand
			// l'exemple ne convient pas.
			for _, p := range core.Presets() {
				if !strings.Contains(ligne, p.Key) {
					t.Errorf("%s ne cite pas le préréglage %q parmi les valeurs possibles",
						nom, p.Key)
				}
			}
		})
	}
}

// ligneDuDrapeau rend la ligne du bloc de drapeaux qui décrit celui demandé.
func ligneDuDrapeau(t *testing.T, contenu, drapeau string) string {
	t.Helper()

	for _, ligne := range strings.Split(contenu, "\n") {
		if strings.HasPrefix(ligne, drapeau) {
			return ligne
		}
	}
	t.Fatalf("aucune ligne ne décrit le drapeau %q : le contrôle ne vérifie plus rien", drapeau)
	return ""
}
