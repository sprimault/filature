// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

// Palette associe des noms à des couleurs. Une forme ne référence que des noms :
// c'est ce qui permet de reteinter tout le jeu sans toucher à la géométrie, et
// à une forme tierce de suivre la palette active sans rien savoir d'elle.
type Palette map[string]string

// RequiredColors est le socle sur lequel toute forme peut compter. Une
// palette qui n'en couvre pas la totalité est refusée : une couleur manquante
// se verrait à l'écran comme un trou, plusieurs écrans plus loin.
//
// Les noms en _detail et marker_outline sont des contours et non des nuances
// d'accompagnement. Un pion se pose sur des sols dont la luminance va du simple
// au triple ; aucun remplissage ne se détache des trois, le contour le fait.
// Une palette qui les remonterait au niveau de son remplissage rendrait les
// pions illisibles sans qu'aucun contrôle ne s'en aperçoive.
//
// backdrop est ce qu'on voit autour du plateau, et n'appartient à aucune forme.
// Il est dans le socle parce qu'une palette qui le laisserait au niveau du bâti
// ferait disparaître les blocs du pourtour, et les pièces avec eux.
var RequiredColors = []string{
	"street", "building", "zone_open", "zone_closed", "backdrop",
	"fugitive_main", "fugitive_detail",
	"inspector_main", "inspector_detail",
	"marker_outline", "trail", "roadblock", "crime_scene",
}

// Les deux fichiers d'un plugin d'apparence. Séparés et tous deux facultatifs :
// une palette seule est le mod le moins cher qui existe, et des formes seules
// suivent la palette active sans rien en savoir.
const (
	ShapesFile  = "shapes.toml"
	PaletteFile = "palette.toml"
)

// ShapeSet rassemble les formes et la palette effectivement actives, après
// application des surcharges.
//
// Origins retient quel plugin a posé chaque forme, et reste vide pour le
// contenu livré — surcharger ce dernier est le cas normal, pas un conflit.
// Sans ce suivi, un conflit entre deux plugins ne pourrait que dire
// « fugitive est déjà définie », ce qui n'aide personne à savoir lequel des
// deux désinstaller.
type ShapeSet struct {
	Shapes  map[string]Shape
	Palette Palette
	Origins map[string]string
}

// fichierFormes est le fichier de formes tel qu'il se décode, avant tout
// contrôle. Le numéro de version vit à sa racine : il qualifie le fichier, pas
// son contenu.
type fichierFormes struct {
	Version int              `toml:"shapes_version"`
	Shape   map[string]Shape `toml:"shape"`
}

// fichierPalette est le fichier de palette tel qu'il se décode. Séparé des
// formes parce qu'un plugin peut ne porter que l'un des deux.
type fichierPalette struct {
	Version int     `toml:"shapes_version"`
	Palette Palette `toml:"palette"`
}

