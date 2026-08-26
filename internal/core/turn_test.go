// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"reflect"
	"testing"
)

// endTurn joue les deux fins de phase qui déclenchent la résolution.
func endTurn(t *testing.T, p *Game) {
	t.Helper()
	if p.Phase == PhaseInspectors {
		if err := p.Apply(firstMove(t, p, SideInspectors, MoveEndPhase)); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.Apply(firstMove(t, p, SideFugitive, MoveEndPhase)); err != nil {
		t.Fatal(err)
	}
}

// TestContactsCapped est le cas limite de tests.md : trois inspecteurs et
// plus, plafond appliqué.
//
// Sans plafond, un encerclement coûterait la moitié de la jauge en un tour et
// la partie se terminerait avant que le fugitif puisse réagir.
func TestContactsCapped(t *testing.T) {
	cas := []struct {
		nom      string
		autour   []Position
		attendue int
	}{
		{"aucun contact", nil, 10},
		{"un seul", []Position{{Column: 2, Row: 1}}, 9},
		{"trois", []Position{{Column: 2, Row: 1}, {Column: 1, Row: 2}, {Column: 3, Row: 2}}, 7},
		{"quatre, plafonnés à trois", []Position{
			{Column: 2, Row: 1}, {Column: 1, Row: 2},
			{Column: 3, Row: 2}, {Column: 2, Row: 3},
		}, 7},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
				Position{Column: 2, Row: 2}, c.autour...)
			p.Extensions = testRegistry()
			endTurn(t, p)

			if p.Fugitive.Stamina != c.attendue {
				t.Errorf("résistance %d, attendu %d", p.Fugitive.Stamina, c.attendue)
			}
		})
	}
}

// TestDiagonalsMakeNoContact vérifie que l'adjacence est orthogonale.
//
// Les diagonales font du fugitif un pion plus rapide, pas un pion plus
// vulnérable : les compter reviendrait à lui reprendre ce qu'elles lui donnent.
func TestDiagonalsMakeNoContact(t *testing.T) {
	p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
		Position{Column: 2, Row: 2},
		Position{Column: 1, Row: 1}, Position{Column: 3, Row: 3})
	p.Extensions = testRegistry()
	endTurn(t, p)

	if p.Fugitive.Stamina != 10 {
		t.Errorf("résistance %d, attendu 10 : une diagonale a fait contact", p.Fugitive.Stamina)
	}
}

// TestTrailPerCellLeft vérifie qu'un déplacement laisse une trace là d'où le
// fugitif vient, avec la direction prise.
func TestTrailPerCellLeft(t *testing.T) {
	p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
		Position{Column: 2, Row: 2})
	p.Extensions = testRegistry()
	depart := p.Fugitive.Position

	c := firstMove(t, p, SideFugitive, MoveStep)
	if err := p.Apply(c); err != nil {
		t.Fatal(err)
	}
	endTurn(t, p)

	trace, posee := p.Trails[depart]
	if !posee {
		t.Fatalf("aucune trace en %v", depart)
	}
	attendue, _ := DirectionTo(depart, c.To)
	if trace.Direction != attendue {
		t.Errorf("direction %d, attendu %d", trace.Direction, attendue)
	}
	if trace.Turn != 3 {
		t.Errorf("trace du tour %d, attendu 3", trace.Turn)
	}
}

// TestTrailExpires vérifie qu'une trace disparaît au bout de sa durée, et pas
// avant.
func TestTrailExpires(t *testing.T) {
	p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
		Position{Column: 2, Row: 2})
	p.Extensions = testRegistry()
	p.Settings.TrailLifetime = 6

	vieille := Position{Column: 0, Row: 0}
	recente := Position{Column: 4, Row: 4}
	p.Trails = map[Position]Trail{
		vieille: {Turn: p.Turn - 6},
		recente: {Turn: p.Turn - 5},
	}

	endTurn(t, p)

	if _, reste := p.Trails[vieille]; reste {
		t.Error("une trace de six tours n'a pas été effacée")
	}
	if _, reste := p.Trails[recente]; !reste {
		t.Error("une trace de cinq tours a été effacée trop tôt")
	}
}

