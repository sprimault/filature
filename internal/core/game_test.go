// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"reflect"
	"testing"
)

// plateauOuvert est un terrain dégagé de la taille demandée, avec ses zones
// d'extraction en périphérie.
//
// Un vrai plateau viendra de Generate, à l'étape 3. Celui-ci suffit à play une
// partie entière, et c'est justement ce que l'injection du plateau permet.
type plateauOuvert struct {
	cote  int
	zones []Zone
}

// ouvert construit un plateau dégagé et ses six zones aux quatre coins et sur
// deux côtés.
func ouvert(cote int) *plateauOuvert {
	b := &plateauOuvert{cote: cote}
	coins := []Position{
		{Column: 0, Row: 0},
		{Column: cote - 1, Row: 0},
		{Column: 0, Row: cote - 1},
		{Column: cote - 1, Row: cote - 1},
		{Column: cote / 2, Row: 0},
		{Column: cote / 2, Row: cote - 1},
	}
	for i, c := range coins {
		b.zones = append(b.zones, Zone{Number: i, Cells: []Position{c}})
	}
	return b
}

// IsStreet accepte toute case dans les bornes.
func (b *plateauOuvert) IsStreet(p Position) bool {
	return p.Column >= 0 && p.Row >= 0 && p.Column < b.cote && p.Row < b.cote
}

// Zones renvoie les six points d'extraction.
func (b *plateauOuvert) Zones() []Zone { return b.zones }

// Seed est figée : le tirage vient de la partie, pas du plateau.
func (b *plateauOuvert) Seed() int64 { return 1 }

// Sight déroule les huit directions à la demande.
//
// Pas de table précalculée ici : sur un terrain sans le moindre bâtiment, la
// ligne se calcule aussi vite qu'elle se lirait, et un plateau d'essai n'a pas
// à porter la mémoire d'un vrai.
func (b *plateauOuvert) Sight(p Position, portee int) []Position {
	if !b.IsStreet(p) || portee <= 0 {
		return nil
	}

	var vues []Position
	for d := Nord; d <= NordOuest; d++ {
		vues = append(vues, lineOfSight(b, p, d, portee)...)
	}
	return vues
}

// CellsWithin énumère le carré autour du centre.
func (b *plateauOuvert) CellsWithin(centre Position, rayon int) []Position {
	var cases []Position
	for ligne := centre.Row - rayon; ligne <= centre.Row+rayon; ligne++ {
		for colonne := centre.Column - rayon; colonne <= centre.Column+rayon; colonne++ {
			if p := (Position{Column: colonne, Row: ligne}); b.IsStreet(p) {
				cases = append(cases, p)
			}
		}
	}
	return cases
}

// testSettings réduit le plateau sans sortir de ce que Validate accepte.
func testSettings() Settings {
	p := SettingsForSize(15)
	p.Range = 7
	return p
}

// TestNewGameRejectsUnplayable vérifie que les manques sont dits plutôt
// que découverts au premier coup.
func TestNewGameRejectsUnplayable(t *testing.T) {
	bon := testSettings()

	cas := []struct {
		nom     string
		plateau Board
		params  Settings
		reg     *Registry
	}{
		{"sans plateau", nil, bon, testRegistry()},
		{"sans registre", ouvert(15), bon, nil},
		{"paramètres refusés", ouvert(15), Settings{}, testRegistry()},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if _, err := NewGame(c.plateau, 1, c.params, c.reg); err == nil {
				t.Error("accepté alors que la partie serait injouable")
			}
		})
	}
}

// TestNewGamePlacesFugitiveAtCentre vérifie que le tirage reste dans le noyau
// central : il doit avoir à traverser, sinon les zones ne servent à rien.
func TestNewGamePlacesFugitiveAtCentre(t *testing.T) {
	params := testSettings()
	milieu := params.Size / 2

	for graine := int64(0); graine < 50; graine++ {
		p, err := NewGame(ouvert(params.Size), graine, params, testRegistry())
		if err != nil {
			t.Fatal(err)
		}
		centre := Position{Column: milieu, Row: milieu}
		if d := ChebyshevDistance(p.Fugitive.Position, centre); d > params.CentreRadius {
			t.Fatalf("graine %d : fugitif à %d du centre, rayon %d", graine, d, params.CentreRadius)
		}
	}
}

