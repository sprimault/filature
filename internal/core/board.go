// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

// Board ne décrit que la topologie : ni pions, ni tours, ni traces.
//
// Ce découplage est la condition du plateau infini de la v2. Une
// implémentation qui génère ses tuiles à la demande se substitue ici sans
// qu'aucune règle change, à condition qu'aucun appelant n'itère sur « toutes
// les cases » — d'où l'absence de méthode de parcours global.
type Board interface {
	IsStreet(p Position) bool
	Zones() []Zone
	Shelters() []Shelter
	Seed() int64
	// Sight renvoie les cases visibles depuis p, table calculée à la
	// génération. Renvoie nil si p n'est pas une rue.
	Sight(p Position, portee int) []Position

	// CellsWithin énumère les rues à portée du centre, en distance de
	// Tchebychev, dans un ordre stable.
	//
	// Bornée par construction, et c'est ce qui la distingue d'un parcours
	// global : un plateau qui génère ses tuiles à la demande y répond aussi
	// bien qu'un plateau clos. C'est par elle que passent le placement des
	// inspecteurs et le tirage du fugitif dans le noyau central.
	CellsWithin(centre Position, rayon int) []Position
}

// Zone est un point d'extraction : un bloc dont au moins cinq cases sont des
// rues. Les six zones sont connues des deux camps dès la mise en place ; seul
// le choix du fugitif est caché.
type Zone struct {
	Number int
	Cells  []Position
	Closed bool
}

// Contains dit si une case appartient à la zone. Le nombre de cases est petit
// et fixe, la recherche linéaire est le bon compromis.
func (z Zone) Contains(p Position) bool {
	for _, c := range z.Cells {
		if c == p {
			return true
		}
	}
	return false
}

// Shelter est un lieu de ressourcement : un bloc au même format qu'une zone,
// dont au moins cinq cases sont des rues.
//
// Le même format, et c'est le point : un joueur n'a rien de neuf à apprendre,
// le générateur perce les deux de la même façon, et une case isolée se serait
// fermée avec un seul inspecteur — ce que le format de bloc interdit.
//
// L'état — actif ou en recharge — n'est pas ici mais dans Game : le plateau est
// en lecture seule, c'est la condition du plateau infini.
type Shelter struct {
	Number int
	Cells  []Position
}

// shelterAt rend le rang du lieu qui couvre une case, NoShelter s'il n'y en a
// pas.
//
// Point unique parce que la mise en place et la résolution de tour posent la
// même question : la première pour savoir si le fugitif naît sur un lieu, la
// seconde pour savoir s'il vient d'y entrer. Deux parcours séparés donneraient
// deux réponses le jour où un lieu cesserait d'être un bloc.
func shelterAt(b Board, at Position) int {
	abris := b.Shelters()
	for i := range abris {
		if abris[i].Contains(at) {
			return i
		}
	}
	return NoShelter
}

// Contains dit si une case appartient au lieu.
func (s Shelter) Contains(p Position) bool {
	for _, c := range s.Cells {
		if c == p {
			return true
		}
	}
	return false
}
