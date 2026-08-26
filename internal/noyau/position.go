// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import (
	"strconv"
	"strings"
)

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

// Cle rend la position sous une forme utilisable en clé JSON.
//
// JSON n'accepte que des chaînes en clé d'objet, et une map indexée par
// Position ne s'y sérialise donc pas. Le format est un contrat : il part dans
// la vue que lisent l'interface, le réseau et les bots.
func (p Position) Cle() string {
	return strconv.Itoa(p.Colonne) + "," + strconv.Itoa(p.Ligne)
}

// PositionDepuisCle relit ce que Cle a écrit, et dit si la chaîne en était une.
//
// Sans elle, les tables de la vue indexées par position — les traces — ne se
// parcourent pas : leurs clés sont du texte, et retrouver la case demanderait à
// chaque lecteur de refaire le découpage à sa façon.
func PositionDepuisCle(cle string) (Position, bool) {
	colonne, ligne, coupee := strings.Cut(cle, ",")
	if !coupee {
		return Position{}, false
	}

	c, err := strconv.Atoi(colonne)
	if err != nil {
		return Position{}, false
	}
	l, err := strconv.Atoi(ligne)
	if err != nil {
		return Position{}, false
	}
	return Position{Colonne: c, Ligne: l}, true
}

// DirectionVers renvoie la direction d'un pas entre deux cases adjacentes, et
// dit si les cases le sont.
//
// Une trace porte la direction prise, et c'est ce qui la rend exploitable : un
// inspecteur qui en découvre une sait vers où chercher. La reconstituer depuis
// le journal évite de la stocker dans le coup, où elle serait redondante avec
// le départ et l'arrivée.
func DirectionVers(de, vers Position) (Direction, bool) {
	ecart := Position{Colonne: vers.Colonne - de.Colonne, Ligne: vers.Ligne - de.Ligne}
	for d, v := range deplacements {
		if v == ecart {
			return Direction(d), true
		}
	}
	return 0, false
}

// EstDiagonale distingue les quatre diagonales des quatre orthogonales.
//
// Le fugitif dispose des huit directions ; les inspecteurs n'ont que les
// orthogonales. C'est là qu'est sa vitesse : à distance égale en terrain
// dégagé, il arrive avant eux.
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
	return max(abs(a.Colonne-b.Colonne), abs(a.Ligne-b.Ligne))
}

// DistanceManhattan compte les pas orthogonaux entre deux cases.
//
// Les deux distances cohabitent et ne se remplacent pas : la portée de vue et
// les déplacements du fugitif se mesurent en Tchebychev, l'adjacence d'un
// contact en Manhattan. Un prototype antérieur les a confondues, et la portée 8
// des rayons en valait 16 côté fugitif — un défaut qui ne se voit pas en
// jouant, seulement en perdant.
func DistanceManhattan(a, b Position) int {
	return abs(a.Colonne-b.Colonne) + abs(a.Ligne-b.Ligne)
}

// abs renvoie la valeur absolue d'un entier.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// signe rend -1, 0 ou 1 selon le sens d'un écart.
func signe(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}
