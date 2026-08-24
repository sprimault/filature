// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package ia

import (
	"context"
	"errors"

	"github.com/sprimault/filature/internal/noyau"
)

// ProtocoleBot est la version parlée par ce binaire.
const ProtocoleBot = 1

// Un bot remplace l'IA du jeu, il ne l'étend pas : le jeu envoie une Vue, le
// bot renvoie un Coup. L'IA livrée parle ce même protocole, ce qui garantit
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

	Protocole  int                     `json:"protocole,omitempty"`
	Camp       noyau.Acteur            `json:"camp,omitempty"`
	Graine     int64                   `json:"graine,omitempty"`
	Parametres *noyau.Parametres       `json:"parametres,omitempty"`
	Greffons   []noyau.EntreeManifeste `json:"greffons,omitempty"`

	Tour     int        `json:"tour,omitempty"`
	BudgetMs int        `json:"budget_ms,omitempty"`
	Vue      *noyau.Vue `json:"vue,omitempty"`

	Nom          string `json:"nom,omitempty"`
	Version      string `json:"version,omitempty"`
	Deterministe bool   `json:"deterministe,omitempty"`

	Coup *noyau.Coup `json:"coup,omitempty"`

	Vainqueur noyau.Acteur `json:"vainqueur,omitempty"`
	Motif     string       `json:"motif,omitempty"`
	Message   string       `json:"message,omitempty"`
}

// Bot pilote un processus externe.
//
// Il ne reçoit que la Vue de son camp, la même que l'interface : il ne voit
// jamais la position cachée du fugitif ni sa zone scellée. Ce n'est pas une
// politesse, c'est la projection qui protège déjà le mode réseau, appliquée au
// même endroit.
type Bot struct {
	nom          string
	deterministe bool
	depassements int
}

// Lancer démarre le processus et échange bonjour/pret.
//
// La graine de la partie lui est transmise : c'est ce qui permet à un bot
// d'être déterministe sans horloge ni entropie système.
func Lancer(ctx context.Context, commande string, args []string, camp noyau.Acteur, p *noyau.Partie) (*Bot, error) {
	return nil, errors.New("à implémenter : étape 9")
}

// Jouer demande un coup et vérifie qu'il figure dans les coups légaux.
//
// Le jeu ne corrige ni n'interprète jamais un coup reçu : un coup illégal
// interrompt la partie et part au journal tel quel. Un budget dépassé une fois
// donne un coup légal tiré au sort ; trois fois, la partie s'arrête. Laisser
// une partie se figer sur un processus tiers est pire qu'un coup au hasard, et
// le journal garde la trace des deux.
func (b *Bot) Jouer(ctx context.Context, v noyau.Vue, budgetMs int) (noyau.Coup, error) {
	return noyau.Coup{}, errors.New("à implémenter : étape 9")
}

// Arreter envoie fin et ferme l'entrée standard, puis tue le processus s'il
// survit une seconde.
func (b *Bot) Arreter(r noyau.Resultat) error {
	return errors.New("à implémenter : étape 9")
}
