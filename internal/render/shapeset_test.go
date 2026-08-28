// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// paletteComplete rend un fichier de palette qui couvre le socle obligatoire.
//
// Les couleurs y valent toutes la même valeur, sauf les sols : ce qui se teste
// ici est la résolution des noms, jamais leur rendu. Les sols font exception
// parce que Validate leur impose un écart de luminance — cinq gris identiques
// seraient une palette invalide, et le test échouerait sur un défaut qu'il ne
// cherche pas.
func paletteComplete() string {
	var b strings.Builder
	b.WriteString("shapes_version = 4\n\n[palette]\n")
	gris := []string{"#e0e0e0", "#b4b4b4", "#888888", "#5c5c5c", "#303030"}
	for _, nom := range RequiredColors {
		valeur := "#808080"
		if rang := slices.Index(Grounds, nom); rang >= 0 {
			valeur = gris[rang]
		}
		b.WriteString(nom + " = \"" + valeur + "\"\n")
	}
	return b.String()
}

// source monte un plugin d'apparence en mémoire.
func source(formes, palette string) fstest.MapFS {
	fs := fstest.MapFS{}
	if formes != "" {
		fs["essai/"+ShapesFile] = &fstest.MapFile{Data: []byte(formes)}
	}
	if palette != "" {
		fs["essai/"+PaletteFile] = &fstest.MapFile{Data: []byte(palette)}
	}
	return fs
}

// lire décode un plugin d'essai et échoue si la lecture même échoue.
func lire(t *testing.T, formes string) *ShapeSet {
	t.Helper()

	j, err := Read(source(formes, paletteComplete()), "essai")
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	return j
}

// manquement dit si un des manquements porte le fragment attendu.
func manquement(errs []error, fragment string) bool {
	for _, e := range errs {
		if strings.Contains(e.Error(), fragment) {
			return true
		}
	}
	return false
}

// TestReadAcceptsAPluginWithoutAppearance vérifie qu'un plugin sans formes ni
// palette se lit sans erreur.
//
// C'est le cas de la plupart des plugins : un plugin de règles ou de langue n'a
// aucune raison de porter ces fichiers, et les exiger le refuserait.
func TestReadAcceptsAPluginWithoutAppearance(t *testing.T) {
	j, err := Read(source("", ""), "essai")
	if err != nil {
		t.Fatalf("un plugin sans apparence est refusé : %v", err)
	}
	if len(j.Shapes) != 0 || len(j.Palette) != 0 {
		t.Errorf("%d formes et %d couleurs pour un plugin qui n'en déclare aucune", len(j.Shapes), len(j.Palette))
	}
}

// TestReadRejectsAnotherContractVersion vérifie qu'un fichier écrit contre une
// autre version est refusé plutôt que lu de travers.
func TestReadRejectsAnotherContractVersion(t *testing.T) {
	_, err := Read(source("shapes_version = 1\n", paletteComplete()), "essai")
	if err == nil {
		t.Fatal("une version inconnue est acceptée")
	}
	if !strings.Contains(err.Error(), "shapes_version") {
		t.Errorf("message %q, attendu qu'il nomme shapes_version", err)
	}
}

// TestReadRejectsUnplacedKeys vérifie qu'une clé que le décodeur ne sait pas
// placer arrête la lecture.
//
// C'est le contrôle le plus utile du décodage : un TOML dont une table ne
// correspond à rien se décode sans erreur et laisse la structure à zéro. Six
// formes livrées se sont retrouvées vides de cette façon, après un renommage
// de clés, sans que rien ne le signale.
func TestReadRejectsUnplacedKeys(t *testing.T) {
	_, err := Read(source(`
shapes_version = 4

[shape.fugitive]
role = "piece"

  [[shape.fugitive.trait]]
  type = "polygon"
`, paletteComplete()), "essai")

	if err == nil {
		t.Fatal("une table orpheline est acceptée en silence")
	}
	if !strings.Contains(err.Error(), "trait") {
		t.Errorf("message %q, attendu qu'il nomme la clé fautive", err)
	}
}

