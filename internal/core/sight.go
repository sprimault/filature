// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

// precomputeSight construit, pour chaque case de rue, la liste des cases
// visibles dans les huit directions, coupée au premier bâtiment.
//
// C'est ce qui rend l'affichage et l'IA tenables : énumérer ce qu'un camp
// couvre se fait à chaque tour et pour chaque pion, et le recalculer à la
// demande reviendrait à dérouler quarante lignes par pion et par tour.
// IsVisible, lui, ne s'en sert pas : une question portant sur une seule case
// se répond plus vite en déroulant sa ligne qu'en filtrant la table.
//
// La table ne dépend que du terrain, et n'est donc bornée par aucune portée. La
// borner supposerait de connaître le plus gros bonus qu'un plugin puisse
// accorder, ce que le noyau ignore — et une table trop courte tronquerait la
// vue sans rien dire. C'est la lecture qui coupe à la portée du pion.
//
// Ne pas la borner coûte moins qu'il n'y paraît : le bâti coupe les lignes bien
// avant la portée, quarante-trois positions par rue en moyenne sur une Ville.
// Environ six cents kilooctets pour un plateau de quarante et un de côté, quatre
// mégaoctets à MaxSize. C'est ce coût qui borne MaxSize, et lui seul : un
// plateau plus vaste demande une autre implémentation de Board, qui calcule à
// la demande au lieu de tout tenir en mémoire.
func precomputeSight(b *BoundedBoard, porteeMax int) map[Position][]Position {
	vues := make(map[Position][]Position)

	for ligne := 0; ligne < b.cote; ligne++ {
		for colonne := 0; colonne < b.cote; colonne++ {
			p := Position{Column: colonne, Row: ligne}
			if !b.IsStreet(p) {
				continue
			}

			var cases []Position
			for d := Nord; d <= NordOuest; d++ {
				cases = append(cases, lineOfSight(b, p, d, porteeMax)...)
			}
			if cases != nil {
				vues[p] = cases
			}
		}
	}
	return vues
}

// lineOfSight déroule une direction depuis une case et s'arrête au premier
// bâtiment, au bord, ou à la portée.
//
// Les diagonales appliquent la même règle d'angle que les déplacements : un
// regard ne se faufile pas entre deux bâtiments en équerre.
//
// La case de départ n'est pas dans le résultat. Elle n'apprend rien à qui la
// regarde depuis elle-même, et l'y mettre huit fois — une par direction —
// obligerait tous les appelants à s'en défendre.
func lineOfSight(b Board, depart Position, d Direction, portee int) []Position {
	var ligne []Position

	courante := depart
	for pas := 0; pas < portee; pas++ {
		suivante := courante.Step(d)
		if !b.IsStreet(suivante) {
			return ligne
		}
		if d.IsDiagonal() {
			if g, h := courante.CornerCells(d); !b.IsStreet(g) && !b.IsStreet(h) {
				return ligne
			}
		}
		ligne = append(ligne, suivante)
		courante = suivante
	}
	return ligne
}

// IsVisible dit si une case est vue depuis une autre.
//
// Trois choses coupent une ligne : un bâtiment, un barrage, et **un autre
// inspecteur**. Cette dernière règle punit l'alignment des pions et force la
// dispersion — cinq inspecteurs en file indienne ne voient qu'avec le premier.
//
// D'où la séparation avec precomputeSight : la table ne dépend que du
// terrain, qui ne bouge pas. L'occlusion par un pion ou un barrage s'applique
// ici, en tronquant la ligne à la première case occupée. Les précalculer
// ensemble supposerait de tout recalculer à chaque déplacement.
//
// Une case occupée qui est elle-même la cible reste vue : ce qui arrête le
// regard, c'est ce qui se tient devant, pas ce qu'on regarde.
func IsVisible(b Board, depuis, cible Position, portee int, occupees map[Position]bool) bool {
	d, aligne := alignment(depuis, cible)
	if !aligne {
		return false
	}

	for _, c := range lineOfSight(b, depuis, d, portee) {
		if c == cible {
			return true
		}
		if occupees[c] {
			return false
		}
	}
	return false
}

// alignment rend la direction menant d'une case à l'autre quand les deux
// partagent une ligne de vue, et dit si elles la partagent.
//
// Les huit directions sont rectilignes : deux cases se voient si elles sont sur
// une même ligne, une même colonne, ou une diagonale exacte. Une case en pas de
// cavalier reste invisible quelle que soit la portée, et une case n'est jamais
// alignée avec elle-même.
func alignment(de, vers Position) (Direction, bool) {
	dc, dl := vers.Column-de.Column, vers.Row-de.Row
	if dc == 0 && dl == 0 {
		return 0, false
	}
	if dc != 0 && dl != 0 && abs(dc) != abs(dl) {
		return 0, false
	}
	return DirectionTo(de, Position{Column: de.Column + sign(dc), Row: de.Row + sign(dl)})
}
