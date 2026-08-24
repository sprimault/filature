// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

// Position repère une case du plateau.
//
// L'origine est arbitraire : le plateau borné place (0,0) au coin nord-ouest,
// la génération par tuiles de la v2 acceptera des coordonnées négatives. Aucun
// code hors de generation.go ne doit supposer que les coordonnées sont
// positives ou bornées.
type Position struct {
	Colonne int
	Ligne   int
}

// Direction indexe les huit directions, dans le sens horaire à partir du nord.
// Les quatre premières sont les orthogonales, seules autorisées aux
// inspecteurs. Cet ordre est un contrat : la sérialisation du journal et les
// manifestes de greffons stockent la valeur numérique.
type Direction uint8

// Les huit directions, dans l'ordre que Direction fige.
const (
	Nord Direction = iota
	Est
	Sud
	Ouest
	NordEst
	SudEst
	SudOuest
	NordOuest
)

// Orthogonales et Diagonales découpent l'énumération selon ce qu'un camp a le
// droit d'emprunter.
var (
	Orthogonales = [4]Direction{Nord, Est, Sud, Ouest}
	Diagonales   = [4]Direction{NordEst, SudEst, SudOuest, NordOuest}
)

// deplacements donne le vecteur de chaque direction, indexé par Direction.
var deplacements = [8]Position{
	{0, -1}, {1, 0}, {0, 1}, {-1, 0},
	{1, -1}, {1, 1}, {-1, 1}, {-1, -1},
}

// Avance renvoie la case voisine dans la direction donnée, sans vérifier
// qu'elle est praticable : c'est le rôle de Partie.CoupsLegaux.
func (p Position) Avance(d Direction) Position {
	v := deplacements[d]
	return Position{p.Colonne + v.Colonne, p.Ligne + v.Ligne}
}

// EstDiagonale distingue les quatre directions réservées au fugitif.
func (d Direction) EstDiagonale() bool { return d >= NordEst }

// Contournement renvoie les deux cases orthogonales par lesquelles une
// diagonale passe. Un déplacement ou une ligne de vue diagonale exige qu'au
// moins l'une des deux soit une rue : sans cette règle, on traverse les angles
// de bâtiments et le bâti ne bloque plus rien.
func (p Position) Contournement(d Direction) (Position, Position) {
	v := deplacements[d]
	return Position{p.Colonne + v.Colonne, p.Ligne}, Position{p.Colonne, p.Ligne + v.Ligne}
}

// DistanceTchebychev est le nombre de tours minimal qu'il faut au fugitif pour
// relier deux cases en terrain dégagé, ses diagonales coûtant un tour comme les
// autres. Sert d'heuristique, jamais de règle.
func DistanceTchebychev(a, b Position) int {
	dc, dl := a.Colonne-b.Colonne, a.Ligne-b.Ligne
	if dc < 0 {
		dc = -dc
	}
	if dl < 0 {
		dl = -dl
	}
	if dc > dl {
		return dc
	}
	return dl
}
