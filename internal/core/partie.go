// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"errors"
	"fmt"
)

// Acteur désigne un camp. Le troisième cas n'existe pas : la partie se joue à
// deux, un plugin ne peut pas en ajouter un.
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

	// DeplacementsFaits est remis à zéro à chaque tour, comme celui des
	// inspecteurs. Le fugitif n'a pas de quota de pions, mais une mobilité
	// qu'un double déplacement porte à deux.
	DeplacementsFaits int `json:"deplacements_faits"`
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
// un plugin qui invente une capacité à durée n'aurait pas de champ où la
// ranger.
type Inspecteur struct {
	Position         Position `json:"position"`
	Capacite         string   `json:"capacite"`
	CapaciteUtilisee bool     `json:"capacite_utilisee"`

	// DeplacementsFaits est remis à zéro à chaque tour. Il tient le quota :
	// savoir combien de pions ont bougé ne suffit pas, il faut savoir
	// lesquels, sinon le même pion consomme les trois places.
	DeplacementsFaits int `json:"deplacements_faits"`
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

	ZonesFermees []int `json:"zones_fermees"`

	// CapaciteJouee dit qu'une capacité a déjà été déclenchée ce tour. La
	// règle en autorise une par tour en plus d'une par pion et par partie :
	// le drapeau du pion ne suffit donc pas.
	CapaciteJouee bool `json:"capacite_jouee"`

	// UsagesDepense compte les emplois des dépenses plafonnées. Générique
	// parce que « usages » est un champ du contrat de plugin : le noyau n'a
	// pas à savoir que celle qui plafonne à deux s'appelle meurtre.
	UsagesDepense map[Depense]int `json:"usages_depense"`

	// EffetsEnAttente est la file des differer posés, résolue en fin de tour
	// avant le test de fin de partie. Elle se sérialise avec le reste : une
	// reprise qui la perdrait escamoterait un barrage déjà annoncé.
	EffetsEnAttente []EffetEnAttente `json:"effets_en_attente"`

	// EffetsActifs porte ce qui modifie temporairement un pion ou le fugitif.
	EffetsActifs []EffetActif `json:"effets_actifs"`

	// FinForcee est le seul moyen qu'un plugin termine une partie sans que le
	// noyau connaisse sa condition de victoire. Resultat la consulte d'abord.
	FinForcee *Resultat `json:"fin_forcee,omitempty"`

	// annulations double le journal, une entrée par coup. Elle ne se sérialise
	// pas — ce sont des fermetures — donc une partie rechargée ne s'annule
	// pas : elle se rejoue, et c'est ce qui vérifie en continu que le journal
	// reste suffisant.
	annulations [][]func()

	Journal    []Coup    `json:"journal"`
	Extensions *Registre `json:"-"`
	alea       *Alea
}

// La distance minimale entre le noyau central et les zones d'extraction reste
// ouverte : elle dépend du taux de rues que la génération obtiendra, donc de
// l'étape 3. Ce rayon la borne par le haut en attendant.

// RayonNoyauCentral borne la zone où le fugitif est tiré au sort.
//
// Assez large pour que les inspecteurs ne puissent pas la couvrir, assez
// resserrée pour qu'aucune sortie ne soit à portée immédiate : il doit avoir à
// traverser, sinon les six points d'extraction ne servent à rien.
const RayonNoyauCentral = 5

// Nouvelle prépare une partie au premier coup de placement.
//
// Le plateau est reçu, pas fabriqué. Le noyau applique des règles à un terrain,
// il ne le produit pas — c'est aussi ce qui rend une partie montable sur un
// plateau d'essai sans passer par la génération, et ce qui laissera la
// génération par tuiles se substituer sans qu'une règle bouge.
//
// La position du fugitif est tirée au sort, comme le veut la mise en place : il
// choisit sa zone d'extraction, jamais sa case de départ.
func Nouvelle(plateau Plateau, graine int64, p Parametres, r *Registre) (*Partie, error) {
	if plateau == nil {
		return nil, errors.New("plateau manquant")
	}
	if r == nil {
		return nil, errors.New("registre manquant")
	}
	if err := p.Valider(); err != nil {
		return nil, fmt.Errorf("parametres: %w", err)
	}

	depart, err := departDuFugitif(plateau, graine, p)
	if err != nil {
		return nil, err
	}

	return &Partie{
		Graine:     graine,
		Parametres: p,
		Plateau:    plateau,
		Phase:      PhasePlacementFugitif,
		Fugitif: Fugitif{
			Position:    depart,
			Resistance:  p.Resistance,
			ZoneScellee: -1,
		},
		Extensions: r,
	}, nil
}

// departDuFugitif tire sa case dans le noyau central.
//
// Le flux nommé isole ce tirage : ajouter un dé ailleurs ne doit pas déplacer
// le fugitif d'une partie qu'on rejoue.
func departDuFugitif(plateau Plateau, graine int64, p Parametres) (Position, error) {
	milieu := p.Cote / 2
	cases := plateau.CasesDans(Position{Colonne: milieu, Ligne: milieu}, RayonNoyauCentral)
	if len(cases) == 0 {
		return Position{}, errors.New("aucune rue au centre du plateau")
	}
	return cases[NouvelAlea(graine, "placement").Entier(len(cases))], nil
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

// L'énumération des coups légaux vit dans coupslegaux.go, la résolution de fin
// de tour et le décompte des contacts dans tour.go.
