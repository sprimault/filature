// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"fmt"
	"math"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

// paletteLivree rend les couleurs du contenu livré.
func paletteLivree(t *testing.T) map[string]string {
	t.Helper()

	var fichier struct {
		Palette map[string]string `toml:"palette"`
	}
	chemin := filepath.Join(racine, "plugins", "base", "palette.toml")
	if _, err := toml.DecodeFile(chemin, &fichier); err != nil {
		t.Fatal(err)
	}
	if len(fichier.Palette) == 0 {
		t.Fatal("la palette livrée est vide : le contrôle ne vérifie plus rien")
	}
	return fichier.Palette
}

// TestSidesAreToldApartByLuminance garde l'écart de clarté entre les deux camps.
//
// La palette livrée l'annonce en toutes lettres, et le chiffre avait dérivé : il
// disait soixante-neuf, celui de l'ancien couple, quand le nouveau en vaut
// quatre-vingt-un. Le lot qui a reposé les couleurs a mis à jour les chiffres
// voisins et pas celui-là.
//
// C'est cet écart qui sépare les camps sur une capture en niveaux de gris ou
// pour qui distingue mal les rouges et les verts — la teinte ne fait que s'y
// ajouter.
func TestSidesAreToldApartByLuminance(t *testing.T) {
	palette := paletteLivree(t)

	ecart := luminance(t, palette["fugitive_main"]) - luminance(t, palette["inspector_main"])
	t.Logf("écart de luminance entre les deux camps : %.0f niveaux", ecart)

	if ecart < 70 || ecart > 95 {
		t.Errorf("écart de %.0f niveaux entre les camps, attendu autour de quatre-vingt-un : "+
			"le commentaire de plugins/base/palette.toml annonce un chiffre qui ne tient plus",
			ecart)
	}
}

// TestOutlinesKeepTheirContrast garde les contrastes que deux fichiers
// versionnés annoncent pour les mêmes couples.
//
// plugins/base/palette.toml justifie des contours presque noirs par ce qu'ils
// tiennent sur chaque sol, et docs/contrat-formes.md publie le même genre de
// valeurs. Les deux avaient divergé : la palette annonçait 6,2 sur une zone
// ouverte pour une valeur réelle entre 8,8 et 9,1, fausse de plus de quarante
// pour cent.
//
// Des plages et non des valeurs : trois contours sont mesurés ensemble, et
// resserrer sur la deuxième décimale rougirait à la première retouche de teinte
// sans que la justification cesse d'être vraie.
func TestOutlinesKeepTheirContrast(t *testing.T) {
	palette := paletteLivree(t)
	contours := []string{"fugitive_detail", "inspector_detail", "marker_outline"}

	for _, cas := range []struct {
		sol      string
		min, max float64
	}{
		{"street", 11, 12},
		{"zone_open", 8.5, 9.5},
		{"zone_closed", 2, 2.5},
	} {
		for _, contour := range contours {
			got := contraste(t, palette[contour], palette[cas.sol])
			t.Logf("%s sur %s : %.2f", contour, cas.sol, got)

			if got < cas.min || got > cas.max {
				t.Errorf("%s sur %s : contraste de %.2f, attendu entre %.1f et %.1f",
					contour, cas.sol, got, cas.min, cas.max)
			}
		}
	}
}

// contraste rend le rapport de contraste WCAG entre deux couleurs.
//
// WCAG et non Rec. 601, contrairement à l'écart entre sols : celui-ci compare
// deux fonds pour savoir s'ils se distinguent, celui-là mesure ce qu'un trait
// posé dessus laisse voir. Les deux modèles répondent à deux questions, et les
// confondre donne des chiffres plausibles et faux.
func contraste(t *testing.T, a, b string) float64 {
	t.Helper()

	x, y := luminanceRelative(t, a)+0.05, luminanceRelative(t, b)+0.05
	return math.Max(x, y) / math.Min(x, y)
}

// luminanceRelative rend la luminance relative WCAG d'une couleur.
func luminanceRelative(t *testing.T, hex string) float64 {
	t.Helper()

	var canaux [3]int
	if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &canaux[0], &canaux[1], &canaux[2]); err != nil {
		t.Fatalf("couleur %q illisible : %v", hex, err)
	}

	var lineaires [3]float64
	for i, n := range canaux {
		s := float64(n) / 255
		if s <= 0.03928 {
			lineaires[i] = s / 12.92
			continue
		}
		lineaires[i] = math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lineaires[0] + 0.7152*lineaires[1] + 0.0722*lineaires[2]
}
