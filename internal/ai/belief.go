// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Package ia porte l'adversaire embarqué et le pilotage des bots externes.
package ai

import "github.com/sprimault/filature/internal/core"

// Croyance est une distribution de probabilité de présence du fugitif sur les
// cases de rue.
//
// Le jeu est à information imparfaite : un minimax classique ne s'applique pas.
// C'est un filtre bayésien discret, mis à jour par trois informations, dont la
// deuxième est de loin la plus riche :
//
//   - propagation : le fugitif a pu se déplace d'une case dans huit directions ;
//   - absence d'observation : toute case vue et vide passe à zéro ;
//   - révélation : tous les quatre tours, la distribution s'effondre sur une case.
//
// C'est cette troisième qui borne l'incertitude quelle que soit la taille du
// plateau, et qui rend l'approche utilisable même sans bords.
type Croyance struct {
	masses map[core.Position]float64
}

// NewBelief part de la connaissance initiale : le fugitif est quelque
// part dans le noyau central.
func NewBelief(v core.View) *Croyance {
	// à implémenter : étape 9
	return nil
}

// Propagate étale la masse d'un tour de déplacement.
func (c *Croyance) Propagate(v core.View) {
	// à implémenter : étape 9
}

// Observe annule la masse des cases vues et vides, et réinjecte autour des
// traces découvertes, pondérée par leur ancienneté.
func (c *Croyance) Observe(v core.View) {
	// à implémenter : étape 9
}

// Centroid sert d'heuristique de convergence. Il n'a de sens que si la masse
// est concentrée : sur une distribution bimodale, il désigne un point où le
// fugitif n'est justement pas.
func (c *Croyance) Centroid() core.Position {
	// à implémenter : étape 9
	return core.Position{}
}
