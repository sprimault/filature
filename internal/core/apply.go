// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"errors"
	"fmt"
	"sort"
)

// ErrIllegalMove est renvoyé pour un coup absent de LegalMoves.
//
// L'appelant doit pouvoir le distinguer d'une panne : un bot qui propose un
// coup illégal interrompt la partie et part au journal, alors qu'une erreur
// interne relève du défaut. Le jeu ne corrige ni n'interprète jamais un coup.
var ErrIllegalMove = errors.New("coup absent des coups legaux")

// ErrRienAAnnuler est renvoyé quand la pile d'annulations est vide.
var ErrRienAAnnuler = errors.New("aucun coup a annuler")

// Apply joue un coup et fait avancer la phase.
//
// Un coup illégal est refusé sans modifier l'état, et un coup qui échoue en
// cours de route est défait avant de rendre la main : l'appelant peut réessayer
// sans repartir d'un instantané.
func (p *Game) Apply(c Move) error {
	if !p.isLegal(c) {
		return ErrIllegalMove
	}

	defaire, err := p.play(c)
	if err != nil {
		return err
	}

	p.Journal = append(p.Journal, c)
	p.annulations = append(p.annulations, defaire)
	return nil
}

// Undo défait le dernier coup.
//
// Ce n'est pas un confort d'interface : c'est ce qui permet à l'IA d'explorer
// des milliers de positions sans copier l'état à chaque nœud.
//
// La pile ne se sérialise pas — ce sont des fermetures. Une partie rechargée ne
// s'annule donc pas : elle se rejoue depuis son journal, qui reste la source de
// vérité.
func (p *Game) Undo() error {
	if len(p.annulations) == 0 {
		return ErrRienAAnnuler
	}

	dernier := len(p.annulations) - 1
	undoAll(p.annulations[dernier])
	p.annulations = truncate(p.annulations)
	p.Journal = truncate(p.Journal)
	return nil
}

// isLegal compare le coup aux coups légaux, par égalité et rien d'autre.
//
// C'est ce qui interdit à un appelant de « presque » play : un coup dont un
// seul champ diffère est un autre coup, et le noyau ne devine pas lequel.
func (p *Game) isLegal(c Move) bool {
	for _, legal := range p.LegalMoves(c.Side) {
		if legal == c {
			return true
		}
	}
	return false
}

// undoAll rappelle les annulations dans l'ordre inverse, celui dont
// dépendent les effets qui tronquent une tranche.
func undoAll(annulations []func()) {
	for i := len(annulations) - 1; i >= 0; i-- {
		annulations[i]()
	}
}

// play exécute un coup et renvoie de quoi le défaire.
//
// Toute modification passe par une annulation empilée au fur et à mesure : en
// cas d'échec en cours de route, ce qui a été fait est défait avant de rendre
// la main.
func (p *Game) play(c Move) ([]func(), error) {
	var faites []func()

	echec := func(err error) ([]func(), error) {
		undoAll(faites)
		return nil, err
	}

	switch c.Type {
	case MovePlace:
		faites = append(faites, p.placeForPhase(c)...)

	case MoveStep:
		defaire, err := p.step(c)
		if err != nil {
			return echec(err)
		}
		faites = append(faites, defaire...)

	case MoveAbility:
		defaire, err := p.triggerAbility(c)
		if err != nil {
			return echec(err)
		}
		faites = append(faites, defaire...)

	case MoveExpense, MoveChangeZone:
		defaire, err := p.spend(c)
		if err != nil {
			return echec(err)
		}
		faites = append(faites, defaire...)

	case MovePass:
		// Passer ne touche à rien : c'est un coup pour que le journal
		// distingue « il a choisi de ne rien faire » de « il n'a rien joué ».

	case MoveEndPhase:
		defaire, err := p.endPhase()
		if err != nil {
			return echec(err)
		}
		faites = append(faites, defaire...)

	default:
		return echec(fmt.Errorf("type de coup inconnu: %s", c.Type))
	}

	return faites, nil
}

// placeForPhase scelle une zone ou pose un inspecteur, selon la phase.
func (p *Game) placeForPhase(c Move) []func() {
	if p.Phase == PhaseFugitiveSetup {
		precedente := p.Fugitive.SealedZone
		p.Fugitive.SealedZone = c.Zone
		phaseName := p.advancePhase(PhaseInspectorsSetup)
		return []func(){phaseName, func() { p.Fugitive.SealedZone = precedente }}
	}

	p.Inspectors = append(p.Inspectors, Inspector{
		Position: c.To,
		Ability:  p.abilityFor(len(p.Inspectors)),
	})
	defaire := []func(){func() { p.Inspectors = truncate(p.Inspectors) }}

	// Le tour 1 commence quand le dernier pion est posé, pas sur une fin de
	// phase : la mise en place n'a rien à rendre.
	if len(p.Inspectors) >= p.Settings.Inspectors {
		tour := p.Turn
		p.Turn = 1
		defaire = append(defaire, p.advancePhase(PhaseInspectors), func() { p.Turn = tour })
	}
	return defaire
}

// abilityFor attribue une capacité au pion d'indice donné.
//
// Les clés du registre sont triées : l'ordre d'une map déciderait sinon de quel
// pion est le Barreur, et deux rejeux du même journal donneraient deux parties
// différentes.
func (p *Game) abilityFor(indice int) string {
	if p.Extensions == nil {
		return ""
	}
	var cles []string
	for cle, c := range p.Extensions.Abilities {
		if c.Camp == SideInspectors {
			cles = append(cles, cle)
		}
	}
	sort.Strings(cles)
	if indice >= len(cles) {
		return ""
	}
	return cles[indice]
}

