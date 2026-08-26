// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"reflect"
	"testing"
)

// plateauTrame est un terrain dont les bâtiments se déclarent case par case, de
// quoi fabriquer un angle fermé ou une impasse sans passer par la génération,
// qui relève de l'étape 3.
type plateauTrame struct {
	cote      string
	batiments map[Position]bool
	zones     []Zone
}

// grid construit un plateau depuis un dessin, « # » pour un bâtiment.
//
// Un dessin vaut mieux qu'une liste de coordonnées : le cas testé se lit d'un
// coup d'œil, et une erreur d'index ne passe pas inaperçue.
func grid(lignes ...string) *plateauTrame {
	b := &plateauTrame{batiments: map[Position]bool{}}
	for l, ligne := range lignes {
		for c, r := range ligne {
			if r == '#' {
				b.batiments[Position{Column: c, Row: l}] = true
			}
		}
	}
	b.cote = lignes[0]
	return b
}

// IsStreet traite le hors-limites comme du bâti, comme le plateau borné.
func (b *plateauTrame) IsStreet(p Position) bool {
	if p.Column < 0 || p.Row < 0 || p.Column >= len(b.cote) || p.Row >= len(b.cote) {
		return false
	}
	return !b.batiments[p]
}

// Zones renvoie les zones déclarées par le cas de test.
func (b *plateauTrame) Zones() []Zone { return b.zones }

// Seed est figée : aucun test de ce fichier ne tire au sort.
func (b *plateauTrame) Seed() int64 { return 1 }

// Sight reste vide : la légalité d'un coup ne dépend pas de ce qu'on voit.
func (b *plateauTrame) Sight(p Position, portee int) []Position { return nil }

