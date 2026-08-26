// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Package rendu porte la projection isométrique et le vocabulaire géométrique
// des plugins d'apparence.
//
// C'est le seul paquet qui convertit entre coordonnées de plateau et
// coordonnées d'écran. Le noyau ne connaît que colonne et ligne ; si une
// coordonnée d'écran apparaît ailleurs, c'est un défaut.
package render

import "errors"

// Dimensions du losange, en unités du contrat de formes. Non modifiables : le
// rapport 2:1 est la projection, pas un réglage.
const (
	LargeurCase = 64
	HauteurCase = 32
)

// ShapesVersion est la version du contrat que ce binaire sait lire. Un plugin
// écrit contre une version inconnue est refusé plutôt que lu de travers.
const ShapesVersion = 2

// TypeTrait énumère les quatre primitives. Le vocabulaire est volontairement
// pauvre : tout ce qui est livré avec le jeu s'y exprime, et une primitive de
// plus entre dans le contrat public sans pouvoir en ressortir.
type TypeTrait string

// Les quatre primitives. TraitPrisme est réservé au rôle bâtiment et ne porte
// qu'une hauteur : son emprise est le losange de la case, jamais déclarée.
const (
	TraitPolygone TypeTrait = "polygone"
	TraitCercle   TypeTrait = "cercle"
	TraitSegment  TypeTrait = "segment"
	TraitPrisme   TypeTrait = "prisme"
)

// Point est une coordonnée dans le repère d'une forme : origine au point
// d'ancrage au sol, y vers le haut.
type Point struct {
	X int
	Y int
}

// Trait est une primitive paramétrée. Les champs inutiles au type restent à
// zéro, comme pour core.Effect et pour la même raison : un enregistrement plat
// se lit et se valide sans hiérarchie de types.
type Trait struct {
	Type             TypeTrait `toml:"type"`
	Points           []Point   `toml:"points"`
	Centre           Point     `toml:"center"`
	Radius           int       `toml:"radius"`
	De               Point     `toml:"from"`
	A                Point     `toml:"to"`
	Epaisseur        int       `toml:"thickness"`
	Emprise          []Point   `toml:"footprint"`
	Hauteur          int       `toml:"height"`
	Couleur          string    `toml:"color"`
	Contour          string    `toml:"outline"`
	EpaisseurContour int       `toml:"outline_thickness"`
	Opacite          int       `toml:"opacity"`
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
	RolePion     Role = "pion"
	RoleBatiment Role = "batiment"
	RoleMarqueur Role = "marqueur"
)

// Gabarit borne ce qu'une forme a le droit d'occuper.
//
// Vérifié au chargement et pas seulement à la publication : une forme qui
// déborde masque les cases derrière elle, ce qui est un avantage de jeu déguisé
// en habillage. Un plugin local est donc soumis au même contrôle qu'un plugin
// catalogué.
//
// Les bornes portent sur le plan du rôle — celui du sol pour une forme plate et
// pour l'emprise d'un prisme, le plan vertical pour un pion. HauteurMax est
// comptée à part parce qu'un prisme mêle les deux : son emprise est au sol, son
// élévation ne l'est pas.
type Gabarit struct {
	Plan       Plan
	XMin, XMax int
	YMin, YMax int
	TraitsMax  int
	HauteurMax int
}

// Plan distingue les deux repères. Les confondre est l'erreur classique du
// contrat : un losange au sol occupe y de -16 à 16, un pion s'élève de 0 à 40,
// et les deux plages n'ont rien à voir.
type Plan string

// Les deux repères.
const (
	PlanSol      Plan = "sol"
	PlanVertical Plan = "vertical"
)

