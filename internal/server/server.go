// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Package server porte le mode réseau.
//
// Le même binaire héberge ou rejoint ; rien n'empêche d'héberger une partie
// tout en en rejoignant une autre, ce sont deux instances du même moteur.
//
// L'hôte fait autorité. Le client n'envoie que des intentions de coup, jamais
// d'état, et ne reçoit que sa View.
package server

import (
	"context"
	"errors"

	"github.com/sprimault/filature/internal/core"
)

// Hote sert une partie à un client distant.
type Hote struct{}

// Host écoute et attend le second joueur.
func Host(ctx context.Context, adresse string, p *core.Game) (*Hote, error) {
	return nil, errors.New("à implémenter : étape 12")
}

// Join se connecte à un hôte et négocie le manifeste.
//
// Les plugins de règles doivent être identiques des deux côtés — comparaison
// par fingerprint, pas par numéro de version. Les plugins d'apparence en sont
// dispensés : deux joueurs peuvent avoir des habillages différents sans que la
// partie diverge, et c'est le contrôle du drapeau regles au chargement qui rend
// cette dispense sûre.
func Join(ctx context.Context, adresse string, manifeste []core.ManifestEntry) error {
	return errors.New("à implémenter : étape 12")
}
