// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Package render porte la projection isométrique et le vocabulaire géométrique
// des plugins d'apparence.
//
// C'est le seul paquet qui convertit entre coordonnées de plateau et
// coordonnées d'écran. Le noyau ne connaît que colonne et ligne ; si une
// coordonnée d'écran apparaît ailleurs, c'est un défaut.
package render

import (
	"errors"
	"fmt"
)

// Dimensions du losange, en unités du contrat de formes. Non modifiables : le
// rapport 2:1 est la projection, pas un réglage.
const (
	TileWidth  = 64
	TileHeight = 32
)

// ShapesVersion est la version du contrat que ce binaire sait lire. Un plugin
// écrit contre une version inconnue est refusé plutôt que lu de travers.
const ShapesVersion = 3

// StrokeType énumère les quatre primitives. Le vocabulaire est volontairement
// pauvre : tout ce qui est livré avec le jeu s'y exprime, et une primitive de
// plus entre dans le contrat public sans pouvoir en ressortir.
type StrokeType string

// Les quatre primitives. StrokePrism est réservé au rôle bâtiment et ne porte
// qu'une hauteur : son emprise est le losange de la case, jamais déclarée.
const (
	StrokePolygon StrokeType = "polygon"
	StrokeCircle  StrokeType = "circle"
	StrokeSegment StrokeType = "segment"
	StrokePrism   StrokeType = "prism"
)

// Point est une coordonnée dans le repère d'une forme : origine au point
// d'ancrage au sol, y vers le haut.
type Point struct {
	X int
	Y int
}

// UnmarshalTOML lit un point écrit en paire, « [-6, 0] ».
//
// Le contrat le veut ainsi : une liste de sommets s'écrit alors
// « [[-7, 0], [7, 0], [5, 22]] », lisible d'un coup d'œil, là où une suite de
// tables « {x = -7, y = 0} » noierait la géométrie sous la syntaxe.
func (p *Point) UnmarshalTOML(v any) error {
	paire, ok := v.([]any)
	if !ok || len(paire) != 2 {
		return fmt.Errorf("point: attendu une paire [x, y], reçu %v", v)
	}

	for i, brut := range paire {
		n, ok := brut.(int64)
		if !ok {
			return fmt.Errorf("point: %v n'est pas un entier", brut)
		}
		if i == 0 {
			p.X = int(n)
		} else {
			p.Y = int(n)
		}
	}
	return nil
}

// Stroke est une primitive paramétrée. Les champs inutiles au type restent à
// zéro, comme pour core.Effect et pour la même raison : un enregistrement plat
// se lit et se valide sans hiérarchie de types.
type Stroke struct {
	Type             StrokeType `toml:"type"`
	Points           []Point    `toml:"points"`
	Center           Point      `toml:"center"`
	Radius           int        `toml:"radius"`
	From             Point      `toml:"from"`
	To               Point      `toml:"to"`
	Thickness        int        `toml:"thickness"`
	Height           int        `toml:"height"`
	Color            string     `toml:"color"`
	Outline          string     `toml:"outline"`
	OutlineThickness int        `toml:"outline_thickness"`
	Opacity          int        `toml:"opacity"`
}

// Role détermine ce qu'un plugin a le droit de redéfinir.
//
// Il n'existe pas de rôle « sol » : le losange est tracé par le moteur et ne se
// redessine pas. Rue et zones sont des noms de couleurs, pas des formes.
//
// La raison est mécanique. Un losange identique partout, c'est une géométrie
// construite une fois pour toutes les cases, un tri en profondeur qui reste
// colonne + ligne, et un ToBoard qui reste une formule fermée. Une tuile de
// forme libre imposerait une boîte englobante par case, un test de survol forme
// par forme, et invaliderait le tri par ordre de peintre.
type Role string

// Les trois rôles. Le sol n'en est pas un : rue, zone_ouverte et zone_fermee
// sont des noms de couleurs, et le losange est tracé par le moteur.
const (
	RolePiece    Role = "piece"
	RoleBuilding Role = "building"
	RoleMarker   Role = "marker"
)

// Template borne ce qu'une forme a le droit d'occuper.
//
// Vérifié au chargement et pas seulement à la publication : une forme qui
// déborde masque les cases derrière elle, ce qui est un avantage de jeu déguisé
// en habillage. Un plugin local est donc soumis au même contrôle qu'un plugin
// catalogué.
//
// Les bornes portent sur le plan du rôle — celui du sol pour une forme plate et
// pour l'emprise d'un prisme, le plan vertical pour un pion. MaxHeight est
// comptée à part parce qu'un prisme mêle les deux : son emprise est au sol, son
// élévation ne l'est pas.
type Template struct {
	Plane      Plane
	XMin, XMax int
	YMin, YMax int
	MaxStrokes int
	MaxHeight  int
}