// TestReadNamesEachShape vérifie que la clé de la table devient le nom de la
// forme, ce que le décodeur TOML ne fait pas seul, et que la lecture n'attribue
// aucune origine.
//
// L'origine désigne le plugin qui a posé une surcharge : c'est Merge qui la
// connaît. Une forme lue sans passer par lui vient du contenu livré, et la
// surcharger doit rester possible sans conflit.
func TestReadNamesEachShape(t *testing.T) {
	j := lire(t, `
shapes_version = 4

[shape.fugitive]
role = "piece"

  [[shape.fugitive.stroke]]
  type = "circle"
  center = [0, 10]
  radius = 5
  color = "fugitive_main"
`)

	if got := j.Shapes["fugitive"].Name; got != "fugitive" {
		t.Errorf("nom %q, attendu fugitive", got)
	}
	if got := j.Origins["fugitive"]; got != "" {
		t.Errorf("origine %q, attendu aucune avant surcharge", got)
	}
}

// TestValidateAcceptsShippedShapes vérifie que le contenu livré passe ses
// propres contrôles.
//
// C'est le test qui donne leur sens aux autres : les formes du jeu sont écrites
// dans le même format qu'un plugin tiers, et si elles ne satisfaisaient pas le
// contrat, c'est le contrat qu'il faudrait reprendre.
func TestValidateAcceptsShippedShapes(t *testing.T) {
	j := lire(t, `
shapes_version = 4

[shape.building]
role = "building"

  [[shape.building.stroke]]
  type = "prism"
  height = 24
  color = "building"

[shape.fugitive]
role = "piece"

  [[shape.fugitive.stroke]]
  type = "polygon"
  points = [[-8, 0], [8, 0], [5, 26], [-5, 26]]
  color = "fugitive_main"
  outline = "fugitive_detail"
  outline_thickness = 2

[shape.trail]
role = "marker"

  [[shape.trail.stroke]]
  type = "segment"
  from = [-10, 0]
  to = [10, 0]
  thickness = 4
  color = "trail"
  outline = "marker_outline"
`)

	if errs := j.Validate(); len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("le contenu livré est refusé : %v", e)
		}
	}
}

