// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import (
	"errors"
	"fmt"
)

// Parametres rassemble tout ce qui est réglable depuis l'interface. Ces valeurs
// sont enregistrées avec la partie : une sauvegarde reste rejouable même si les
// valeurs par défaut changent.
type Parametres struct {
	Cote                int `json:"cote"`
	Portee              int `json:"portee"`
	Tours               int `json:"tours"`
	Resistance          int `json:"resistance"`
	Inspecteurs         int `json:"inspecteurs"`
	PionsParTour        int `json:"pions_par_tour"`
	PeriodeRevelation   int `json:"periode_revelation"`
	Zones               int `json:"zones"`
	DureeTrace          int `json:"duree_trace"`
	DebutEtranglement   int `json:"debut_etranglement"`
	PeriodeEtranglement int `json:"periode_etranglement"`
}

// ParametresDefaut correspond au préréglage « Ville ». Les valeurs sont
// justifiées dans docs/regles.md ; celle qui bougera en premier est
// PionsParTour.
func ParametresDefaut() Parametres {
	return Parametres{
		Cote:                41,
		Portee:              8,
		Tours:               40,
		Resistance:          10,
		Inspecteurs:         5,
		PionsParTour:        3,
		PeriodeRevelation:   4,
		Zones:               6,
		DureeTrace:          6,
		DebutEtranglement:   30,
		PeriodeEtranglement: 2,
	}
}

// Bornes des paramètres, chacune avec ce qui la fixe.
const (
	// CoteMin laisse la place au noyau central de départ et aux zones qui
	// l'entourent. En dessous, le fugitif naît à portée d'un point
	// d'extraction et la traversée qui fait le jeu disparaît.
	CoteMin = 2*RayonNoyauCentral + CoteZone + 2

	// CoteMax borne PlateauBorne, et lui seul.
	//
	// Cette implémentation précalcule la vision de chaque case : le coût suit
	// le nombre de rues multiplié par la portée, soit environ un mégaoctet et
	// demi à 41, six fois plus à 81. C'est la limite d'un plateau qu'on tient
	// entier en mémoire, pas celle du jeu.
	//
	// Une carte plus grande ne s'obtient pas en levant ce plafond mais en
	// écrivant une autre implémentation de Plateau, qui génère ses tuiles et
	// calcule la vision à la demande. L'interface est faite pour ça, et aucune
	// règle n'aurait à bouger — c'est le plateau infini prévu en v2.
	CoteMax = 81

	// PorteeMin est le minimum pour qu'un inspecteur voie autre chose que ses
	// voisines immédiates. Le maximum est relatif au plateau : voir la moitié
	// du terrain depuis un pion rendrait la fuite impossible.
	PorteeMin = 3
)

// Valider refuse les combinaisons qui rendent la partie injouable plutôt que de
// les laisser produire un plateau invalide ou une partie sans issue.
//
// Chaque refus dit la valeur reçue et ce qui était attendu : « côté hors
// bornes » oblige à ouvrir le code pour savoir lesquelles.
func (p Parametres) Valider() error {
	switch {
	case p.Cote < CoteMin || p.Cote > CoteMax:
		return fmt.Errorf("cote de %d, attendu entre %d et %d", p.Cote, CoteMin, CoteMax)
	case p.Zones <= p.Inspecteurs:
		return fmt.Errorf("%d zones pour %d inspecteurs : il en faut davantage, "+
			"sinon tout est couvrable", p.Zones, p.Inspecteurs)
	case p.Portee < PorteeMin || p.Portee > p.Cote/2:
		return fmt.Errorf("portee de %d, attendu entre %d et %d sur un plateau de %d",
			p.Portee, PorteeMin, p.Cote/2, p.Cote)
	case p.DebutEtranglement >= p.Tours:
		return fmt.Errorf("etranglement au tour %d pour une partie de %d tours : "+
			"il doit commencer avant la fin", p.DebutEtranglement, p.Tours)
	case p.PionsParTour < 1 || p.PionsParTour > p.Inspecteurs:
		return fmt.Errorf("%d pions deplacables pour %d inspecteurs",
			p.PionsParTour, p.Inspecteurs)
	}
	return nil
}

// PlateauBorne est l'implémentation v1 : une grille dense, cases hors limites
// traitées comme bâtiments.
type PlateauBorne struct {
	graine int64
	cote   int
	rues   []bool
	zones  []Zone
	vues   map[Position][]Position
}

// La génération elle-même vit dans trame.go.

