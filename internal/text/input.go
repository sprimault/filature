// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package text

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/sprimault/filature/internal/core"
)

// ErrQuit dit que le joueur a demandé à sortir, ce qui n'est pas une panne.
var ErrQuit = errors.New("partie interrompue")

// Moves énumère les coups légaux, numérotés à partir de un.
//
// Numérotés et non décrits par une syntaxe à apprendre : la liste vient de
// LegalMoves, donc tout ce qui s'y trouve est jouable et rien d'autre ne
// l'est. Le joueur choisit dans ce que la règle autorise au lieu de proposer un
// coup que le jeu refusera.
func Moves(coups []core.Move) string {
	var s strings.Builder
	for i, c := range coups {
		fmt.Fprintf(&s, "%3d. %s\n", i+1, Describe(c))
	}
	return s.String()
}

// Describe rend un coup en une ligne lisible.
//
// Une capacité et une dépense nomment la case qu'elles désignent, quand elles en
// désignent une : le Barreur propose un coup par voisine praticable et le leurre
// un par couple de cases, et sans elle la liste offre au joueur des lignes
// identiques entre lesquelles il ne peut pas choisir.
func Describe(c core.Move) string {
	switch c.Type {
	case core.MovePlace:
		if c.Side == core.SideFugitive {
			return fmt.Sprintf("sceller la zone %d", c.Zone)
		}
		return fmt.Sprintf("placer %s en (%d,%d)",
			PieceLetter(c.Piece), c.To.Column, c.To.Row)

	case core.MoveStep:
		qui := "le fugitif"
		if c.Side == core.SideInspectors {
			qui = PieceLetter(c.Piece)
		}
		return fmt.Sprintf("déplacer %s vers (%d,%d) %s",
			qui, c.To.Column, c.To.Row, direction(c.From, c.To))

	case core.MoveAbility:
		rendu := fmt.Sprintf("capacité %s de %s", c.Ability, PieceLetter(c.Piece))
		if c.From != c.To {
			rendu += fmt.Sprintf(" sur (%d,%d) %s", c.To.Column, c.To.Row, direction(c.From, c.To))
		}
		return rendu

	case core.MoveExpense:
		rendu := fmt.Sprintf("dépenser %s", c.Expense)
		if c.From != c.To {
			rendu += fmt.Sprintf(" en (%d,%d) %s", c.From.Column, c.From.Row, direction(c.From, c.To))
		}
		return rendu

	case core.MoveChangeZone:
		return fmt.Sprintf("resceller vers la zone %d", c.Zone)

	case core.MovePass:
		return "passer son déplacement"

	case core.MoveEndPhase:
		return "rendre la main"

	default:
		return string(c.Type)
	}
}

// direction nomme le sens d'un déplacement, quand il en a un.
//
// Une flèche cardinale se lit plus vite qu'un couple de coordonnées lorsqu'on
// choisit parmi huit cases voisines qui ne diffèrent que d'une unité.
func direction(de, vers core.Position) string {
	d, adjacentes := core.DirectionTo(de, vers)
	if !adjacentes {
		return ""
	}
	noms := [...]string{"nord", "est", "sud", "ouest",
		"nord-est", "sud-est", "sud-ouest", "nord-ouest"}
	return noms[d]
}

// ReadMove demande un numéro jusqu'à en obtenir un valide.
//
// Une saisie hors liste est redemandée plutôt que refusée : c'est une faute de
// frappe, pas une tentative de tricher, et le noyau n'a jamais à voir un coup
// illégal. La fin d'entrée vaut abandon, ce qui rend le mode texte pilotable
// par un fichier.
func ReadMove(entree io.Reader, sortie io.Writer, coups []core.Move) (core.Move, error) {
	if len(coups) == 0 {
		return core.Move{}, errors.New("aucun coup legal")
	}

	lecteur := bufio.NewScanner(entree)
	for {
		// L'invite est un confort : si la sortie est morte, c'est la lecture
		// juste en dessous qui le dira, et elle, on la vérifie.
		_, _ = fmt.Fprintf(sortie, "Coup (1-%d, q pour abandonner) > ", len(coups))

		if !lecteur.Scan() {
			if err := lecteur.Err(); err != nil {
				return core.Move{}, err
			}
			return core.Move{}, ErrQuit
		}

		saisie := strings.TrimSpace(lecteur.Text())
		if saisie == "q" || saisie == "Q" {
			return core.Move{}, ErrQuit
		}

		n, err := strconv.Atoi(saisie)
		if err != nil || n < 1 || n > len(coups) {
			_, _ = fmt.Fprintf(sortie, "« %s » n'est pas un numéro de la liste.\n", saisie)
			continue
		}
		return coups[n-1], nil
	}
}