// step bouge un pion et décompte son déplacement.
func (p *Game) step(c Move) ([]func(), error) {
	ctx := EffectContext{Side: c.Side, Piece: c.Piece, Case: c.To}
	defaire, err := p.ApplyOneEffect(Effect{Type: EffectMove, Target: targetOf(c.Side)}, ctx)
	if err != nil {
		return nil, err
	}

	if c.Side == SideFugitive {
		p.Fugitive.StepsTaken++
		return []func(){defaire, func() { p.Fugitive.StepsTaken-- }}, nil
	}
	pion := c.Piece
	p.Inspectors[pion].StepsTaken++
	return []func(){defaire, func() { p.Inspectors[pion].StepsTaken-- }}, nil
}

// targetOf traduit un camp en cible d'effet.
func targetOf(a Side) Target {
	if a == SideFugitive {
		return TargetFugitive
	}
	return TargetCurrentPiece
}

// triggerAbility applique les effets d'une capacité et la marked employée.
//
// Deux marques et non une : le pion ne la rejouera plus de la partie, et le
// camp ne déclenchera plus rien ce tour-ci. La règle pose les deux limites.
func (p *Game) triggerAbility(c Move) ([]func(), error) {
	capacite, connue := p.Extensions.Abilities[c.Ability]
	if !connue {
		return nil, fmt.Errorf("capacite inconnue: %s", c.Ability)
	}

	ctx := EffectContext{Side: c.Side, Piece: c.Piece, Case: c.To, Zone: c.Zone, AutrePion: c.Piece}
	defaire, err := p.applyEffects(capacite.Effects, ctx)
	if err != nil {
		return nil, err
	}

	pion := c.Piece
	p.Inspectors[pion].AbilityUsed = true
	p.AbilityPlayed = true
	return append(defaire,
		func() { p.Inspectors[pion].AbilityUsed = false },
		func() { p.AbilityPlayed = false },
	), nil
}

// spend prélève le coût d'une dépense et applique ses effets.
func (p *Game) spend(c Move) ([]func(), error) {
	cle := c.Expense
	if c.Type == MoveChangeZone {
		cle = ExpenseChangeZone
	}

	depense, connue := p.Extensions.Expenses[cle]
	if !connue {
		return nil, fmt.Errorf("depense inconnue: %s", cle)
	}

	faites := []func(){p.adjustStamina(-depense.Cost)}

	// Le compteur d'emplois est générique : « usages » est un champ du
	// contrat, et le noyau n'a pas à savoir que la dépense plafonnée s'appelle
	// meurtre.
	if depense.Uses > 0 {
		if p.ExpenseUses == nil {
			p.ExpenseUses = map[Expense]int{}
		}
		p.ExpenseUses[cle]++
		faites = append(faites, func() { p.ExpenseUses[cle]-- })
	}

	ctx := EffectContext{Side: c.Side, Case: p.Fugitive.Position, Zone: c.Zone}
	effets, err := p.applyEffects(depense.Effects, ctx)
	if err != nil {
		undoAll(faites)
		return nil, err
	}
	return append(faites, effets...), nil
}

// applyEffects déroule une suite d'effets, en défaisant tout si l'un échoue.
func (p *Game) applyEffects(effets []Effect, ctx EffectContext) ([]func(), error) {
	var faites []func()
	for _, e := range effets {
		defaire, err := p.ApplyOneEffect(e, ctx)
		if err != nil {
			undoAll(faites)
			return nil, err
		}
		faites = append(faites, defaire)
	}
	return faites, nil
}

// endPhase rend la main au camp next, et résout le tour quand les deux ont
// joué.
//
// L'ordre est un contrat : la résolution n'a lieu qu'après la phase du fugitif,
// jamais entre les deux.
func (p *Game) endPhase() ([]func(), error) {
	// La main passe au fugitif, et c'est le seul moment où son immobilisation
	// se constate : plus tard dans son tour, l'absence de déplacement veut dire
	// qu'il a déjà bougé.
	if p.Phase == PhaseInspectors {
		defaire := []func(){p.advancePhase(PhaseFugitive)}
		if _, fini := p.Outcome(); fini {
			defaire = append(defaire, p.advancePhase(PhaseOver))
		}
		return defaire, nil
	}

	defaire := []func(){p.advancePhase(PhaseInspectors)}
	defaire = append(defaire, p.resolveTurnEnd()...)

	tour := p.Turn
	p.Turn++
	defaire = append(defaire, func() { p.Turn = tour })
	defaire = append(defaire, p.resetQuotas()...)

	// Le résultat lui-même n'est pas retenu : il se recalcule, et le stocker en
	// ferait un cache que rien ne réconcilie.
	if _, fini := p.Outcome(); fini {
		defaire = append(defaire, p.advancePhase(PhaseOver))
	}
	return defaire, nil
}

// resetQuotas remet à zéro ce qui vaut pour un tour.
func (p *Game) resetQuotas() []func() {
	var defaire []func()

	if faits := p.Fugitive.StepsTaken; faits != 0 {
		p.Fugitive.StepsTaken = 0
		defaire = append(defaire, func() { p.Fugitive.StepsTaken = faits })
	}
	for i := range p.Inspectors {
		if faits := p.Inspectors[i].StepsTaken; faits != 0 {
			pion := i
			p.Inspectors[i].StepsTaken = 0
			defaire = append(defaire, func() { p.Inspectors[pion].StepsTaken = faits })
		}
	}
	if p.AbilityPlayed {
		p.AbilityPlayed = false
		defaire = append(defaire, func() { p.AbilityPlayed = true })
	}
	return defaire
}

// advancePhase change la phase et renvoie de quoi la rendre.
func (p *Game) advancePhase(suivante Phase) func() {
	precedente := p.Phase
	p.Phase = suivante
	return func() { p.Phase = precedente }
}
