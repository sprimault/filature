// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Package texte rend une vue de partie en caractères, et relit un coup saisi.
//
// Il ne connaît que core.View, jamais l'état complet : c'est la même contrainte
// que pour un bot ou pour le réseau, et elle vaut aussi en partie locale. Ce
// qu'un affichage ne peut pas montrer depuis une vue est un manque du contrat
// de vue, pas une raison d'aller lire ailleurs.
package text

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sprimault/filature/internal/core"
)

// Les caractères du plateau, du plus prioritaire au moins.
//
// Un pion couvre ce qu'il piétine : savoir qu'un inspecteur se tient sur une
// trace importe plus que la trace. Les zones passent en dernier parce qu'elles
// occupent neuf cases et masqueraient tout le reste.
const (
	CarBatiment   = '#'
	CarRue        = '.'
	CarFugitif    = 'F'
	CarBarrage    = 'X'
	CarScene      = '!'
	CarTrace      = '+'
	CarInspecteur = 'A' // A, B, C… selon le rang du pion
)

// PieceLetter nomme un inspecteur par son rang, A pour le premier.
//
// Bornée à l'alphabet : un rang au-delà ne peut venir que d'un coup mal formé —
// d'un bot, par exemple — et le dire en clair vaut mieux qu'un caractère
// aberrant dont personne ne saura d'où il sort.
func PieceLetter(rang int) string {
	const dernier = 'Z' - CarInspecteur
	if rang < 0 || rang > dernier {
		return fmt.Sprintf("pion %d", rang)
	}
	return string(rune(CarInspecteur + rang))
}

// Board dessine le terrain et ce que la vue en montre.
//
// Les lettres numérotent les inspecteurs dans l'ordre de la vue, qui est celui
// des coups légaux : « déplacer B » désigne sans ambiguïté le pion que l'écran
// montre en B.
func Board(v core.View) string {
	cote := v.Settings.Size
	grille := make([][]rune, cote)
	for l := range grille {
		grille[l] = make([]rune, cote)
		for c := range grille[l] {
			grille[l][c] = CarBatiment
		}
	}

	poser := func(p core.Position, car rune) {
		if p.Column >= 0 && p.Row >= 0 && p.Column < cote && p.Row < cote {
			grille[p.Row][p.Column] = car
		}
	}

	for _, p := range v.Streets {
		poser(p, CarRue)
	}
	for _, z := range v.Zones {
		for _, c := range z.Cells {
			poser(c, rune('0'+z.Number%10))
		}
	}
	for cle := range v.KnownTrails {
		if p, ok := core.PositionFromKey(cle); ok {
			poser(p, CarTrace)
		}
	}
	for _, s := range v.CrimeScenes {
		poser(s.Position, CarScene)
	}
	for _, p := range v.Roadblocks {
		poser(p, CarBarrage)
	}
	for i, insp := range v.Inspectors {
		poser(insp.Position, CarInspecteur+rune(i))
	}
	if v.PositionFugitif != nil {
		poser(*v.PositionFugitif, CarFugitif)
	}

	var sortie strings.Builder
	sortie.WriteString(header(cote))
	for l, ligne := range grille {
		fmt.Fprintf(&sortie, "%3d %s\n", l, string(ligne))
	}
	return sortie.String()
}

// header écrit les dizaines puis les unités des colonnes.
//
// Deux lignes plutôt qu'une : au-delà de dix colonnes, un seul chiffre ne situe
// plus rien, et un plateau de quarante et un de côté se lit à la colonne près
// quand on cherche pourquoi un coup est refusé.
func header(cote int) string {
	var dizaines, unites strings.Builder
	dizaines.WriteString("    ")
	unites.WriteString("    ")

	for c := 0; c < cote; c++ {
		if c >= 10 {
			fmt.Fprintf(&dizaines, "%d", c/10%10)
		} else {
			dizaines.WriteByte(' ')
		}
		fmt.Fprintf(&unites, "%d", c%10)
	}
	return dizaines.String() + "\n" + unites.String() + "\n"
}

// Status résume ce qui ne tient pas sur le plateau.
//
// La résistance n'apparaît que si la vue la porte : côté inspecteurs elle est
// absente, et afficher un zéro dirait que le fugitif est à bout au lieu de dire
// qu'on n'en sait rien.
func Status(v core.View) string {
	var s strings.Builder

	fmt.Fprintf(&s, "Tour %d", v.Turn)
	if v.Settings.Turns > 0 {
		fmt.Fprintf(&s, "/%d", v.Settings.Turns)
	}
	fmt.Fprintf(&s, " — %s — %s\n", phaseName(v.Phase), sideName(v.Side))

	if v.Stamina != nil {
		fmt.Fprintf(&s, "Résistance : %d\n", *v.Stamina)
	}
	if v.SealedZone != nil {
		fmt.Fprintf(&s, "Zone scellée : %d\n", *v.SealedZone)
	}

	if fermees := closedZones(v); fermees != "" {
		fmt.Fprintf(&s, "Zones fermées : %s\n", fermees)
	}
	if len(v.ZonesAnnoncees) > 0 {
		fmt.Fprintf(&s, "Zones annoncées : %s\n", numbers(v.ZonesAnnoncees))
	}
	if v.ProchaineReveal > 0 {
		fmt.Fprintf(&s, "Révélation dans %d tour(s)\n", v.ProchaineReveal)
	}
	if v.SilencePaye {
		s.WriteString("Le fugitif a payé le silence\n")
	}

	for i, insp := range v.Inspectors {
		fmt.Fprintf(&s, "%s en (%d,%d)", PieceLetter(i), insp.Position.Column, insp.Position.Row)
		if insp.Ability != "" {
			fmt.Fprintf(&s, ", %s", insp.Ability)
			if insp.AbilityUsed {
				s.WriteString(" (utilisée)")
			}
		}
		s.WriteString("\n")
	}

	return s.String()
}

