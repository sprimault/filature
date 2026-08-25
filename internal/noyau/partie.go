// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import (
	"errors"
	"sort"
)

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
// un greffon qui invente une capacité à durée n'aurait pas de champ où la
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
	// parce que « usages » est un champ du contrat de greffon : le noyau n'a
	// pas à savoir que celle qui plafonne à deux s'appelle meurtre.
	UsagesDepense map[Depense]int `json:"usages_depense"`

	// EffetsEnAttente est la file des differer posés, résolue en fin de tour
	// avant le test de fin de partie. Elle se sérialise avec le reste : une
	// reprise qui la perdrait escamoterait un barrage déjà annoncé.
	EffetsEnAttente []EffetEnAttente `json:"effets_en_attente"`

	// EffetsActifs porte ce qui modifie temporairement un pion ou le fugitif.
	EffetsActifs []EffetActif `json:"effets_actifs"`

	// FinForcee est le seul moyen qu'un greffon termine une partie sans que le
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
	switch p.Phase {
	case PhasePlacementFugitif:
		if a == CampFugitif {
			return p.coupsSceller()
		}
	case PhasePlacementInspecteurs:
		if a == CampInspecteurs {
			return p.coupsPlacer()
		}
	case PhaseInspecteurs:
		if a == CampInspecteurs {
			return p.coupsInspecteurs()
		}
	case PhaseFugitif:
		if a == CampFugitif {
			return p.coupsFugitif()
		}
	}
	return nil
}

// PionsDeplaces compte les inspecteurs qui ont bougé ce tour.
//
// Calculé et non stocké : le total se déduit des pions, et deux sources pour
// un même chiffre finiraient par se contredire.
func (p *Partie) PionsDeplaces() int {
	n := 0
	for _, i := range p.Inspecteurs {
		if i.DeplacementsFaits > 0 {
			n++
		}
	}
	return n
}

// coupsSceller énumère les zones que le fugitif peut choisir à la mise en
// place. Sa case, elle, est tirée au sort : il ne la choisit jamais.
func (p *Partie) coupsSceller() []Coup {
	zones := p.Plateau.Zones()
	coups := make([]Coup, 0, len(zones))
	for _, z := range zones {
		coups = append(coups, Coup{Tour: p.Tour, Acteur: CampFugitif, Type: CoupPlacer, Zone: z.Numero})
	}
	return coups
}

// coupsPlacer énumère où poser le prochain inspecteur.
//
// La case du fugitif n'est pas retirée de la liste, bien qu'il soit déjà sur le
// plateau : l'en exclure dirait aux inspecteurs où il n'est pas, ce qui est une
// fuite exactement comme dire où il est. Un pion qui tombe dessus l'a trouvé
// par hasard, et c'est un résultat de partie, pas un coup illégal.
func (p *Partie) coupsPlacer() []Coup {
	if len(p.Inspecteurs) >= p.Parametres.Inspecteurs {
		return nil
	}
	rayon := p.Parametres.Cote / 2
	centre := Position{Colonne: rayon, Ligne: rayon}

	cases := p.Plateau.CasesDans(centre, rayon)
	coups := make([]Coup, 0, len(cases))
	for _, c := range cases {
		if !p.EstPraticable(c) {
			continue
		}
		coups = append(coups, Coup{
			Tour:    p.Tour,
			Acteur:  CampInspecteurs,
			Type:    CoupPlacer,
			Pion:    len(p.Inspecteurs),
			Arrivee: c,
		})
	}
	return coups
}

// coupsInspecteurs énumère déplacements et capacités de la phase.
//
// Le quota porte sur le nombre de pions distincts, pas sur le nombre de
// déplacements : un pion déjà entamé continue sur sa mobilité propre sans
// prendre une place de plus.
func (p *Partie) coupsInspecteurs() []Coup {
	var coups []Coup
	placeLibre := p.PionsDeplaces() < p.Parametres.PionsParTour

	for i := range p.Inspecteurs {
		entame := p.Inspecteurs[i].DeplacementsFaits > 0
		if !entame && !placeLibre {
			continue
		}
		if p.Inspecteurs[i].DeplacementsFaits >= p.MobiliteDe(CampInspecteurs, i) {
			continue
		}
		depart := p.Inspecteurs[i].Position
		for _, d := range Orthogonales {
			arrivee := depart.Avance(d)
			if !p.EstPraticable(arrivee) {
				continue
			}
			coups = append(coups, Coup{
				Tour: p.Tour, Acteur: CampInspecteurs, Type: CoupDeplacer,
				Pion: i, Depart: depart, Arrivee: arrivee,
			})
		}
	}

	coups = append(coups, p.coupsCapacite()...)
	return append(coups, Coup{Tour: p.Tour, Acteur: CampInspecteurs, Type: CoupFinDePhase})
}