// Read lit les formes et la palette d'un plugin.
//
// Les deux fichiers sont facultatifs et indépendants : un plugin qui n'en porte
// aucun rend un jeu vide, ce qui n'est pas une erreur — la plupart des plugins
// ne touchent pas à l'apparence.
//
// Aucun contrôle de contenu ici : Read refuse ce qui ne se décode pas, Validate
// juge ce qui s'est décodé. Les séparer permet de lister tous les manquements
// d'un plugin d'un coup, là où un décodage qui validerait au fil de l'eau
// s'arrêterait au premier.
func Read(source fs.FS, dossier string) (*ShapeSet, error) {
	j := &ShapeSet{
		Shapes:  map[string]Shape{},
		Palette: Palette{},
		Origins: map[string]string{},
	}

	var formes fichierFormes
	presentFormes, err := decoder(source, dossier, ShapesFile, &formes)
	if err != nil {
		return nil, err
	}
	var palette fichierPalette
	presentPalette, err := decoder(source, dossier, PaletteFile, &palette)
	if err != nil {
		return nil, err
	}

	// La version se contrôle dès que le fichier existe, et non seulement s'il
	// porte quelque chose : un fichier d'une autre version dont on ne sait rien
	// lire se décode en structure vide, ce qui passerait pour un plugin qui ne
	// touche pas à l'apparence.
	for _, f := range []struct {
		nom     string
		version int
		present bool
	}{
		{ShapesFile, formes.Version, presentFormes},
		{PaletteFile, palette.Version, presentPalette},
	} {
		if f.present && f.version != ShapesVersion {
			return nil, fmt.Errorf("%s: shapes_version %d, ce binaire lit la %d",
				path.Join(dossier, f.nom), f.version, ShapesVersion)
		}
	}

	// La clé de la table porte le nom de la forme ; le champ le reçoit ici
	// parce que le décodeur TOML ne le connaît pas.
	//
	// Origins reste vide : Read lit un jeu de formes, il ne dit pas d'où vient
	// une surcharge. C'est Merge qui l'attribue, et lui seul en a besoin.
	for nom, forme := range formes.Shape {
		forme.Name = nom
		j.Shapes[nom] = forme
	}
	for nom, valeur := range palette.Palette {
		j.Palette[nom] = valeur
	}
	return j, nil
}

// decoder lit un fichier facultatif, dit s'il existait, et signale tout champ
// qu'il n'a pas placé.
//
// Ce dernier point est le contrôle qui compte : un TOML dont une table ne
// correspond à rien se décode sans erreur et laisse la structure à zéro. C'est
// ainsi que six formes livrées se sont retrouvées vides sans que rien ne le
// dise, après un renommage de clés.
func decoder(source fs.FS, dossier, nom string, dans any) (present bool, err error) {
	chemin := path.Join(dossier, nom)

	brut, err := fs.ReadFile(source, chemin)
	switch {
	// Absent vaut « ce plugin ne touche pas à cela », qui est le cas ordinaire :
	// la plupart des plugins ne portent ni formes ni palette.
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("%s: %w", chemin, err)
	}

	meta, err := toml.Decode(string(brut), dans)
	if err != nil {
		return true, fmt.Errorf("%s: %w", chemin, err)
	}
	if reste := meta.Undecoded(); len(reste) > 0 {
		cles := make([]string, len(reste))
		for i, k := range reste {
			cles[i] = k.String()
		}
		slices.Sort(cles)
		return true, fmt.Errorf("%s: clés inconnues, non appliquées : %s", chemin, strings.Join(cles, ", "))
	}
	return true, nil
}

// Validate applique les contrôles du contrat : gabarit, plafond de traits,
// nombre de sommets, résolution des couleurs, absence de valeur hexadécimale.
//
// Renvoie tous les manquements plutôt que le premier : quelqu'un qui met au
// point un plugin veut la liste, pas un aller-retour par erreur. Ils sortent
// dans l'ordre des noms de formes, sans quoi deux exécutions sur le même
// plugin donneraient deux listes différentes — le parcours d'une map n'a pas
// d'ordre.
func (j *ShapeSet) Validate() []error {
	var manquements []error

	for _, nom := range triees(j.Shapes) {
		manquements = append(manquements, j.validerForme(nom, j.Shapes[nom])...)
	}

	// Un ensemble entièrement vide n'a rien à respecter : il ne peint rien. Ce
	// cas ne se produit qu'en test, le contenu livré portant toujours les deux
	// fichiers — et si ce n'était plus vrai, c'est le binaire qui serait en
	// cause, ce que le paquet plugins vérifie de son côté.
	if len(j.Shapes) == 0 && len(j.Palette) == 0 {
		return manquements
	}

	for _, nom := range RequiredColors {
		if _, ok := j.Palette[nom]; !ok {
			manquements = append(manquements, fmt.Errorf("palette: couleur obligatoire absente : %s", nom))
		}
	}
	return manquements
}