// CellsWithin énumère les rues du carré, ligne par ligne.
func (b *plateauTrame) CellsWithin(centre Position, rayon int) []Position {
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

// gameOn monte une partie en phase fugitif sur un plateau dessiné.
func gameOn(b *plateauTrame, fugitif Position, inspecteurs ...Position) *Game {
	p := &Game{
		Settings: DefaultSettings(),
		Board:    b,
		Turn:     3,
		Phase:    PhaseFugitive,
		Fugitive: Fugitive{Position: fugitif, Stamina: 10},
	}
	p.Settings.Size = len(b.cote)
	for _, pos := range inspecteurs {
		p.Inspectors = append(p.Inspectors, Inspector{Position: pos})
	}
	return p
}

// arrivals extrait les destinations des déplacements, pour compare sans se
// soucier des autres champs.
func arrivals(coups []Move) []Position {
	var positions []Position
	for _, c := range coups {
		if c.Type == MoveStep {
			positions = append(positions, c.To)
		}
	}
	return positions
}

// TestFugitiveEightDirections vérifie qu'en terrain dégagé le fugitif dispose de
// ses huit directions, là où un inspecteur n'en a que quatre.
func TestFugitiveEightDirections(t *testing.T) {
	p := gameOn(grid(
		".....",
		".....",
		".....",
		".....",
		".....",
	), Position{Column: 2, Row: 2})

	if got := len(arrivals(p.LegalMoves(SideFugitive))); got != 8 {
		t.Errorf("%d déplacements, attendu 8", got)
	}
}

// TestDiagonalBlockedByClosedCorner est le cas limite de docs/regles.md §2.
//
// Une diagonale exige qu'au moins une des deux cases orthogonales
// intermédiaires soit une rue. Sans cette règle, on traverse les angles de
// bâtiments et le bâti ne bloque plus rien.
func TestDiagonalBlockedByClosedCorner(t *testing.T) {
	// Le coin nord-est du fugitif est fermé par deux bâtiments en équerre, en
	// (2,1) et (3,2). La case visée (3,1) est libre, et c'est tout l'intérêt :
	// elle serait atteignable si seule la praticabilité comptait.
	p := gameOn(grid(
		".....",
		"..#..",
		"...#.",
		".....",
		".....",
	), Position{Column: 2, Row: 2})

	if !p.IsWalkable(Position{Column: 3, Row: 1}) {
		t.Fatal("la case visée doit être libre, sinon le test ne prouve rien")
	}

	for _, a := range arrivals(p.LegalMoves(SideFugitive)) {
		if a == (Position{Column: 3, Row: 1}) {
			t.Error("la diagonale traverse un angle fermé")
		}
	}
}

// TestDiagonalOpenedByOneSide vérifie l'autre moitié de la règle : une
// seule des deux cases intermédiaires suffit.
func TestDiagonalOpenedByOneSide(t *testing.T) {
	// Seul (3,2) est bâti ; (2,1) reste libre, et suffit.
	p := gameOn(grid(
		".....",
		".....",
		"...#.",
		".....",
		".....",
	), Position{Column: 2, Row: 2})

	trouve := false
	for _, a := range arrivals(p.LegalMoves(SideFugitive)) {
		if a == (Position{Column: 3, Row: 1}) {
			trouve = true
		}
	}
	if !trouve {
		t.Error("la diagonale devrait passer par la case orthogonale libre")
	}
}

// TestFugitiveCannotEnterInspector applique la décision de docs/regles.md
// §5 : il y serait à l'abri de tout contact, l'adjacence n'incluant pas la
// superposition.
func TestFugitiveCannotEnterInspector(t *testing.T) {
	p := gameOn(grid(
		".....",
		".....",
		".....",
		".....",
		".....",
	), Position{Column: 2, Row: 2}, Position{Column: 3, Row: 2})

	for _, a := range arrivals(p.LegalMoves(SideFugitive)) {
		if a == (Position{Column: 3, Row: 2}) {
			t.Error("le fugitif peut se poser sous un inspecteur")
		}
	}
}

// TestFugitiveWithNoLegalStep est le cas limite qui décide d'une fin de
// partie : le fugitif muré ne peut plus bouger.
//
// Il lui reste ses dépenses et le passage, mais aucun déplacement — c'est ce
// que l'arbitre lira pour conclure.
func TestFugitiveWithNoLegalStep(t *testing.T) {
	p := gameOn(grid(
		"#####",
		"#####",
		"##.##",
		"#####",
		"#####",
	), Position{Column: 2, Row: 2})

	if got := arrivals(p.LegalMoves(SideFugitive)); got != nil {
		t.Errorf("%v, attendu aucun déplacement", got)
	}
}

// TestQuotaCountsDistinctPieces vérifie que le quota compte les pions et
// non les déplacements — ce qu'un simple compteur ne savait pas faire.
func TestQuotaCountsDistinctPieces(t *testing.T) {
	p := gameOn(grid(
		".....",
		".....",
		".....",
		".....",
		".....",
	), Position{Column: 4, Row: 4},
		Position{Column: 0, Row: 0},
		Position{Column: 1, Row: 0},
		Position{Column: 2, Row: 0},
		Position{Column: 3, Row: 0})
	p.Phase = PhaseInspectors
	p.Settings.PiecesPerTurn = 2

	// Deux pions distincts ont bougé : le quota est atteint, et les deux
	// autres n'ont plus de coup.
	p.Inspectors[0].StepsTaken = 1
	p.Inspectors[1].StepsTaken = 1

	for _, c := range p.LegalMoves(SideInspectors) {
		if c.Type == MoveStep && c.Piece >= 2 {
			t.Errorf("le pion %d bouge alors que le quota est atteint", c.Piece)
		}
	}
}

// TestStartedPieceContinuesOutsideQuota vérifie qu'un pion déjà déplacé poursuit sur
// sa mobilité propre sans prendre une place de plus.
func TestStartedPieceContinuesOutsideQuota(t *testing.T) {
	p := gameOn(grid(
		".....",
		".....",
		".....",
		".....",
		".....",
	), Position{Column: 4, Row: 4},
		Position{Column: 1, Row: 1},
		Position{Column: 3, Row: 1})
	p.Phase = PhaseInspectors
	p.Settings.PiecesPerTurn = 1

	// Le pion 0 a bougé et porte un bonus de mobilité : il lui reste un pas,
	// alors que le quota d'un seul pion est déjà consommé.
	p.Inspectors[0].StepsTaken = 1
	p.ActiveEffects = []ActiveEffect{{
		Effect:        Effect{Type: EffectChangeMobility, Target: TargetCurrentPiece, Value: 1, Duration: 1},
		EffectContext: EffectContext{Side: SideInspectors, Piece: 0},
		Echeance:      p.Turn,
	}}

	var pions []int
	for _, c := range p.LegalMoves(SideInspectors) {
		if c.Type == MoveStep {
			pions = append(pions, c.Piece)
		}
	}
	for _, pion := range pions {
		if pion != 0 {
			t.Errorf("le pion %d bouge alors que seul le pion entamé le peut", pion)
		}
	}
	if len(pions) == 0 {
		t.Error("le pion entamé devrait pouvoir continuer")
	}
}

// TestInspectorsOrthogonalOnly vérifie qu'ils n'ont pas les diagonales.
func TestInspectorsOrthogonalOnly(t *testing.T) {
	p := gameOn(grid(
		".....",
		".....",
		".....",
		".....",
		".....",
	), Position{Column: 4, Row: 4}, Position{Column: 2, Row: 2})
	p.Phase = PhaseInspectors

	if got := len(arrivals(p.LegalMoves(SideInspectors))); got != 4 {
		t.Errorf("%d déplacements, attendu 4", got)
	}
}

// TestInspectorsCanStack applique l'autre moitié de la décision : eux
// n'ont aucune raison d'être séparés, et ils n'y gagnent rien.
func TestInspectorsCanStack(t *testing.T) {
	p := gameOn(grid(
		".....",
		".....",
		".....",
		".....",
		".....",
	), Position{Column: 4, Row: 4},
		Position{Column: 2, Row: 2},
		Position{Column: 3, Row: 2})
	p.Phase = PhaseInspectors

	trouve := false
	for _, c := range p.LegalMoves(SideInspectors) {
		if c.Type == MoveStep && c.Piece == 0 && c.To == (Position{Column: 3, Row: 2}) {
			trouve = true
		}
	}
	if !trouve {
		t.Error("un inspecteur devrait pouvoir rejoindre la case d'un autre")
	}
}

// TestNoMoveOutOfTurn vérifie qu'un camp interrogé pendant la phase de
// l'autre n'obtient rien. C'est le serveur qui s'en sert pour refuser un coup
// arrivé trop tôt.
func TestNoMoveOutOfTurn(t *testing.T) {
	p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
		Position{Column: 2, Row: 2}, Position{Column: 0, Row: 0})

	if got := p.LegalMoves(SideInspectors); got != nil {
		t.Errorf("%d coups pour les inspecteurs pendant la phase du fugitif", len(got))
	}

	p.Phase = PhaseInspectors
	if got := p.LegalMoves(SideFugitive); got != nil {
		t.Errorf("%d coups pour le fugitif pendant la phase des inspecteurs", len(got))
	}

	p.Phase = PhaseOver
	if got := p.LegalMoves(SideFugitive); got != nil {
		t.Errorf("%d coups sur une partie terminée", len(got))
	}
}

