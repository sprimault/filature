// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import (
	"reflect"
	"testing"
)

// finirLeTour joue les deux fins de phase qui déclenchent la résolution.
func finirLeTour(t *testing.T, p *Partie) {
	t.Helper()
	if p.Phase == PhaseInspecteurs {
		if err := p.Appliquer(premierCoup(t, p, CampInspecteurs, CoupFinDePhase)); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.Appliquer(premierCoup(t, p, CampFugitif, CoupFinDePhase)); err != nil {
		t.Fatal(err)
	}
}

// TestContactsPlafonnes est le cas limite de tests.md : trois inspecteurs et
// plus, plafond appliqué.
//
// Sans plafond, un encerclement coûterait la moitié de la jauge en un tour et
// la partie se terminerait avant que le fugitif puisse réagir.
func TestContactsPlafonnes(t *testing.T) {
	cas := []struct {
		nom      string
		autour   []Position
		attendue int
	}{
		{"aucun contact", nil, 10},
		{"un seul", []Position{{Colonne: 2, Ligne: 1}}, 9},
		{"trois", []Position{{Colonne: 2, Ligne: 1}, {Colonne: 1, Ligne: 2}, {Colonne: 3, Ligne: 2}}, 7},
		{"quatre, plafonnés à trois", []Position{
			{Colonne: 2, Ligne: 1}, {Colonne: 1, Ligne: 2},
			{Colonne: 3, Ligne: 2}, {Colonne: 2, Ligne: 3},
		}, 7},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			p := partieSur(trame(".....", ".....", ".....", ".....", "....."),
				Position{Colonne: 2, Ligne: 2}, c.autour...)
			p.Extensions = registreDEssai()
			finirLeTour(t, p)

			if p.Fugitif.Resistance != c.attendue {
				t.Errorf("résistance %d, attendu %d", p.Fugitif.Resistance, c.attendue)
			}
		})
	}
}

// TestDiagonalesNeFontPasContact vérifie que l'adjacence est orthogonale.
//
// Les diagonales font du fugitif un pion plus rapide, pas un pion plus
// vulnérable : les compter reviendrait à lui reprendre ce qu'elles lui donnent.
func TestDiagonalesNeFontPasContact(t *testing.T) {
	p := partieSur(trame(".....", ".....", ".....", ".....", "....."),
		Position{Colonne: 2, Ligne: 2},
		Position{Colonne: 1, Ligne: 1}, Position{Colonne: 3, Ligne: 3})
	p.Extensions = registreDEssai()
	finirLeTour(t, p)

	if p.Fugitif.Resistance != 10 {
		t.Errorf("résistance %d, attendu 10 : une diagonale a fait contact", p.Fugitif.Resistance)
	}
}

// TestTraceParCaseQuittee vérifie qu'un déplacement laisse une trace là d'où le
// fugitif vient, avec la direction prise.
func TestTraceParCaseQuittee(t *testing.T) {
	p := partieSur(trame(".....", ".....", ".....", ".....", "....."),
		Position{Colonne: 2, Ligne: 2})
	p.Extensions = registreDEssai()
	depart := p.Fugitif.Position

	c := premierCoup(t, p, CampFugitif, CoupDeplacer)
	if err := p.Appliquer(c); err != nil {
		t.Fatal(err)
	}
	finirLeTour(t, p)

	trace, posee := p.Traces[depart]
	if !posee {
		t.Fatalf("aucune trace en %v", depart)
	}
	attendue, _ := DirectionVers(depart, c.Arrivee)
	if trace.Direction != attendue {
		t.Errorf("direction %d, attendu %d", trace.Direction, attendue)
	}
	if trace.Tour != 3 {
		t.Errorf("trace du tour %d, attendu 3", trace.Tour)
	}
}

// TestTraceExpire vérifie qu'une trace disparaît au bout de sa durée, et pas
// avant.
func TestTraceExpire(t *testing.T) {
	p := partieSur(trame(".....", ".....", ".....", ".....", "....."),
		Position{Colonne: 2, Ligne: 2})
	p.Extensions = registreDEssai()
	p.Parametres.DureeTrace = 6

	vieille := Position{Colonne: 0, Ligne: 0}
	recente := Position{Colonne: 4, Ligne: 4}
	p.Traces = map[Position]Trace{
		vieille: {Tour: p.Tour - 6},
		recente: {Tour: p.Tour - 5},
	}

	finirLeTour(t, p)

	if _, reste := p.Traces[vieille]; reste {
		t.Error("une trace de six tours n'a pas été effacée")
	}
	if _, reste := p.Traces[recente]; !reste {
		t.Error("une trace de cinq tours a été effacée trop tôt")
	}
}

