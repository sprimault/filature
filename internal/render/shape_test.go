// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package render

import "testing"

// TestSpanKeepsTheTwoToOneRatio vérifie qu'une emprise reste deux fois plus
// large que haute.
//
// C'est l'erreur que Span existe pour empêcher, et elle ne se voit pas : un
// calcul qui confond cases d'écran et cases de plateau donne un résultat
// plausible, seulement faux d'un facteur deux.
func TestSpanKeepsTheTwoToOneRatio(t *testing.T) {
	for _, cases := range []int{1, 9, 17, 41} {
		l, h := Span(cases)
		if l != 2*h {
			t.Errorf("Span(%d) = %d x %d, attendu un rapport 2:1", cases, l, h)
		}
		if l != cases*TileWidth {
			t.Errorf("Span(%d) largeur = %d, attendu %d", cases, l, cases*TileWidth)
		}
	}
}

// TestSpanAgreesWithToScreen vérifie que l'emprise annoncée est bien celle que
// la projection produit.
//
// Les deux fonctions doivent rester d'accord : Span n'est utile que si elle
// décrit la même géométrie que ToScreen, sans quoi dimensionner une fenêtre
// avec l'une et y placer des cases avec l'autre laisserait un écart silencieux.
func TestSpanAgreesWithToScreen(t *testing.T) {
	const cotes = 17

	// Les quatre coins d'un carré de cotes cases, projetés.
	gauche, _ := ToScreen(0, cotes-1)
	droite, _ := ToScreen(cotes-1, 0)
	_, haut := ToScreen(0, 0)
	_, bas := ToScreen(cotes-1, cotes-1)

	l, h := Span(cotes)
	if got := droite - gauche + TileWidth; got != l {
		t.Errorf("largeur projetée %d, Span annonce %d", got, l)
	}
	if got := bas - haut + TileHeight; got != h {
		t.Errorf("hauteur projetée %d, Span annonce %d", got, h)
	}
}

// TestStrokeWidthStaysWithinItsBounds vérifie le plancher et le plafond.
//
// Le plancher est ce qui empêche un trait de passer sous le pixel au dézoom, le
// plafond ce qui l'empêche d'avaler la forme quand elle devient minuscule. Les
// deux se contredisent sur une très petite forme, et c'est le plancher qui
// l'emporte : mieux vaut un trait trop épais qu'un trait absent.
func TestStrokeWidthStaysWithinItsBounds(t *testing.T) {
	cas := []struct {
		nom      string
		unites   int
		echelle  float64
		minForme int
		attendu  float64
	}{
		{"taille nominale, sous le plafond", 2, 1, 14, 2},
		{"demi-echelle", 2, 0.5, 14, 1},
		{"plafond atteint", 8, 1, 24, 4},
		{"plancher atteint", 1, 0.25, 14, 1},
		{"plancher l'emporte sur le plafond", 2, 0.1, 6, 1},
	}

	for _, c := range cas {
		if got := StrokeWidth(c.unites, c.echelle, c.minForme); got != c.attendu {
			t.Errorf("%s : StrokeWidth(%d, %g, %d) = %g, attendu %g",
				c.nom, c.unites, c.echelle, c.minForme, got, c.attendu)
		}
	}
}

// TestStrokeWidthLeavesTheCoreVisible vérifie la propriété que les deux bornes
// existent pour tenir : un pion garde un remplissage majoritaire.
//
// C'est la couleur qui dit à quel camp il appartient. Un noyau minoritaire ne
// serait pas un défaut d'esthétique mais une perte d'information.
func TestStrokeWidthLeavesTheCoreVisible(t *testing.T) {
	// La tête du fugitif, la plus petite forme d'un pion livré.
	const diametre = 14

	for _, taille := range []int{64, 48, 32, 24} {
		echelle := float64(taille) / TileWidth
		noyau := diametre * echelle
		bordure := StrokeWidth(2, echelle, diametre) + StrokeWidth(RimWidth, echelle, diametre)

		if part := noyau / (noyau + 2*bordure); part <= 0.5 {
			t.Errorf("à %d px par case, le noyau ne fait que %.0f %% du pion", taille, 100*part)
		}
	}
}

// TestRequiredColorsCoverBothOutlines vérifie que le socle porte les contours.
//
// Ce sont eux qui détachent une forme d'un sol dont la luminance varie du
// simple au triple ; les oublier ne casse rien au chargement et rend les pièces
// illisibles à l'écran.
func TestRequiredColorsCoverBothOutlines(t *testing.T) {
	presentes := map[string]bool{}
	for _, c := range RequiredColors {
		presentes[c] = true
	}

	for _, attendue := range []string{"backdrop", "marker_outline", "fugitive_detail", "inspector_detail"} {
		if !presentes[attendue] {
			t.Errorf("%s manque au socle obligatoire", attendue)
		}
	}
}
