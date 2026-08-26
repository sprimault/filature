// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"math"
	"testing"
)

// TestMemeGraineMemeSuite est l'invariant de déterminisme.
//
// Deux générateurs partis de la même graine et du même flux donnent la même
// suite, sinon un rejeu diverge de son journal et la reprise, le débogage et
// l'entraînement de l'IA tombent ensemble.
func TestMemeGraineMemeSuite(t *testing.T) {
	a := NouvelAlea(178342119, "plateau")
	b := NouvelAlea(178342119, "plateau")

	for i := 0; i < 1000; i++ {
		if x, y := a.Entier(1000), b.Entier(1000); x != y {
			t.Fatalf("tirage %d : %d puis %d", i, x, y)
		}
	}
}

// TestSuiteDeReference fige ce que le générateur doit rendre, pour toujours.
//
// Les autres tests vérifient qu'il est cohérent avec lui-même : ils passeraient
// encore si l'algorithme changeait. Celui-ci est le seul qui s'y oppose, et
// c'est ce qu'on veut — une suite différente périmerait toutes les parties
// enregistrées, tous les plateaux de référence et toute comparaison d'IA.
//
// Ces valeurs ont été relevées sous Windows et vérifiées identiques sous
// Linux/amd64 : le générateur ne dépend ni de la plateforme, ni de la taille
// d'un int.
func TestSuiteDeReference(t *testing.T) {
	cas := []struct {
		flux    string
		attendu []int
	}{
		{"plateau", []int{276, 153, 191, 631, 908, 681, 809, 801}},
		{"ia", []int{391, 551, 161, 480, 683, 275, 78, 696}},
	}

	for _, c := range cas {
		t.Run(c.flux, func(t *testing.T) {
			a := NouvelAlea(178342119, c.flux)
			for i, attendu := range c.attendu {
				if got := a.Entier(1000); got != attendu {
					t.Fatalf("tirage %d : %d, attendu %d", i, got, attendu)
				}
			}
		})
	}

	melange := []int{0, 1, 2, 3, 4, 5}
	Melanger(NouvelAlea(99, "etranglement"), melange)
	for i, attendu := range []int{3, 5, 0, 1, 4, 2} {
		if melange[i] != attendu {
			t.Fatalf("mélange %v, attendu [3 5 0 1 4 2]", melange)
		}
	}
}

// TestFluxIndependants vérifie ce pour quoi les flux nommés existent.
//
// Ajouter un tirage dans la génération du plateau ne doit pas décaler ceux de
// l'IA. Deux flux consommés inégalement continuent donc chacun sur sa suite.
func TestFluxIndependants(t *testing.T) {
	plateau := NouvelAlea(42, "plateau")
	ia := NouvelAlea(42, "ia")
	temoin := NouvelAlea(42, "ia")

	// Le plateau consomme cent tirages, l'IA aucun.
	for i := 0; i < 100; i++ {
		plateau.Entier(6)
	}

	for i := 0; i < 50; i++ {
		if x, y := ia.Entier(6), temoin.Entier(6); x != y {
			t.Fatalf("le flux ia a été décalé au tirage %d : %d contre %d", i, x, y)
		}
	}
}

// TestFluxDifferentsDonnentDesSuitesDifferentes vérifie l'autre moitié : deux
// noms ne doivent pas retomber sur la même suite.
func TestFluxDifferentsDonnentDesSuitesDifferentes(t *testing.T) {
	a := NouvelAlea(7, "plateau")
	b := NouvelAlea(7, "ia")

	identiques := 0
	for i := 0; i < 100; i++ {
		if a.Entier(1000) == b.Entier(1000) {
			identiques++
		}
	}
	// Sur mille valeurs, une poignée de coïncidences est normale ; une suite
	// identique ne l'est pas.
	if identiques > 10 {
		t.Errorf("%d tirages identiques sur 100 : les flux se confondent", identiques)
	}
}

// TestGrainesDifferentesDonnentDesSuitesDifferentes vérifie que la graine porte
// bien la partie.
func TestGrainesDifferentesDonnentDesSuitesDifferentes(t *testing.T) {
	a := NouvelAlea(1, "plateau")
	b := NouvelAlea(2, "plateau")

	identiques := 0
	for i := 0; i < 100; i++ {
		if a.Entier(1000) == b.Entier(1000) {
			identiques++
		}
	}
	if identiques > 10 {
		t.Errorf("%d tirages identiques sur 100 : la graine ne sépare pas", identiques)
	}
}

