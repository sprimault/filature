// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

// Plateau ne décrit que la topologie : ni pions, ni tours, ni traces.
//
// Ce découplage est la condition du plateau infini de la v2. Une
// implémentation qui génère ses tuiles à la demande se substitue ici sans
// qu'aucune règle change, à condition qu'aucun appelant n'itère sur « toutes
// les cases » — d'où l'absence de méthode de parcours global.
type Plateau interface {
	EstRue(p Position) bool
	Zones() []Zone
	Graine() int64
	// Vision renvoie les cases visibles depuis p, table calculée à la
	// génération. Renvoie nil si p n'est pas une rue.
	Vision(p Position, portee int) []Position
}

// Zone est un point d'extraction : un bloc dont au moins cinq cases sont des
// rues. Les six zones sont connues des deux camps dès la mise en place ; seul
// le choix du fugitif est caché.
type Zone struct {
	Numero int
	Cases  []Position
	Fermee bool
}

// Contient dit si une case appartient à la zone. Le nombre de cases est petit
// et fixe, la recherche linéaire est le bon compromis.
func (z Zone) Contient(p Position) bool {
	for _, c := range z.Cases {
		if c == p {
			return true
		}
	}
	return false
}
