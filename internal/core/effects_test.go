// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"reflect"
	"testing"
)

// plateauNu est un terrain sans bâtiment, de quoi exercer les effets sans
// dépendre de la génération, qui relève de l'étape 3.
type plateauNu struct{ cote int }

// IsStreet accepte toute case dans les bornes.
func (b plateauNu) IsStreet(p Position) bool {
	return p.Column >= 0 && p.Row >= 0 && p.Column < b.cote && p.Row < b.cote
}

// Zones renvoie une unique zone, suffisante pour les bascules.
func (b plateauNu) Zones() []Zone {
	return []Zone{{Number: 0, Cells: []Position{{Column: 1, Row: 1}}}}
}

// Seed est figée : aucun test de ce fichier ne tire au sort.
func (b plateauNu) Seed() int64 { return 1 }

// Sight reste vide : ce sont les effets qu'on applique ici, pas la vue.
func (b plateauNu) Sight(p Position, portee int) []Position { return nil }

// CellsWithin énumère les cases du carré, dans l'ordre du plateau borné.
func (b plateauNu) CellsWithin(centre Position, rayon int) []Position {
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

// testGame monte une partie au tour 5, avec trois inspecteurs et un
// fugitif, de quoi appliquer n'importe quelle primitive.
func testGame() *Game {
	return &Game{
		Seed:     1,
		Settings: DefaultSettings(),
		Board:    plateauNu{cote: 9},
		Turn:     5,
		Phase:    PhaseInspectors,
		Fugitive: Fugitive{
			Position: Position{Column: 4, Row: 4},
			Stamina:  10,
		},
		Inspectors: []Inspector{
			{Position: Position{Column: 0, Row: 0}, Ability: "lookout"},
			{Position: Position{Column: 1, Row: 0}, Ability: "runner"},
			{Position: Position{Column: 2, Row: 0}, Ability: "chief"},
		},
		Trails: map[Position]Trail{
			{Column: 3, Row: 4}: {Turn: 4, Direction: Est},
			{Column: 2, Row: 4}: {Turn: 1, Direction: Est},
		},
		Roadblocks: map[Position]int{},
		Openings:   map[Position]int{},
	}
}

// casEffet décrit une primitive à appliquer et ce qu'elle doit produire.
type casEffet struct {
	nom string

	// prepare pose l'état que la primitive exige pour agir. Une primitive qui
	// n'a rien à défaire sort par une branche vide et ne prouve rien : c'est
	// exactement ce qui a laissé ouvrir_zone sans couverture.
	prepare func(*Game)

	effet   Effect
	ctx     EffectContext
	verifie func(*testing.T, *Game)
}

// gameFor monte une partie d'essai dans l'état qu'un cas réclame.
func (c casEffet) gameFor() *Game {
	p := testGame()
	if c.prepare != nil {
		c.prepare(p)
	}
	return p
}

// gameWithSetups monte une partie qui satisfait tous les cas à la fois,
// pour ceux qui les enchaînent sur un même état.
func gameWithSetups() *Game {
	p := testGame()
	for _, c := range tousLesCas() {
		if c.prepare != nil {
			c.prepare(p)
		}
	}
	return p
}

// tousLesCas couvre les dix-neuf primitives du vocabulaire. Une primitive
// absente d'ici est une primitive dont personne ne vérifie qu'elle se défait.
func tousLesCas() []casEffet {
	inspecteur := EffectContext{Side: SideInspectors, Piece: 1}
	fugitif := EffectContext{Side: SideFugitive}
	uneCase := Position{Column: 6, Row: 6}

	return []casEffet{
		{
			nom:   "deplacer",
			effet: Effect{Type: EffectMove, Target: TargetCurrentPiece},
			ctx:   EffectContext{Side: SideInspectors, Piece: 1, Case: uneCase},
			verifie: func(t *testing.T, p *Game) {
				if p.Inspectors[1].Position != uneCase {
					t.Errorf("pion en %v, attendu %v", p.Inspectors[1].Position, uneCase)
				}
			},
		},
		{
			nom:   "teleporter le fugitif",
			effet: Effect{Type: EffectTeleport, Target: TargetFugitive},
			ctx:   EffectContext{Side: SideFugitive, Case: uneCase},
			verifie: func(t *testing.T, p *Game) {
				if p.Fugitive.Position != uneCase {
					t.Errorf("fugitif en %v, attendu %v", p.Fugitive.Position, uneCase)
				}
			},
		},
		{
			nom:   "modifier_portee",
			effet: Effect{Type: EffectChangeRange, Target: TargetCurrentPiece, Value: 8, Duration: 1},
			ctx:   inspecteur,
			verifie: func(t *testing.T, p *Game) {
				if got := p.RangeOf(1); got != p.Settings.Range+8 {
					t.Errorf("portée du pion visé %d, attendu %d", got, p.Settings.Range+8)
				}
				if got := p.RangeOf(0); got != p.Settings.Range {
					t.Errorf("portée d'un autre pion %d, attendu %d", got, p.Settings.Range)
				}
			},
		},
		{
			nom:   "modifier_mobilite",
			effet: Effect{Type: EffectChangeMobility, Target: TargetFugitive, Value: 1, Duration: 1},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Game) {
				if got := p.MobilityOf(SideFugitive, 0); got != 2 {
					t.Errorf("mobilité du fugitif %d, attendu 2", got)
				}
			},
		},
		{
			nom:   "bloquer_case",
			effet: Effect{Type: EffectBlockCell, Target: TargetCell, Duration: 3},
			ctx:   EffectContext{Side: SideInspectors, Piece: 1, Case: uneCase},
			verifie: func(t *testing.T, p *Game) {
				if p.IsWalkable(uneCase) {
					t.Error("la case barrée reste praticable")
				}
			},
		},
		{
			nom:   "ouvrir_case",
			effet: Effect{Type: EffectOpenCell, Target: TargetCell, Duration: 3},
			ctx:   EffectContext{Side: SideInspectors, Piece: 1, Case: Position{Column: 99, Row: 99}},
			verifie: func(t *testing.T, p *Game) {
				if !p.IsWalkable(Position{Column: 99, Row: 99}) {
					t.Error("la case percée reste impraticable")
				}
			},
		},
		{
			nom:   "reveler_traces",
			effet: Effect{Type: EffectRevealTrails, Target: TargetCurrentPiece, Radius: 2},
			ctx:   inspecteur,
			verifie: func(t *testing.T, p *Game) {
				if got := p.TrailRadiusOf(1); got != 2 {
					t.Errorf("rayon du Traqueur %d, attendu 2", got)
				}
				if got := p.TrailRadiusOf(0); got != 1 {
					t.Errorf("rayon d'un autre pion %d, attendu 1", got)
				}
			},
		},
		{
			nom:   "reveler_position",
			effet: Effect{Type: EffectRevealPosition, Target: TargetFugitive},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Game) {
				if !p.Fugitive.Visible {
					t.Error("le fugitif reste caché après révélation")
				}
			},
		},
		{
			nom:   "marquer_scene",
			effet: Effect{Type: EffectMarkCrimeScene, Target: TargetFugitive},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Game) {
				attendue := CrimeScene{Position: Position{Column: 4, Row: 4}, Turn: 5}
				if !reflect.DeepEqual(p.CrimeScenes, []CrimeScene{attendue}) {
					t.Errorf("scènes %v, attendu [%v]", p.CrimeScenes, attendue)
				}
			},
		},
		{
			nom:   "annuler_revelation",
			effet: Effect{Type: EffectCancelReveal, Target: TargetFugitive},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Game) {
				if !p.Fugitive.SilenceBought {
					t.Error("le silence n'est pas enregistré")
				}
			},
		},
		{
			nom:   "partager_vue",
			effet: Effect{Type: EffectShareView, Target: TargetOtherPiece, Duration: 2},
			ctx:   EffectContext{Side: SideInspectors, Piece: 2, AutrePion: 0},
			verifie: func(t *testing.T, p *Game) {
				if got := p.SharedViewOf(2); !reflect.DeepEqual(got, []int{0}) {
					t.Errorf("vue partagée %v, attendu [0]", got)
				}
			},
		},
		{
			nom:   "couter_resistance",
			effet: Effect{Type: EffectCostStamina, Target: TargetFugitive, Value: 3},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Game) {
				if p.Fugitive.Stamina != 7 {
					t.Errorf("résistance %d, attendu 7", p.Fugitive.Stamina)
				}
			},
		},
		{
			nom:   "rendre_resistance",
			effet: Effect{Type: EffectRestoreStamina, Target: TargetFugitive, Value: 2},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Game) {
				if p.Fugitive.Stamina != 12 {
					t.Errorf("résistance %d, attendu 12", p.Fugitive.Stamina)
				}
			},
		},
		{
			nom:   "effacer_traces",
			effet: Effect{Type: EffectEraseTrails, Target: TargetFugitive, Duration: 3},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Game) {
				if _, reste := p.Trails[Position{Column: 3, Row: 4}]; reste {
					t.Error("la trace du tour 4 devait être effacée")
				}
				if _, reste := p.Trails[Position{Column: 2, Row: 4}]; !reste {
					t.Error("la trace du tour 1 est trop vieille pour être effacée")
				}
			},
		},
		{
			nom:   "fermer_zone",
			effet: Effect{Type: EffectCloseZone, Target: TargetZone},
			ctx:   EffectContext{Side: SideInspectors, Piece: 1, Zone: 3},
			verifie: func(t *testing.T, p *Game) {
				if !reflect.DeepEqual(p.ClosedZones, []int{3}) {
					t.Errorf("zones fermées %v, attendu [3]", p.ClosedZones)
				}
			},
		},
		{
			nom: "ouvrir_zone",
			// Trois zones fermées et celle qu'on rouvre au milieu : c'est son
			// rang qui compte. L'annulation doit la remettre entre 5 et 2, pas
			// au bout — cet ordre part dans le journal, et un rejeu qui le
			// retrouve différent n'est plus le même octet pour octet.
			prepare: func(p *Game) { p.ClosedZones = []int{5, 1, 2} },
			effet:   Effect{Type: EffectOpenZone, Target: TargetZone},
			ctx:     EffectContext{Side: SideInspectors, Piece: 1, Zone: 1},
			verifie: func(t *testing.T, p *Game) {
				if !reflect.DeepEqual(p.ClosedZones, []int{5, 2}) {
					t.Errorf("zones fermées %v, attendu [5 2]", p.ClosedZones)
				}
			},
		},
		{
			nom:   "sceller_zone",
			effet: Effect{Type: EffectSealZone, Target: TargetZone},
			ctx:   EffectContext{Side: SideFugitive, Zone: 4},
			verifie: func(t *testing.T, p *Game) {
				if p.Fugitive.SealedZone != 4 {
					t.Errorf("zone scellée %d, attendu 4", p.Fugitive.SealedZone)
				}
			},
		},
		{
			nom:   "differer",
			effet: Effect{Type: EffectDefer, Duration: 2, Announced: true, Then: []Effect{{Type: EffectCloseZone, Target: TargetZone}}},
			ctx:   EffectContext{Side: SideInspectors, Piece: 1, Zone: 2},
			verifie: func(t *testing.T, p *Game) {
				if len(p.PendingEffects) != 1 {
					t.Fatalf("%d effets en attente, attendu 1", len(p.PendingEffects))
				}
				if got := p.PendingEffects[0].Turn; got != 7 {
					t.Errorf("échéance au tour %d, attendu 7", got)
				}
				if !p.PendingEffects[0].Announced {
					t.Error("l'annonce n'est pas conservée")
				}
			},
		},
		{
			nom:   "fin_partie",
			effet: Effect{Type: EffectEndGame, Target: TargetFugitive},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Game) {
				r, fini := p.Outcome()
				if !fini {
					t.Fatal("la partie devait être terminée")
				}
				if r.Winner != SideFugitive || r.Reason != OutcomePlugin {
					t.Errorf("résultat %+v, attendu fugitif/plugin", r)
				}
			},
		},
	}
}

