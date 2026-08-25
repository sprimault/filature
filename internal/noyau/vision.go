// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

// precalculerVision construit, pour chaque case de rue, la liste des cases
// visibles dans les huit directions, coupée au premier bâtiment.
//
// C'est ce qui rend l'IA utilisable : tester « le fugitif est-il vu » devient
// une lecture de table au lieu de huit parcours. Sur un plateau de 41 cases de
// côté, de l'ordre du mégaoctet et demi — environ sept cent cinquante rues,
// huit directions, seize cases de portée avec le bonus du Guetteur.
//
// C'est ce coût qui borne CoteMax, et lui seul : un plateau plus vaste demande
// une autre implémentation de Plateau, qui calcule à la demande au lieu de tout
// tenir en mémoire.
//
// La table est calculée pour la portée maximale atteignable — capacité du
// Guetteur comprise — et tronquée à la lecture. Recalculer à chaque
// changement de portée coûterait plus cher que stocker.
func precalculerVision(b *PlateauBorne, porteeMax int) map[Position][]Position {
	// à implémenter : étape 4
	return nil
}

// ligneDeVue déroule une direction depuis une case et s'arrête au premier
// bâtiment, au bord, ou à la portée.
//
// Les diagonales appliquent la même règle d'angle que les déplacements : un
// regard ne se faufile pas entre deux bâtiments en équerre.
func ligneDeVue(b *PlateauBorne, depart Position, d Direction, portee int) []Position {
	// à implémenter : étape 4
	return nil
}

// EstVisible dit si une case est vue depuis une autre.
//
// Trois choses coupent une ligne : un bâtiment, un barrage, et **un autre
// inspecteur**. Cette dernière règle punit l'alignement des pions et force la
// dispersion — cinq inspecteurs en file indienne ne voient qu'avec le premier.
//
// D'où la séparation avec precalculerVision : la table ne dépend que du
// terrain, qui ne bouge pas. L'occlusion par un pion ou un barrage s'applique
// ici, en tronquant la ligne à la première case occupée. Les précalculer
// ensemble supposerait de tout recalculer à chaque déplacement.
func EstVisible(b Plateau, depuis, cible Position, portee int, occupees map[Position]bool) bool {
	// à implémenter : étape 4
	return false
}
