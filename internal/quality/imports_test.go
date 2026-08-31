// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// interdits liste, par paquet, ce qu'il n'a pas le droit d'importer, avec la
// raison du refus — c'est elle que lira celui dont le test échoue.
//
// Les invariants du projet sont écrits dans docs/architecture.md et rappelés en
// commentaire à plusieurs endroits. Aucun n'était outillé : ni depguard dans
// .golangci.yml, ni contrôle ici. Ils tenaient par la relecture, et une
// relecture ne se déclenche pas au moment où quelqu'un ajoute un import.
var interdits = map[string][]struct {
	prefixe string
	raison  string
}{
	"internal/core": {
		{"github.com/sprimault/evasion/internal/", "le noyau est une feuille du graphe : il applique des règles à un terrain, il ne dépend de rien du dépôt"},
		{"math/rand", "core.Random est le seul générateur autorisé, sans quoi un rejeu diverge de son journal"},
		{"crypto/rand", "core.Random est le seul générateur autorisé, sans quoi un rejeu diverge de son journal"},
		{"time", "l'horloge n'entre pas dans le noyau : une décision qui en dépend ne se rejoue pas"},
		{"github.com/hajimehoshi/ebiten", "aucune coordonnée d'écran dans l'état, et les runners d'intégration sont sans écran"},
	},
	"internal/ai": {
		{"math/rand", "core.Random est le seul générateur autorisé, l'IA comprise"},
		{"crypto/rand", "core.Random est le seul générateur autorisé, l'IA comprise"},
		{"github.com/hajimehoshi/ebiten", "l'IA tourne sans écran, y compris dans la simulation d'équilibrage"},
	},
}

// TestPackagesKeepTheirImportsClean garde les invariants qui se lisent sur les
// imports d'un paquet.
//
// Les fichiers de test sont exclus : ils mesurent des durées, tirent au hasard
// pour bâtir des cas, et rien de ce qu'ils importent n'entre dans le binaire.
func TestPackagesKeepTheirImportsClean(t *testing.T) {
	for paquet, refus := range interdits {
		fset := token.NewFileSet()
		dossier := filepath.Join(racine, filepath.FromSlash(paquet))

		entrees, err := os.ReadDir(dossier)
		if err != nil {
			t.Fatalf("%s : %v", paquet, err)
		}

		for _, e := range entrees {
			nom := e.Name()
			if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
				continue
			}

			fichier, err := parser.ParseFile(fset, filepath.Join(dossier, nom), nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("%s : %v", nom, err)
			}
			for _, imp := range fichier.Imports {
				chemin := strings.Trim(imp.Path.Value, `"`)
				for _, r := range refus {
					if strings.HasPrefix(chemin, r.prefixe) {
						t.Errorf("%s importe %q — %s", nom, chemin, r.raison)
					}
				}
			}
		}
	}
}