// validerForme contrôle une forme et rend tous ses manquements.
func (j *ShapeSet) validerForme(nom string, f Shape) []error {
	cle := "shape." + nom

	gabarit, connu := templates[f.Role]
	if !connu {
		return []error{fmt.Errorf("%s.role: %q inconnu, attendu %s", cle, f.Role, rolesConnus())}
	}

	var manquements []error
	if len(f.Strokes) == 0 {
		manquements = append(manquements, fmt.Errorf("%s.stroke: aucun trait", cle))
	}
	if len(f.Strokes) > gabarit.MaxStrokes {
		manquements = append(manquements, fmt.Errorf("%s.stroke: %d traits, le rôle %s en accepte %d",
			cle, len(f.Strokes), f.Role, gabarit.MaxStrokes))
	}

	for i, t := range f.Strokes {
		manquements = append(manquements, j.validerTrait(fmt.Sprintf("%s.stroke[%d]", cle, i), t, f.Role, gabarit)...)
	}

	// Les variantes d'état suivent le même gabarit : une forme qui déborderait
	// une fois surlignée masquerait ses voisines à ce moment-là, ce qui est le
	// même avantage de jeu, simplement intermittent.
	for _, etat := range triees(f.Variants) {
		for i, t := range f.Variants[etat] {
			manquements = append(manquements,
				j.validerTrait(fmt.Sprintf("%s.%s[%d]", cle, etat, i), t, f.Role, gabarit)...)
		}
	}
	return manquements
}

// validerTrait contrôle une primitive : son type, sa géométrie, ses couleurs.
func (j *ShapeSet) validerTrait(cle string, t Stroke, role Role, gabarit Template) []error {
	var manquements []error

	for champ, nom := range map[string]string{"color": t.Color, "outline": t.Outline} {
		if nom == "" {
			continue
		}
		if strings.HasPrefix(nom, "#") {
			manquements = append(manquements, fmt.Errorf("%s.%s: %q est une valeur hexadécimale, un nom de palette est attendu", cle, champ, nom))
			continue
		}
		if _, ok := j.Palette[nom]; !ok {
			manquements = append(manquements, fmt.Errorf("%s.%s: couleur %q absente de la palette", cle, champ, nom))
		}
	}

	borner := func(sous string, p Point) {
		if p.X < gabarit.XMin || p.X > gabarit.XMax || p.Y < gabarit.YMin || p.Y > gabarit.YMax {
			manquements = append(manquements, fmt.Errorf("%s%s: (%d, %d) hors du gabarit du rôle %s, x de %d à %d et y de %d à %d",
				cle, sous, p.X, p.Y, role, gabarit.XMin, gabarit.XMax, gabarit.YMin, gabarit.YMax))
		}
	}

	switch t.Type {
	case StrokePolygon:
		if n := len(t.Points); n < 3 || n > 32 {
			manquements = append(manquements, fmt.Errorf("%s.points: %d sommets, attendu de 3 à 32", cle, n))
		}
		for i, p := range t.Points {
			borner(fmt.Sprintf(".points[%d]", i), p)
		}

	case StrokeCircle:
		if t.Radius < 1 {
			manquements = append(manquements, fmt.Errorf("%s.radius: %d, attendu au moins 1", cle, t.Radius))
		}
		// Les quatre extrémités et non le seul centre : c'est le disque entier
		// qui doit tenir dans le gabarit, pas son point d'ancrage.
		for _, p := range []Point{
			{t.Center.X - t.Radius, t.Center.Y}, {t.Center.X + t.Radius, t.Center.Y},
			{t.Center.X, t.Center.Y - t.Radius}, {t.Center.X, t.Center.Y + t.Radius},
		} {
			borner(".center", p)
		}

	case StrokeSegment:
		if t.Thickness < 1 || t.Thickness > 8 {
			manquements = append(manquements, fmt.Errorf("%s.thickness: %d, attendu de 1 à 8", cle, t.Thickness))
		}
		borner(".from", t.From)
		borner(".to", t.To)

	case StrokePrism:
		if role != RoleBuilding {
			manquements = append(manquements, fmt.Errorf("%s.type: prism est réservé au rôle %s", cle, RoleBuilding))
		}
		if t.Height < 1 || t.Height > gabarit.MaxHeight {
			manquements = append(manquements, fmt.Errorf("%s.height: %d, attendu de 1 à %d", cle, t.Height, gabarit.MaxHeight))
		}

	default:
		manquements = append(manquements, fmt.Errorf("%s.type: %q inconnu, attendu %s", cle, t.Type, typesConnus()))
	}

	// Un bâtiment ne déclare qu'une élévation : lui laisser une géométrie
	// permettrait de déborder sur les cases voisines et de masquer ce que
	// l'adversaire doit voir.
	if role == RoleBuilding && t.Type != StrokePrism {
		manquements = append(manquements, fmt.Errorf("%s.type: le rôle %s n'accepte que prism, reçu %q", cle, RoleBuilding, t.Type))
	}

	if t.OutlineThickness < 0 || t.OutlineThickness > 4 {
		manquements = append(manquements, fmt.Errorf("%s.outline_thickness: %d, attendu de 1 à 4", cle, t.OutlineThickness))
	}
	if t.Opacity < 0 || t.Opacity > 100 {
		manquements = append(manquements, fmt.Errorf("%s.opacity: %d, attendu de 0 à 100", cle, t.Opacity))
	}
	return manquements
}