// TestPeriodicReveal vérifie le battement qui empêche la partie de
// devenir une loterie.
func TestPeriodicReveal(t *testing.T) {
	p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
		Position{Column: 2, Row: 2})
	p.Extensions = testRegistry()
	p.Settings.RevealPeriod = 4
	p.Turn = 4

	endTurn(t, p)
	if !p.Fugitive.Visible {
		t.Error("le fugitif devait être révélé au tour 4")
	}
}

// TestSilenceConsumesTheReveal est le cas limite de tests.md : un tour de
// révélation qui tombe le même tour qu'un silence acheté.
//
// Les inspecteurs apprennent qu'il a payé, pas où il est. Le silence est
// consommé, donc la révélation suivante le trouvera à découvert.
func TestSilenceConsumesTheReveal(t *testing.T) {
	p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
		Position{Column: 2, Row: 2})
	p.Extensions = testRegistry()
	p.Settings.RevealPeriod = 4
	p.Turn = 4
	p.Fugitive.SilenceBought = true

	endTurn(t, p)

	if p.Fugitive.Visible {
		t.Error("le silence n'a pas empêché la révélation")
	}
	if p.Fugitive.SilenceBought {
		t.Error("le silence n'a pas été consommé : il couvrirait toutes les révélations")
	}
}

// TestNoRevealOutOfPeriod vérifie qu'un tour ordinaire ne révèle rien.
func TestNoRevealOutOfPeriod(t *testing.T) {
	p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
		Position{Column: 2, Row: 2})
	p.Extensions = testRegistry()
	p.Settings.RevealPeriod = 4
	p.Turn = 3

	endTurn(t, p)
	if p.Fugitive.Visible {
		t.Error("le fugitif a été révélé hors période")
	}
}

// TestDeferredComesDue vérifie qu'un effet posé pour plus tard s'applique
// au bon tour, et pas avant.
func TestDeferredComesDue(t *testing.T) {
	p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
		Position{Column: 2, Row: 2})
	p.Extensions = testRegistry()
	p.PendingEffects = []PendingEffect{{
		Effects:       []Effect{{Type: EffectCloseZone, Target: TargetZone}},
		Turn:          p.Turn + 1,
		Announced:     true,
		EffectContext: EffectContext{Side: SideInspectors, Zone: 2},
	}}

	endTurn(t, p)
	if len(p.ClosedZones) != 0 {
		t.Fatal("le différé s'est appliqué avant son échéance")
	}
	if len(p.PendingEffects) != 1 {
		t.Fatal("le différé a quitté la file avant son échéance")
	}

	endTurn(t, p)
	if !reflect.DeepEqual(p.ClosedZones, []int{2}) {
		t.Errorf("zones fermées %v, attendu [2]", p.ClosedZones)
	}
	if len(p.PendingEffects) != 0 {
		t.Error("le différé résolu reste dans la file")
	}
}

