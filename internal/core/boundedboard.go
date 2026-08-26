// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"errors"
	"fmt"
)

// Settings rassemble tout ce qui est réglable depuis l'interface. Ces valeurs
// sont enregistrées avec la partie : une sauvegarde reste rejouable même si les
// valeurs par défaut changent.
type Settings struct {
	Size                int `json:"size"`
	Range               int `json:"range"`
	Turns               int `json:"turns"`
	Stamina             int `json:"stamina"`
	Inspectors          int `json:"inspectors"`
	PiecesPerTurn       int `json:"pieces_per_turn"`
	RevealPeriod        int `json:"reveal_period"`
	Zones               int `json:"zones"`
	TrailLifetime       int `json:"trail_lifetime"`
	StranglingStart     int `json:"strangling_start"`
	PeriodeEtranglement int `json:"strangling_period"`
}

// DefaultSettings correspond au préréglage « Ville ». Les valeurs sont
// justifiées dans docs/regles.md ; celle qui bougera en premier est
// PiecesPerTurn.
func DefaultSettings() Settings {
	return Settings{
		Size:                41,
		Range:               8,
		Turns:               40,
		Stamina:             10,
		Inspectors:          5,
		PiecesPerTurn:       3,
		RevealPeriod:        4,
		Zones:               6,
		TrailLifetime:       6,
		StranglingStart:     30,
		PeriodeEtranglement: 2,
	}
}

// Bornes des paramètres, chacune avec ce qui la fixe.
const (
	// MinSize laisse la place au noyau central de départ et aux zones qui
	// l'entourent. En dessous, le fugitif naît à portée d'un point
	// d'extraction et la traversée qui fait le jeu disparaît.
	MinSize = 2*CentreRadius + ZoneSize + 2

	// MaxSize borne BoundedBoard, et lui seul.
	//
	// Cette implémentation précalcule la vision de chaque case : le coût suit
	// le nombre de rues multiplié par la portée, soit environ un mégaoctet et
	// demi à 41, six fois plus à 81. C'est la limite d'un plateau qu'on tient
	// entier en mémoire, pas celle du jeu.
	//
	// Une carte plus grande ne s'obtient pas en levant ce plafond mais en
	// écrivant une autre implémentation de Board, qui génère ses tuiles et
	// calcule la vision à la demande. L'interface est faite pour ça, et aucune
	// règle n'aurait à bouger — c'est le plateau infini prévu en v2.
	MaxSize = 81

	// MinRange est le minimum pour qu'un inspecteur voie autre chose que ses
	// voisines immédiates. Le maximum est relatif au plateau : voir la moitié
	// du terrain depuis un pion rendrait la fuite impossible.
	MinRange = 3
)

// Validate refuse les combinaisons qui rendent la partie injouable plutôt que de
// les laisser produire un plateau invalide ou une partie sans issue.
//
// Chaque refus dit la valeur reçue et ce qui était attendu : « côté hors
// bornes » oblige à open le code pour savoir lesquelles.
func (p Settings) Validate() error {
	switch {
	case p.Size < MinSize || p.Size > MaxSize:
		return fmt.Errorf("cote de %d, attendu entre %d et %d", p.Size, MinSize, MaxSize)
	case p.Zones <= p.Inspectors:
		return fmt.Errorf("%d zones pour %d inspecteurs : il en faut davantage, "+
			"sinon tout est couvrable", p.Zones, p.Inspectors)
	case p.Range < MinRange || p.Range > p.Size/2:
		return fmt.Errorf("portee de %d, attendu entre %d et %d sur un plateau de %d",
			p.Range, MinRange, p.Size/2, p.Size)
	case p.StranglingStart >= p.Turns:
		return fmt.Errorf("etranglement au tour %d pour une partie de %d tours : "+
			"il doit commencer avant la fin", p.StranglingStart, p.Turns)
	case p.PiecesPerTurn < 1 || p.PiecesPerTurn > p.Inspectors:
		return fmt.Errorf("%d pions deplacables pour %d inspecteurs",
			p.PiecesPerTurn, p.Inspectors)
	}
	return nil
}

// BoundedBoard est l'implémentation v1 : une grille dense, cases hors limites
// traitées comme bâtiments.
type BoundedBoard struct {
	graine int64
	cote   int
	rues   []bool
	zones  []Zone
	vues   map[Position][]Position
}

// La génération elle-même vit dans grid.go.