// TestRevelationPeriodique vérifie le battement qui empêche la partie de
// devenir une loterie.
func TestRevelationPeriodique(t *testing.T) {
	p := partieSur(trame(".....", ".....", ".....", ".....", "....."),
		Position{Colonne: 2, Ligne: 2})
	p.Extensions = registreDEssai()
	p.Parametres.PeriodeRevelation = 4
	p.Tour = 4

	finirLeTour(t, p)
	if !p.Fugitif.Visible {
		t.Error("le fugitif devait être révélé au tour 4")
	}
}

// TestSilenceConsommeLaRevelation est le cas limite de tests.md : un tour de
// révélation qui tombe le même tour qu'un silence acheté.
//
// Les inspecteurs apprennent qu'il a payé, pas où il est. Le silence est
// consommé, donc la révélation suivante le trouvera à découvert.
func TestSilenceConsommeLaRevelation(t *testing.T) {
	p := partieSur(trame(".....", ".....", ".....", ".....", "....."),
		Position{Colonne: 2, Ligne: 2})
	p.Extensions = registreDEssai()
	p.Parametres.PeriodeRevelation = 4
	p.Tour = 4
	p.Fugitif.SilenceAchete = true

	finirLeTour(t, p)

	if p.Fugitif.Visible {
		t.Error("le silence n'a pas empêché la révélation")
	}
	if p.Fugitif.SilenceAchete {
		t.Error("le silence n'a pas été consommé : il couvrirait toutes les révélations")
	}
}

// TestPasDeRevelationHorsPeriode vérifie qu'un tour ordinaire ne révèle rien.
func TestPasDeRevelationHorsPeriode(t *testing.T) {
	p := partieSur(trame(".....", ".....", ".....", ".....", "....."),
		Position{Colonne: 2, Ligne: 2})
	p.Extensions = registreDEssai()
	p.Parametres.PeriodeRevelation = 4
	p.Tour = 3

	finirLeTour(t, p)
	if p.Fugitif.Visible {
		t.Error("le fugitif a été révélé hors période")
	}
}

// TestDiffereArriveAEcheance vérifie qu'un effet posé pour plus tard s'applique
// au bon tour, et pas avant.
func TestDiffereArriveAEcheance(t *testing.T) {
	p := partieSur(trame(".....", ".....", ".....", ".....", "....."),
		Position{Colonne: 2, Ligne: 2})
	p.Extensions = registreDEssai()
	p.EffetsEnAttente = []EffetEnAttente{{
		Effets:   []Effet{{Type: EffetFermerZone, Cible: CibleZone}},
		Tour:     p.Tour + 1,
		Annonce:  true,
		Contexte: Contexte{Acteur: CampInspecteurs, Zone: 2},
	}}

	finirLeTour(t, p)
	if len(p.ZonesFermees) != 0 {
		t.Fatal("le différé s'est appliqué avant son échéance")
	}
	if len(p.EffetsEnAttente) != 1 {
		t.Fatal("le différé a quitté la file avant son échéance")
	}

	finirLeTour(t, p)
	if !reflect.DeepEqual(p.ZonesFermees, []int{2}) {
		t.Errorf("zones fermées %v, attendu [2]", p.ZonesFermees)
	}
	if len(p.EffetsEnAttente) != 0 {
		t.Error("le différé résolu reste dans la file")
	}
}