// TestStranglingTriggersMode vérifie que le noyau donne la cadence et la
// cible, et que le mode fait le reste.
func TestStranglingTriggersMode(t *testing.T) {
	b := grid(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{{Number: 0}, {Number: 1}, {Number: 2}}
	p := gameOn(b, Position{Column: 2, Row: 2})
	p.Seed = 99
	p.Settings.StranglingStart = 3
	p.Settings.StranglingPeriod = 2
	p.Extensions = testRegistry()
	p.Extensions.Modes = map[string]Mode{"etranglement": {
		Name: "Étranglement", Trigger: OnStrangling,
		Effects: []Effect{{
			Type: EffectDefer, Duration: 2, Announced: true,
			Then: []Effect{{Type: EffectCloseZone, Target: TargetZone}},
		}},
	}}

	endTurn(t, p)

	if len(p.PendingEffects) != 1 {
		t.Fatalf("%d effets en attente, attendu 1", len(p.PendingEffects))
	}
	if !p.PendingEffects[0].Announced {
		t.Error("la fermeture n'est pas annoncée : elle tomberait sans prévenir")
	}
	if got := p.PendingEffects[0].Turn; got != 5 {
		t.Errorf("échéance au tour %d, attendu 5", got)
	}
}

// TestStranglingOrderDeterministic vérifie que la graine décide, et elle
// seule.
func TestStranglingOrderDeterministic(t *testing.T) {
	construire := func() *Game {
		b := grid(".....", ".....", ".....", ".....", ".....")
		b.zones = []Zone{{Number: 0}, {Number: 1}, {Number: 2}, {Number: 3}, {Number: 4}, {Number: 5}}
		p := gameOn(b, Position{Column: 2, Row: 2})
		p.Seed = 178342119
		return p
	}

	premier := construire().stranglingOrder()
	for i := 0; i < 10; i++ {
		if got := construire().stranglingOrder(); !reflect.DeepEqual(got, premier) {
			t.Fatalf("ordre %v puis %v", premier, got)
		}
	}

	autre := construire()
	autre.Seed = 1
	if reflect.DeepEqual(autre.stranglingOrder(), premier) {
		t.Error("deux graines donnent le même ordre de fermeture")
	}
}

// TestStranglingCadence vérifie quand l'étranglement mord, et quand il se
// tait.
func TestStranglingCadence(t *testing.T) {
	b := grid(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{{Number: 0}, {Number: 1}, {Number: 2}}

	cas := []struct {
		tour   int
		attend bool
	}{
		{2, false}, // avant le début
		{3, true},  // premier tour d'étranglement
		{4, false}, // hors cadence
		{5, true},  // deux tours plus tard
		{7, true},  // la troisième et dernière zone
		{9, false}, // plus de zone à fermer
	}

	for _, c := range cas {
		p := gameOn(b, Position{Column: 2, Row: 2})
		p.Seed = 5
		p.Turn = c.tour
		p.Settings.StranglingStart = 3
		p.Settings.StranglingPeriod = 2

		if _, vise := p.zoneToStrangle(); vise != c.attend {
			t.Errorf("tour %d : visé=%v, attendu %v", c.tour, vise, c.attend)
		}
	}
}

// TestTurnEndUndoes est l'invariant de réversibilité étendu à la
// résolution.
//
// Contacts, traces, révélation et différés modifient tous l'état : un seul qui
// oublierait son annulation ferait diverger l'exploration de l'IA, puis le
// rejeu du journal.
func TestTurnEndUndoes(t *testing.T) {
	construire := func() *Game {
		p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
			Position{Column: 2, Row: 2},
			Position{Column: 2, Row: 1}, Position{Column: 1, Row: 2})
		p.Extensions = testRegistry()
		p.Settings.RevealPeriod = 4
		p.Turn = 4
		p.Trails = map[Position]Trail{{Column: 0, Row: 0}: {Turn: 0}}
		p.PendingEffects = []PendingEffect{{
			Effects:       []Effect{{Type: EffectCloseZone, Target: TargetZone}},
			Turn:          4,
			EffectContext: EffectContext{Side: SideInspectors, Zone: 1},
		}}
		return p
	}

	p, avant := construire(), construire()

	// Un déplacement pour qu'une trace soit déposée, puis le tour se résout.
	if err := p.Apply(firstMove(t, p, SideFugitive, MoveStep)); err != nil {
		t.Fatal(err)
	}
	endTurn(t, p)

	// Deux coups joués : le déplacement, puis la fin de phase du fugitif qui
	// emporte toute la résolution.
	for i := 0; i < 2; i++ {
		if err := p.Undo(); err != nil {
			t.Fatalf("annulation %d : %v", i, err)
		}
	}

	p.annulations, avant.annulations = nil, nil
	if !reflect.DeepEqual(p, avant) {
		t.Errorf("l'état diffère après annulation\n  obtenu : %+v\n  attendu: %+v", p, avant)
	}
}
