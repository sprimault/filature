// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import "testing"

// partieAvecZone monte une partie dont la zone scellée est un bloc connu, de
// quoi éprouver l'extraction sans dépendre de la génération.
func partieAvecZone(fugitif Position, inspecteurs ...Position) *Partie {
	b := trame(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{
		{Numero: 0, Cases: []Position{{Colonne: 0, Ligne: 0}, {Colonne: 1, Ligne: 0}}},
		{Numero: 1, Cases: []Position{{Colonne: 4, Ligne: 4}, {Colonne: 3, Ligne: 4}}},
	}
	p := partieSur(b, fugitif, inspecteurs...)
	p.Fugitif.ZoneScellee = 1
	p.Extensions = registreDEssai()
	return p
}

// TestExtractionDemandeDeuxTours vérifie qu'arriver ne suffit pas.
//
// Le délai est ce qui donne aux inspecteurs une chance de venir neutraliser la
// zone : sans lui, l'extraction serait une arrivée et non un pari.
func TestExtractionDemandeDeuxTours(t *testing.T) {
	p := partieAvecZone(Position{Colonne: 4, Ligne: 4})

	finirLeTour(t, p)
	if _, fini := p.Resultat(); fini {
		t.Fatal("la partie s'achève dès le premier tour dans la zone")
	}
	if p.Fugitif.ToursDansZone != 1 {
		t.Fatalf("compte à %d, attendu 1", p.Fugitif.ToursDansZone)
	}

	finirLeTour(t, p)
	r, fini := p.Resultat()
	if !fini {
		t.Fatal("l'extraction n'aboutit pas au second tour")
	}
	if r.Vainqueur != CampFugitif || r.Motif != MotifExtraction {
		t.Errorf("résultat %+v, attendu fugitif/extraction", r)
	}
}

// TestInspecteurNeutraliseLaZone est le cas limite de tests.md : extraction
// interrompue par un inspecteur arrivant sur la zone.
//
// Camper est une stratégie valide — mais un inspecteur assis sur une zone est
// un inspecteur qui ne cherche pas.
func TestInspecteurNeutraliseLaZone(t *testing.T) {
	p := partieAvecZone(Position{Colonne: 4, Ligne: 4}, Position{Colonne: 0, Ligne: 4})

	finirLeTour(t, p)
	if p.Fugitif.ToursDansZone != 1 {
		t.Fatalf("compte à %d, attendu 1", p.Fugitif.ToursDansZone)
	}

	// L'inspecteur se pose sur l'autre case de la zone scellée.
	p.Inspecteurs[0].Position = Position{Colonne: 3, Ligne: 4}
	finirLeTour(t, p)

	if p.Fugitif.ToursDansZone != 0 {
		t.Errorf("compte à %d, attendu 0 : la zone est neutralisée", p.Fugitif.ToursDansZone)
	}
	if _, fini := p.Resultat(); fini {
		t.Error("la partie s'achève alors que la zone est occupée")
	}
}

// TestZoneFermeeInterromptLExtraction vérifie l'autre neutralisation : une zone
// que l'étranglement referme ne vaut plus rien, même occupée par le fugitif.
func TestZoneFermeeInterromptLExtraction(t *testing.T) {
	p := partieAvecZone(Position{Colonne: 4, Ligne: 4})

	finirLeTour(t, p)
	if p.Fugitif.ToursDansZone != 1 {
		t.Fatalf("compte à %d, attendu 1", p.Fugitif.ToursDansZone)
	}

	p.ZonesFermees = []int{1}
	finirLeTour(t, p)

	if p.Fugitif.ToursDansZone != 0 {
		t.Errorf("compte à %d, attendu 0 : la zone est fermée", p.Fugitif.ToursDansZone)
	}
}

// TestSortirDeLaZoneRemetLeCompteAZero vérifie qu'on ne cumule pas des
// passages : les deux tours doivent être consécutifs.
func TestSortirDeLaZoneRemetLeCompteAZero(t *testing.T) {
	p := partieAvecZone(Position{Colonne: 4, Ligne: 4})

	finirLeTour(t, p)
	p.Fugitif.Position = Position{Colonne: 2, Ligne: 2}
	finirLeTour(t, p)

	if p.Fugitif.ToursDansZone != 0 {
		t.Errorf("compte à %d, attendu 0 après être sorti", p.Fugitif.ToursDansZone)
	}
}

// TestResistanceEpuisee vérifie la première victoire des inspecteurs.
func TestResistanceEpuisee(t *testing.T) {
	p := partieAvecZone(Position{Colonne: 2, Ligne: 2})
	p.Fugitif.Resistance = 0

	r, fini := p.Resultat()
	if !fini {
		t.Fatal("une résistance nulle ne termine pas la partie")
	}
	if r.Vainqueur != CampInspecteurs || r.Motif != MotifResistance {
		t.Errorf("résultat %+v, attendu inspecteurs/resistance_epuisee", r)
	}
}

// TestTempsEcoule vérifie que le dernier tour se joue avant de conclure.
//
// Le test porte sur le tour dépassé et non atteint : sinon le tour quarante ne
// serait jamais joué, et l'extraction engagée à quarante ne pourrait jamais
// aboutir.
func TestTempsEcoule(t *testing.T) {
	p := partieAvecZone(Position{Colonne: 2, Ligne: 2})
	p.Parametres.Tours = 40

	p.Tour = 40
	if _, fini := p.Resultat(); fini {
		t.Error("la partie s'achève avant que le dernier tour soit joué")
	}

	p.Tour = 41
	r, fini := p.Resultat()
	if !fini {
		t.Fatal("le temps écoulé ne termine pas la partie")
	}
	if r.Vainqueur != CampInspecteurs || r.Motif != MotifTempsEcoule {
		t.Errorf("résultat %+v, attendu inspecteurs/temps_ecoule", r)
	}
}

// TestExtractionAuDernierTour est le cas limite de tests.md : dernier tour
// atteint avec l'extraction engagée.
//
// L'ordre des tests décide : une extraction achevée le tour où le temps
// s'épuise est une victoire du fugitif, pas un temps écoulé.
func TestExtractionAuDernierTour(t *testing.T) {
	p := partieAvecZone(Position{Colonne: 2, Ligne: 2})
	p.Parametres.Tours = 40
	p.Tour = 41
	p.Fugitif.ToursDansZone = ToursPourExtraction

	r, fini := p.Resultat()
	if !fini {
		t.Fatal("la partie ne se termine pas")
	}
	if r.Motif != MotifExtraction {
		t.Errorf("motif %s, attendu extraction : elle précède le temps écoulé", r.Motif)
	}
}

// TestFugitifBloque vérifie la troisième victoire des inspecteurs, et qu'elle
// ne se constate qu'en début de phase du fugitif.
func TestFugitifBloque(t *testing.T) {
	b := trame(
		"#####",
		"#####",
		"##.##",
		"#####",
		"#####",
	)
	b.zones = []Zone{{Numero: 1, Cases: []Position{{Colonne: 0, Ligne: 0}}}}
	p := partieSur(b, Position{Colonne: 2, Ligne: 2})
	p.Fugitif.ZoneScellee = 1
	p.Extensions = registreDEssai()

	r, fini := p.Resultat()
	if !fini {
		t.Fatal("un fugitif muré ne termine pas la partie")
	}
	if r.Vainqueur != CampInspecteurs || r.Motif != MotifBlocage {
		t.Errorf("résultat %+v, attendu inspecteurs/fugitif_bloque", r)
	}
}

// TestBlocageNeCompteQuEnDebutDePhase vérifie qu'un fugitif ayant déjà bougé
// n'est pas déclaré bloqué faute de second déplacement.
func TestBlocageNeCompteQuEnDebutDePhase(t *testing.T) {
	p := partieAvecZone(Position{Colonne: 2, Ligne: 2})
	p.Fugitif.DeplacementsFaits = 1

	if _, fini := p.Resultat(); fini {
		t.Error("un fugitif ayant déjà bougé est déclaré bloqué")
	}
}

// TestBlocageDetecteAuPassageDeMain vérifie que la main rendue par les
// inspecteurs constate l'immobilisation.
func TestBlocageDetecteAuPassageDeMain(t *testing.T) {
	b := trame(
		"#####",
		"##.##",
		"#####",
		"#####",
		"#####",
	)
	b.zones = []Zone{{Numero: 1, Cases: []Position{{Colonne: 0, Ligne: 0}}}}
	p := partieSur(b, Position{Colonne: 2, Ligne: 1})
	p.Phase = PhaseInspecteurs
	p.Fugitif.ZoneScellee = 1
	p.Extensions = registreDEssai()

	if err := p.Appliquer(premierCoup(t, p, CampInspecteurs, CoupFinDePhase)); err != nil {
		t.Fatal(err)
	}
	if p.Phase != PhaseTerminee {
		t.Errorf("phase %s, attendu terminee", p.Phase)
	}
}

// TestFinForceeLEmporte vérifie qu'un greffon conclut avant les conditions du
// noyau, dont il ne connaît pas les siennes.
func TestFinForceeLEmporte(t *testing.T) {
	p := partieAvecZone(Position{Colonne: 2, Ligne: 2})
	p.Fugitif.Resistance = 0
	p.FinForcee = &Resultat{Vainqueur: CampFugitif, Motif: MotifGreffon, Tour: 3}

	r, fini := p.Resultat()
	if !fini {
		t.Fatal("la fin forcée ne termine pas la partie")
	}
	if r.Motif != MotifGreffon {
		t.Errorf("motif %s, attendu greffon", r.Motif)
	}
}

// TestPartieEnCours vérifie qu'une position ordinaire ne conclut rien.
func TestPartieEnCours(t *testing.T) {
	p := partieAvecZone(Position{Colonne: 2, Ligne: 2})
	if r, fini := p.Resultat(); fini {
		t.Errorf("partie déclarée finie : %+v", r)
	}
}
