// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import (
	"errors"
	"fmt"
	"sort"
)

// ErrCoupIllegal est renvoyé pour un coup absent de CoupsLegaux.
//
// L'appelant doit pouvoir le distinguer d'une panne : un bot qui propose un
// coup illégal interrompt la partie et part au journal, alors qu'une erreur
// interne relève du défaut. Le jeu ne corrige ni n'interprète jamais un coup.
var ErrCoupIllegal = errors.New("coup absent des coups legaux")

// ErrRienAAnnuler est renvoyé quand la pile d'annulations est vide.
var ErrRienAAnnuler = errors.New("aucun coup a annuler")

// Appliquer joue un coup et fait avancer la phase.
//
// Un coup illégal est refusé sans modifier l'état, et un coup qui échoue en
// cours de route est défait avant de rendre la main : l'appelant peut réessayer
// sans repartir d'un instantané.
func (p *Partie) Appliquer(c Coup) error {
	if !p.estLegal(c) {
		return ErrCoupIllegal
	}

	defaire, err := p.jouer(c)
	if err != nil {
		return err
	}

	p.Journal = append(p.Journal, c)
	p.annulations = append(p.annulations, defaire)
	return nil
}

// Annuler défait le dernier coup.
//
// Ce n'est pas un confort d'interface : c'est ce qui permet à l'IA d'explorer
// des milliers de positions sans copier l'état à chaque nœud.
//
// La pile ne se sérialise pas — ce sont des fermetures. Une partie rechargée ne
// s'annule donc pas : elle se rejoue depuis son journal, qui reste la source de
// vérité.
func (p *Partie) Annuler() error {
	if len(p.annulations) == 0 {
		return ErrRienAAnnuler
	}

	dernier := len(p.annulations) - 1
	defaireTout(p.annulations[dernier])
	p.annulations = tronquer(p.annulations)
	p.Journal = tronquer(p.Journal)
	return nil
}

// estLegal compare le coup aux coups légaux, par égalité et rien d'autre.
//
// C'est ce qui interdit à un appelant de « presque » jouer : un coup dont un
// seul champ diffère est un autre coup, et le noyau ne devine pas lequel.
func (p *Partie) estLegal(c Coup) bool {
	for _, legal := range p.CoupsLegaux(c.Acteur) {
		if legal == c {
			return true
		}
	}
	return false
}

// defaireTout rappelle les annulations dans l'ordre inverse, celui dont
// dépendent les effets qui tronquent une tranche.
func defaireTout(annulations []func()) {
	for i := len(annulations) - 1; i >= 0; i-- {
		annulations[i]()
	}
}

// jouer exécute un coup et renvoie de quoi le défaire.
//
// Toute modification passe par une annulation empilée au fur et à mesure : en
// cas d'échec en cours de route, ce qui a été fait est défait avant de rendre
// la main.
func (p *Partie) jouer(c Coup) ([]func(), error) {
	var faites []func()

	echec := func(err error) ([]func(), error) {
		defaireTout(faites)
		return nil, err
	}

	switch c.Type {
	case CoupPlacer:
		faites = append(faites, p.placerSelonPhase(c)...)

	case CoupDeplacer:
		defaire, err := p.deplacer(c)
		if err != nil {
			return echec(err)
		}
		faites = append(faites, defaire...)

	case CoupCapacite:
		defaire, err := p.declencherCapacite(c)
		if err != nil {
			return echec(err)
		}
		faites = append(faites, defaire...)

	case CoupDepense, CoupChangerZone:
		defaire, err := p.depenser(c)
		if err != nil {
			return echec(err)
		}
		faites = append(faites, defaire...)

	case CoupPasser:
		// Passer ne touche à rien : c'est un coup pour que le journal
		// distingue « il a choisi de ne rien faire » de « il n'a rien joué ».

	case CoupFinDePhase:
		defaire, err := p.finirPhase()
		if err != nil {
			return echec(err)
		}
		faites = append(faites, defaire...)

	default:
		return echec(fmt.Errorf("type de coup inconnu: %s", c.Type))
	}

	return faites, nil
}