// valider vérifie qu'une seule composante connexe couvre toutes les rues, que
// chaque zone est atteignable et que le taux de rues tombe dans ses bornes.
//
// Un plateau qui échoue est jeté, jamais rapiécé : boucher un trou en ouvrant
// des cases au hasard donnerait un terrain que personne n'a dessiné, et dont on
// ne saurait plus dire s'il tient les critères pour de bon ou par accident.
func (b *PlateauBorne) valider(p Parametres) error {
	rues := b.compterRues()
	if rues == 0 {
		return errors.New("plateau sans rue")
	}

	// Le taux se mesure en centièmes plutôt qu'en flottants : deux machines
	// doivent trancher pareil, et une comparaison de flottants au bord de la
	// fourchette est le genre de chose qui diverge.
	taux := rues * 100 / (b.cote * b.cote)
	if taux < TauxRuesMin || taux > TauxRuesMax {
		return fmt.Errorf("taux de rues de %d %%, hors des bornes %d a %d",
			taux, TauxRuesMin, TauxRuesMax)
	}

	atteintes := b.parcourir()
	if len(atteintes) != rues {
		return fmt.Errorf("%d rues isolees du reste du plateau", rues-len(atteintes))
	}

	for _, z := range b.zones {
		if !b.zoneAtteignable(z, atteintes) {
			return fmt.Errorf("la zone %d n'a aucune case praticable atteignable", z.Numero)
		}
	}
	return nil
}

// compterRues totalise les cases praticables.
func (b *PlateauBorne) compterRues() int {
	n := 0
	for _, rue := range b.rues {
		if rue {
			n++
		}
	}
	return n
}

// parcourir rend les cases atteignables depuis la première rue, en n'empruntant
// que les orthogonales.
//
// Orthogonales et non diagonales : les inspecteurs ne se déplacent que
// ainsi, et un plateau où seul le fugitif circulerait ne serait pas jouable
// pour l'autre camp.
func (b *PlateauBorne) parcourir() map[Position]bool {
	depart, trouve := b.premiereRue()
	if !trouve {
		return nil
	}

	atteintes := map[Position]bool{depart: true}
	file := []Position{depart}
	for len(file) > 0 {
		courante := file[0]
		file = file[1:]

		for _, d := range Orthogonales {
			voisine := courante.Avance(d)
			if b.EstRue(voisine) && !atteintes[voisine] {
				atteintes[voisine] = true
				file = append(file, voisine)
			}
		}
	}
	return atteintes
}

// premiereRue rend la première case praticable, dans l'ordre de la grille.
func (b *PlateauBorne) premiereRue() (Position, bool) {
	for i, rue := range b.rues {
		if rue {
			return Position{Colonne: i % b.cote, Ligne: i / b.cote}, true
		}
	}
	return Position{}, false
}

// zoneAtteignable dit si l'on peut entrer dans une zone.
//
// Une seule case suffit : le fugitif doit pouvoir s'y tenir, pas la parcourir
// entière.
func (b *PlateauBorne) zoneAtteignable(z Zone, atteintes map[Position]bool) bool {
	for _, c := range z.Cases {
		if atteintes[c] {
			return true
		}
	}
	return false
}

// EstRue dit si une case est praticable. Hors plateau vaut bâtiment, ce qui
// évite un test de bornes à chaque appelant.
func (b *PlateauBorne) EstRue(p Position) bool {
	return b.dedans(p) && b.rues[p.Ligne*b.cote+p.Colonne]
}

// dedans dit si une position tombe sur le plateau.
//
// Les parcours de voisinage sortent des bords en permanence : c'est ce test qui
// leur évite d'avoir à s'en soucier.
func (b *PlateauBorne) dedans(p Position) bool {
	return p.Colonne >= 0 && p.Ligne >= 0 && p.Colonne < b.cote && p.Ligne < b.cote
}

// CasesDans énumère les rues autour d'un centre, ligne par ligne puis colonne
// par colonne. L'ordre est un contrat : il détermine celui des coups légaux, et
// donc ce qu'un rejeu doit retrouver.
func (b *PlateauBorne) CasesDans(centre Position, rayon int) []Position {
	var cases []Position
	for ligne := centre.Ligne - rayon; ligne <= centre.Ligne+rayon; ligne++ {
		for colonne := centre.Colonne - rayon; colonne <= centre.Colonne+rayon; colonne++ {
			if p := (Position{Colonne: colonne, Ligne: ligne}); b.EstRue(p) {
				cases = append(cases, p)
			}
		}
	}
	return cases
}

// Zones renvoie les zones d'extraction, connues des deux camps dès le début.
func (b *PlateauBorne) Zones() []Zone { return b.zones }

// Graine renvoie la graine dont ce plateau est issu.
func (b *PlateauBorne) Graine() int64 { return b.graine }

// Vision renvoie les cases vues depuis p, terrain seul. L'occlusion par les
// pions et les barrages s'applique à la lecture : la précalculer supposerait de
// tout recalculer à chaque déplacement.
//
// La table est calculée sans borne de portée, celle-ci s'appliquant ici. Elle
// se mesure en Tchebychev, comme les déplacements du fugitif : à portée 8, huit
// pas en diagonale sont dans la vue. Le prototype Python comptait en Manhattan
// et lui donnait de fait le double.
func (b *PlateauBorne) Vision(p Position, portee int) []Position {
	if portee <= 0 {
		return nil
	}

	var vues []Position
	for _, c := range b.vues[p] {
		if DistanceTchebychev(p, c) <= portee {
			vues = append(vues, c)
		}
	}
	return vues
}
