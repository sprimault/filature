// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"errors"
	"fmt"
)

// Side désigne un camp. Le troisième cas n'existe pas : la partie se joue à
// deux, un plugin ne peut pas en ajouter un.
type Side string

// Les deux camps.
const (
	SideFugitive   Side = "fugitive"
	SideInspectors Side = "inspectors"
)

// Phase découpe le tour. La mise en place est une phase comme une autre pour
// que le placement passe par le journal et soit donc rejouable.
type Phase string

// Les cinq phases, dans l'ordre où elles s'enchaînent. Les inspecteurs jouent
// avant le fugitif : c'est ce qui compense leurs trois déplacements contre un.
const (
	PhaseFugitiveSetup   Phase = "fugitive_setup"
	PhaseInspectorsSetup Phase = "inspectors_setup"
	PhaseInspectors      Phase = "inspectors"
	PhaseFugitive        Phase = "fugitive"
	PhaseOver            Phase = "over"
)

// Trail est le passage du fugitif sur une case. Elle n'est jamais visible à
// distance : un inspecteur la découvre en occupant la case ou une case
// orthogonalement adjacente.
type Trail struct {
	Turn      int       `json:"turn"`
	Direction Direction `json:"direction"`
}

// Fugitive porte l'état du camp caché. SealedZone est l'information la plus
// sensible du jeu : elle ne doit jamais franchir ViewFor côté inspecteurs.
type Fugitive struct {
	Position      Position `json:"position"`
	Stamina       int      `json:"stamina"`
	Visible       bool     `json:"spotted"`
	SealedZone    int      `json:"sealed_zone"`
	TurnsInZone   int      `json:"turns_in_zone"`
	SilenceBought bool     `json:"silence_bought"`

	// StepsTaken est remis à zéro à chaque tour, comme celui des
	// inspecteurs. Le fugitif n'a pas de quota de pions, mais une mobilité
	// qu'un double déplacement porte à deux.
	StepsTaken int `json:"steps_taken"`
}

// CrimeScene est un lieu de meurtre. Contrairement à une trace, elle est connue des
// deux camps dès qu'elle existe : c'est ce que le fugitif achète en payant.
//
// Elle ne contraint personne. Les inspecteurs sont libres de l'ignorer, et
// c'est tout l'intérêt — ils savent où il était, ils doivent parier sur ce que
// ça dit de sa destination.
type CrimeScene struct {
	Position Position `json:"position"`
	Turn     int      `json:"turn"`
}

// Inspector porte un pion et sa capacité, utilisable une fois par partie.
//
// Aucun bonus n'est stocké ici : portée, mobilité et rayon de détection se
// lisent par RangeOf, MobilityOf et TrailRadiusOf, qui agrègent les effets en
// cours. Un entier figé dans le pion serait un cache que rien ne réconcilie, et
// un plugin qui invente une capacité à durée n'aurait pas de champ où la
// ranger.
type Inspector struct {
	Position    Position `json:"position"`
	Ability     string   `json:"ability"`
	AbilityUsed bool     `json:"ability_used"`

	// StepsTaken est remis à zéro à chaque tour. Il tient le quota :
	// savoir combien de pions ont bougé ne suffit pas, il faut savoir
	// lesquels, sinon le même pion consomme les trois places.
	StepsTaken int `json:"steps_taken"`
}