// Plane distingue les deux repères. Les confondre est l'erreur classique du
// contrat : un losange au sol occupe y de -16 à 16, un pion s'élève de 0 à 40,
// et les deux plages n'ont rien à voir.
type Plane string

// Les deux repères.
const (
	PlaneGround   Plane = "ground"
	PlaneVertical Plane = "vertical"
)

// templates borne ce que chaque rôle a le droit d'occuper.
var templates = map[Role]Template{
	RolePiece:  {PlaneVertical, -24, 24, 0, 40, 24, 0},
	RoleMarker: {PlaneGround, -24, 24, -12, 12, 8, 0},
	// Un bâtiment ne déclare qu'une hauteur et une couleur : son emprise est le
	// losange de la case. Une emprise libre permettrait de déborder sur les
	// voisines et de masquer ce que l'adversaire doit voir.
	RoleBuilding: {PlaneVertical, 0, 0, 0, 0, 1, 24},
}

// Shape est un dessin nommé, plus ses variantes d'état facultatives.
type Shape struct {
	// La clé de la table porte le nom : « [shape.fugitive] » nomme la forme,
	// et le champ le reçoit au chargement.
	Name string `toml:"-"`

	Role Role `toml:"role"`

	// Les tags sont explicites et non déduits du nom du champ. La déduction
	// est insensible à la casse mais pas à la langue : elle a laissé passer
	// des formes vides quand « trait » est devenu « stroke », sans que rien
	// ne le signale — un TOML dont on ignore une table se décode sans erreur.
	Strokes  []Stroke            `toml:"stroke"`
	Variants map[string][]Stroke `toml:"variant"`
}

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

// ShapeSet rassemble les formes et la palette effectivement actives, après
// application des surcharges.
type ShapeSet struct {
	Shapes  map[string]Shape
	Palette Palette
}

// Validate applique les contrôles du contrat : gabarit, plafond de traits,
// nombre de sommets, résolution des couleurs, absence de valeur hexadécimale.
//
// Renvoie tous les manquements plutôt que le premier : quelqu'un qui met au
// point un plugin veut la liste, pas un aller-retour par erreur.
func (j *ShapeSet) Validate() []error {
	return []error{errors.New("à implémenter : étape 6")}
}

// Merge applique une surcharge partielle sur le jeu de base.
//
// Un plugin ne déclare que ce qu'il remplace, le reste retombe sur le contenu
// livré — sans quoi changer un seul pion obligerait à livrer les quarante
// formes, et personne ne le ferait. Deux plugins qui redéfinissent la même
// forme sont un conflit, jamais un écrasement silencieux.
func (j *ShapeSet) Merge(nom string, autre *ShapeSet) error {
	return errors.New("à implémenter : étape 6")
}

// RimColor est le liseré que le moteur pose sous le contour des pions et des
// marqueurs. Un plugin n'a rien à en déclarer, et ne peut pas le retirer.
//
// Un contour seul ne suffit pas, et c'est contre-intuitif. Il tient contre le
// sol, qui est clair, mais un pion se dessine par-dessus les cubes situés
// devant lui : sa moitié supérieure est en permanence sur du bâti sombre, où un
// contour sombre ne se voit plus — 1,25 sur une face latérale. L'inverse est
// vrai d'un contour clair, qui meurt sur la rue. Aucune couleur unique ne
// couvre une plage de fonds qui va de 15 à 210 en luminance.
//
// Les deux ensemble la couvrent : quel que soit le fond, l'un des deux traits
// tranche, et le pire cas remonte de 1,10 à 5,03.
//
// Valeur fixe et non nom de palette : c'est de l'éclairage, au même titre que
// les coefficients de faces d'un prisme. Une palette qui pourrait la déplacer
// pourrait rendre les pions illisibles, ce que le liseré existe pour empêcher.
const RimColor = "#e8e2d4"

// RimWidth est l'épaisseur du liseré, en unités de contrat. Comme toute
// épaisseur de trait, elle est encadrée par StrokeWidth.
const RimWidth = 2

// MinStrokePixels et MaxStrokeRatio encadrent l'épaisseur de tout trait de
// contour, celui d'un plugin comme le liseré du moteur.
//
// Les deux bornes traitent le même défaut par ses deux bouts, et l'une sans
// l'autre ne fait que le déplacer. Sans plancher, une épaisseur mise à l'échelle
// passe sous le pixel au dézoom : l'antialiasing la mêle à ce qu'elle devait
// séparer, et le contraste réel s'effondre bien avant la valeur calculée. Sans
// plafond, une épaisseur fixe finit par occuper la forme entière — à 24 pixels
// par case, la tête d'un pion en fait quatre et demi, et deux pixels de liseré
// n'en détachent plus rien : ils avalent la couleur, qui est le seul signal
// d'appartenance à un camp.
//
// Le plafond porte sur la plus petite dimension du trait et non sur la case ni
// sur la forme entière : c'est ce trait-là que le contour doit laisser voir, et
// un trait long et fin serait avalé si l'on bornait par sa plus grande.
//
// Chaque trait est encadré pour lui-même, jamais leur somme : borner le total
// reviendrait à écraser le liseré dès que le plafond mord, alors que c'est lui
// qui porte le contraste sur le bâti.
const (
	MinStrokePixels = 1
	MaxStrokeRatio  = 6
)