// Merge applique une surcharge partielle sur le jeu de base.
//
// Un plugin ne déclare que ce qu'il remplace, le reste retombe sur le contenu
// livré — sans quoi changer un seul pion obligerait à livrer les quarante
// formes, et personne ne le ferait. Deux plugins qui redéfinissent la même
// forme sont un conflit, jamais un écrasement silencieux.
//
// Les couleurs ne suivent pas cette règle : une palette se remplace nom par nom
// sans conflit, parce qu'un plugin de formes qui ajoute un nom doit livrer la
// palette qui le définit, et que deux palettes qui reteintent la même chose est
// le cas normal — le joueur choisit celle qu'il installe.
func (j *ShapeSet) Merge(nom string, autre *ShapeSet) error {
	// Le conflit se juge sur Origins et non sur Shapes : une forme sans origine
	// vient du contenu livré, et la surcharger est précisément ce qu'un plugin
	// d'apparence est fait pour faire.
	for _, forme := range triees(autre.Shapes) {
		if origine := j.Origins[forme]; origine != "" && origine != nom {
			return fmt.Errorf("shape.%s: définie par %s et par %s", forme, origine, nom)
		}
	}

	for _, forme := range triees(autre.Shapes) {
		j.Shapes[forme] = autre.Shapes[forme]
		j.Origins[forme] = nom
	}
	for _, couleur := range triees(autre.Palette) {
		j.Palette[couleur] = autre.Palette[couleur]
	}
	return nil
}

// triees rend les clés d'une map dans un ordre stable.
//
// Le parcours d'une map n'a pas d'ordre en Go : sans tri, deux exécutions sur
// le même plugin rendraient ses manquements dans deux ordres différents, et un
// conflit entre trois plugins désignerait un coupable différent à chaque fois.
func triees[V any](m map[string]V) []string {
	cles := make([]string, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	slices.Sort(cles)
	return cles
}

// rolesConnus énonce les rôles acceptés, pour qu'une erreur donne la liste
// plutôt qu'un adjectif.
func rolesConnus() string {
	noms := make([]string, 0, len(templates))
	for r := range templates {
		noms = append(noms, string(r))
	}
	slices.Sort(noms)
	return strings.Join(noms, ", ")
}

// typesConnus énonce les quatre primitives, pour la même raison.
func typesConnus() string {
	return strings.Join([]string{
		string(StrokePolygon), string(StrokeCircle), string(StrokeSegment), string(StrokePrism),
	}, ", ")
}