// Game porte l'intégralité de l'état.
//
// Toute information dérivable — visibilité, coups légaux, carte de croyance —
// est recalculée, jamais stockée : c'est ce qui garantit que rejouer le journal
// reconstruit exactement le même état.
type Game struct {
	Seed       int64       `json:"seed"`
	Settings   Settings    `json:"settings"`
	Board      Board       `json:"-"`
	Turn       int         `json:"turn"`
	Phase      Phase       `json:"phase"`
	Fugitive   Fugitive    `json:"fugitive"`
	Inspectors []Inspector `json:"inspectors"`

	Trails      map[Position]Trail `json:"trails"`
	CrimeScenes []CrimeScene       `json:"crime_scenes"`

	// Roadblocks et Openings sont les deux altérations du terrain, en tours
	// d'expiration. Le plateau est en lecture seule — c'est la condition du
	// plateau infini — donc ce qui le modifie vit ici, par-dessus.
	Roadblocks map[Position]int `json:"roadblocks"`
	Openings   map[Position]int `json:"openings"`

	ClosedZones []int `json:"closed_zones"`

	// AbilityPlayed dit qu'une capacité a déjà été déclenchée ce tour. La
	// règle en autorise une par tour en plus d'une par pion et par partie :
	// le drapeau du pion ne suffit donc pas.
	AbilityPlayed bool `json:"ability_played"`

	// ExpenseUses compte les emplois des dépenses plafonnées. Générique
	// parce que « usages » est un champ du contrat de plugin : le noyau n'a
	// pas à savoir que celle qui plafonne à deux s'appelle meurtre.
	ExpenseUses map[Expense]int `json:"expense_uses"`

	// PendingEffects est la file des differer posés, résolue en fin de tour
	// avant le test de fin de partie. Elle se sérialise avec le reste : une
	// reprise qui la perdrait escamoterait un barrage déjà annoncé.
	PendingEffects []PendingEffect `json:"pending_effects"`

	// ActiveEffects porte ce qui modifie temporairement un pion ou le fugitif.
	ActiveEffects []ActiveEffect `json:"active_effects"`

	// ForcedOutcome est le seul moyen qu'un plugin termine une partie sans que le
	// noyau connaisse sa condition de victoire. Outcome la consulte d'abord.
	ForcedOutcome *Outcome `json:"forced_outcome,omitempty"`

	// annulations double le journal, une entrée par coup. Elle ne se sérialise
	// pas — ce sont des fermetures — donc une partie rechargée ne s'annule
	// pas : elle se rejoue, et c'est ce qui vérifie en continu que le journal
	// reste suffisant.
	annulations [][]func()

	Journal    []Move    `json:"journal"`
	Extensions *Registry `json:"-"`
	alea       *Random
}

// La distance minimale entre le noyau central et les zones d'extraction reste
// ouverte : elle dépend du taux de rues que la génération obtiendra, donc de
// l'étape 3. Ce rayon la borne par le haut en attendant.

// CentreRadius borne la zone où le fugitif est tiré au sort.
//
// Assez large pour que les inspecteurs ne puissent pas la couvrir, assez
// resserrée pour qu'aucune sortie ne soit à portée immédiate : il doit avoir à
// traverser, sinon les six points d'extraction ne servent à rien.
const CentreRadius = 5

// NewGame prépare une partie au premier coup de placement.
//
// Le plateau est reçu, pas fabriqué. Le noyau applique des règles à un terrain,
// il ne le produit pas — c'est aussi ce qui rend une partie montable sur un
// plateau d'essai sans passer par la génération, et ce qui laissera la
// génération par tuiles se substituer sans qu'une règle bouge.
//
// La position du fugitif est tirée au sort, comme le veut la mise en place : il
// choisit sa zone d'extraction, jamais sa case de départ.
func NewGame(plateau Board, graine int64, p Settings, r *Registry) (*Game, error) {
	if plateau == nil {
		return nil, errors.New("plateau manquant")
	}
	if r == nil {
		return nil, errors.New("registre manquant")
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("parametres: %w", err)
	}

	depart, err := fugitiveStart(plateau, graine, p)
	if err != nil {
		return nil, err
	}

	return &Game{
		Seed:     graine,
		Settings: p,
		Board:    plateau,
		Phase:    PhaseFugitiveSetup,
		Fugitive: Fugitive{
			Position:   depart,
			Stamina:    p.Stamina,
			SealedZone: -1,
		},
		Extensions: r,
	}, nil
}

// fugitiveStart tire sa case dans le noyau central.
//
// Le flux nommé isole ce tirage : ajouter un dé ailleurs ne doit pas déplace
// le fugitif d'une partie qu'on rejoue.
func fugitiveStart(plateau Board, graine int64, p Settings) (Position, error) {
	milieu := p.Size / 2
	cases := plateau.CellsWithin(Position{Column: milieu, Row: milieu}, CentreRadius)
	if len(cases) == 0 {
		return Position{}, errors.New("aucune rue au centre du plateau")
	}
	return cases[NewRandom(graine, "setup").Int(len(cases))], nil
}

// IsWalkable dit si une case peut être occupée et traversée du regard.
//
// Trois couches, dans cet ordre : le terrain, les percements d'un ouvrir_case,
// les barrages. Un barrage l'emporte sur tout — sans quoi rouvrir une case déjà
// barrée dépendrait de l'ordre d'application, et le rejeu du journal cesserait
// d'être reproductible.
func (p *Game) IsWalkable(pos Position) bool {
	if _, barre := p.Roadblocks[pos]; barre {
		return false
	}
	if _, ouverte := p.Openings[pos]; ouverte {
		return true
	}
	return p.Board.IsStreet(pos)
}

// L'énumération des coups légaux vit dans legalmoves.go, la résolution de fin
// de tour et le décompte des contacts dans turn.go.