// gabarits borne ce que chaque rôle a le droit d'occuper.
var gabarits = map[Role]Gabarit{
	RolePion:     {PlanVertical, -24, 24, 0, 40, 24, 0},
	RoleMarqueur: {PlanSol, -24, 24, -12, 12, 8, 0},
	// Un bâtiment ne déclare qu'une hauteur et une couleur : son emprise est le
	// losange de la case. Une emprise libre permettrait de déborder sur les
	// voisines et de masquer ce que l'adversaire doit voir.
	RoleBatiment: {PlanVertical, 0, 0, 0, 0, 1, 24},
}

// Forme est un dessin nommé, plus ses variantes d'état facultatives.
type Forme struct {
	Name      string
	Role      Role
	Traits    []Trait
	Variantes map[string][]Trait
}

// Palette associe des noms à des couleurs. Une forme ne référence que des noms :
// c'est ce qui permet de reteinter tout le jeu sans toucher à la géométrie, et
// à une forme tierce de suivre la palette active sans rien savoir d'elle.
type Palette map[string]string

// CouleursObligatoires est le socle sur lequel toute forme peut compter. Une
// palette qui n'en couvre pas la totalité est refusée : une couleur manquante
// se verrait à l'écran comme un trou, plusieurs écrans plus loin.
var CouleursObligatoires = []string{
	"rue", "batiment", "zone_ouverte", "zone_fermee",
	"fugitif_principal", "fugitif_detail",
	"inspecteur_principal", "inspecteur_detail",
	"trace", "barrage", "scene",
}

// Jeu rassemble les formes et la palette effectivement actives, après
// application des surcharges.
type Jeu struct {
	Formes  map[string]Forme
	Palette Palette
}

// Validate applique les contrôles du contrat : gabarit, plafond de traits,
// nombre de sommets, résolution des couleurs, absence de valeur hexadécimale.
//
// Renvoie tous les manquements plutôt que le premier : quelqu'un qui met au
// point un plugin veut la liste, pas un aller-retour par erreur.
func (j *Jeu) Validate() []error {
	return []error{errors.New("à implémenter : étape 6")}
}

// Merge applique une surcharge partielle sur le jeu de base.
//
// Un plugin ne déclare que ce qu'il remplace, le reste retombe sur le contenu
// livré — sans quoi changer un seul pion obligerait à livrer les quarante
// formes, et personne ne le ferait. Deux plugins qui redéfinissent la même
// forme sont un conflit, jamais un écrasement silencieux.
func (j *Jeu) Merge(nom string, autre *Jeu) error {
	return errors.New("à implémenter : étape 6")
}

// AmplitudeGrainSol est l'écart de luminosité appliqué au sol, en pourcents.
//
// Un plateau de couleurs pleines est plat à l'œil sur seize cents cases. Le
// grain est dérivé de la position et de la graine, donc stable : le retirer au
// sort à l'affichage produirait un scintillement au défilement, pire que l'uni.
//
// Luminosité seule, jamais la teinte, et le sol seulement — pions et bâtiments
// gardent leur couleur exacte, qui doit rester identifiable d'un coup d'œil.
// Moveé en vue de débogage, où l'uniformité aide à lire les surcouches.
const AmplitudeGrainSol = 5

// GroundGrain renvoie l'écart de luminosité d'une case, dans
// [-AmplitudeGrainSol, +AmplitudeGrainSol].
//
// Ce n'est pas une primitive du contrat et aucun plugin n'a à en tenir compte :
// tous en bénéficient, y compris ceux qui ne changent que la palette.
func GroundGrain(graine int64, colonne, ligne int) int {
	// à implémenter : étape 7
	return 0
}

// ToScreen projette une case du plateau en coordonnées d'écran.
func ToScreen(colonne, ligne int) (x, y int) {
	return (colonne - ligne) * LargeurCase / 2, (colonne + ligne) * HauteurCase / 2
}

// ToBoard est la transformation inverse, pour le clic.
//
// L'arrondi doit tomber sur la case que l'utilisateur croit viser, y compris
// près des arêtes du losange où le calcul naïf se trompe d'une case.
func ToBoard(x, y int) (colonne, ligne int) {
	// à implémenter : étape 7
	return 0, 0
}