// TestEtranglementDeclencheLeMode vérifie que le noyau donne la cadence et la
// cible, et que le mode fait le reste.
func TestEtranglementDeclencheLeMode(t *testing.T) {
	b := trame(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{{Numero: 0}, {Numero: 1}, {Numero: 2}}
	p := partieSur(b, Position{Colonne: 2, Ligne: 2})
	p.Graine = 99
	p.Parametres.DebutEtranglement = 3
	p.Parametres.PeriodeEtranglement = 2
	p.Extensions = registreDEssai()
	p.Extensions.Modes = map[string]Mode{"etranglement": {
		Nom: "Étranglement", Declenchement: SurEtranglement,
		Effets: []Effet{{
			Type: EffetDifferer, Duree: 2, Annonce: true,
			Puis: []Effet{{Type: EffetFermerZone, Cible: CibleZone}},
		}},
	}}

	finirLeTour(t, p)

	if len(p.EffetsEnAttente) != 1 {
		t.Fatalf("%d effets en attente, attendu 1", len(p.EffetsEnAttente))
	}
	if !p.EffetsEnAttente[0].Annonce {
		t.Error("la fermeture n'est pas annoncée : elle tomberait sans prévenir")
	}
	if got := p.EffetsEnAttente[0].Tour; got != 5 {
		t.Errorf("échéance au tour %d, attendu 5", got)
	}
}

// TestOrdreEtranglementDeterministe vérifie que la graine décide, et elle
// seule.
func TestOrdreEtranglementDeterministe(t *testing.T) {
	construire := func() *Partie {
		b := trame(".....", ".....", ".....", ".....", ".....")
		b.zones = []Zone{{Numero: 0}, {Numero: 1}, {Numero: 2}, {Numero: 3}, {Numero: 4}, {Numero: 5}}
		p := partieSur(b, Position{Colonne: 2, Ligne: 2})
		p.Graine = 178342119
		return p
	}

	premier := construire().ordreEtranglement()
	for i := 0; i < 10; i++ {
		if got := construire().ordreEtranglement(); !reflect.DeepEqual(got, premier) {
			t.Fatalf("ordre %v puis %v", premier, got)
		}
	}

	autre := construire()
	autre.Graine = 1
	if reflect.DeepEqual(autre.ordreEtranglement(), premier) {
		t.Error("deux graines donnent le même ordre de fermeture")
	}
}

// TestCadenceEtranglement vérifie quand l'étranglement mord, et quand il se
// tait.
func TestCadenceEtranglement(t *testing.T) {
	b := trame(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{{Numero: 0}, {Numero: 1}, {Numero: 2}}

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
		p := partieSur(b, Position{Colonne: 2, Ligne: 2})
		p.Graine = 5
		p.Tour = c.tour
		p.Parametres.DebutEtranglement = 3
		p.Parametres.PeriodeEtranglement = 2

		if _, vise := p.zoneAEtrangler(); vise != c.attend {
			t.Errorf("tour %d : visé=%v, attendu %v", c.tour, vise, c.attend)
		}
	}
}

// TestFinDeTourSeDefait est l'invariant de réversibilité étendu à la
// résolution.
//
// Contacts, traces, révélation et différés modifient tous l'état : un seul qui
// oublierait son annulation ferait diverger l'exploration de l'IA, puis le
// rejeu du journal.
func TestFinDeTourSeDefait(t *testing.T) {
	construire := func() *Partie {
		p := partieSur(trame(".....", ".....", ".....", ".....", "....."),
			Position{Colonne: 2, Ligne: 2},
			Position{Colonne: 2, Ligne: 1}, Position{Colonne: 1, Ligne: 2})
		p.Extensions = registreDEssai()
		p.Parametres.PeriodeRevelation = 4
		p.Tour = 4
		p.Traces = map[Position]Trace{{Colonne: 0, Ligne: 0}: {Tour: 0}}
		p.EffetsEnAttente = []EffetEnAttente{{
			Effets:   []Effet{{Type: EffetFermerZone, Cible: CibleZone}},
			Tour:     4,
			Contexte: Contexte{Acteur: CampInspecteurs, Zone: 1},
		}}
		return p
	}

	p, avant := construire(), construire()

	// Un déplacement pour qu'une trace soit déposée, puis le tour se résout.
	if err := p.Appliquer(premierCoup(t, p, CampFugitif, CoupDeplacer)); err != nil {
		t.Fatal(err)
	}
	finirLeTour(t, p)

	// Deux coups joués : le déplacement, puis la fin de phase du fugitif qui
	// emporte toute la résolution.
	for i := 0; i < 2; i++ {
		if err := p.Annuler(); err != nil {
			t.Fatalf("annulation %d : %v", i, err)
		}
	}

	p.annulations, avant.annulations = nil, nil
	if !reflect.DeepEqual(p, avant) {
		t.Errorf("l'état diffère après annulation\n  obtenu : %+v\n  attendu: %+v", p, avant)
	}
}
