// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Package texte rend une vue de partie en caractères, et relit un coup saisi.
//
// Il ne connaît que noyau.Vue, jamais l'état complet : c'est la même contrainte
// que pour un bot ou pour le réseau, et elle vaut aussi en partie locale. Ce
// qu'un affichage ne peut pas montrer depuis une vue est un manque du contrat
// de vue, pas une raison d'aller lire ailleurs.
package texte

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sprimault/filature/internal/noyau"
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

// LettrePion nomme un inspecteur par son rang, A pour le premier.
//
// Bornée à l'alphabet : un rang au-delà ne peut venir que d'un coup mal formé —
// d'un bot, par exemple — et le dire en clair vaut mieux qu'un caractère
// aberrant dont personne ne saura d'où il sort.
func LettrePion(rang int) string {
	const dernier = 'Z' - CarInspecteur
	if rang < 0 || rang > dernier {
		return fmt.Sprintf("pion %d", rang)
	}
	return string(rune(CarInspecteur + rang))
}

// Plateau dessine le terrain et ce que la vue en montre.
//
// Les lettres numérotent les inspecteurs dans l'ordre de la vue, qui est celui
// des coups légaux : « déplacer B » désigne sans ambiguïté le pion que l'écran
// montre en B.
func Plateau(v noyau.Vue) string {
	cote := v.Parametres.Cote
	grille := make([][]rune, cote)
	for l := range grille {
		grille[l] = make([]rune, cote)
		for c := range grille[l] {
			grille[l][c] = CarBatiment
		}
	}

	poser := func(p noyau.Position, car rune) {
		if p.Colonne >= 0 && p.Ligne >= 0 && p.Colonne < cote && p.Ligne < cote {
			grille[p.Ligne][p.Colonne] = car
		}
	}

	for _, p := range v.Rues {
		poser(p, CarRue)
	}
	for _, z := range v.Zones {
		for _, c := range z.Cases {
			poser(c, rune('0'+z.Numero%10))
		}
	}
	for cle := range v.TracesConnues {
		if p, ok := noyau.PositionDepuisCle(cle); ok {
			poser(p, CarTrace)
		}
	}
	for _, s := range v.Scenes {
		poser(s.Position, CarScene)
	}
	for _, p := range v.Barrages {
		poser(p, CarBarrage)
	}
	for i, insp := range v.Inspecteurs {
		poser(insp.Position, CarInspecteur+rune(i))
	}
	if v.PositionFugitif != nil {
		poser(*v.PositionFugitif, CarFugitif)
	}

	var sortie strings.Builder
	sortie.WriteString(enTete(cote))
	for l, ligne := range grille {
		fmt.Fprintf(&sortie, "%3d %s\n", l, string(ligne))
	}
	return sortie.String()
}

