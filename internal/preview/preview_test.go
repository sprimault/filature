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
// forme sur les trois sols.
//
// Les trois et non le seul sol de rue : leurs luminances vont de 210 à 82, et
// une forme lisible sur l'un peut disparaître sur l'autre. Une planche qui n'en
// montrerait qu'un laisserait passer exactement ce qu'elle sert à voir.
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
	if err := Board(&premier, formes, registre, "quartier"); err != nil {
		t.Fatal(err)
	}
	if err := Board(&second, formes, registre, "quartier"); err != nil {
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
	if err := Board(&svg, formes, registre, "quartier"); err != nil {
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
