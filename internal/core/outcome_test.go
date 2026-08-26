// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import "testing"

// gameWithZone monte une partie dont la zone scellée est un bloc connu, de
// quoi éprouver l'extraction sans dépendre de la génération.
func gameWithZone(fugitif Position, inspecteurs ...Position) *Game {
	b := grid(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{
		{Number: 0, Cells: []Position{{Column: 0, Row: 0}, {Column: 1, Row: 0}}},
		{Number: 1, Cells: []Position{{Column: 4, Row: 4}, {Column: 3, Row: 4}}},
	}
	p := gameOn(b, fugitif, inspecteurs...)
	p.Fugitive.SealedZone = 1
	p.Extensions = testRegistry()
	return p
}

// TestExtractionTakesTwoTurns vérifie qu'arriver ne suffit pas.
//
// Le délai est ce qui donne aux inspecteurs une chance de venir neutraliser la
// zone : sans lui, l'extraction serait une arrivée et non un pari.
func TestExtractionTakesTwoTurns(t *testing.T) {
	p := gameWithZone(Position{Column: 4, Row: 4})

	endTurn(t, p)
	if _, fini := p.Outcome(); fini {
		t.Fatal("la partie s'achève dès le premier tour dans la zone")
	}
	if p.Fugitive.TurnsInZone != 1 {
		t.Fatalf("compte à %d, attendu 1", p.Fugitive.TurnsInZone)
	}

	endTurn(t, p)
	r, fini := p.Outcome()
	if !fini {
		t.Fatal("l'extraction n'aboutit pas au second tour")
	}
	if r.Winner != SideFugitive || r.Reason != OutcomeExtraction {
		t.Errorf("résultat %+v, attendu fugitif/extraction", r)
	}
}

// TestInspectorNeutralisesZone est le cas limite de tests.md : extraction
// interrompue par un inspecteur arrivant sur la zone.
//
// Camper est une stratégie valide — mais un inspecteur assis sur une zone est
// un inspecteur qui ne cherche pas.
func TestInspectorNeutralisesZone(t *testing.T) {
	p := gameWithZone(Position{Column: 4, Row: 4}, Position{Column: 0, Row: 4})

	endTurn(t, p)
	if p.Fugitive.TurnsInZone != 1 {
		t.Fatalf("compte à %d, attendu 1", p.Fugitive.TurnsInZone)
	}

	// L'inspecteur se pose sur l'autre case de la zone scellée.
	p.Inspectors[0].Position = Position{Column: 3, Row: 4}
	endTurn(t, p)

	if p.Fugitive.TurnsInZone != 0 {
		t.Errorf("compte à %d, attendu 0 : la zone est neutralisée", p.Fugitive.TurnsInZone)
	}
	if _, fini := p.Outcome(); fini {
		t.Error("la partie s'achève alors que la zone est occupée")
	}
}

// TestClosedZoneInterruptsExtraction vérifie l'autre neutralisation : une zone
// que l'étranglement referme ne vaut plus rien, même occupée par le fugitif.
func TestClosedZoneInterruptsExtraction(t *testing.T) {
	p := gameWithZone(Position{Column: 4, Row: 4})

	endTurn(t, p)
	if p.Fugitive.TurnsInZone != 1 {
		t.Fatalf("compte à %d, attendu 1", p.Fugitive.TurnsInZone)
	}

	p.ClosedZones = []int{1}
	endTurn(t, p)

	if p.Fugitive.TurnsInZone != 0 {
		t.Errorf("compte à %d, attendu 0 : la zone est fermée", p.Fugitive.TurnsInZone)
	}
}

// TestLeavingZoneResetsCount vérifie qu'on ne cumule pas des
// passages : les deux tours doivent être consécutifs.
func TestLeavingZoneResetsCount(t *testing.T) {
	p := gameWithZone(Position{Column: 4, Row: 4})

	endTurn(t, p)
	p.Fugitive.Position = Position{Column: 2, Row: 2}
	endTurn(t, p)

	if p.Fugitive.TurnsInZone != 0 {
		t.Errorf("compte à %d, attendu 0 après être sorti", p.Fugitive.TurnsInZone)
	}
}

// TestStaminaSpent vérifie la première victoire des inspecteurs.
func TestStaminaSpent(t *testing.T) {
	p := gameWithZone(Position{Column: 2, Row: 2})
	p.Fugitive.Stamina = 0

	r, fini := p.Outcome()
	if !fini {
		t.Fatal("une résistance nulle ne termine pas la partie")
	}
	if r.Winner != SideInspectors || r.Reason != OutcomeStaminaSpent {
		t.Errorf("résultat %+v, attendu inspecteurs/resistance_epuisee", r)
	}
}

// TestTimeUp vérifie que le dernier tour se joue avant de conclure.
//
// Le test porte sur le tour dépassé et non atteint : sinon le tour quarante ne
// serait jamais joué, et l'extraction engagée à quarante ne pourrait jamais
// aboutir.
func TestTimeUp(t *testing.T) {
	p := gameWithZone(Position{Column: 2, Row: 2})
	p.Settings.Turns = 40

	p.Turn = 40
	if _, fini := p.Outcome(); fini {
		t.Error("la partie s'achève avant que le dernier tour soit joué")
	}

	p.Turn = 41
	r, fini := p.Outcome()
	if !fini {
		t.Fatal("le temps écoulé ne termine pas la partie")
	}
	if r.Winner != SideInspectors || r.Reason != OutcomeTimeUp {
		t.Errorf("résultat %+v, attendu inspecteurs/temps_ecoule", r)
	}
}

// TestExtractionOnLastTurn est le cas limite de tests.md : dernier tour
// atteint avec l'extraction engagée.
//
// L'ordre des tests décide : une extraction achevée le tour où le temps
// s'épuise est une victoire du fugitif, pas un temps écoulé.
func TestExtractionOnLastTurn(t *testing.T) {
	p := gameWithZone(Position{Column: 2, Row: 2})
	p.Settings.Turns = 40
	p.Turn = 41
	p.Fugitive.TurnsInZone = TurnsToExtract

	r, fini := p.Outcome()
	if !fini {
		t.Fatal("la partie ne se termine pas")
	}
	if r.Reason != OutcomeExtraction {
		t.Errorf("motif %s, attendu extraction : elle précède le temps écoulé", r.Reason)
	}
}

// TestFugitiveCornered vérifie la troisième victoire des inspecteurs, et qu'elle
// ne se constate qu'en début de phase du fugitif.
func TestFugitiveCornered(t *testing.T) {
	b := grid(
		"#####",
		"#####",
		"##.##",
		"#####",
		"#####",
	)
	b.zones = []Zone{{Number: 1, Cells: []Position{{Column: 0, Row: 0}}}}
	p := gameOn(b, Position{Column: 2, Row: 2})
	p.Fugitive.SealedZone = 1
	p.Extensions = testRegistry()

	r, fini := p.Outcome()
	if !fini {
		t.Fatal("un fugitif muré ne termine pas la partie")
	}
	if r.Winner != SideInspectors || r.Reason != OutcomeCornered {
		t.Errorf("résultat %+v, attendu inspecteurs/fugitif_bloque", r)
	}
}

// TestCorneredOnlyCountsAtPhaseStart vérifie qu'un fugitif ayant déjà bougé
// n'est pas déclaré bloqué faute de second déplacement.
func TestCorneredOnlyCountsAtPhaseStart(t *testing.T) {
	p := gameWithZone(Position{Column: 2, Row: 2})
	p.Fugitive.StepsTaken = 1

	if _, fini := p.Outcome(); fini {
		t.Error("un fugitif ayant déjà bougé est déclaré bloqué")
	}
}

// TestCorneredDetectedOnHandover vérifie que la main rendue par les
// inspecteurs constate l'immobilisation.
func TestCorneredDetectedOnHandover(t *testing.T) {
	b := grid(
		"#####",
		"##.##",
		"#####",
		"#####",
		"#####",
	)
	b.zones = []Zone{{Number: 1, Cells: []Position{{Column: 0, Row: 0}}}}
	p := gameOn(b, Position{Column: 2, Row: 1})
	p.Phase = PhaseInspectors
	p.Fugitive.SealedZone = 1
	p.Extensions = testRegistry()

	if err := p.Apply(firstMove(t, p, SideInspectors, MoveEndPhase)); err != nil {
		t.Fatal(err)
	}
	if p.Phase != PhaseOver {
		t.Errorf("phase %s, attendu terminee", p.Phase)
	}
}

// TestForcedOutcomeWins vérifie qu'un plugin conclut avant les conditions du
// noyau, dont il ne connaît pas les siennes.
func TestForcedOutcomeWins(t *testing.T) {
	p := gameWithZone(Position{Column: 2, Row: 2})
	p.Fugitive.Stamina = 0
	p.ForcedOutcome = &Outcome{Winner: SideFugitive, Reason: OutcomePlugin, Turn: 3}

	r, fini := p.Outcome()
	if !fini {
		t.Fatal("la fin forcée ne termine pas la partie")
	}
	if r.Reason != OutcomePlugin {
		t.Errorf("motif %s, attendu plugin", r.Reason)
	}
}

// TestGameInProgress vérifie qu'une position ordinaire ne conclut rien.
func TestGameInProgress(t *testing.T) {
	p := gameWithZone(Position{Column: 2, Row: 2})
	if r, fini := p.Outcome(); fini {
		t.Errorf("partie déclarée finie : %+v", r)
	}
}
