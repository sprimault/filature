// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package texte

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/sprimault/filature/internal/noyau"
)

// ErrQuitter dit que le joueur a demandé à sortir, ce qui n'est pas une panne.
var ErrQuitter = errors.New("partie interrompue")

// Coups liste les coups légaux, numérotés à partir de un.
//
// Numérotés et non décrits par une syntaxe à apprendre : la liste vient de
// CoupsLegaux, donc tout ce qui s'y trouve est jouable et rien d'autre ne
// l'est. Le joueur choisit dans ce que la règle autorise au lieu de proposer un
// coup que le jeu refusera.
func Coups(coups []noyau.Coup) string {
	var s strings.Builder
	for i, c := range coups {
		fmt.Fprintf(&s, "%3d. %s\n", i+1, Decrire(c))
	}
	return s.String()
}

// Decrire rend un coup en une ligne lisible.
func Decrire(c noyau.Coup) string {
	switch c.Type {
	case noyau.CoupPlacer:
		if c.Acteur == noyau.CampFugitif {
			return fmt.Sprintf("sceller la zone %d", c.Zone)
		}
		return fmt.Sprintf("placer %s en (%d,%d)",
			LettrePion(c.Pion), c.Arrivee.Colonne, c.Arrivee.Ligne)

	case noyau.CoupDeplacer:
		qui := "le fugitif"
		if c.Acteur == noyau.CampInspecteurs {
			qui = LettrePion(c.Pion)
		}
		return fmt.Sprintf("déplacer %s vers (%d,%d) %s",
			qui, c.Arrivee.Colonne, c.Arrivee.Ligne, direction(c.Depart, c.Arrivee))

	case noyau.CoupCapacite:
		return fmt.Sprintf("capacité %s de %s", c.Capacite, LettrePion(c.Pion))

	case noyau.CoupDepense:
		return fmt.Sprintf("dépenser %s", c.Depense)

	case noyau.CoupChangerZone:
		return fmt.Sprintf("resceller vers la zone %d", c.Zone)

	case noyau.CoupPasser:
		return "passer son déplacement"

	case noyau.CoupFinDePhase:
		return "rendre la main"

	default:
		return string(c.Type)
	}
}

// direction nomme le sens d'un déplacement, quand il en a un.
//
// Une flèche cardinale se lit plus vite qu'un couple de coordonnées lorsqu'on
// choisit parmi huit cases voisines qui ne diffèrent que d'une unité.
func direction(de, vers noyau.Position) string {
	d, adjacentes := noyau.DirectionVers(de, vers)
	if !adjacentes {
		return ""
	}
	noms := [...]string{"nord", "est", "sud", "ouest",
		"nord-est", "sud-est", "sud-ouest", "nord-ouest"}
	return noms[d]
}

// LireCoup demande un numéro jusqu'à en obtenir un valide.
//
// Une saisie hors liste est redemandée plutôt que refusée : c'est une faute de
// frappe, pas une tentative de tricher, et le noyau n'a jamais à voir un coup
// illégal. La fin d'entrée vaut abandon, ce qui rend le mode texte pilotable
// par un fichier.
func LireCoup(entree io.Reader, sortie io.Writer, coups []noyau.Coup) (noyau.Coup, error) {
	if len(coups) == 0 {
		return noyau.Coup{}, errors.New("aucun coup legal")
	}

	lecteur := bufio.NewScanner(entree)
	for {
		// L'invite est un confort : si la sortie est morte, c'est la lecture
		// juste en dessous qui le dira, et elle, on la vérifie.
		_, _ = fmt.Fprintf(sortie, "Coup (1-%d, q pour abandonner) > ", len(coups))

		if !lecteur.Scan() {
			if err := lecteur.Err(); err != nil {
				return noyau.Coup{}, err
			}
			return noyau.Coup{}, ErrQuitter
		}

		saisie := strings.TrimSpace(lecteur.Text())
		if saisie == "q" || saisie == "Q" {
			return noyau.Coup{}, ErrQuitter
		}

		n, err := strconv.Atoi(saisie)
		if err != nil || n < 1 || n > len(coups) {
			_, _ = fmt.Fprintf(sortie, "« %s » n'est pas un numéro de la liste.\n", saisie)
			continue
		}
		return coups[n-1], nil
	}
}
