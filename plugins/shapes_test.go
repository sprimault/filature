// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"io/fs"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/sprimault/evasion/internal/render"
)

// formesLivrees décode shapes.toml et palette.toml du contenu embarqué.
//
// Signale tout champ que le décodeur n'a pas su placer. C'est le contrôle qui
// compte : un TOML dont une table ne correspond à rien se décode sans erreur et
// laisse la structure à zéro, ce qui donne des formes vides que rien n'annonce.
func formesLivrees(t *testing.T) (map[string]render.Shape, render.Palette, int) {
	t.Helper()

	var formes struct {
		Version int                     `toml:"shapes_version"`
		Shape   map[string]render.Shape `toml:"shape"`
	}
	decoder(t, "base/shapes.toml", &formes)

	var palette struct {
		Version int            `toml:"shapes_version"`
		Palette render.Palette `toml:"palette"`
	}
	decoder(t, "base/palette.toml", &palette)

	if palette.Version != formes.Version {
		t.Errorf("versions divergentes : formes %d, palette %d", formes.Version, palette.Version)
	}
	return formes.Shape, palette.Palette, formes.Version
}

// decoder lit un fichier embarqué et échoue sur tout champ non placé.
func decoder(t *testing.T, chemin string, dans any) {
	t.Helper()

	brut, err := fs.ReadFile(Shipped(), chemin)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := toml.Decode(string(brut), dans)
	if err != nil {
		t.Fatalf("%s: %v", chemin, err)
	}
	for _, cle := range meta.Undecoded() {
		t.Errorf("%s: la clé %s n'a été placée nulle part", chemin, cle)
	}
}

// TestShippedShapesMatchBinaryContract vérifie que le contenu livré annonce la
// version que ce binaire sait lire.
//
// Les deux fichiers portent le numéro et le code en porte un troisième : les
// laisser diverger reviendrait à refuser au démarrage le contenu du dépôt.
func TestShippedShapesMatchBinaryContract(t *testing.T) {
	_, _, version := formesLivrees(t)

	if version != render.ShapesVersion {
		t.Errorf("contenu livré en version %d, binaire en version %d", version, render.ShapesVersion)
	}
}

// TestShippedPaletteCoversRequiredColors vérifie que la palette livrée couvre
// le socle obligatoire.
//
// Une couleur manquante ne fait rien échouer : elle se voit à l'écran comme un
// trou, plusieurs écrans plus loin.
func TestShippedPaletteCoversRequiredColors(t *testing.T) {
	_, palette, _ := formesLivrees(t)

	for _, nom := range render.RequiredColors {
		if _, ok := palette[nom]; !ok {
			t.Errorf("couleur obligatoire absente de la palette : %s", nom)
		}
	}

	// Et l'inverse, qui est le sens par lequel le défaut est arrivé : les deux
	// couleurs des lieux de ressourcement ont été livrées dans la palette sans
	// entrer dans le socle. Une palette tierce qui les oubliait passait alors la
	// validation, et le rendu cherchait un nom absent sur toutes les cases
	// d'abri. Une couleur qui existe ici sans être exigée n'est garantie par
	// personne.
	exigees := map[string]bool{}
	for _, nom := range render.RequiredColors {
		exigees[nom] = true
	}
	for nom := range palette {
		if !exigees[nom] {
			t.Errorf("la palette livrée porte %s, que RequiredColors n'exige pas : "+
				"une palette tierce peut l'omettre sans être refusée", nom)
		}
	}
}

// TestShippedShapesResolveTheirColors vérifie que tout nom référencé par une
// forme existe dans la palette livrée.
//
// Vaut pour les contours autant que pour les remplissages : c'est le contour
// qui détache un pion d'un sol sombre, et un contour non résolu le rendrait
// illisible là précisément.
func TestShippedShapesResolveTheirColors(t *testing.T) {
	formes, palette, _ := formesLivrees(t)

	for nom, f := range formes {
		for i, trait := range f.Strokes {
			for cle, couleur := range map[string]string{"color": trait.Color, "outline": trait.Outline} {
				if couleur == "" {
					continue
				}
				if _, ok := palette[couleur]; !ok {
					t.Errorf("shape.%s.stroke[%d].%s : couleur %q absente de la palette", nom, i, cle, couleur)
				}
			}
		}
	}
}

// TestShippedShapesCarryStrokes vérifie qu'aucune forme livrée n'est vide.
//
// Une forme sans trait ne se dessine pas et ne se signale pas non plus : le
// plateau se rend, la pièce manque. Le cas s'est produit en renommant les
// tables du fichier sans renommer les champs qui les reçoivent.
func TestShippedShapesCarryStrokes(t *testing.T) {
	formes, _, _ := formesLivrees(t)

	if len(formes) == 0 {
		t.Fatal("aucune forme livrée")
	}
	for nom, f := range formes {
		if len(f.Strokes) == 0 {
			t.Errorf("shape.%s ne porte aucun trait", nom)
		}
		if f.Role == "" {
			t.Errorf("shape.%s ne déclare pas de rôle", nom)
		}
	}
}