// coupsCapacite énumère les capacités déclenchables ce tour.
//
// Les clés du registre sont triées avant parcours : l'ordre d'une map n'est pas
// stable en Go, et il déciderait ici de l'ordre des coups légaux — donc de ce
// qu'un rejeu doit retrouver.
func (p *Partie) coupsCapacite() []Coup {
	if p.Extensions == nil || p.CapaciteJouee {
		return nil
	}

	cles := make([]string, 0, len(p.Extensions.Capacites))
	for cle := range p.Extensions.Capacites {
		cles = append(cles, cle)
	}
	sort.Strings(cles)

	var coups []Coup
	for _, cle := range cles {
		c := p.Extensions.Capacites[cle]
		if c.Camp != CampInspecteurs || c.Passive || c.Declenchement != SurPhaseInspecteurs {
			continue
		}
		for i := range p.Inspecteurs {
			if p.Inspecteurs[i].Capacite != cle || p.Inspecteurs[i].CapaciteUtilisee {
				continue
			}
			coups = append(coups, Coup{
				Tour: p.Tour, Acteur: CampInspecteurs, Type: CoupCapacite,
				Pion: i, Capacite: cle,
			})
		}
	}
	return coups
}

// coupsFugitif énumère déplacements, dépenses et changement de zone.
//
// Une case occupée par un inspecteur lui est fermée : il y serait à l'abri de
// tout contact, l'adjacence n'incluant pas la superposition.
func (p *Partie) coupsFugitif() []Coup {
	var coups []Coup
	depart := p.Fugitif.Position

	if p.Fugitif.DeplacementsFaits < p.MobiliteDe(CampFugitif, 0) {
		for d := Nord; d <= NordOuest; d++ {
			arrivee := depart.Avance(d)
			if !p.EstPraticable(arrivee) || p.occupee(arrivee) {
				continue
			}
			// Un angle fermé ne se franchit pas : la diagonale exige qu'au
			// moins une des deux cases orthogonales intermédiaires soit une
			// rue, sinon le bâti ne bloque plus rien.
			if d.EstDiagonale() {
				a, b := depart.Contournement(d)
				if !p.EstPraticable(a) && !p.EstPraticable(b) {
					continue
				}
			}
			coups = append(coups, Coup{
				Tour: p.Tour, Acteur: CampFugitif, Type: CoupDeplacer,
				Depart: depart, Arrivee: arrivee,
			})
		}
	}

	coups = append(coups, p.coupsDepense()...)
	coups = append(coups, Coup{Tour: p.Tour, Acteur: CampFugitif, Type: CoupPasser})
	return append(coups, Coup{Tour: p.Tour, Acteur: CampFugitif, Type: CoupFinDePhase})
}

// coupsDepense énumère ce que le fugitif peut acheter avec sa résistance.
//
// Le changement de zone est une dépense comme une autre, mais porte un type de
// coup distinct : il désigne une zone, que les autres n'ont pas à porter.
func (p *Partie) coupsDepense() []Coup {
	if p.Extensions == nil {
		return nil
	}

	cles := make([]Depense, 0, len(p.Extensions.Depenses))
	for cle := range p.Extensions.Depenses {
		cles = append(cles, cle)
	}
	sort.Slice(cles, func(i, j int) bool { return cles[i] < cles[j] })

	var coups []Coup
	for _, cle := range cles {
		d := p.Extensions.Depenses[cle]
		if d.Camp != CampFugitif || d.Cout > p.Fugitif.Resistance {
			continue
		}
		if d.Usages > 0 && p.UsagesDepense[cle] >= d.Usages {
			continue
		}
		if cle == DepenseSilence && p.Fugitif.SilenceAchete {
			continue
		}
		coups = append(coups, Coup{
			Tour: p.Tour, Acteur: CampFugitif, Type: CoupDepense, Depense: cle,
		})
	}

	return append(coups, p.coupsChangerZone()...)
}

// coupsChangerZone énumère les zones vers lesquelles le fugitif peut resceller.
func (p *Partie) coupsChangerZone() []Coup {
	if p.Extensions == nil {
		return nil
	}
	d, connue := p.Extensions.Depenses[DepenseChangerZone]
	if !connue || d.Cout > p.Fugitif.Resistance {
		return nil
	}

	var coups []Coup
	for _, z := range p.Plateau.Zones() {
		if z.Numero == p.Fugitif.ZoneScellee {
			continue
		}
		coups = append(coups, Coup{
			Tour: p.Tour, Acteur: CampFugitif, Type: CoupChangerZone, Zone: z.Numero,
		})
	}
	return coups
}

// occupee dit si un inspecteur tient la case.
func (p *Partie) occupee(pos Position) bool {
	for _, i := range p.Inspecteurs {
		if i.Position == pos {
			return true
		}
	}
	return false
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