// TestApplyEffects vérifie que chaque primitive produit ce qu'elle annonce.
func TestApplyEffects(t *testing.T) {
	for _, cas := range tousLesCas() {
		t.Run(cas.nom, func(t *testing.T) {
			p := cas.gameFor()
			annuler, err := p.ApplyOneEffect(cas.effet, cas.ctx)
			if err != nil {
				t.Fatalf("application refusée : %v", err)
			}
			if annuler == nil {
				t.Fatal("aucune annulation renvoyée")
			}
			cas.verifie(t, p)
		})
	}
}

// TestUndoRestoresState est l'invariant de réversibilité.
//
// Ce n'est pas un confort d'interface : l'IA explore des milliers de positions
// en appliquant puis défaisant, sans copier l'état. Un effet qui laisse la
// moindre trace après annulation fait diverger l'exploration, puis le rejeu du
// journal.
func TestUndoRestoresState(t *testing.T) {
	for _, cas := range tousLesCas() {
		t.Run(cas.nom, func(t *testing.T) {
			p := cas.gameFor()
			avant := cas.gameFor()

			annuler, err := p.ApplyOneEffect(cas.effet, cas.ctx)
			if err != nil {
				t.Fatalf("application refusée : %v", err)
			}
			annuler()

			if !reflect.DeepEqual(p, avant) {
				t.Errorf("l'état diffère après annulation\n  obtenu : %+v\n  attendu: %+v", p, avant)
			}
		})
	}
}

