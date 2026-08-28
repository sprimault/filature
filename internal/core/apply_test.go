// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"errors"
	"reflect"
	"testing"
)

// testRegistry reproduit ce que plugins/base déclare, en plus court : de
// quoi exercer capacités et dépenses sans dépendre du chargeur, qui relève de
// l'étape 8.
func testRegistry() *Registry {
	return &Registry{
		Abilities: map[string]Ability{
			"blocker": {
				Name: "Barreur", Camp: SideInspectors, Uses: 1,
				Trigger: OnInspectorsPhase,
				Effects: []Effect{{Type: EffectBlockCell, Target: TargetCell, Duration: 3}},
			},
			"lookout": {
				Name: "Guetteur", Camp: SideInspectors, Uses: 1,
				Trigger: OnInspectorsPhase,
				Effects: []Effect{{Type: EffectChangeRange, Target: TargetCurrentPiece, Value: 8, Duration: 1}},
			},
		},
		Expenses: map[Expense]Ability{
			ExpenseSilence: {
				Name: "Silence", Camp: SideFugitive, Cost: 3,
				Trigger: OnFugitivePhase,
				Effects: []Effect{{Type: EffectCancelReveal, Target: TargetFugitive}},
			},
			ExpenseDecoy: {
				Name: "Leurre", Camp: SideFugitive, Cost: 1, Uses: 2,
				Trigger: OnFugitivePhase,
				Effects: []Effect{{Type: EffectDecoyTrail, Target: TargetCell}},
			},
			ExpenseChangeZone: {
				Name: "Changement de zone", Camp: SideFugitive, Cost: 2,
				Trigger: OnFugitivePhase,
				Effects: []Effect{{Type: EffectSealZone, Target: TargetZone}},
			},
		},
	}
}

// playableGame monte une partie en phase fugitif, registre compris.
func playableGame() *Game {
	b := grid(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{{Number: 0}, {Number: 1}, {Number: 2}}
	p := gameOn(b, Position{Column: 2, Row: 2},
		Position{Column: 0, Row: 0}, Position{Column: 4, Row: 0})
	p.Extensions = testRegistry()
	return p
}

// firstMove rend le premier coup légal du type demandé.
func firstMove(t *testing.T, p *Game, a Side, typ MoveType) Move {
	t.Helper()
	for _, c := range p.LegalMoves(a) {
		if c.Type == typ {
			return c
		}
	}
	t.Fatalf("aucun coup de type %s", typ)
	return Move{}
}

// stepTo rend le déplacement d'un pion d'inspecteur vers une case donnée.
func stepTo(t *testing.T, p *Game, pion int, arrivee Position) Move {
	t.Helper()
	for _, c := range p.LegalMoves(SideInspectors) {
		if c.Type == MoveStep && c.Piece == pion && c.To == arrivee {
			return c
		}
	}
	t.Fatalf("aucun déplacement du pion %d vers %v", pion, arrivee)
	return Move{}
}

// TestIllegalMoveRejected vérifie qu'un coup absent de la liste est rejeté, et
// qu'il ne laisse aucune trace.
//
// C'est ce que le serveur oppose à un bot fautif : le jeu ne corrige ni
// n'interprète, il refuse.
func TestIllegalMoveRejected(t *testing.T) {
	p := playableGame()
	avant := *p

	faux := Move{Turn: p.Turn, Side: SideFugitive, Type: MoveStep,
		From: p.Fugitive.Position, To: Position{Column: 9, Row: 9}}

	if err := p.Apply(faux); !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("erreur %v, attendu ErrIllegalMove", err)
	}
	if len(p.Journal) != 0 {
		t.Error("un coup refusé est entré au journal")
	}
	if p.Fugitive != avant.Fugitive {
		t.Error("un coup refusé a modifié l'état")
	}
}

// TestNearlyLegalMoveRejected vérifie que la comparaison est stricte : un coup
// dont un seul champ diffère est un autre coup.
func TestNearlyLegalMoveRejected(t *testing.T) {
	p := playableGame()
	c := firstMove(t, p, SideFugitive, MoveStep)
	c.Turn++

	if err := p.Apply(c); !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("erreur %v, attendu ErrIllegalMove", err)
	}
}

