// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import "math"

// Random est le seul générateur autorisé dans le noyau, l'IA et les plugins.
//
// L'entropie système et l'horloge sont hors d'atteinte partout ailleurs : une
// partie doit se rejouer à l'identique depuis son journal, sinon la reprise,
// le débogage et l'entraînement de l'IA tombent tous les trois. Un plugin qui
// veut de l'aléatoire passe par ici, alimenté par la graine de la partie.
//
// L'algorithme est splitmix64 : un état de soixante-quatre bits, aucune table,
// aucune dépendance. Sa qualité statistique dépasse de loin ce qu'un jeu de
// plateau demande, et sa brièveté est ce qui compte ici — le générateur doit
// donner la même suite sur toutes les cibles, y compris WebAssembly, et un code
// qu'on relit en entier est un code dont on peut l'affirmer.
type Random struct {
	etat uint64
}

// Constantes de splitmix64. Elles ne se choisissent pas : les changer donnerait
// un autre générateur, et périmerait toutes les parties enregistrées.
const (
	aleaIncrement = 0x9E3779B97F4A7C15
	aleaMelange1  = 0xBF58476D1CE4E5B9
	aleaMelange2  = 0x94D049BB133111EB
)

// NewRandom dérive un flux nommé de la graine de la partie.
//
// Les flux nommés évitent qu'ajouter un tirage dans la génération décale tous
// les tirages de l'IA : deux noms différents partent d'états sans rapport, et
// consommer l'un ne touche pas l'autre. Le nom est haché en FNV-1a, puis mêlé à
// la graine.
func NewRandom(graine int64, flux string) *Random {
	const (
		fnvBase    = 14695981039346656037
		fnvPremier = 1099511628211
	)

	h := uint64(fnvBase)
	for i := 0; i < len(flux); i++ {
		h ^= uint64(flux[i])
		h *= fnvPremier
	}
	// La conversion cherche le motif de bits, pas la valeur : une graine
	// négative doit donner un état comme un autre, et c'est précisément le
	// débordement qui le garantit.
	return &Random{etat: h ^ uint64(graine)} // #nosec G115
}

// Int renvoie une valeur dans [0, n[.
//
// Un n nul ou négatif renvoie zéro plutôt que de paniquer : le noyau ne
// s'arrête pas sur une erreur d'appel, et un plugin ne doit pas pouvoir faire
// tomber la partie en demandant un tirage vide.
func (a *Random) Int(n int) int {
	if n <= 0 {
		return 0
	}

	// Le reste d'une division sur toute la plage favoriserait les petites
	// valeurs. Le biais est minuscule, mais l'équilibrage compare des milliers
	// de parties : il s'y verrait, et on le chercherait ailleurs. On rejette
	// donc les tirages du dernier bloc incomplet.
	// Le seuil vaut 2^64 mod n, écrit sans passer par un négatif converti.
	borne := uint64(n)
	seuil := (math.MaxUint64 - borne + 1) % borne
	for {
		if v := a.suivant(); v >= seuil {
			// Le reste est inférieur à borne, elle-même issue d'un int :
			// la valeur tient donc dans un int par construction.
			return int(v % borne) // #nosec G115
		}
	}
}

// suivant avance l'état et renvoie soixante-quatre bits mélangés.
func (a *Random) suivant() uint64 {
	a.etat += aleaIncrement
	z := a.etat
	z = (z ^ (z >> 30)) * aleaMelange1
	z = (z ^ (z >> 27)) * aleaMelange2
	return z ^ (z >> 31)
}

// Shuffle permute une tranche sur place, par Fisher-Yates.
//
// À préférer systématiquement à un parcours de map, dont l'ordre n'est pas
// déterministe en Go : c'est la façon la plus courante de faire diverger un
// rejeu sans que rien ne le signale.
func Shuffle[T any](a *Random, s []T) {
	for i := len(s) - 1; i > 0; i-- {
		j := a.Int(i + 1)
		s[i], s[j] = s[j], s[i]
	}
}