// closedZones rend les zones dont la vue dit qu'elles le sont.
func closedZones(v core.View) string {
	var fermees []int
	for _, z := range v.Zones {
		if z.Closed {
			fermees = append(fermees, z.Number)
		}
	}
	return numbers(fermees)
}

// numbers joint des entiers par des espaces.
func numbers(n []int) string {
	if len(n) == 0 {
		return ""
	}
	textes := make([]string, len(n))
	for i, v := range n {
		textes[i] = fmt.Sprint(v)
	}
	return strings.Join(textes, " ")
}

// phase rend le nom d'une phase en français.
func phaseName(p core.Phase) string {
	switch p {
	case core.PhaseFugitiveSetup:
		return "le fugitif scelle sa zone"
	case core.PhaseInspectorsSetup:
		return "placement des inspecteurs"
	case core.PhaseInspectors:
		return "aux inspecteurs"
	case core.PhaseFugitive:
		return "au fugitif"
	case core.PhaseOver:
		return "partie terminée"
	default:
		return string(p)
	}
}

// camp rend le nom d'un camp en français.
func sideName(a core.Side) string {
	switch a {
	case core.SideFugitive:
		return "vue du fugitif"
	case core.SideInspectors:
		return "vue des inspecteurs"
	default:
		return "vue complète"
	}
}

// Ending rend le résultat d'une partie.
func Ending(r core.Outcome) string {
	motifs := map[string]string{
		core.OutcomeExtraction:   "le fugitif s'est extrait",
		core.OutcomeStaminaSpent: "le fugitif est à bout",
		core.OutcomeCornered:     "le fugitif est pris",
		core.OutcomeTimeUp:       "le temps est écoulé",
		core.OutcomePlugin:       "une règle de plugin y a mis fin",
	}
	motif := motifs[r.Reason]
	if motif == "" {
		motif = r.Reason
	}

	// Le vainqueur ne se nomme pas comme une vue : « les inspecteurs
	// l'emportent », pas « vue des inspecteurs ». Et l'accord suit — l'un est
	// seul, les autres sont cinq.
	vainqueur := "le fugitif l'emporte"
	if r.Winner == core.SideInspectors {
		vainqueur = "les inspecteurs l'emportent"
	}
	return fmt.Sprintf("Tour %d — %s : %s", r.Turn, vainqueur, motif)
}

// Merge superpose les deux vues d'une même partie, pour qui regarde sans
// play.
//
// Un spectateur doit voir ce qu'aucun des deux camps ne sait, mais le lui
// donner depuis l'état complet ouvrirait une porte sur la position cachée. La
// superposition des vues filtrées y suffit — et si quelque chose y manquait,
// ce serait un trou dans le contrat de vue, visible à l'écran plutôt que
// silencieux.
func Merge(fugitif, inspecteurs core.View) core.View {
	v := inspecteurs
	v.Side = ""

	v.PositionFugitif = fugitif.PositionFugitif
	v.SealedZone = fugitif.SealedZone
	v.Stamina = fugitif.Stamina
	v.LegalMoves = nil

	// Chaque camp connaît des traces que l'autre ignore : le fugitif les
	// siennes, les inspecteurs celles qu'ils ont découvertes.
	v.KnownTrails = map[string]core.Trail{}
	for cle, t := range inspecteurs.KnownTrails {
		v.KnownTrails[cle] = t
	}
	for cle, t := range fugitif.KnownTrails {
		v.KnownTrails[cle] = t
	}

	v.Streets = union(inspecteurs.Streets, fugitif.Streets)
	return v
}

// union rassemble deux listes de positions sans doublon, dans un ordre stable.
func union(a, b []core.Position) []core.Position {
	vues := make(map[core.Position]bool, len(a)+len(b))
	for _, p := range append(append([]core.Position{}, a...), b...) {
		vues[p] = true
	}

	toutes := make([]core.Position, 0, len(vues))
	for p := range vues {
		toutes = append(toutes, p)
	}
	sort.Slice(toutes, func(i, j int) bool {
		if toutes[i].Row != toutes[j].Row {
			return toutes[i].Row < toutes[j].Row
		}
		return toutes[i].Column < toutes[j].Column
	})
	return toutes
}
