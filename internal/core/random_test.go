// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"math"
	"testing"
)

// TestSameSeedSameSequence est l'invariant de déterminisme.
//
// Deux générateurs partis de la même graine et du même flux donnent la même
// suite, sinon un rejeu diverge de son journal et la reprise, le débogage et
// l'entraînement de l'IA tombent ensemble.
func TestSameSeedSameSequence(t *testing.T) {
	a := NewRandom(178342119, "board")
	b := NewRandom(178342119, "board")

	for i := 0; i < 1000; i++ {
		if x, y := a.Int(1000), b.Int(1000); x != y {
			t.Fatalf("tirage %d : %d puis %d", i, x, y)
		}
	}
}

// TestReferenceSequence fige ce que le générateur doit rendre, pour toujours.
//
// Les autres tests vérifient qu'il est cohérent avec lui-même : ils passeraient
// encore si l'algorithme changeait. Celui-ci est le seul qui s'y oppose, et
// c'est ce qu'on veut — une suite différente périmerait toutes les parties
// enregistrées, tous les plateaux de référence et toute comparaison d'IA.
//
// Ces valeurs ont été relevées sous Windows et vérifiées identiques sous
// Linux/amd64 : le générateur ne dépend ni de la plateforme, ni de la taille
// d'un int.
//
// Les noms de flux qui suivent sont des entrées figées, et non ceux que le jeu
// emploie : ils entrent dans le hachage, donc les renommer changerait les
// valeurs attendues. Les régénérer ferait dire au test que le code actuel se
// comporte comme lui-même, ce qui est vrai de n'importe quel code.
func TestReferenceSequence(t *testing.T) {
	cas := []struct {
		flux    string
		attendu []int
	}{
		{"plateau", []int{276, 153, 191, 631, 908, 681, 809, 801}},
		{"ia", []int{391, 551, 161, 480, 683, 275, 78, 696}},
	}

	for _, c := range cas {
		t.Run(c.flux, func(t *testing.T) {
			a := NewRandom(178342119, c.flux)
			for i, attendu := range c.attendu {
				if got := a.Int(1000); got != attendu {
					t.Fatalf("tirage %d : %d, attendu %d", i, got, attendu)
				}
			}
		})
	}

	melange := []int{0, 1, 2, 3, 4, 5}
	Shuffle(NewRandom(99, "etranglement"), melange)
	for i, attendu := range []int{3, 5, 0, 1, 4, 2} {
		if melange[i] != attendu {
			t.Fatalf("mélange %v, attendu [3 5 0 1 4 2]", melange)
		}
	}
}

// TestStreamsAreIndependent vérifie ce pour quoi les flux nommés existent.
//
// Ajouter un tirage dans la génération du plateau ne doit pas décaler ceux de
// l'IA. Deux flux consommés inégalement continuent donc chacun sur sa suite.
func TestStreamsAreIndependent(t *testing.T) {
	plateau := NewRandom(42, "board")
	ia := NewRandom(42, "ai")
	temoin := NewRandom(42, "ai")

	// Le plateau consomme cent tirages, l'IA aucun.
	for i := 0; i < 100; i++ {
		plateau.Int(6)
	}

	for i := 0; i < 50; i++ {
		if x, y := ia.Int(6), temoin.Int(6); x != y {
			t.Fatalf("le flux ia a été décalé au tirage %d : %d contre %d", i, x, y)
		}
	}
}

// TestDifferentStreamsDifferentSequences vérifie l'autre moitié : deux
// noms ne doivent pas retomber sur la même suite.
func TestDifferentStreamsDifferentSequences(t *testing.T) {
	a := NewRandom(7, "board")
	b := NewRandom(7, "ai")

	identiques := 0
	for i := 0; i < 100; i++ {
		if a.Int(1000) == b.Int(1000) {
			identiques++
		}
	}
	// Sur mille valeurs, une poignée de coïncidences est normale ; une suite
	// identique ne l'est pas.
	if identiques > 10 {
		t.Errorf("%d tirages identiques sur 100 : les flux se confondent", identiques)
	}
}

// TestDifferentSeedsDifferentSequences vérifie que la graine porte
// bien la partie.
func TestDifferentSeedsDifferentSequences(t *testing.T) {
	a := NewRandom(1, "board")
	b := NewRandom(2, "board")

	identiques := 0
	for i := 0; i < 100; i++ {
		if a.Int(1000) == b.Int(1000) {
			identiques++
		}
	}
	if identiques > 10 {
		t.Errorf("%d tirages identiques sur 100 : la graine ne sépare pas", identiques)
	}
}