// enTete écrit les dizaines puis les unités des colonnes.
//
// Deux lignes plutôt qu'une : au-delà de dix colonnes, un seul chiffre ne situe
// plus rien, et un plateau de quarante et un de côté se lit à la colonne près
// quand on cherche pourquoi un coup est refusé.
func enTete(cote int) string {
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

// Etat résume ce qui ne tient pas sur le plateau.
//
// La résistance n'apparaît que si la vue la porte : côté inspecteurs elle est
// absente, et afficher un zéro dirait que le fugitif est à bout au lieu de dire
// qu'on n'en sait rien.
func Etat(v noyau.Vue) string {
	var s strings.Builder

	fmt.Fprintf(&s, "Tour %d", v.Tour)
	if v.Parametres.Tours > 0 {
		fmt.Fprintf(&s, "/%d", v.Parametres.Tours)
	}
	fmt.Fprintf(&s, " — %s — %s\n", phase(v.Phase), camp(v.Acteur))

	if v.Resistance != nil {
		fmt.Fprintf(&s, "Résistance : %d\n", *v.Resistance)
	}
	if v.ZoneScellee != nil {
		fmt.Fprintf(&s, "Zone scellée : %d\n", *v.ZoneScellee)
	}

	if fermees := zonesFermees(v); fermees != "" {
		fmt.Fprintf(&s, "Zones fermées : %s\n", fermees)
	}
	if len(v.ZonesAnnoncees) > 0 {
		fmt.Fprintf(&s, "Zones annoncées : %s\n", nombres(v.ZonesAnnoncees))
	}
	if v.ProchaineReveal > 0 {
		fmt.Fprintf(&s, "Révélation dans %d tour(s)\n", v.ProchaineReveal)
	}
	if v.SilencePaye {
		s.WriteString("Le fugitif a payé le silence\n")
	}

	for i, insp := range v.Inspecteurs {
		fmt.Fprintf(&s, "%s en (%d,%d)", LettrePion(i), insp.Position.Colonne, insp.Position.Ligne)
		if insp.Capacite != "" {
			fmt.Fprintf(&s, ", %s", insp.Capacite)
			if insp.CapaciteUtilisee {
				s.WriteString(" (utilisée)")
			}
		}
		s.WriteString("\n")
	}

	return s.String()
}

// zonesFermees rend les zones dont la vue dit qu'elles le sont.
func zonesFermees(v noyau.Vue) string {
	var fermees []int
	for _, z := range v.Zones {
		if z.Fermee {
			fermees = append(fermees, z.Numero)
		}
	}
	return nombres(fermees)
}

// nombres joint des entiers par des espaces.
func nombres(n []int) string {
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
func phase(p noyau.Phase) string {
	switch p {
	case noyau.PhasePlacementFugitif:
		return "le fugitif scelle sa zone"
	case noyau.PhasePlacementInspecteurs:
		return "placement des inspecteurs"
	case noyau.PhaseInspecteurs:
		return "aux inspecteurs"
	case noyau.PhaseFugitif:
		return "au fugitif"
	case noyau.PhaseTerminee:
		return "partie terminée"
	default:
		return string(p)
	}
}

// camp rend le nom d'un camp en français.
func camp(a noyau.Acteur) string {
	switch a {
	case noyau.CampFugitif:
		return "vue du fugitif"
	case noyau.CampInspecteurs:
		return "vue des inspecteurs"
	default:
		return "vue complète"
	}
}

// Fin rend le résultat d'une partie.
func Fin(r noyau.Resultat) string {
	motifs := map[string]string{
		noyau.MotifExtraction:  "le fugitif s'est extrait",
		noyau.MotifResistance:  "le fugitif est à bout",
		noyau.MotifBlocage:     "le fugitif est pris",
		noyau.MotifTempsEcoule: "le temps est écoulé",
		noyau.MotifGreffon:     "une règle de greffon y a mis fin",
	}
	motif := motifs[r.Motif]
	if motif == "" {
		motif = r.Motif
	}
	return fmt.Sprintf("Tour %d — %s l'emportent : %s", r.Tour, camp(r.Vainqueur), motif)
}

// Fusionner superpose les deux vues d'une même partie, pour qui regarde sans
// jouer.
//
// Un spectateur doit voir ce qu'aucun des deux camps ne sait, mais le lui
// donner depuis l'état complet ouvrirait une porte sur la position cachée. La
// superposition des vues filtrées y suffit — et si quelque chose y manquait,
// ce serait un trou dans le contrat de vue, visible à l'écran plutôt que
// silencieux.
func Fusionner(fugitif, inspecteurs noyau.Vue) noyau.Vue {
	v := inspecteurs
	v.Acteur = ""

	v.PositionFugitif = fugitif.PositionFugitif
	v.ZoneScellee = fugitif.ZoneScellee
	v.Resistance = fugitif.Resistance
	v.CoupsLegaux = nil

	// Chaque camp connaît des traces que l'autre ignore : le fugitif les
	// siennes, les inspecteurs celles qu'ils ont découvertes.
	v.TracesConnues = map[string]noyau.Trace{}
	for cle, t := range inspecteurs.TracesConnues {
		v.TracesConnues[cle] = t
	}
	for cle, t := range fugitif.TracesConnues {
		v.TracesConnues[cle] = t
	}

	v.Rues = union(inspecteurs.Rues, fugitif.Rues)
	return v
}

// union rassemble deux listes de positions sans doublon, dans un ordre stable.
func union(a, b []noyau.Position) []noyau.Position {
	vues := make(map[noyau.Position]bool, len(a)+len(b))
	for _, p := range append(append([]noyau.Position{}, a...), b...) {
		vues[p] = true
	}

	toutes := make([]noyau.Position, 0, len(vues))
	for p := range vues {
		toutes = append(toutes, p)
	}
	sort.Slice(toutes, func(i, j int) bool {
		if toutes[i].Ligne != toutes[j].Ligne {
			return toutes[i].Ligne < toutes[j].Ligne
		}
		return toutes[i].Colonne < toutes[j].Colonne
	})
	return toutes
}
