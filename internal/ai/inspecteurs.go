// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"errors"

	"github.com/sprimault/filature/internal/core"
)

// Poids pondère les critères d'évaluation. Les niveaux sont des jeux de poids
// différents, plus un bruit décroissant — pas des algorithmes différents.
//
// L'amélioration au fil des parties est un ajustement évolutionnaire de ces
// poids : des milliers de parties IA contre IA en tâche de fond, les jeux
// gagnants conservés. Léger, explicable, et suffisant pour un jeu à quelques
// milliers de cases — inutile d'embarquer un réseau de neurones.
type Poids struct {
	Couverture   float64 `json:"couverture"`
	Interception float64 `json:"interception"`
	Dispersion   float64 `json:"dispersion"`
	GardeZones   float64 `json:"garde_zones"`
	Centroid     float64 `json:"barycentre"`
	Bruit        float64 `json:"bruit"`
}

// Inspectors est l'IA embarquée du sideName des poursuivants.
//
// Elle se branche sur la même signature qu'un bot externe : c'est ce qui
// garantit que le protocole est suffisant, puisque le jeu s'en sert lui-même.
type Inspectors struct {
	poids    Poids
	croyance *Croyance
}

// Play choisit un coup en minimisant l'espérance de masse résiduelle après
// déplacement, sous contrainte de couvrir les zones d'extraction.
func (i *Inspectors) Play(v core.View, a *core.Random) (core.Move, error) {
	return core.Move{}, errors.New("à implémenter : étape 9")
}