// placerSelonPhase scelle une zone ou pose un inspecteur, selon la phase.
func (p *Partie) placerSelonPhase(c Coup) []func() {
	if p.Phase == PhasePlacementFugitif {
		precedente := p.Fugitif.ZoneScellee
		p.Fugitif.ZoneScellee = c.Zone
		phase := p.avancerPhase(PhasePlacementInspecteurs)
		return []func(){phase, func() { p.Fugitif.ZoneScellee = precedente }}
	}

	p.Inspecteurs = append(p.Inspecteurs, Inspecteur{
		Position: c.Arrivee,
		Capacite: p.capacitePour(len(p.Inspecteurs)),
	})
	defaire := []func(){func() { p.Inspecteurs = tronquer(p.Inspecteurs) }}

	// Le tour 1 commence quand le dernier pion est posé, pas sur une fin de
	// phase : la mise en place n'a rien à rendre.
	if len(p.Inspecteurs) >= p.Parametres.Inspecteurs {
		tour := p.Tour
		p.Tour = 1
		defaire = append(defaire, p.avancerPhase(PhaseInspecteurs), func() { p.Tour = tour })
	}
	return defaire
}

// capacitePour attribue une capacité au pion d'indice donné.
//
// Les clés du registre sont triées : l'ordre d'une map déciderait sinon de quel
// pion est le Barreur, et deux rejeux du même journal donneraient deux parties
// différentes.
func (p *Partie) capacitePour(indice int) string {
	if p.Extensions == nil {
		return ""
	}
	var cles []string
	for cle, c := range p.Extensions.Capacites {
		if c.Camp == CampInspecteurs {
			cles = append(cles, cle)
		}
	}
	sort.Strings(cles)
	if indice >= len(cles) {
		return ""
	}
	return cles[indice]
}

// deplacer bouge un pion et décompte son déplacement.
func (p *Partie) deplacer(c Coup) ([]func(), error) {
	ctx := Contexte{Acteur: c.Acteur, Pion: c.Pion, Case: c.Arrivee}
	defaire, err := p.Appliquer1Effet(Effet{Type: EffetDeplacer, Cible: cibleDe(c.Acteur)}, ctx)
	if err != nil {
		return nil, err
	}

	if c.Acteur == CampFugitif {
		p.Fugitif.DeplacementsFaits++
		return []func(){defaire, func() { p.Fugitif.DeplacementsFaits-- }}, nil
	}
	pion := c.Pion
	p.Inspecteurs[pion].DeplacementsFaits++
	return []func(){defaire, func() { p.Inspecteurs[pion].DeplacementsFaits-- }}, nil
}

// cibleDe traduit un camp en cible d'effet.
func cibleDe(a Acteur) Cible {
	if a == CampFugitif {
		return CibleFugitif
	}
	return CiblePionCourant
}

// declencherCapacite applique les effets d'une capacité et la marque employée.
//
// Deux marques et non une : le pion ne la rejouera plus de la partie, et le
// camp ne déclenchera plus rien ce tour-ci. La règle pose les deux limites.
func (p *Partie) declencherCapacite(c Coup) ([]func(), error) {
	capacite, connue := p.Extensions.Capacites[c.Capacite]
	if !connue {
		return nil, fmt.Errorf("capacite inconnue: %s", c.Capacite)
	}

	ctx := Contexte{Acteur: c.Acteur, Pion: c.Pion, Case: c.Arrivee, Zone: c.Zone, AutrePion: c.Pion}
	defaire, err := p.appliquerEffets(capacite.Effets, ctx)
	if err != nil {
		return nil, err
	}

	pion := c.Pion
	p.Inspecteurs[pion].CapaciteUtilisee = true
	p.CapaciteJouee = true
	return append(defaire,
		func() { p.Inspecteurs[pion].CapaciteUtilisee = false },
		func() { p.CapaciteJouee = false },
	), nil
}

