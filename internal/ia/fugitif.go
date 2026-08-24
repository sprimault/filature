// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package ia

import (
	"errors"

	"github.com/sprimault/filature/internal/noyau"
)

// Fugitif est le miroir de l'IA des inspecteurs : maximiser la masse de
// croyance résiduelle, éviter les zones couvertes, arbitrer les dépenses de
// résistance.
//
// Elle sert autant de mode démonstration que de partenaire d'entraînement pour
// l'équilibrage.
type Fugitif struct {
	poids Poids
}

// Jouer choisit un déplacement et, le cas échéant, une dépense.
func (f *Fugitif) Jouer(v noyau.Vue, a *noyau.Alea) (noyau.Coup, error) {
	return noyau.Coup{}, errors.New("à implémenter : étape 10")
}