// TestStepCountsAndUndoes vérifie qu'un déplacement avance le pion,
// consomme sa mobilité, et se défait entièrement.
func TestStepCountsAndUndoes(t *testing.T) {
	p := playableGame()
	depart := p.Fugitive.Position

	c := firstMove(t, p, SideFugitive, MoveStep)
	if err := p.Apply(c); err != nil {
		t.Fatal(err)
	}
	if p.Fugitive.Position != c.To {
		t.Errorf("fugitif en %v, attendu %v", p.Fugitive.Position, c.To)
	}
	if p.Fugitive.StepsTaken != 1 {
		t.Errorf("%d déplacements comptés, attendu 1", p.Fugitive.StepsTaken)
	}

	if err := p.Undo(); err != nil {
		t.Fatal(err)
	}
	if p.Fugitive.Position != depart || p.Fugitive.StepsTaken != 0 {
		t.Error("l'annulation n'a pas rendu la position ou le compteur")
	}
	if len(p.Journal) != 0 {
		t.Error("le journal garde un coup annulé")
	}
}

// TestMobilitySpentAfterOneStep vérifie qu'un fugitif sans bonus ne joue qu'un
// déplacement par tour.
func TestMobilitySpentAfterOneStep(t *testing.T) {
	p := playableGame()
	if err := p.Apply(firstMove(t, p, SideFugitive, MoveStep)); err != nil {
		t.Fatal(err)
	}
	for _, c := range p.LegalMoves(SideFugitive) {
		if c.Type == MoveStep {
			t.Fatal("un second déplacement est proposé sans bonus de mobilité")
		}
	}
}

// TestExpenseCostsAndUndoes vérifie le prélèvement, l'effet, et le retour en
// arrière complet.
func TestExpenseCostsAndUndoes(t *testing.T) {
	p := playableGame()

	var silence Move
	for _, c := range p.LegalMoves(SideFugitive) {
		if c.Type == MoveExpense && c.Expense == ExpenseSilence {
			silence = c
		}
	}
	if silence.Type == "" {
		t.Fatal("le silence n'est pas proposé")
	}

	if err := p.Apply(silence); err != nil {
		t.Fatal(err)
	}
	if p.Fugitive.Stamina != 7 {
		t.Errorf("résistance %d, attendu 7", p.Fugitive.Stamina)
	}
	if !p.Fugitive.SilenceBought {
		t.Error("le silence n'a pas pris effet")
	}

	if err := p.Undo(); err != nil {
		t.Fatal(err)
	}
	if p.Fugitive.Stamina != 10 || p.Fugitive.SilenceBought {
		t.Error("l'annulation n'a pas rendu la résistance ou défait l'effet")
	}
}

// TestUsesCapped vérifie que le compteur d'emplois est générique : le noyau
// n'a pas à savoir laquelle des dépenses est plafonnée, ni à combien.
//
// Le plafond a changé de dépense — c'était le meurtre, c'est le leurre — sans
// qu'une ligne du noyau bouge, ce qui est exactement ce que ce test garde.
func TestUsesCapped(t *testing.T) {
	p := playableGame()
	p.Fugitive.Stamina = 20

	for i := 0; i < 2; i++ {
		var leurre Move
		for _, c := range p.LegalMoves(SideFugitive) {
			if c.Type == MoveExpense && c.Expense == ExpenseDecoy {
				leurre = c
				break
			}
		}
		if leurre.Type == "" {
			t.Fatalf("le leurre n'est plus proposé au %dᵉ emploi", i+1)
		}
		if err := p.Apply(leurre); err != nil {
			t.Fatal(err)
		}
	}

	if p.ExpenseUses[ExpenseDecoy] != 2 {
		t.Errorf("%d emplois comptés, attendu 2", p.ExpenseUses[ExpenseDecoy])
	}
	for _, c := range p.LegalMoves(SideFugitive) {
		if c.Type == MoveExpense && c.Expense == ExpenseDecoy {
			t.Error("un troisième leurre est proposé")
		}
	}
}

// TestOneAbilityPerTurn vérifie les deux limites de la règle : une par
// pion et par partie, et une seule par tour tous pions confondus.
func TestOneAbilityPerTurn(t *testing.T) {
	p := playableGame()
	p.Phase = PhaseInspectors
	p.Inspectors[0].Ability = "lookout"
	p.Inspectors[1].Ability = "blocker"

	c := firstMove(t, p, SideInspectors, MoveAbility)
	if err := p.Apply(c); err != nil {
		t.Fatal(err)
	}
	if !p.AbilityPlayed {
		t.Error("la capacité du tour n'est pas marquée")
	}
	for _, l := range p.LegalMoves(SideInspectors) {
		if l.Type == MoveAbility {
			t.Error("une seconde capacité est proposée dans le même tour")
		}
	}

	if err := p.Undo(); err != nil {
		t.Fatal(err)
	}
	if p.AbilityPlayed || p.Inspectors[c.Piece].AbilityUsed {
		t.Error("l'annulation n'a pas rendu les marques de capacité")
	}
}