// TestSealAZone vérifie que la mise en place propose les zones, et non des
// cases : la position de départ du fugitif est tirée au sort, il ne la choisit
// jamais.
func TestSealAZone(t *testing.T) {
	b := grid(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{{Number: 0}, {Number: 1}, {Number: 2}}
	p := gameOn(b, Position{Column: 2, Row: 2})
	p.Phase = PhaseFugitiveSetup

	coups := p.LegalMoves(SideFugitive)
	if len(coups) != 3 {
		t.Fatalf("%d coups, attendu une par zone", len(coups))
	}
	for _, c := range coups {
		if c.Type != MovePlace {
			t.Errorf("type %s, attendu placer", c.Type)
		}
		if c.To != (Position{}) {
			t.Error("sceller une zone ne désigne pas de case")
		}
	}
}

// TestSetupDoesNotExcludeFugitiveCell protège l'invariant de la vue filtrée
// à la mise en place.
//
// Retirer sa case de la liste dirait aux inspecteurs où il n'est pas, ce qui
// est une fuite exactement comme dire où il est.
func TestSetupDoesNotExcludeFugitiveCell(t *testing.T) {
	cache := Position{Column: 2, Row: 2}
	p := gameOn(grid(".....", ".....", ".....", ".....", "....."), cache)
	p.Phase = PhaseInspectorsSetup
	p.Settings.Inspectors = 2

	trouve := false
	for _, c := range p.LegalMoves(SideInspectors) {
		if c.To == cache {
			trouve = true
		}
	}
	if !trouve {
		t.Error("la case du fugitif est absente des placements, ce qui la trahit")
	}
}

// TestSetupStopsAtCount vérifie qu'on ne pose pas un sixième pion.
func TestSetupStopsAtCount(t *testing.T) {
	p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
		Position{Column: 2, Row: 2},
		Position{Column: 0, Row: 0}, Position{Column: 1, Row: 0})
	p.Phase = PhaseInspectorsSetup
	p.Settings.Inspectors = 2

	if got := p.LegalMoves(SideInspectors); got != nil {
		t.Errorf("%d coups alors que les deux pions sont placés", len(got))
	}
}