// TestValidateCatchesEachBreach vérifie qu'aucun manquement du contrat ne passe.
//
// Un cas par règle, et chacun nomme la clé fautive : un auteur de plugin qui
// reçoit « cible invalide » ne sait pas où chercher, celui qui reçoit
// « shape.essai.stroke[0].points[2] » corrige en une minute.
func TestValidateCatchesEachBreach(t *testing.T) {
	cas := []struct {
		nom      string
		formes   string
		fragment string
	}{
		{
			nom: "point hors du gabarit",
			formes: `[shape.essai]
role = "marker"
  [[shape.essai.stroke]]
  type = "polygon"
  points = [[-10, 0], [10, 0], [0, 30]]
  color = "trail"`,
			fragment: "points[2]",
		},
		{
			nom: "polygone à deux sommets",
			formes: `[shape.essai]
role = "marker"
  [[shape.essai.stroke]]
  type = "polygon"
  points = [[-10, 0], [10, 0]]
  color = "trail"`,
			fragment: "2 sommets",
		},
		{
			nom: "couleur absente de la palette",
			formes: `[shape.essai]
role = "marker"
  [[shape.essai.stroke]]
  type = "segment"
  from = [-5, 0]
  to = [5, 0]
  thickness = 2
  color = "mauve"`,
			fragment: `couleur "mauve" absente`,
		},
		{
			nom: "valeur hexadécimale dans une forme",
			formes: `[shape.essai]
role = "marker"
  [[shape.essai.stroke]]
  type = "segment"
  from = [-5, 0]
  to = [5, 0]
  thickness = 2
  color = "#ff0000"`,
			fragment: "hexadécimale",
		},
		{
			nom: "contour non résolu",
			formes: `[shape.essai]
role = "marker"
  [[shape.essai.stroke]]
  type = "segment"
  from = [-5, 0]
  to = [5, 0]
  thickness = 2
  color = "trail"
  outline = "inconnu"`,
			fragment: "outline",
		},
		{
			nom: "rôle absent",
			formes: `[shape.essai]
  [[shape.essai.stroke]]
  type = "segment"
  from = [-5, 0]
  to = [5, 0]
  thickness = 2
  color = "trail"`,
			fragment: "role",
		},
		{
			nom: "rôle inconnu",
			formes: `[shape.essai]
role = "sol"
  [[shape.essai.stroke]]
  type = "segment"
  from = [-5, 0]
  to = [5, 0]
  thickness = 2
  color = "trail"`,
			fragment: "role",
		},
		{
			nom: "prisme hors du rôle bâtiment",
			formes: `[shape.essai]
role = "marker"
  [[shape.essai.stroke]]
  type = "prism"
  height = 10
  color = "building"`,
			fragment: "prism est réservé",
		},
		{
			nom: "bâtiment qui déclare une géométrie",
			formes: `[shape.essai]
role = "building"
  [[shape.essai.stroke]]
  type = "polygon"
  points = [[0, 0], [1, 0], [0, 1]]
  color = "building"`,
			fragment: "n'accepte que prism",
		},
		{
			nom: "bâtiment plus haut que le plafond",
			formes: `[shape.essai]
role = "building"
  [[shape.essai.stroke]]
  type = "prism"
  height = 32
  color = "building"`,
			fragment: "height",
		},
		{
			nom: "marqueur au-dessus de son plafond de traits",
			formes: "[shape.essai]\nrole = \"marker\"\n" + strings.Repeat(`[[shape.essai.stroke]]
type = "segment"
from = [-5, 0]
to = [5, 0]
thickness = 2
color = "trail"
`, 9),
			fragment: "9 traits",
		},
		{
			nom: "forme sans trait",
			formes: `[shape.essai]
role = "marker"`,
			fragment: "aucun trait",
		},
		{
			nom: "cercle qui déborde par son rayon",
			formes: `[shape.essai]
role = "marker"
  [[shape.essai.stroke]]
  type = "circle"
  center = [0, 0]
  radius = 20
  color = "trail"`,
			fragment: "hors du gabarit",
		},
		{
			nom: "opacité hors bornes",
			formes: `[shape.essai]
role = "marker"
  [[shape.essai.stroke]]
  type = "segment"
  from = [-5, 0]
  to = [5, 0]
  thickness = 2
  color = "trail"
  opacity = 150`,
			fragment: "opacity",
		},
		{
			nom: "cercle sans rayon",
			formes: `[shape.essai]
role = "marker"
  [[shape.essai.stroke]]
  type = "circle"
  center = [0, 0]
  radius = 0
  color = "trail"`,
			fragment: "radius",
		},
		{
			nom: "épaisseur de contour hors bornes",
			formes: `[shape.essai]
role = "marker"
  [[shape.essai.stroke]]
  type = "segment"
  from = [-5, 0]
  to = [5, 0]
  thickness = 2
  color = "trail"
  outline = "marker_outline"
  outline_thickness = 5`,
			fragment: "outline_thickness",
		},
		{
			nom: "épaisseur de contour nulle",
			formes: `[shape.essai]
role = "marker"
  [[shape.essai.stroke]]
  type = "segment"
  from = [-5, 0]
  to = [5, 0]
  thickness = 2
  color = "trail"
  outline = "marker_outline"
  outline_thickness = 0`,
			fragment: "outline_thickness",
		},
		{
			nom: "type de trait inconnu",
			formes: `[shape.essai]
role = "marker"
  [[shape.essai.stroke]]
  type = "ellipse"
  color = "trail"`,
			fragment: `"ellipse" inconnu`,
		},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			// Le rôle n'est plus injecté quand il manque : c'est un champ
			// obligatoire, et un harnais qui le complète masquerait le seul
			// contrôle qui le vérifie.
			j, err := Read(source("shapes_version = 4\n\n"+c.formes, paletteComplete()), "essai")
			if err != nil {
				t.Fatalf("lecture : %v", err)
			}

			errs := j.Validate()
			if len(errs) == 0 {
				t.Fatal("accepté sans manquement")
			}
			if !manquement(errs, c.fragment) {
				t.Errorf("manquements %v, attendu qu'un porte %q", errs, c.fragment)
			}
			if !manquement(errs, "shape.essai") {
				t.Errorf("manquements %v, attendu qu'un nomme la forme fautive", errs)
			}
		})
	}
}

// TestValidateWantsEveryRequiredColor vérifie qu'une palette incomplète est
// refusée, couleur par couleur.
func TestValidateWantsEveryRequiredColor(t *testing.T) {
	j, err := Read(source("", "shapes_version = 4\n\n[palette]\nstreet = \"#d8d2c4\"\n"), "essai")
	if err != nil {
		t.Fatal(err)
	}

	errs := j.Validate()
	if len(errs) != len(RequiredColors)-1 {
		t.Errorf("%d manquements pour %d couleurs absentes", len(errs), len(RequiredColors)-1)
	}
	if !manquement(errs, "backdrop") {
		t.Errorf("manquements %v, attendu qu'ils nomment chaque couleur absente", errs)
	}
}

