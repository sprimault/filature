// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package preview

import (
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/core"
	"github.com/sprimault/filature/internal/loader"
	"github.com/sprimault/filature/internal/render"
	"github.com/sprimault/filature/plugins"
)

// livre charge l'apparence du contenu embarqué, telle qu'une partie l'utilise.
func livre(t *testing.T) (*render.ShapeSet, *core.Registry) {
	t.Helper()

	registre, formes, err := loader.Load(plugins.Shipped(), "")
	if err != nil {
		t.Fatalf("chargement du contenu livré : %v", err)
	}
	return formes, registre
}

// TestShapesDrawsEveryShapeOnEveryGround vérifie que la planche montre chaque
// forme sur chacun des sols.
//
// Tous et non le seul sol de rue : leurs luminances vont de 213 à 85, et une
// forme lisible sur l'un peut disparaître sur l'autre. Une planche qui n'en
// montrerait qu'une partie laisserait passer exactement ce qu'elle sert à voir
// — ce qu'elle a fait tant qu'elle en oubliait deux.
//
// Le compte se prend sur render.Grounds et non sur un littéral : c'est la seule
// liste de sols du dépôt, et une planche qui suivrait la sienne pourrait la
// laisser diverger sans que ce test bronche.
func TestShapesDrawsEveryShapeOnEveryGround(t *testing.T) {
	formes, _ := livre(t)

	var svg strings.Builder
	if err := Shapes(&svg, formes, ""); err != nil {
		t.Fatal(err)
	}

	rendu := svg.String()
	for nom := range formes.Shapes {
		if n := strings.Count(rendu, ">"+nom+"<"); n != len(sols) {
			t.Errorf("%s apparaît %d fois, attendu une par sol", nom, n)
		}
	}
	for _, sol := range sols {
		if !strings.Contains(rendu, formes.Palette[sol]) {
			t.Errorf("le sol %s n'est pas peint", sol)
		}
	}
}

// TestShapesMarksWhatThePluginOverrides vérifie qu'une forme surchargée se
// distingue de celles qui retombent sur le contenu livré.
//
// C'est le seul moyen de voir qu'une clé mal orthographiée n'a rien surchargé :
// elle passe la validation, ne change rien, et sans marque l'auteur croirait
// son plugin actif.
func TestShapesMarksWhatThePluginOverrides(t *testing.T) {
	formes, _ := livre(t)
	formes.Origins["fugitive"] = "mien"

	var svg strings.Builder
	if err := Shapes(&svg, formes, "mien"); err != nil {
		t.Fatal(err)
	}

	rendu := svg.String()
	if !strings.Contains(rendu, ">fugitive *<") {
		t.Error("la forme surchargée n'est pas marquée")
	}
	if strings.Contains(rendu, ">inspector *<") {
		t.Error("une forme du contenu livré est marquée comme surchargée")
	}
	if !strings.Contains(rendu, "surchargée par mien") {
		t.Error("la légende de la marque manque")
	}
}

// TestShapesEscapesNames vérifie qu'un nom venu d'un fichier tiers ne casse pas
// le document.
//
// Un chevron non échappé produirait un SVG invalide, et l'aperçu servirait
// justement à comprendre pourquoi.
func TestShapesEscapesNames(t *testing.T) {
	formes, _ := livre(t)
	formes.Shapes["a<b>"] = formes.Shapes["trail"]

	var svg strings.Builder
	if err := Shapes(&svg, formes, ""); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(svg.String(), ">a<b><") {
		t.Error("le nom entre tel quel dans le document")
	}
	if !strings.Contains(svg.String(), "a&lt;b&gt;") {
		t.Error("le nom échappé est absent")
	}
}

// TestBoardIsStable vérifie que deux aperçus du même plugin sont identiques.
//
// La graine est figée pour cela : sans stabilité, le diff de deux aperçus
// n'apprendrait rien, et on ne saurait pas si un changement vient du plugin ou
// du tirage.
func TestBoardIsStable(t *testing.T) {
	formes, registre := livre(t)

	var premier, second strings.Builder
	if err := Board(&premier, formes, registre, "district"); err != nil {
		t.Fatal(err)
	}
	if err := Board(&second, formes, registre, "district"); err != nil {
		t.Fatal(err)
	}

	if premier.String() != second.String() {
		t.Error("deux aperçus du même plugin diffèrent")
	}
}

// TestBoardShowsPiecesAndMarkers vérifie que le plateau porte de quoi juger.
//
// Une partie est jouée avant d'être rendue précisément pour cela : au premier
// tour il n'y a ni trace, ni inspecteur placé, et l'aperçu ne montrerait qu'un
// décor.
func TestBoardShowsPiecesAndMarkers(t *testing.T) {
	formes, registre := livre(t)

	var svg strings.Builder
	if err := Board(&svg, formes, registre, "district"); err != nil {
		t.Fatal(err)
	}

	rendu := svg.String()
	for nom, couleur := range map[string]string{
		"fugitif":    formes.Palette["fugitive_main"],
		"inspecteur": formes.Palette["inspector_main"],
		"trace":      formes.Palette["trail"],
		"fond":       formes.Palette["backdrop"],
	} {
		if !strings.Contains(rendu, couleur) {
			t.Errorf("le %s n'apparaît pas sur le plateau", nom)
		}
	}
}

