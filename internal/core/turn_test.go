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

// gameWithShelter monte une partie dont un lieu de ressourcement occupe le coin
// haut-gauche, le fugitif posé où le cas le demande.
func gameWithShelter(fugitif Position, inspecteurs ...Position) *Game {
	b := grid(".....", ".....", ".....", ".....", ".....")
	b.abris = []Shelter{{Number: 0, Cells: []Position{
		{Column: 0, Row: 0}, {Column: 1, Row: 0}, {Column: 0, Row: 1},
	}}}

	p := gameOn(b, fugitif, inspecteurs...)
	p.Extensions = testRegistry()
	p.ShelterReady = []int{ShelterActive}
	p.Fugitive.Stamina = 5

	// Une partie assez longue pour qu'un lieu ait le temps de revenir : le
	// préréglage d'un plateau de cinq cases ne dure que cinq tours, moins que
	// la recharge.
	p.Settings.Turns = 3 * p.Settings.ShelterRecharge
	return p
}

// TestShelterRestoresThenRecharges vérifie qu'un lieu rend ses points une fois,
// puis se tait le temps de sa recharge.
func TestShelterRestoresThenRecharges(t *testing.T) {
	p := gameWithShelter(Position{Column: 0, Row: 0})
	depart := p.Turn

	endTurn(t, p)
	if p.Fugitive.Stamina != 5+p.Settings.ShelterGain {
		t.Fatalf("résistance %d, attendu %d", p.Fugitive.Stamina, 5+p.Settings.ShelterGain)
	}
	if attendu := depart + p.Settings.ShelterRecharge; p.ShelterReady[0] != attendu {
		t.Fatalf("recharge jusqu'au tour %d, attendu %d", p.ShelterReady[0], attendu)
	}

	// Le fugitif n'a pas bougé : le lieu ne doit plus rien rendre.
	avant := p.Fugitive.Stamina
	endTurn(t, p)
	if p.Fugitive.Stamina != avant {
		t.Errorf("résistance %d, attendu %d : un lieu en recharge rend encore",
			p.Fugitive.Stamina, avant)
	}
}

// TestShelterComesBack vérifie qu'un lieu redevient actif au tour dit.
//
// C'est ce qui distingue la recharge de l'usage unique, et ce qui rend la
// ressource disponible pour la percée finale plutôt que consommée trop tôt.
func TestShelterComesBack(t *testing.T) {
	p := gameWithShelter(Position{Column: 0, Row: 0})
	endTurn(t, p)

	avant := p.Fugitive.Stamina
	p.Turn = p.ShelterReady[0]
	endTurn(t, p)

	if p.Fugitive.Stamina != avant+p.Settings.ShelterGain {
		t.Errorf("résistance %d, attendu %d : le lieu n'est pas revenu",
			p.Fugitive.Stamina, avant+p.Settings.ShelterGain)
	}
}

// TestShelterIsNoSanctuary vérifie qu'un fugitif mis à zéro sur un lieu ne s'y
// relève pas.
//
// Le ressourcement passe après les contacts, et il le faut : des inspecteurs
// qui l'acculent dessus l'emportent, sans quoi le lieu serait un refuge où la
// capture et l'épuisement n'ont plus de prise.
func TestShelterIsNoSanctuary(t *testing.T) {
	p := gameWithShelter(Position{Column: 0, Row: 0},
		Position{Column: 1, Row: 0}, Position{Column: 0, Row: 1})
	p.Fugitive.Stamina = 2

	endTurn(t, p)

	if p.Fugitive.Stamina > 0 {
		t.Errorf("résistance %d, attendu au plus 0 : le lieu a servi de refuge",
			p.Fugitive.Stamina)
	}
	if _, fini := p.Outcome(); !fini {
		t.Error("la partie continue alors que la résistance est épuisée")
	}
}