// TestValidateChecksVariantsToo vérifie qu'une variante d'état est soumise au
// même gabarit que l'état normal.
//
// Une forme qui ne déborde qu'une fois surlignée masque ses voisines à ce
// moment-là : c'est le même avantage de jeu, simplement intermittent.
func TestValidateChecksVariantsToo(t *testing.T) {
	j := lire(t, `
shapes_version = 4

[shape.essai]
role = "marker"

  [[shape.essai.stroke]]
  type = "segment"
  from = [-5, 0]
  to = [5, 0]
  thickness = 2
  color = "trail"

  [[shape.essai.highlighted]]
  type = "polygon"
  points = [[-10, 0], [10, 0], [0, 40]]
  color = "trail"
`)

	manquements := j.Validate()
	if !manquement(manquements, "highlighted") {
		t.Error("une variante hors gabarit passe")
	}

	// Le chemin nommé doit être celui d'un fichier chargeable : le message
	// désignait « shape.essai.highlighted » sous une clé « variant » que
	// personne ne pouvait écrire, et renvoyait donc l'auteur nulle part.
	if !manquement(manquements, "shape.essai.highlighted[0]") {
		t.Errorf("manquements %v, attendu qu'un nomme la clé telle qu'elle s'écrit", manquements)
	}
}

// TestVariantUnderAnUnknownKeyIsRefused vérifie que l'ancienne écriture ne
// passe plus en silence.
//
// Le Go décodait les variantes sous « variant.<état> » quand le schéma et le
// document déclaraient deux propriétés nommées. Une forme écrite d'après le
// contrat perdait donc sa variante sans un mot ; l'inverse vaut désormais, et
// il vaut mieux qu'il se dise.
func TestVariantUnderAnUnknownKeyIsRefused(t *testing.T) {
	_, err := Read(source(`shapes_version = 4

[shape.essai]
role = "marker"

  [[shape.essai.stroke]]
  type = "segment"
  from = [-5, 0]
  to = [5, 0]
  thickness = 2
  color = "trail"

  [[shape.essai.variant.highlighted]]
  type = "circle"
  center = [0, 0]
  radius = 2
  color = "trail"
`, paletteComplete()), "essai")

	if err == nil {
		t.Fatal("une variante sous une clé inconnue est acceptée")
	}
	if !strings.Contains(err.Error(), "variant") {
		t.Errorf("refusé, mais sans nommer la clé fautive :\n%v", err)
	}
}

// TestValidateListsEverythingAtOnce vérifie qu'un plugin abîmé rend tous ses
// manquements et non le premier.
//
// C'est ce qui fait la différence entre corriger un plugin en une passe et le
// corriger en autant d'allers-retours qu'il a de fautes.
func TestValidateListsEverythingAtOnce(t *testing.T) {
	j := lire(t, `
shapes_version = 4

[shape.un]
role = "marker"

  [[shape.un.stroke]]
  type = "polygon"
  points = [[-10, 0], [10, 0], [0, 30]]
  color = "mauve"

[shape.deux]
role = "marker"

  [[shape.deux.stroke]]
  type = "segment"
  from = [-5, 0]
  to = [5, 0]
  thickness = 99
  color = "#ff0000"
`)

	errs := j.Validate()
	for _, attendu := range []string{"shape.un", "shape.deux", "mauve", "hexadécimale", "thickness"} {
		if !manquement(errs, attendu) {
			t.Errorf("manquements %v, attendu qu'un porte %q", errs, attendu)
		}
	}
}

// TestValidateIsStable vérifie que deux passes rendent les manquements dans le
// même ordre.
//
// Le parcours d'une map n'a pas d'ordre en Go : sans tri, un auteur de plugin
// verrait sa liste se réordonner à chaque exécution, et un diff de sortie
// deviendrait illisible.
func TestValidateIsStable(t *testing.T) {
	j := lire(t, `
shapes_version = 4

[shape.aaa]
role = "marker"

[shape.bbb]
role = "marker"

[shape.ccc]
role = "marker"
`)

	premier := j.Validate()
	for i := range 20 {
		suivant := j.Validate()
		if len(suivant) != len(premier) {
			t.Fatalf("passe %d : %d manquements contre %d", i, len(suivant), len(premier))
		}
		for k := range premier {
			if suivant[k].Error() != premier[k].Error() {
				t.Fatalf("passe %d, rang %d : %q contre %q", i, k, suivant[k], premier[k])
			}
		}
	}
}

