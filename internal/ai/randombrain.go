// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"errors"

	"github.com/sprimault/filature/internal/core"
)

// ErrNoLegalMove dit qu'on a demandé un coup dans une position qui n'en offre
// aucun.
//
// Ce n'est pas au cerveau de trancher : une position sans coup est une fin de
// partie, et c'est l'arbitre qui la constate.
var ErrNoLegalMove = errors.New("aucun coup legal")

// RandomBrain choisit au hasard parmi les coups que la règle autorise.
//
// C'est le bot minimal de docs/protocole-bot.md §9, écrit en Go. Il ne cherche
// pas à jouer : il sert d'adversaire tant que l'IA des étapes 9 et 10 n'existe
// pas, et de test de conformité — s'il joue cent parties sans qu'un coup soit
// refusé, c'est que LegalMoves et Apply s'accordent.
//
// Il honore BrainFactory, la signature que l'IA véritable utilisera : elle se
// branchera au même endroit sans que la boucle de jeu bouge.
type RandomBrain struct{}

// Play choisit une entrée de la liste des coups légaux.
//
// Le tirage passe par core.Random et non par math/rand : deux parties lancées
// sur la même graine doivent produire la même suite de coups, sans quoi ni le
// rejeu du journal ni la comparaison de deux versions d'IA ne veulent dire
// quelque chose.
//
// Aucun filtre, aucune préférence — pas même écarter l'abandon de tour. Un
// cerveau qui trierait un peu donnerait l'illusion d'une stratégie et
// masquerait ce que l'IA de l'étape 9 apporte vraiment.
func (RandomBrain) Play(v core.View, a *core.Random) (core.Move, error) {
	if len(v.LegalMoves) == 0 {
		return core.Move{}, ErrNoLegalMove
	}
	return v.LegalMoves[a.Int(len(v.LegalMoves))], nil
}
