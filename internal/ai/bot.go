// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"errors"
	"fmt"

	"github.com/sprimault/filature/internal/core"
)

// BotProtocol est la version parlée par ce binaire.
//
// Passée à 3 quand les six types de message sont passés à l'anglais. Les
// versions 1 et 2 nommaient un coup « coup » dans le schéma et « move » dans la
// documentation, et les cinq autres types en français : un auteur de bot ne
// pouvait deviner ni la règle ni l'exception.
//
// Passée à 4 quand la vue du fugitif a gagné le compte de ses dépenses
// plafonnées et le leurre qu'il vient d'armer. Un bot de la version 3 ne voit ni
// l'un ni l'autre, et il ne pouvait pas les recalculer : deux états qui laissent
// des vues indiscernables ne se déduisent pas.
const BotProtocol = 4

// Un bot remplace l'IA du jeu, il ne l'étend pas : le jeu envoie une View, le
// bot renvoie un Move. L'IA livrée parle ce même protocole, ce qui garantit
// qu'il est suffisant — s'il manquait quelque chose, le jeu ne pourrait pas
// jouer contre lui-même.

// Message est l'enveloppe échangée en JSON Lines sur les entrée et sortie
// standard du processus.
//
// Un objet par ligne, sans indentation : pas de longueur préfixée, pas de
// négociation, pas de socket. Un bot s'écrit en trente lignes de Python et se
// rejoue à la main avec un fichier et une redirection.
type Message struct {
	Type string `json:"type"`

	Protocole int                  `json:"protocol,omitempty"`
	Camp      core.Side            `json:"side,omitempty"`
	Seed      int64                `json:"seed,omitempty"`
	Settings  *core.Settings       `json:"settings,omitempty"`
	Plugins   []core.ManifestEntry `json:"plugins,omitempty"`

	Turn     int        `json:"turn,omitempty"`
	BudgetMs int        `json:"budget_ms,omitempty"`
	View     *core.View `json:"view,omitempty"`

	Name         string `json:"name,omitempty"`
	Version      string `json:"version,omitempty"`
	Author       string `json:"author,omitempty"`
	Deterministe bool   `json:"deterministic,omitempty"`

	Move *core.Move `json:"move,omitempty"`

	Winner  core.Side `json:"winner,omitempty"`
	Reason  string    `json:"reason,omitempty"`
	Message string    `json:"message,omitempty"`
}

// Bot pilote un processus externe.
//
// Il ne reçoit que la View de son camp, la même que l'interface : il ne voit
// jamais la position cachée du fugitif ni sa zone scellée. Ce n'est pas une
// politesse, c'est la projection qui protège déjà le mode réseau, appliquée au
// même endroit.
type Bot struct {
	nom          string
	deterministe bool
	depassements int
}

// Start démarre le processus et échange hello/ready.
//
// La graine de la partie lui est transmise : c'est ce qui permet à un bot
// d'être déterministe sans horloge ni entropie système.
//
// checkProtocol est appelé sur le ready reçu, avant tout autre message.
func Start(ctx context.Context, command string, args []string, sideName core.Side, p *core.Game) (*Bot, error) {
	return nil, errors.New("à implémenter : étape 9")
}

// checkProtocol accepte ou écarte la version qu'un bot annonce dans son ready.
//
// **Au premier message et non au troisième.** Un bot écrit contre une version
// antérieure enverrait sinon des types que ce binaire ne connaît plus, et
// échouerait sur un « type inconnu » qui ne dit pas ce qui s'est passé. Le
// message porte les deux numéros et ce qu'il y a à faire : c'est tout ce dont
// son auteur dispose, le jeu n'ayant aucun moyen de le joindre.
func checkProtocol(annoncee int) error {
	if annoncee == BotProtocol {
		return nil
	}
	if annoncee < BotProtocol {
		return fmt.Errorf("le bot annonce le protocole %d, ce binaire parle le %d : "+
			"mettre le bot a jour, ou jouer avec une version du jeu qui parle le %d",
			annoncee, BotProtocol, annoncee)
	}
	return fmt.Errorf("le bot annonce le protocole %d, ce binaire parle le %d : "+
		"mettre le jeu a jour", annoncee, BotProtocol)
}

// Play demande un coup et vérifie qu'il figure dans les coups légaux.
//
// Le jeu ne corrige ni n'interprète jamais un coup reçu : un coup illégal
// interrompt la partie et part au journal tel quel. Un budget dépassé une fois
// donne un coup légal choisi au sort ; trois fois, la partie s'arrête. Laisser
// une partie se figer sur un processus tiers est pire qu'un coup au hasard, et
// le journal garde la trace des deux.
func (b *Bot) Play(ctx context.Context, v core.View, budgetMs int) (core.Move, error) {
	return core.Move{}, errors.New("à implémenter : étape 9")
}

// Stop envoie over et ferme l'entrée standard, puis tue le processus s'il
// survit une seconde.
func (b *Bot) Stop(r core.Outcome) error {
	return errors.New("à implémenter : étape 9")
}