// TestIntStaysWithinBounds vérifie qu'aucun tirage ne sort de [0, n[.
func TestIntStaysWithinBounds(t *testing.T) {
	a := NewRandom(3, "essai")
	for _, n := range []int{1, 2, 3, 6, 41, 1000} {
		for i := 0; i < 5000; i++ {
			if v := a.Int(n); v < 0 || v >= n {
				t.Fatalf("Entier(%d) a rendu %d", n, v)
			}
		}
	}
}

// TestIntWithInvalidBound vérifie qu'un appel fautif ne fait pas tomber la
// partie. Un plugin ne doit pas pouvoir l'arrêter en demandant un tirage vide.
func TestIntWithInvalidBound(t *testing.T) {
	a := NewRandom(3, "essai")
	for _, n := range []int{0, -1, math.MinInt} {
		if v := a.Int(n); v != 0 {
			t.Errorf("Entier(%d) a rendu %d, attendu 0", n, v)
		}
	}
}

// TestUnbiasedDistribution éprouve le rejet du dernier bloc incomplet.
//
// Un simple reste favoriserait les petites valeurs. Le biais serait minuscule
// et invisible en jouant, mais l'équilibrage compare des milliers de parties :
// il s'y verrait, et on le chercherait ailleurs.
func TestUnbiasedDistribution(t *testing.T) {
	const faces, tirages = 6, 600000
	compte := make([]int, faces)

	a := NewRandom(20260825, "des")
	for i := 0; i < tirages; i++ {
		compte[a.Int(faces)]++
	}

	attendu := float64(tirages) / faces
	for face, n := range compte {
		if ecart := math.Abs(float64(n)-attendu) / attendu; ecart > 0.02 {
			t.Errorf("face %d sortie %d fois, écart de %.1f %% sur %.0f attendus",
				face, n, ecart*100, attendu)
		}
	}
}

// TestShuffleKeepsElements vérifie qu'un mélange permute sans rien perdre
// ni rien inventer.
func TestShuffleKeepsElements(t *testing.T) {
	depart := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	melange := append([]int(nil), depart...)

	Shuffle(NewRandom(5, "essai"), melange)

	vus := map[int]int{}
	for _, v := range melange {
		vus[v]++
	}
	if len(melange) != len(depart) || len(vus) != len(depart) {
		t.Fatalf("le mélange a changé le contenu : %v", melange)
	}
}

// TestShuffleIsDeterministic vérifie que deux mélanges de même graine donnent
// la même permutation — ce dont dépend l'ordre de fermeture des zones.
func TestShuffleIsDeterministic(t *testing.T) {
	a := []int{0, 1, 2, 3, 4, 5}
	b := []int{0, 1, 2, 3, 4, 5}

	Shuffle(NewRandom(99, "strangling"), a)
	Shuffle(NewRandom(99, "strangling"), b)

	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("permutations différentes : %v contre %v", a, b)
		}
	}
}

// TestShuffleActuallyPermutes vérifie qu'un mélange ne rend pas l'ordre de
// départ. Un Shuffle vide passerait tous les autres tests.
func TestShuffleActuallyPermutes(t *testing.T) {
	inchange := 0
	for graine := int64(0); graine < 20; graine++ {
		s := []int{0, 1, 2, 3, 4, 5, 6, 7}
		Shuffle(NewRandom(graine, "essai"), s)

		identique := true
		for i, v := range s {
			if v != i {
				identique = false
				break
			}
		}
		if identique {
			inchange++
		}
	}
	if inchange > 1 {
		t.Errorf("%d mélanges sur 20 ont rendu l'ordre de départ", inchange)
	}
}

// TestShuffleHandlesShortSlices vérifie qu'une tranche vide ou d'un
// seul élément ne fait rien de fâcheux.
func TestShuffleHandlesShortSlices(t *testing.T) {
	a := NewRandom(1, "essai")
	Shuffle(a, []int(nil))
	Shuffle(a, []int{7})

	seul := []int{7}
	Shuffle(a, seul)
	if seul[0] != 7 {
		t.Error("une tranche d'un élément a été altérée")
	}
}