// TestUndoInSequence vérifie que plusieurs effets se défont dans l'ordre
// inverse, ce dont les annulations qui tronquent une tranche dépendent.
func TestUndoInSequence(t *testing.T) {
	// Les préparations sont posées toutes ensemble avant la chaîne : appliquée
	// entre deux effets, une préparation modifierait l'état sans annulation
	// correspondante, et la comparaison finale échouerait sans qu'aucun effet
	// soit en cause.
	p := gameWithSetups()
	avant := gameWithSetups()

	var annulations []func()
	for _, cas := range tousLesCas() {
		annuler, err := p.ApplyOneEffect(cas.effet, cas.ctx)
		if err != nil {
			t.Fatalf("%s refusé : %v", cas.nom, err)
		}
		annulations = append(annulations, annuler)
	}

	for i := len(annulations) - 1; i >= 0; i-- {
		annulations[i]()
	}

	if !reflect.DeepEqual(p, avant) {
		t.Errorf("l'état diffère après annulation en chaîne\n  obtenu : %+v\n  attendu: %+v", p, avant)
	}
}

// TestUnknownEffectFails vérifie qu'un type hors vocabulaire est refusé plutôt
// qu'ignoré : un plugin entré sans validation doit s'entendre dire non.
func TestUnknownEffectFails(t *testing.T) {
	p := testGame()
	if _, err := p.ApplyOneEffect(Effect{Type: "voler"}, EffectContext{Side: SideFugitive}); err == nil {
		t.Fatal("un effet inconnu a été accepté")
	}
}