// TestEntierResteDansLesBornes vérifie qu'aucun tirage ne sort de [0, n[.
func TestEntierResteDansLesBornes(t *testing.T) {
	a := NouvelAlea(3, "essai")
	for _, n := range []int{1, 2, 3, 6, 41, 1000} {
		for i := 0; i < 5000; i++ {
			if v := a.Entier(n); v < 0 || v >= n {
				t.Fatalf("Entier(%d) a rendu %d", n, v)
			}
		}
	}
}

// TestEntierBorneInvalide vérifie qu'un appel fautif ne fait pas tomber la
// partie. Un plugin ne doit pas pouvoir l'arrêter en demandant un tirage vide.
func TestEntierBorneInvalide(t *testing.T) {
	a := NouvelAlea(3, "essai")
	for _, n := range []int{0, -1, math.MinInt} {
		if v := a.Entier(n); v != 0 {
			t.Errorf("Entier(%d) a rendu %d, attendu 0", n, v)
		}
	}
}

// TestRepartitionSansBiais éprouve le rejet du dernier bloc incomplet.
//
// Un simple reste favoriserait les petites valeurs. Le biais serait minuscule
// et invisible en jouant, mais l'équilibrage compare des milliers de parties :
// il s'y verrait, et on le chercherait ailleurs.
func TestRepartitionSansBiais(t *testing.T) {
	const faces, tirages = 6, 600000
	compte := make([]int, faces)

	a := NouvelAlea(20260825, "des")
	for i := 0; i < tirages; i++ {
		compte[a.Entier(faces)]++
	}

	attendu := float64(tirages) / faces
	for face, n := range compte {
		if ecart := math.Abs(float64(n)-attendu) / attendu; ecart > 0.02 {
			t.Errorf("face %d sortie %d fois, écart de %.1f %% sur %.0f attendus",
				face, n, ecart*100, attendu)
		}
	}
}

// TestMelangerGardeLesElements vérifie qu'un mélange permute sans rien perdre
// ni rien inventer.
func TestMelangerGardeLesElements(t *testing.T) {
	depart := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	melange := append([]int(nil), depart...)

	Melanger(NouvelAlea(5, "essai"), melange)

	vus := map[int]int{}
	for _, v := range melange {
		vus[v]++
	}
	if len(melange) != len(depart) || len(vus) != len(depart) {
		t.Fatalf("le mélange a changé le contenu : %v", melange)
	}
}

// TestMelangerEstDeterministe vérifie que deux mélanges de même graine donnent
// la même permutation — ce dont dépend l'ordre de fermeture des zones.
func TestMelangerEstDeterministe(t *testing.T) {
	a := []int{0, 1, 2, 3, 4, 5}
	b := []int{0, 1, 2, 3, 4, 5}

	Melanger(NouvelAlea(99, "etranglement"), a)
	Melanger(NouvelAlea(99, "etranglement"), b)

	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("permutations différentes : %v contre %v", a, b)
		}
	}
}

// TestMelangerPermuteVraiment vérifie qu'un mélange ne rend pas l'ordre de
// départ. Un Melanger vide passerait tous les autres tests.
func TestMelangerPermuteVraiment(t *testing.T) {
	inchange := 0
	for graine := int64(0); graine < 20; graine++ {
		s := []int{0, 1, 2, 3, 4, 5, 6, 7}
		Melanger(NouvelAlea(graine, "essai"), s)

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

// TestMelangerSupporteLesTranchesCourtes vérifie qu'une tranche vide ou d'un
// seul élément ne fait rien de fâcheux.
func TestMelangerSupporteLesTranchesCourtes(t *testing.T) {
	a := NouvelAlea(1, "essai")
	Melanger(a, []int(nil))
	Melanger(a, []int{7})

	seul := []int{7}
	Melanger(a, seul)
	if seul[0] != 7 {
		t.Error("une tranche d'un élément a été altérée")
	}
}