// validate vérifie qu'une seule composante connexe couvre toutes les rues, que
// chaque zone est atteignable et que le taux de rues tombe dans ses bornes.
//
// Un plateau qui échoue est jeté, jamais rapiécé : boucher un trou en ouvrant
// des cases au hasard donnerait un terrain que personne n'a dessiné, et dont on
// ne saurait plus dire s'il tient les critères pour de bon ou par accident.
func (b *BoundedBoard) validate(p Settings) error {
	rues := b.countStreets()
	if rues == 0 {
		return errors.New("plateau sans rue")
	}

	// Le taux se mesure en centièmes plutôt qu'en flottants : deux machines
	// doivent trancher pareil, et une comparaison de flottants au bord de la
	// fourchette est le genre de chose qui diverge.
	taux := rues * 100 / (b.cote * b.cote)
	if taux < MinStreetRatio || taux > MaxStreetRatio {
		return fmt.Errorf("taux de rues de %d %%, hors des bornes %d a %d",
			taux, MinStreetRatio, MaxStreetRatio)
	}

	atteintes := b.walk()
	if len(atteintes) != rues {
		return fmt.Errorf("%d rues isolees du reste du plateau", rues-len(atteintes))
	}

	for _, z := range b.zones {
		if !b.zoneReachable(z, atteintes) {
			return fmt.Errorf("la zone %d n'a aucune case praticable atteignable", z.Number)
		}
	}
	return nil
}

// countStreets totalise les cases praticables.
func (b *BoundedBoard) countStreets() int {
	n := 0
	for _, rue := range b.rues {
		if rue {
			n++
		}
	}
	return n
}

// walk rend les cases atteignables depuis la première rue, en n'empruntant
// que les orthogonales.
//
// Orthogonales et non diagonales : les inspecteurs ne se déplacent que
// ainsi, et un plateau où seul le fugitif circulerait ne serait pas jouable
// pour l'autre camp.
func (b *BoundedBoard) walk() map[Position]bool {
	depart, trouve := b.firstStreet()
	if !trouve {
		return nil
	}

	atteintes := map[Position]bool{depart: true}
	file := []Position{depart}
	for len(file) > 0 {
		courante := file[0]
		file = file[1:]

		for _, d := range Orthogonales {
			voisine := courante.Step(d)
			if b.IsStreet(voisine) && !atteintes[voisine] {
				atteintes[voisine] = true
				file = append(file, voisine)
			}
		}
	}
	return atteintes
}

// firstStreet rend la première case praticable, dans l'ordre de la grille.
func (b *BoundedBoard) firstStreet() (Position, bool) {
	for i, rue := range b.rues {
		if rue {
			return Position{Column: i % b.cote, Row: i / b.cote}, true
		}
	}
	return Position{}, false
}

// zoneReachable dit si l'on peut entrer dans une zone.
//
// Une seule case suffit : le fugitif doit pouvoir s'y tenir, pas la walk
// entière.
func (b *BoundedBoard) zoneReachable(z Zone, atteintes map[Position]bool) bool {
	for _, c := range z.Cells {
		if atteintes[c] {
			return true
		}
	}
	return false
}

// IsStreet dit si une case est praticable. Hors plateau vaut bâtiment, ce qui
// évite un test de bornes à chaque appelant.
func (b *BoundedBoard) IsStreet(p Position) bool {
	return b.inside(p) && b.rues[p.Row*b.cote+p.Column]
}

// inside dit si une position tombe sur le plateau.
//
// Les parcours de voisinage sortent des bords en permanence : c'est ce test qui
// leur évite d'avoir à s'en soucier.
func (b *BoundedBoard) inside(p Position) bool {
	return p.Column >= 0 && p.Row >= 0 && p.Column < b.cote && p.Row < b.cote
}

// CellsWithin énumère les rues autour d'un centre, ligne par ligne puis colonne
// par colonne. L'ordre est un contrat : il détermine celui des coups légaux, et
// donc ce qu'un rejeu doit retrouver.
func (b *BoundedBoard) CellsWithin(centre Position, rayon int) []Position {
	var cases []Position
	for ligne := centre.Row - rayon; ligne <= centre.Row+rayon; ligne++ {
		for colonne := centre.Column - rayon; colonne <= centre.Column+rayon; colonne++ {
			if p := (Position{Column: colonne, Row: ligne}); b.IsStreet(p) {
				cases = append(cases, p)
			}
		}
	}
	return cases
}

// Zones renvoie les zones d'extraction, connues des deux camps dès le début.
func (b *BoundedBoard) Zones() []Zone { return b.zones }

// Seed renvoie la graine dont ce plateau est issu.
func (b *BoundedBoard) Seed() int64 { return b.graine }

// Sight renvoie les cases vues depuis p, terrain seul. L'occlusion par les
// pions et les barrages s'applique à la lecture : la précalculer supposerait de
// tout recalculer à chaque déplacement.
//
// La table est calculée sans borne de portée, celle-ci s'appliquant ici. Elle
// se mesure en Tchebychev, comme les déplacements du fugitif : à portée 8, huit
// pas en diagonale sont dans la vue. Le prototype Python comptait en Manhattan
// et lui donnait de fait le double.
func (b *BoundedBoard) Sight(p Position, portee int) []Position {
	if portee <= 0 {
		return nil
	}

	var vues []Position
	for _, c := range b.vues[p] {
		if ChebyshevDistance(p, c) <= portee {
			vues = append(vues, c)
		}
	}
	return vues
}
