// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"testing"

	"github.com/sprimault/filature/internal/loader"
	"github.com/sprimault/filature/internal/render"
	"github.com/sprimault/filature/plugins"
)

// TestPieceHeadsAtTheFloorScale vérifie la taille des têtes de pions au
// plancher de rendu, que deux textes citent.
//
// docs/contrat-formes.md et la godoc de StrokeWidth s'en servent pour justifier
// le plafond d'épaisseur : à cette échelle, deux pixels de liseré avaleraient
// une tête. L'argument ne vaut que si le chiffre est le bon, et il a annoncé
// « quatre et demi » pour deux têtes qui font 5,25 et 3,75 — une valeur qui
// n'est ni l'une ni l'autre, et qu'aucune géométrie livrée ne produit.
//
// Mesuré sur les formes livrées plutôt que recopié : c'est shapes.toml qui
// décide, et il bougera à l'étape 7 quand le rendu existera pour de bon.
func TestPieceHeadsAtTheFloorScale(t *testing.T) {
	_, formes, err := loader.Load(plugins.Shipped(), "")
	if err != nil {
		t.Fatalf("chargement du contenu livré : %v", err)
	}

	attendues := map[string]float64{"fugitive": 5.25, "inspector": 3.75}
	for nom, attendu := range attendues {
		forme, present := formes.Shapes[nom]
		if !present {
			t.Errorf("la forme %s a disparu du contenu livré", nom)
			continue
		}

		tete := 0
		for _, trait := range forme.Strokes {
			if trait.Type == render.StrokeCircle {
				tete = max(tete, 2*trait.Radius)
			}
		}
		if tete == 0 {
			t.Errorf("%s n'a plus de tête : aucun cercle dans ses traits", nom)
			continue
		}

		if got := float64(tete) * render.MinRenderScale; got != attendu {
			t.Errorf("tête de %s à %.2f pixels au plancher de rendu, les textes disent %.2f",
				nom, got, attendu)
		}
	}
}
