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