// TestChangeZone vérifie que le fugitif peut resceller ailleurs, et
// seulement ailleurs.
//
// La règle le facture 2 points et l'étranglement peut l'y contraindre en
// fermant sa zone : sans ce coup, une partie se perd sur une règle documentée
// mais injouable.
func TestChangeZone(t *testing.T) {
	b := grid(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{{Number: 0}, {Number: 1}, {Number: 2}}
	p := gameOn(b, Position{Column: 2, Row: 2})
	p.Fugitive.SealedZone = 1
	p.Extensions = &Registry{Depenses: map[Expense]Ability{
		ExpenseChangeZone: {Name: "Changement de zone", Camp: SideFugitive, Cost: 2},
	}}

	var zones []int
	for _, c := range p.LegalMoves(SideFugitive) {
		if c.Type == MoveChangeZone {
			zones = append(zones, c.Zone)
		}
	}
	if !reflect.DeepEqual(zones, []int{0, 2}) {
		t.Errorf("zones proposées %v, attendu [0 2]", zones)
	}
}

// TestChangeZoneTooExpensive vérifie qu'une dépense inabordable n'est pas
// proposée. Le fugitif à bout de résistance doit voir ce qu'il ne peut plus
// faire disparaître de ses coups, pas échouer en le tentant.
func TestChangeZoneTooExpensive(t *testing.T) {
	b := grid(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{{Number: 0}, {Number: 1}}
	p := gameOn(b, Position{Column: 2, Row: 2})
	p.Fugitive.Stamina = 1
	p.Extensions = &Registry{Depenses: map[Expense]Ability{
		ExpenseChangeZone: {Name: "Changement de zone", Camp: SideFugitive, Cost: 2},
	}}

	for _, c := range p.LegalMoves(SideFugitive) {
		if c.Type == MoveChangeZone {
			t.Error("une dépense à 2 est proposée avec 1 point de résistance")
		}
	}
}

// TestStableOrder vérifie que deux appels rendent la même liste dans le même
// ordre.
//
// L'ordre des coups légaux décide de ce qu'un rejeu doit retrouver et de ce que
// l'IA explore en premier. Un parcours de map le rendrait instable sans que
// rien ne le signale, puisque chaque exécution serait cohérente avec elle-même.
func TestStableOrder(t *testing.T) {
	p := gameOn(grid(".....", ".....", ".....", ".....", "....."),
		Position{Column: 2, Row: 2}, Position{Column: 0, Row: 0})

	premier := p.LegalMoves(SideFugitive)
	for i := 0; i < 20; i++ {
		if got := p.LegalMoves(SideFugitive); !reflect.DeepEqual(got, premier) {
			t.Fatalf("l'ordre a changé à l'appel %d", i)
		}
	}
}

// TestEndPhaseAlwaysOffered vérifie qu'un camp peut toujours rendre la
// main, même sans rien d'autre à play. Sans ce coup, une partie où plus rien
// n'est possible se figerait.
func TestEndPhaseAlwaysOffered(t *testing.T) {
	p := gameOn(grid("#####", "#####", "##.##", "#####", "#####"),
		Position{Column: 2, Row: 2})

	fin := false
	for _, c := range p.LegalMoves(SideFugitive) {
		if c.Type == MoveEndPhase {
			fin = true
		}
	}
	if !fin {
		t.Error("le fugitif muré ne peut pas rendre la main")
	}
}