// depenser prélève le coût d'une dépense et applique ses effets.
func (p *Partie) depenser(c Coup) ([]func(), error) {
	cle := c.Depense
	if c.Type == CoupChangerZone {
		cle = DepenseChangerZone
	}

	depense, connue := p.Extensions.Depenses[cle]
	if !connue {
		return nil, fmt.Errorf("depense inconnue: %s", cle)
	}

	faites := []func(){p.ajusterResistance(-depense.Cout)}

	// Le compteur d'emplois est générique : « usages » est un champ du
	// contrat, et le noyau n'a pas à savoir que la dépense plafonnée s'appelle
	// meurtre.
	if depense.Usages > 0 {
		if p.UsagesDepense == nil {
			p.UsagesDepense = map[Depense]int{}
		}
		p.UsagesDepense[cle]++
		faites = append(faites, func() { p.UsagesDepense[cle]-- })
	}

	ctx := Contexte{Acteur: c.Acteur, Case: p.Fugitif.Position, Zone: c.Zone}
	effets, err := p.appliquerEffets(depense.Effets, ctx)
	if err != nil {
		defaireTout(faites)
		return nil, err
	}
	return append(faites, effets...), nil
}

// appliquerEffets déroule une suite d'effets, en défaisant tout si l'un échoue.
func (p *Partie) appliquerEffets(effets []Effet, ctx Contexte) ([]func(), error) {
	var faites []func()
	for _, e := range effets {
		defaire, err := p.Appliquer1Effet(e, ctx)
		if err != nil {
			defaireTout(faites)
			return nil, err
		}
		faites = append(faites, defaire)
	}
	return faites, nil
}

// finirPhase rend la main au camp suivant, et résout le tour quand les deux ont
// joué.
//
// L'ordre est un contrat : la résolution n'a lieu qu'après la phase du fugitif,
// jamais entre les deux.
func (p *Partie) finirPhase() ([]func(), error) {
	if p.Phase == PhaseInspecteurs {
		return []func(){p.avancerPhase(PhaseFugitif)}, nil
	}

	defaire := []func(){p.avancerPhase(PhaseInspecteurs)}
	defaire = append(defaire, p.resoudreFinDeTour()...)

	tour := p.Tour
	p.Tour++
	defaire = append(defaire, func() { p.Tour = tour })
	defaire = append(defaire, p.rouvrirLesQuotas()...)

	// Le résultat lui-même n'est pas retenu : il se recalcule, et le stocker en
	// ferait un cache que rien ne réconcilie.
	if _, fini := p.Resultat(); fini {
		defaire = append(defaire, p.avancerPhase(PhaseTerminee))
	}
	return defaire, nil
}

// rouvrirLesQuotas remet à zéro ce qui vaut pour un tour.
func (p *Partie) rouvrirLesQuotas() []func() {
	var defaire []func()

	if faits := p.Fugitif.DeplacementsFaits; faits != 0 {
		p.Fugitif.DeplacementsFaits = 0
		defaire = append(defaire, func() { p.Fugitif.DeplacementsFaits = faits })
	}
	for i := range p.Inspecteurs {
		if faits := p.Inspecteurs[i].DeplacementsFaits; faits != 0 {
			pion := i
			p.Inspecteurs[i].DeplacementsFaits = 0
			defaire = append(defaire, func() { p.Inspecteurs[pion].DeplacementsFaits = faits })
		}
	}
	if p.CapaciteJouee {
		p.CapaciteJouee = false
		defaire = append(defaire, func() { p.CapaciteJouee = true })
	}
	return defaire
}

// avancerPhase change la phase et renvoie de quoi la rendre.
func (p *Partie) avancerPhase(suivante Phase) func() {
	precedente := p.Phase
	p.Phase = suivante
	return func() { p.Phase = precedente }
}