// TestRoadblockLandsOnTheChosenCell vérifie que le barrage se pose là où le
// coup le dit, et nulle part ailleurs.
func TestRoadblockLandsOnTheChosenCell(t *testing.T) {
	p := playableGame()
	p.Phase = PhaseInspectors
	p.Inspectors[0].Ability = "blocker"

	c := firstMove(t, p, SideInspectors, MoveAbility)
	if ChebyshevDistance(c.To, p.Inspectors[0].Position) != 1 {
		t.Fatalf("le coup porte la case %v, hors des voisines du pion", c.To)
	}
	if err := p.Apply(c); err != nil {
		t.Fatal(err)
	}

	if _, barre := p.Roadblocks[c.To]; !barre {
		t.Errorf("barrages %v, attendu un en %v", p.Roadblocks, c.To)
	}
	if len(p.Roadblocks) != 1 {
		t.Errorf("%d barrages posés, un seul attendu", len(p.Roadblocks))
	}

	if err := p.Undo(); err != nil {
		t.Fatal(err)
	}
	if len(p.Roadblocks) != 0 {
		t.Errorf("%d barrages après annulation", len(p.Roadblocks))
	}
}

// TestAbilitySpentForTheWholeGame vérifie que les deux marques n'ont pas la
// même portée : le camp retrouve son droit au tour suivant, le pion jamais.
//
// docs/regles.md §5 : « une seule par tour, une seule fois par pion et par
// partie ». Le test voisin ne lit AbilityUsed qu'après une annulation, donc il
// passerait tout aussi bien si la marque du pion n'était jamais posée.
func TestAbilitySpentForTheWholeGame(t *testing.T) {
	p := playableGame()
	p.Phase = PhaseInspectors
	p.Inspectors[0].Ability = "lookout"
	p.Inspectors[1].Ability = "lookout"

	c := firstMove(t, p, SideInspectors, MoveAbility)
	if err := p.Apply(c); err != nil {
		t.Fatal(err)
	}
	if !p.Inspectors[c.Piece].AbilityUsed {
		t.Fatalf("le pion %d n'est pas marqué comme ayant employé sa capacité", c.Piece)
	}

	endTurn(t, p)

	if p.AbilityPlayed {
		t.Error("le camp n'a pas retrouvé son droit de capacité au tour suivant")
	}
	if !p.Inspectors[c.Piece].AbilityUsed {
		t.Errorf("le pion %d a retrouvé une capacité qui s'épuise pour la partie", c.Piece)
	}

	// L'autre pion porte la même capacité et ne l'a pas jouée : c'est lui, et
	// lui seul, que la liste doit offrir.
	var offerts []int
	for _, l := range p.LegalMoves(SideInspectors) {
		if l.Type == MoveAbility {
			offerts = append(offerts, l.Piece)
		}
	}
	for _, pion := range offerts {
		if pion == c.Piece {
			t.Errorf("le pion %d se voit reproposer sa capacité", pion)
		}
	}
	if len(offerts) == 0 {
		t.Error("le pion qui n'a rien joué n'a plus de capacité non plus")
	}
}

// TestSpottingRewardsOncePerTurn vérifie que le pas gagné au repérage ne se
// gagne qu'une fois par tour et par pion.
//
// Sans cette garde, un pion qui entre et sort d'une ligne de vue empocherait un
// pas à chaque aller : la compensation de la téléportation supprimée
// (docs/regles.md §2) deviendrait un déplacement illimité.
func TestSpottingRewardsOncePerTurn(t *testing.T) {
	p := gameOn(grid(
		".....",
		".....",
		".....",
		".....",
		".....",
	), Position{Column: 4, Row: 2}, Position{Column: 1, Row: 1})
	p.Phase = PhaseInspectors

	// (1,1) ne voit rien : le fugitif n'est ni sur sa ligne, ni sur sa
	// colonne, ni sur une de ses diagonales.
	if err := p.Apply(stepTo(t, p, 0, Position{Column: 1, Row: 2})); err != nil {
		t.Fatal(err)
	}
	if got := p.MobilityOf(SideInspectors, 0); got != 2 {
		t.Fatalf("mobilité %d après le repérage, attendu 2", got)
	}

	// Le second pas le laisse en vue, plus près encore. Le pas gagné ne se
	// regagne pas.
	if err := p.Apply(stepTo(t, p, 0, Position{Column: 2, Row: 2})); err != nil {
		t.Fatal(err)
	}
	if got := p.MobilityOf(SideInspectors, 0); got != 2 {
		t.Errorf("mobilité %d après un second repérage, attendu 2", got)
	}
	for _, c := range p.LegalMoves(SideInspectors) {
		if c.Type == MoveStep && c.Piece == 0 {
			t.Errorf("troisième déplacement vers %v offert au même pion", c.To)
		}
	}
}

