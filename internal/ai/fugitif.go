// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"errors"

	"github.com/sprimault/filature/internal/core"
)

// Fugitive est le miroir de l'IA des inspecteurs : maximiser la masse de
// croyance résiduelle, éviter les zones couvertes, arbitrer les dépenses de
// résistance.
//
// Elle sert autant de mode démonstration que de partenaire d'entraînement pour
// l'équilibrage.
type Fugitive struct {
	poids Poids
}

// Play choisit un déplacement et, le cas échéant, une dépense.
func (f *Fugitive) Play(v core.View, a *core.Random) (core.Move, error) {
	return core.Move{}, errors.New("à implémenter : étape 10")
}