// MinRenderScale est le rapport en dessous duquel le rendu ne garantit plus la
// lisibilité des pions. Vaut 24 pixels par case.
//
// Le plancher est un minimum en pixels : il ne dépend pas de la taille du trait
// mais de l'échelle, et finit donc par commander partout. En dessous d'un
// quart, la bordure d'un pion l'emporte sur son remplissage — 47 % de noyau à
// 16 pixels par case — et la couleur cesse de dire à quel camp il appartient.
//
// La vue isométrique ne cherche pas à montrer le plateau entier : elle garde
// une échelle de travail et défile en suivant le jeu, la vue d'ensemble étant
// portée par la carte à plat affichée à côté d'elle. Ce plancher n'est donc pas
// un compromis d'affichage mais un garde-fou : l'échelle est bornée par le haut
// bien avant de l'atteindre, et il ne se déclenche pas en jeu normal.
//
// La borne haute est le champ du pion sélectionné, que la vue doit contenir en
// entier : Span(2*portée+1) donne la place qu'il demande. Elle vaut 55 pixels
// par case dans le pire cas prévu — une fenêtre de 1280 sur le plus grand
// préréglage —, soit le double du plancher.
//
// Cette portée est la portée **nominale** du préréglage, jamais celle du tour
// en cours. Une capacité qui double la vue ferait sinon tomber le plafond d'un
// tiers, donc dézoomerait tout le plateau au moment de son déclenchement pour
// le rezoomer au tour suivant : une caméra qui bouge seule quand on active une
// capacité est déroutante, et c'est la carte à plat qui montre ce que la vue
// isométrique ne peut plus contenir.
const MinRenderScale = 0.375

// StrokeWidth donne l'épaisseur d'un trait à l'écran, en pixels.
//
// unites est l'épaisseur déclarée, echelle le rapport entre la case rendue et
// ses 64 pixels nominaux, minForme la plus petite dimension de la forme en
// unités de contrat.
func StrokeWidth(unites int, echelle float64, minForme int) float64 {
	e := float64(unites) * echelle
	if plafond := float64(minForme) * echelle / MaxStrokeRatio; e > plafond {
		e = plafond
	}
	return max(e, MinStrokePixels)
}

// GroundGrainAmplitude est l'écart de luminosité appliqué au sol, en pourcents.
//
// Un plateau de couleurs pleines est plat à l'œil sur seize cents cases. Le
// grain est dérivé de la position et de la graine, donc stable : le retirer au
// sort à l'affichage produirait un scintillement au défilement, pire que l'uni.
//
// Luminosité seule, jamais la teinte, et le sol seulement — pions et bâtiments
// gardent leur couleur exacte, qui doit rester identifiable d'un coup d'œil.
// Coupé en vue à plat, où l'uniformité aide à lire les surcouches.
const GroundGrainAmplitude = 5

// GroundGrain renvoie l'écart de luminosité d'une case, dans
// [-GroundGrainAmplitude, +GroundGrainAmplitude].
//
// Ce n'est pas une primitive du contrat et aucun plugin n'a à en tenir compte :
// tous en bénéficient, y compris ceux qui ne changent que la palette.
func GroundGrain(graine int64, colonne, ligne int) int {
	// à implémenter : étape 7
	return 0
}

// ToScreen projette une case du plateau en coordonnées d'écran.
func ToScreen(colonne, ligne int) (x, y int) {
	return (colonne - ligne) * TileWidth / 2, (colonne + ligne) * TileHeight / 2
}

// Span donne l'emprise projetée d'un carré de cases, en unités du contrat.
//
// À écrire partout où l'on veut savoir quelle place occupe un nombre de cases :
// dimensionner une fenêtre, tenir un champ de vision, comparer une forme au
// terrain. Le rapport 2:1 fait qu'un carré de n cases est deux fois plus large
// que haut, et l'oublier donne un résultat plausible et faux — le facteur écrit
// à la main s'est trompé trois fois de suite en écrivant ce paquet.
func Span(cases int) (largeur, hauteur int) {
	return cases * TileWidth, cases * TileHeight
}

// ToBoard est la transformation inverse, pour le clic.
//
// L'arrondi doit tomber sur la case que l'utilisateur croit viser, y compris
// près des arêtes du losange où le calcul naïf se trompe d'une case.
func ToBoard(x, y int) (colonne, ligne int) {
	// à implémenter : étape 7
	return 0, 0
}