// TestBoardRefusesUnknownPreset vérifie qu'un préréglage inconnu est refusé
// plutôt que remplacé par un défaut silencieux.
func TestBoardRefusesUnknownPreset(t *testing.T) {
	formes, registre := livre(t)

	err := Board(&strings.Builder{}, formes, registre, "metropole")
	if err == nil {
		t.Fatal("un préréglage inconnu est accepté")
	}
	if !strings.Contains(err.Error(), "metropole") {
		t.Errorf("message %q, attendu qu'il nomme le préréglage", err)
	}
}

// TestStrokeWidthsFollowTheContract vérifie que l'aperçu encadre ses traits
// comme le rendu le fera.
//
// Un aperçu qui s'en écarterait montrerait autre chose que le jeu, ce qui est
// pire que pas d'aperçu du tout : l'auteur validerait des formes sur une image
// qui ment.
func TestStrokeWidthsFollowTheContract(t *testing.T) {
	t.Run("le plafond porte sur la plus petite dimension", func(t *testing.T) {
		long := render.Stroke{
			Type:   render.StrokePolygon,
			Points: []render.Point{{X: -20, Y: 0}, {X: 20, Y: 0}, {X: 20, Y: 2}, {X: -20, Y: 2}},
		}
		if got := minDimension(long); got != 2 {
			t.Errorf("minDimension = %d, attendu la hauteur du trait", got)
		}
	})

	t.Run("un cercle est mesuré par son diamètre", func(t *testing.T) {
		if got := minDimension(render.Stroke{Type: render.StrokeCircle, Radius: 7}); got != 14 {
			t.Errorf("minDimension = %d, attendu 14", got)
		}
	})
}

// TestTransparentStrokeIsDrawnTransparent vérifie qu'une opacité nulle produit
// un attribut, et qu'une opacité absente n'en produit pas.
//
// Les deux se confondaient sur un entier : une forme déclarée transparente
// s'affichait pleine, alors que le contrat et le schéma acceptent la valeur.
func TestTransparentStrokeIsDrawnTransparent(t *testing.T) {
	segment := func(opacite *int) string {
		var s strings.Builder
		trait(&s, render.Stroke{
			Type: render.StrokeSegment, From: render.Point{X: -5}, To: render.Point{X: 5},
			Thickness: 2, Color: "trail", Opacity: opacite,
		}, 0, 0, 1, render.Palette{"trail": "#f0e6c8"})
		return s.String()
	}

	zero := 0
	if got := segment(&zero); !strings.Contains(got, `opacity="0.00"`) {
		t.Errorf("une opacité nulle se dessine pleine :\n%s", got)
	}
	if got := segment(nil); strings.Contains(got, "opacity=") {
		t.Errorf("une opacité absente pose un attribut :\n%s", got)
	}
}

// TestRimKeepsItsWidthWithoutOutline vérifie que le liseré garde son épaisseur
// quand aucun contour n'est déclaré.
//
// La couche du liseré est tracée à contour + liseré puis recouverte par le
// contour ; sans contour, rien ne repeignait cette bande et le liseré faisait
// trois unités au lieu de deux — six avec une épaisseur déclarée de quatre.
// Son épaisseur devenait pilotée par le plugin, ce que le contrat exclut.
func TestRimKeepsItsWidthWithoutOutline(t *testing.T) {
	// Un trait assez large pour que le plafond d'épaisseur ne morde pas :
	// à MaxStrokeRatio, une épaisseur de 4 ne se distingue d'une épaisseur de 1
	// qu'au-delà de vingt-quatre unités de corps, et un segment fin les rendrait
	// toutes deux à la même valeur plafonnée.
	segment := func(epaisseur *int) string {
		var s strings.Builder
		trait(&s, render.Stroke{
			Type: render.StrokeSegment, From: render.Point{X: -20}, To: render.Point{X: 20},
			Thickness: 4 * render.MaxStrokeRatio, Color: "trail", OutlineThickness: epaisseur,
		}, 0, 0, 1, render.Palette{"trail": "#f0e6c8"})
		return s.String()
	}

	// Une épaisseur qui ne borde rien ne doit rien changer, quelle qu'elle soit.
	un, quatre := 1, 4
	nu := segment(nil)
	for _, e := range []*int{&un, &quatre} {
		if got := segment(e); got != nu {
			t.Errorf("outline_thickness = %d change le tracé sans contour à border :\n%s\nattendu :\n%s",
				*e, got, nu)
		}
	}
}

// TestTintNeverExceedsWhite vérifie qu'un coefficient de face ne déborde pas
// d'un canal.
func TestTintNeverExceedsWhite(t *testing.T) {
	if got := teinte("#c0c0c0", 1.5); got != "#ffffff" {
		t.Errorf("teinte = %s, attendu que les canaux soient bornés", got)
	}
	if got := teinte("pas une couleur", 1.5); got != "#000000" {
		t.Errorf("teinte = %s, attendu un repli plutôt qu'un échec", got)
	}
}
