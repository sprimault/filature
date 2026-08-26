// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package server

import "github.com/sprimault/filature/internal/core"

// Enveloppe est le message échangé en WebSocket, dans les deux sens.
//
// Les noms de types partent sur le réseau et se retrouvent dans les traces :
// ils ne se renomment pas sans casser la compatibilité entre deux versions du
// jeu.
type Enveloppe struct {
	Type string `json:"type"`

	// client vers serveur
	Jeton string     `json:"jeton,omitempty"`
	Role  core.Side  `json:"role,omitempty"`
	Move  *core.Move `json:"coup,omitempty"`

	// serveur vers client
	View    *core.View    `json:"vue,omitempty"`
	Outcome *core.Outcome `json:"resultat,omitempty"`
	Message string        `json:"message,omitempty"`

	Manifeste []core.ManifestEntry `json:"manifeste,omitempty"`
}

// Types de messages. La reconnexion passe par le jeton de session : le serveur
// rejoue le journal et renvoie une View complète, il n'y a aucun état de session
// à conserver côté client.
const (
	MsgRejoindre   = "rejoindre"
	MsgChoisirRole = "choisir_role"
	MsgJouerCoup   = "jouer_coup"
	MsgAbandonner  = "abandonner"

	MsgVue        = "vue"
	MsgCoupJoue   = "coup_joue"
	MsgTourChange = "tour_change"
	MsgRevelation = "revelation"
	MsgFinPartie  = "fin_partie"
	MsgErreur     = "erreur"
)
