// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/core"
)

// TestMeasurementBlockNamesThePresets vérifie que le bloc de mesures des règles
// nomme les préréglages comme le code les nomme.
//
// docs/regles.md §15 affirme sa propre provenance — « ce bloc est réécrit par la
// mesure elle-même, ne pas le modifier à la main ». Sa première colonne a
// pourtant porté « quartier », « faubourg » et « ville » longtemps après que
// Presets ait rendu district, outskirts et city : le bloc ne pouvait donc pas
// venir du harnais, qui écrit la clé telle quelle.
//
// Le contrôle s'arrête aux noms. Les chiffres, eux, sont des conséquences : un
// test qui rougirait quand un taux bouge rougirait au premier changement voulu
// de génération, et se désactiverait au troisième passage. C'est le seul écart
// assumé au modèle des autres contrôles de ce paquet.
func TestMeasurementBlockNamesThePresets(t *testing.T) {
	contenu, err := os.ReadFile(filepath.Join(racine, "docs", "regles.md"))
	if err != nil {
		t.Fatal(err)
	}

	const debut, fin = "<!-- mesures:début -->", "<!-- mesures:fin -->"
	_, apres, ouvert := strings.Cut(string(contenu), debut)
	bloc, ferme, _ := strings.Cut(apres, fin)
	if !ouvert || ferme == "" {
		t.Fatal("les marqueurs du bloc de mesures ont disparu de docs/regles.md")
	}

	// La première cellule de chaque ligne de tableau, hors en-tête et
	// séparateur : c'est là que le harnais écrit la clé.
	var nommes []string
	for _, ligne := range strings.Split(bloc, "\n") {
		if !strings.HasPrefix(ligne, "| ") || strings.HasPrefix(ligne, "|---") {
			continue
		}
		cellule := strings.TrimSpace(strings.Split(ligne, "|")[1])
		if cellule == "Préréglage" {
			continue
		}
		nommes = append(nommes, cellule)
	}

	var attendus []string
	for _, p := range core.Presets() {
		attendus = append(attendus, p.Key)
	}
	slices.Sort(attendus)
	slices.Sort(nommes)

	if !slices.Equal(attendus, nommes) {
		t.Errorf("le bloc de mesures nomme %v, les préréglages sont %v : "+
			"le bloc n'a pas été régénéré depuis que les clés ont changé",
			nommes, attendus)
	}
}