// TestMergeOverridesOnlyWhatIsDeclared vérifie qu'une surcharge partielle
// laisse le reste en place.
//
// C'est ce qui rend un plugin d'apparence écrivable : sans cela, changer un
// seul pion obligerait à livrer toutes les formes, et personne ne le ferait.
func TestMergeOverridesOnlyWhatIsDeclared(t *testing.T) {
	base := lire(t, `
shapes_version = 4

[shape.fugitive]
role = "piece"
  [[shape.fugitive.stroke]]
  type = "circle"
  center = [0, 10]
  radius = 5
  color = "fugitive_main"

[shape.inspector]
role = "piece"
  [[shape.inspector.stroke]]
  type = "circle"
  center = [0, 10]
  radius = 5
  color = "inspector_main"
`)

	surcharge, err := Read(source(`
shapes_version = 4

[shape.fugitive]
role = "piece"
  [[shape.fugitive.stroke]]
  type = "circle"
  center = [0, 12]
  radius = 7
  color = "fugitive_main"
`, ""), "essai")
	if err != nil {
		t.Fatal(err)
	}

	if err := base.Merge("mes-vehicules", surcharge); err != nil {
		t.Fatalf("surcharge refusée : %v", err)
	}

	if got := base.Shapes["fugitive"].Strokes[0].Radius; got != 7 {
		t.Errorf("rayon du fugitif %d, attendu celui de la surcharge", got)
	}
	if got := base.Shapes["inspector"].Strokes[0].Radius; got != 5 {
		t.Errorf("rayon de l'inspecteur %d, la surcharge ne le déclarait pas", got)
	}
	if got := base.Origins["fugitive"]; got != "mes-vehicules" {
		t.Errorf("origine %q, attendu le plugin surchargeant", got)
	}
}

// TestMergeRefusesTwoPluginsOnTheSameShape vérifie qu'un conflit est signalé et
// non résolu en silence.
//
// Deux plugins qui redéfinissent la même forme donnent un résultat qui dépend
// de l'ordre de chargement : le joueur verrait l'un ou l'autre sans savoir
// pourquoi, et l'auteur écarté n'aurait aucun moyen de s'en apercevoir.
func TestMergeRefusesTwoPluginsOnTheSameShape(t *testing.T) {
	forme := `
shapes_version = 4

[shape.fugitive]
role = "piece"
  [[shape.fugitive.stroke]]
  type = "circle"
  center = [0, 10]
  radius = 5
  color = "fugitive_main"
`
	base := lire(t, forme)

	premier, err := Read(source(forme, ""), "essai")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Read(source(forme, ""), "essai")
	if err != nil {
		t.Fatal(err)
	}

	if err := base.Merge("un", premier); err != nil {
		t.Fatalf("première surcharge refusée : %v", err)
	}

	err = base.Merge("deux", second)
	if err == nil {
		t.Fatal("deux plugins redéfinissent la même forme sans conflit")
	}
	for _, attendu := range []string{"fugitive", "un", "deux"} {
		if !strings.Contains(err.Error(), attendu) {
			t.Errorf("message %q, attendu qu'il nomme %q", err, attendu)
		}
	}
}

// TestMergeAcceptsTwoPalettes vérifie que deux palettes ne sont pas un conflit.
//
// Contrairement aux formes : un plugin de formes qui ajoute un nom doit livrer
// la palette qui le définit, et deux palettes qui reteintent la même chose est
// le cas normal — le joueur choisit celle qu'il installe.
func TestMergeAcceptsTwoPalettes(t *testing.T) {
	base := lire(t, "shapes_version = 4\n")

	autre, err := Read(source("", "shapes_version = 4\n\n[palette]\nstreet = \"#000000\"\n"), "essai")
	if err != nil {
		t.Fatal(err)
	}

	if err := base.Merge("sombre", autre); err != nil {
		t.Fatalf("deux palettes sont refusées : %v", err)
	}
	if got := base.Palette["street"]; got != "#000000" {
		t.Errorf("street vaut %q, attendu la valeur surchargée", got)
	}
}