// TestTurnEndResetsQuotas vérifie que la résolution remet les compteurs à
// zéro, et que l'annulation les rétablit.
func TestTurnEndResetsQuotas(t *testing.T) {
	p := playableGame()
	p.Phase = PhaseInspectors
	p.Inspectors[0].StepsTaken = 1
	p.AbilityPlayed = true
	tour := p.Turn

	// Les inspecteurs rendent la main, puis le fugitif : le tour se résout.
	if err := p.Apply(firstMove(t, p, SideInspectors, MoveEndPhase)); err != nil {
		t.Fatal(err)
	}
	if p.Phase != PhaseFugitive {
		t.Fatalf("phase %s, attendu fugitif", p.Phase)
	}
	if p.Turn != tour {
		t.Error("le tour a avancé entre les deux phases")
	}

	if err := p.Apply(firstMove(t, p, SideFugitive, MoveEndPhase)); err != nil {
		t.Fatal(err)
	}
	if p.Turn != tour+1 {
		t.Errorf("tour %d, attendu %d", p.Turn, tour+1)
	}
	if p.Inspectors[0].StepsTaken != 0 || p.AbilityPlayed {
		t.Error("les quotas n'ont pas été rouverts")
	}

	if err := p.Undo(); err != nil {
		t.Fatal(err)
	}
	if p.Turn != tour || p.Inspectors[0].StepsTaken != 1 || !p.AbilityPlayed {
		t.Error("l'annulation de la fin de tour n'a pas tout rendu")
	}
}

// TestUndoWithNoMove vérifie qu'annuler sur une partie vierge le dit.
func TestUndoWithNoMove(t *testing.T) {
	if err := playableGame().Undo(); !errors.Is(err, ErrNothingToUndo) {
		t.Fatalf("erreur %v, attendu ErrNothingToUndo", err)
	}
}

// TestWholeGameUndoes est l'invariant de réversibilité au niveau du coup.
//
// Une suite de coups quelconques, puis autant d'annulations, doit rendre un
// état identique à l'original. C'est ce dont l'IA dépend pour explorer sans
// copier l'état, et ce qui casse en silence dès qu'un coup oublie une
// annulation.
func TestWholeGameUndoes(t *testing.T) {
	p := playableGame()
	avant := playableGame()

	// Une partie qui traverse les deux phases, un déplacement, une dépense et
	// une résolution de tour.
	joues := 0
	for _, choix := range []struct {
		acteur Side
		typ    MoveType
	}{
		{SideFugitive, MoveStep},
		{SideFugitive, MoveExpense},
		{SideFugitive, MoveEndPhase},
		{SideInspectors, MoveStep},
		{SideInspectors, MoveEndPhase},
		{SideFugitive, MoveStep},
		{SideFugitive, MoveEndPhase},
	} {
		c := firstMove(t, p, choix.acteur, choix.typ)
		if err := p.Apply(c); err != nil {
			t.Fatalf("coup %d refusé : %v", joues, err)
		}
		joues++
	}

	for i := 0; i < joues; i++ {
		if err := p.Undo(); err != nil {
			t.Fatalf("annulation %d refusée : %v", i, err)
		}
	}

	// Les fermetures ne se comparent pas : la pile doit être vide des deux
	// côtés, et le reste identique.
	if len(p.annulations) != 0 {
		t.Errorf("%d annulations restantes", len(p.annulations))
	}
	p.annulations, avant.annulations = nil, nil
	if !reflect.DeepEqual(p, avant) {
		t.Errorf("l'état diffère après annulation complète\n  obtenu : %+v\n  attendu: %+v", p, avant)
	}
}