// TestPieceOutOfBoundsFails vérifie qu'un index de pion invalide est refusé.
// Sans ce contrôle, un contexte mal formé ferait paniquer le noyau.
func TestPieceOutOfBoundsFails(t *testing.T) {
	p := testGame()
	ctx := EffectContext{Side: SideInspectors, Piece: 9}
	if _, err := p.ApplyOneEffect(Effect{Type: EffectMove}, ctx); err == nil {
		t.Fatal("un pion hors bornes a été accepté")
	}
}

// TestEffectExpires vérifie qu'un effet à durée cesse de compter passé son
// échéance, et qu'un effet sans durée ne cesse jamais.
func TestEffectExpires(t *testing.T) {
	p := testGame()
	ctx := EffectContext{Side: SideInspectors, Piece: 0}

	if _, err := p.ApplyOneEffect(Effect{Type: EffectChangeRange, Target: TargetCurrentPiece, Value: 8, Duration: 1}, ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ApplyOneEffect(Effect{Type: EffectRevealTrails, Target: TargetCurrentPiece, Radius: 2}, ctx); err != nil {
		t.Fatal(err)
	}

	if got := p.RangeOf(0); got != p.Settings.Range+8 {
		t.Errorf("au tour du déclenchement, portée %d", got)
	}

	p.Turn++
	if got := p.RangeOf(0); got != p.Settings.Range {
		t.Errorf("au tour suivant, portée %d, attendu %d", got, p.Settings.Range)
	}
	if got := p.TrailRadiusOf(0); got != 2 {
		t.Errorf("un effet sans durée a expiré : rayon %d, attendu 2", got)
	}
}

// TestNegativeMobilityImmobilises vérifie que le vocabulaire tient sa promesse :
// une valeur négative est légale, et modifier_mobilite à -1 cloue le pion.
func TestNegativeMobilityImmobilises(t *testing.T) {
	p := testGame()
	ctx := EffectContext{Side: SideInspectors, Piece: 0}

	if _, err := p.ApplyOneEffect(Effect{Type: EffectChangeMobility, Target: TargetCurrentPiece, Value: -1, Duration: 1}, ctx); err != nil {
		t.Fatal(err)
	}
	if got := p.MobilityOf(SideInspectors, 0); got != 0 {
		t.Errorf("mobilité %d, attendu 0", got)
	}
}

// TestStaminaFloor vérifie que la résistance ne passe pas sous zéro, et
// que l'annulation rend la valeur d'origine malgré ce plafonnement.
func TestStaminaFloor(t *testing.T) {
	p := testGame()
	annuler, err := p.ApplyOneEffect(
		Effect{Type: EffectCostStamina, Target: TargetFugitive, Value: 99},
		EffectContext{Side: SideFugitive})
	if err != nil {
		t.Fatal(err)
	}
	if p.Fugitive.Stamina != 0 {
		t.Errorf("résistance %d, attendu 0", p.Fugitive.Stamina)
	}
	annuler()
	if p.Fugitive.Stamina != 10 {
		t.Errorf("après annulation, résistance %d, attendu 10", p.Fugitive.Stamina)
	}
}

// TestRoadblockBeatsOpening fixe l'ordre des couches de terrain. Sans
// priorité déclarée, le résultat dépendrait de l'ordre d'application et le
// rejeu du journal cesserait d'être reproductible.
func TestRoadblockBeatsOpening(t *testing.T) {
	p := testGame()
	pos := Position{Column: 3, Row: 3}
	ctx := EffectContext{Side: SideInspectors, Piece: 0, Case: pos}

	if _, err := p.ApplyOneEffect(Effect{Type: EffectOpenCell, Target: TargetCell, Duration: 3}, ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ApplyOneEffect(Effect{Type: EffectBlockCell, Target: TargetCell, Duration: 3}, ctx); err != nil {
		t.Fatal(err)
	}
	if p.IsWalkable(pos) {
		t.Error("le barrage doit l'emporter sur le percement")
	}
}