// TestSharedCellIsContact vérifie qu'un inspecteur sur la case du fugitif est
// au contact.
//
// Rien ne lui interdit d'y marcher, et rien ne doit le lui interdire : un coup
// refusé parce qu'un fugitif invisible s'y trouve reviendrait à le lui
// annoncer. Le pion le plus proche possible serait alors le seul du plateau à
// ne rien faire de sa position.
func TestSharedCellIsContact(t *testing.T) {
	p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
		Position{Column: 2, Row: 2}, Position{Column: 2, Row: 2})
	p.Extensions = testRegistry()
	endTurn(t, p)

	if p.Fugitive.Stamina != 9 {
		t.Errorf("résistance %d, attendu 9 : la case commune ne compte pas",
			p.Fugitive.Stamina)
	}
}

// TestCaptureNeedsTwoResolutions vérifie qu'un contact isolé ne capture pas, et
// que le même inspecteur maintenu capture.
//
// C'est ce qui sépare la capture du blocage : en terrain ouvert le fugitif rompt
// le contact d'une diagonale, qui porte la distance orthogonale de un à trois
// quand le poursuivant n'en regagne qu'un. Ce qui se maintient, c'est le
// talonnage dans un couloir.
func TestCaptureNeedsTwoResolutions(t *testing.T) {
	p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
		Position{Column: 2, Row: 2}, Position{Column: 2, Row: 1})
	p.Extensions = testRegistry()

	endTurn(t, p)
	if p.Fugitive.Captured {
		t.Fatal("capturé dès le premier contact")
	}
	if _, fini := p.Outcome(); fini {
		t.Fatal("la partie s'achève au premier contact")
	}

	endTurn(t, p)
	if !p.Fugitive.Captured {
		t.Fatal("le contact tenu deux résolutions ne capture pas")
	}
	r, fini := p.Outcome()
	if !fini || r.Winner != SideInspectors || r.Reason != OutcomeCaptured {
		t.Errorf("résultat %+v, attendu inspecteurs/captured", r)
	}
}

// TestBrokenContactDoesNotCapture vérifie qu'un pion qui lâche prise ne
// capture pas, même si un autre prend le relais.
//
// La règle demande le *même* inspecteur deux fois : deux pions qui se relaient
// autour du fugitif le fatiguent, ils ne l'arrêtent pas.
func TestBrokenContactDoesNotCapture(t *testing.T) {
	p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
		Position{Column: 2, Row: 2}, Position{Column: 2, Row: 1}, Position{Column: 4, Row: 4})
	p.Extensions = testRegistry()

	endTurn(t, p)

	// Le premier s'écarte, le second vient prendre sa place.
	p.Inspectors[0].Position = Position{Column: 0, Row: 0}
	p.Inspectors[1].Position = Position{Column: 1, Row: 2}
	endTurn(t, p)

	if p.Fugitive.Captured {
		t.Error("capturé par deux pions qui se sont relayés")
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
	b.zones = []Zone{{Number: 0}, {Number: 1}, {Number: 2}, {Number: 3}}
	p := gameOn(b, Position{Column: 2, Row: 2})
	p.Seed = 99
	p.Settings.StranglingStart = 3
	p.Settings.StranglingPeriod = 2
	p.Settings.ZonesLeftOpen = 3
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

// TestStranglingCadence vérifie quand l'étranglement mord, quand il se tait, et
// qu'il s'arrête avant d'avoir tout fermé.
//
// Six zones dont trois restent ouvertes : l'entonnoir doit créer un
// rendez-vous, pas un verrou. Avec une seule issue, le camp entier s'y assied
// et il n'y a plus d'arbitrage.
func TestStranglingCadence(t *testing.T) {
	b := grid(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{{Number: 0}, {Number: 1}, {Number: 2},
		{Number: 3}, {Number: 4}, {Number: 5}}

	cas := []struct {
		tour   int
		attend bool
	}{
		{2, false}, // avant le début
		{3, true},  // première fermeture
		{4, false}, // hors cadence
		{5, true},  // deux tours plus tard
		{7, true},  // la troisième et dernière
		{9, false}, // il reste trois zones, l'étranglement s'arrête
		{11, false},
	}

	for _, c := range cas {
		p := gameOn(b, Position{Column: 2, Row: 2})
		p.Seed = 5
		p.Turn = c.tour
		p.Settings.StranglingStart = 3
		p.Settings.StranglingPeriod = 2
		p.Settings.ZonesLeftOpen = 3

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
