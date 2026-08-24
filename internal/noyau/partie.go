// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import "errors"

// Acteur désigne un camp. Le troisième cas n'existe pas : la partie se joue à
// deux, un greffon ne peut pas en ajouter un.
type Acteur string

// Les deux camps.
const (
	CampFugitif     Acteur = "fugitif"
	CampInspecteurs Acteur = "inspecteurs"
)

// Phase découpe le tour. La mise en place est une phase comme une autre pour
// que le placement passe par le journal et soit donc rejouable.
type Phase string

// Les cinq phases, dans l'ordre où elles s'enchaînent. Les inspecteurs jouent
// avant le fugitif : c'est ce qui compense leurs trois déplacements contre un.
const (
	PhasePlacementFugitif     Phase = "placement_fugitif"
	PhasePlacementInspecteurs Phase = "placement_inspecteurs"
	PhaseInspecteurs          Phase = "inspecteurs"
	PhaseFugitif              Phase = "fugitif"
	PhaseTerminee             Phase = "terminee"
)

// Trace est le passage du fugitif sur une case. Elle n'est jamais visible à
// distance : un inspecteur la découvre en occupant la case ou une case
// orthogonalement adjacente.
type Trace struct {
	Tour      int       `json:"tour"`
	Direction Direction `json:"direction"`
}

// Fugitif porte l'état du camp caché. ZoneScellee est l'information la plus
// sensible du jeu : elle ne doit jamais franchir VuePour côté inspecteurs.
type Fugitif struct {
	Position      Position `json:"position"`
	Resistance    int      `json:"resistance"`
	Visible       bool     `json:"visible"`
	ZoneScellee   int      `json:"zone_scellee"`
	ToursDansZone int      `json:"tours_dans_zone"`
	SilenceAchete bool     `json:"silence_achete"`
	Meurtres      int      `json:"meurtres"`
}

// Scene est un lieu de meurtre. Contrairement à une trace, elle est connue des
// deux camps dès qu'elle existe : c'est ce que le fugitif achète en payant.
//
// Elle ne contraint personne. Les inspecteurs sont libres de l'ignorer, et
// c'est tout l'intérêt — ils savent où il était, ils doivent parier sur ce que
// ça dit de sa destination.
type Scene struct {
	Position Position `json:"position"`
	Tour     int      `json:"tour"`
}

// Inspecteur porte un pion et sa capacité, utilisable une fois par partie.
//
// Aucun bonus n'est stocké ici : portée, mobilité et rayon de détection se
// lisent par PorteeDe, MobiliteDe et RayonTracesDe, qui agrègent les effets en
// cours. Un entier figé dans le pion serait un cache que rien ne réconcilie, et
// un greffon qui invente une capacité à durée n'aurait pas de champ où la
// ranger.
type Inspecteur struct {
	Position         Position `json:"position"`
	Capacite         string   `json:"capacite"`
	CapaciteUtilisee bool     `json:"capacite_utilisee"`
}

// Partie porte l'intégralité de l'état.
//
// Toute information dérivable — visibilité, coups légaux, carte de croyance —
// est recalculée, jamais stockée : c'est ce qui garantit que rejouer le journal
// reconstruit exactement le même état.
type Partie struct {
	Graine      int64        `json:"graine"`
	Parametres  Parametres   `json:"parametres"`
	Plateau     Plateau      `json:"-"`
	Tour        int          `json:"tour"`
	Phase       Phase        `json:"phase"`
	Fugitif     Fugitif      `json:"fugitif"`
	Inspecteurs []Inspecteur `json:"inspecteurs"`

	Traces map[Position]Trace `json:"traces"`
	Scenes []Scene            `json:"scenes"`

	// Barrages et Ouvertures sont les deux altérations du terrain, en tours
	// d'expiration. Le plateau est en lecture seule — c'est la condition du
	// plateau infini — donc ce qui le modifie vit ici, par-dessus.
	Barrages   map[Position]int `json:"barrages"`
	Ouvertures map[Position]int `json:"ouvertures"`

	PionsDeplaces int   `json:"pions_deplaces"`
	ZonesFermees  []int `json:"zones_fermees"`

	// EffetsEnAttente est la file des differer posés, résolue en fin de tour
	// avant le test de fin de partie. Elle se sérialise avec le reste : une
	// reprise qui la perdrait escamoterait un barrage déjà annoncé.
	EffetsEnAttente []EffetEnAttente `json:"effets_en_attente"`

	// EffetsActifs porte ce qui modifie temporairement un pion ou le fugitif.
	EffetsActifs []EffetActif `json:"effets_actifs"`

	// FinForcee est le seul moyen qu'un greffon termine une partie sans que le
	// noyau connaisse sa condition de victoire. Resultat la consulte d'abord.
	FinForcee *Resultat `json:"fin_forcee,omitempty"`

	Journal    []Coup    `json:"journal"`
	Extensions *Registre `json:"-"`
	alea       *Alea
}

// Nouvelle prépare une partie au premier coup de placement. Le plateau est
// généré ici, donc la graine renvoyée peut différer de celle demandée.
func Nouvelle(graine int64, p Parametres, r *Registre) (*Partie, error) {
	return nil, errors.New("à implémenter : étape 1")
}

// EstPraticable dit si une case peut être occupée et traversée du regard.
//
// Trois couches, dans cet ordre : le terrain, les percements d'un ouvrir_case,
// les barrages. Un barrage l'emporte sur tout — sans quoi rouvrir une case déjà
// barrée dépendrait de l'ordre d'application, et le rejeu du journal cesserait
// d'être reproductible.
func (p *Partie) EstPraticable(pos Position) bool {
	if _, barre := p.Barrages[pos]; barre {
		return false
	}
	if _, ouverte := p.Ouvertures[pos]; ouverte {
		return true
	}
	return p.Plateau.EstRue(pos)
}

// CoupsLegaux énumère ce que l'acteur peut jouer dans la phase courante.
//
// C'est la seule source de vérité sur la légalité : l'interface s'en sert pour
// surligner les cases, l'IA pour explorer, le serveur pour valider ce qui
// arrive du réseau. Aucun de ces trois ne réimplémente la règle.
func (p *Partie) CoupsLegaux(a Acteur) []Coup {
	// à implémenter : étape 1
	return nil
}

// Appliquer joue un coup et fait avancer la phase. Un coup illégal est refusé
// sans modifier l'état : l'appelant peut réessayer sans repartir d'un
// instantané.
func (p *Partie) Appliquer(c Coup) error {
	return errors.New("à implémenter : étape 1")
}

// Annuler défait le dernier coup.
//
// Ce n'est pas un confort d'interface : c'est ce qui permet à l'IA d'explorer
// des milliers de positions sans copier l'état à chaque nœud. Toute
// modification d'état doit donc rester réversible, y compris celles d'un
// greffon.
func (p *Partie) Annuler() error {
	return errors.New("à implémenter : étape 1")
}

// resoudreFinDeTour enchaîne visibilité, contacts, traces, révélation,
// étranglement et test de fin. L'ordre est un contrat : le décompte des
// contacts a lieu après le déplacement du fugitif, pas avant.
func (p *Partie) resoudreFinDeTour() {
	// à implémenter : étape 1
}

// contacts compte les inspecteurs orthogonalement adjacents au fugitif. Les
// diagonales ne comptent pas, et le total est plafonné : être encerclé doit
// faire très mal sans être instantanément fatal.
func (p *Partie) contacts() int {
	// à implémenter : étape 1
	return 0
}
