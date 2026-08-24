// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

// Alea est le seul générateur autorisé dans le noyau, l'IA et les greffons.
//
// L'entropie système et l'horloge sont hors d'atteinte partout ailleurs : une
// partie doit se rejouer à l'identique depuis son journal, sinon la reprise,
// le débogage et l'entraînement de l'IA tombent tous les trois. Un greffon qui
// veut de l'aléatoire passe par ici, alimenté par la graine de la partie.
type Alea struct {
	etat uint64
}

// NouvelAlea dérive un flux nommé de la graine de la partie. Les flux nommés
// évitent qu'ajouter un tirage dans la génération décale tous les tirages de
// l'IA.
func NouvelAlea(graine int64, flux string) *Alea {
	return nil
}

// Entier renvoie une valeur dans [0, n[.
func (a *Alea) Entier(n int) int { return 0 }

// Melanger permute une tranche sur place. À préférer systématiquement à un
// parcours de map, dont l'ordre n'est pas déterministe en Go.
func Melanger[T any](a *Alea, s []T) {}