// TestNewGameIsDeterministic vérifie que la graine décide seule du départ.
func TestNewGameIsDeterministic(t *testing.T) {
	params := testSettings()

	a, err := NewGame(ouvert(params.Size), 178342119, params, testRegistry())
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewGame(ouvert(params.Size), 178342119, params, testRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if a.Fugitive.Position != b.Fugitive.Position {
		t.Errorf("départs %v et %v pour la même graine", a.Fugitive.Position, b.Fugitive.Position)
	}

	autre, err := NewGame(ouvert(params.Size), 1, params, testRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if autre.Fugitive.Position == a.Fugitive.Position {
		t.Log("deux graines donnent le même départ, ce qui arrive")
	}
}

// TestNewGameSealsNoZone vérifie que le fugitif choisit, et que rien
// n'est décidé pour lui.
func TestNewGameSealsNoZone(t *testing.T) {
	params := testSettings()
	p, err := NewGame(ouvert(params.Size), 7, params, testRegistry())
	if err != nil {
		t.Fatal(err)
	}

	if p.Phase != PhaseFugitiveSetup {
		t.Errorf("phase %s, attendu placement_fugitif", p.Phase)
	}
	if _, scellee := p.sealedZone(); scellee {
		t.Error("une zone est déjà scellée avant que le fugitif ait choisi")
	}
	if p.Fugitive.Stamina != params.Stamina {
		t.Errorf("résistance %d, attendu %d", p.Fugitive.Stamina, params.Stamina)
	}
}

// jouerUnePartie déroule une partie entière en choisissant au hasard parmi les
// coups légaux, et rend le journal.
//
// Le hasard vient d'Random : deux appels sur la même graine jouent la même
// partie, ce qui rend l'échec reproductible.
func jouerUnePartie(t *testing.T, graine int64) *Game {
	t.Helper()

	params := testSettings()
	p, err := NewGame(ouvert(params.Size), graine, params, testRegistry())
	if err != nil {
		t.Fatal(err)
	}

	des := NewRandom(graine, "player")
	for coups := 0; coups < 20000; coups++ {
		if _, fini := p.Outcome(); fini || p.Phase == PhaseOver {
			return p
		}

		acteur := SideFugitive
		if p.Phase == PhaseInspectorsSetup || p.Phase == PhaseInspectors {
			acteur = SideInspectors
		}

		legaux := p.LegalMoves(acteur)
		if len(legaux) == 0 {
			t.Fatalf("aucun coup légal au tour %d, phase %s", p.Turn, p.Phase)
		}
		if err := p.Apply(legaux[des.Int(len(legaux))]); err != nil {
			t.Fatalf("coup refusé au tour %d : %v", p.Turn, err)
		}
	}

	t.Fatal("la partie ne se termine pas")
	return nil
}

// TestFullGamePlaysOut est le critère de livraison de l'étape 1 : une
// partie entière se joue depuis des appels Go, sans interface.
//
// Elle est jouée au hasard sur vingt graines figées. Ce qui est vérifié n'est
// pas qui gagne, mais qu'aucune ne s'enlise, qu'aucun coup légal n'est refusé,
// et que chaque fin porte un motif connu.
func TestFullGamePlaysOut(t *testing.T) {
	motifs := map[string]int{}

	for graine := int64(1); graine <= 20; graine++ {
		p := jouerUnePartie(t, graine)

		r, fini := p.Outcome()
		if !fini {
			t.Fatalf("graine %d : la partie s'arrête sans résultat", graine)
		}
		switch r.Reason {
		case OutcomeExtraction, OutcomeCaptured, OutcomeStaminaSpent, OutcomeCornered, OutcomeTimeUp:
			motifs[r.Reason]++
		default:
			t.Errorf("graine %d : motif inattendu %q", graine, r.Reason)
		}
		if r.Winner != SideFugitive && r.Winner != SideInspectors {
			t.Errorf("graine %d : vainqueur %q", graine, r.Winner)
		}
		if len(p.Journal) == 0 {
			t.Errorf("graine %d : journal vide", graine)
		}
	}

	t.Logf("motifs de fin sur vingt parties : %v", motifs)
}

// TestSameSeedSameJournal est le filet du déterminisme.
//
// Deux parties menées de la même façon sur la même graine produisent la même
// suite de coups. Sans cela, rejouer un journal ne reconstruit pas la partie, et
// la reprise comme l'entraînement de l'IA tombent ensemble.
func TestSameSeedSameJournal(t *testing.T) {
	for graine := int64(1); graine <= 5; graine++ {
		a := jouerUnePartie(t, graine)
		b := jouerUnePartie(t, graine)

		if !reflect.DeepEqual(a.Journal, b.Journal) {
			t.Fatalf("graine %d : les journaux divergent après %d et %d coups",
				graine, len(a.Journal), len(b.Journal))
		}
		if a.Fugitive != b.Fugitive || a.Turn != b.Turn {
			t.Errorf("graine %d : états finaux différents", graine)
		}
	}
}

// TestFullGameUndoes vérifie que la réversibilité tient sur une partie
// entière, et pas seulement sur un coup.
//
// C'est ce dont l'IA dépendra pour explorer : des milliers de coups appliqués
// puis défaits, sans jamais copier l'état.
func TestFullGameUndoes(t *testing.T) {
	params := testSettings()
	depart, err := NewGame(ouvert(params.Size), 3, params, testRegistry())
	if err != nil {
		t.Fatal(err)
	}
	avant := *depart

	des := NewRandom(3, "player")
	joues := 0
	for ; joues < 60; joues++ {
		if _, fini := depart.Outcome(); fini {
			break
		}
		acteur := SideFugitive
		if depart.Phase == PhaseInspectorsSetup || depart.Phase == PhaseInspectors {
			acteur = SideInspectors
		}
		legaux := depart.LegalMoves(acteur)
		if len(legaux) == 0 {
			break
		}
		if err := depart.Apply(legaux[des.Int(len(legaux))]); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < joues; i++ {
		if err := depart.Undo(); err != nil {
			t.Fatalf("annulation %d : %v", i, err)
		}
	}

	depart.annulations = nil
	avant.annulations = nil
	if !reflect.DeepEqual(*depart, avant) {
		t.Error("l'état diffère après avoir défait toute la partie")
	}
}
