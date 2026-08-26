// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// partieCachee monte une partie où tout ce qui est secret a une valeur
// reconnaissable, de quoi repérer une fuite dans la vue quelle qu'en soit la
// forme.
func partieCachee() *Game {
	b := grid(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{
		{Number: 0, Cells: []Position{{Column: 0, Row: 0}}},
		{Number: 3, Cells: []Position{{Column: 4, Row: 4}}},
	}
	p := partieSur(b, Position{Column: 2, Row: 2},
		Position{Column: 0, Row: 4}, Position{Column: 4, Row: 0})
	p.Extensions = registreDEssai()
	p.Fugitive.SealedZone = 3
	p.Fugitive.Stamina = 7
	p.Trails = map[Position]Trail{
		// Hors de portée des deux inspecteurs.
		{Column: 2, Row: 1}: {Turn: 2, Direction: Sud},
		// Adjacente à l'inspecteur 0, donc découverte.
		{Column: 1, Row: 4}: {Turn: 2, Direction: Est},
	}
	return p
}

// TestInspectorsViewLeaksNothing est le test que la feuille de route exige à
// l'étape 2.
//
// Il ne regarde pas les champs un par un : il sérialise la vue et cherche les
// valeurs secrètes dans le JSON produit. Un champ ajouté à Game et recopié
// par mégarde serait attrapé, ce qu'une list de vérifications nommées ne
// ferait pas.
func TestInspectorsViewLeaksNothing(t *testing.T) {
	p := partieCachee()
	p.Fugitive.Visible = false

	v := p.ViewFor(SideInspectors)

	if v.PositionFugitif != nil {
		t.Errorf("la position du fugitif est dans la vue : %v", *v.PositionFugitif)
	}
	if v.SealedZone != nil {
		t.Errorf("la zone scellée est dans la vue : %d", *v.SealedZone)
	}
	if v.Stamina != nil {
		t.Errorf("la résistance est dans la vue : %d", *v.Stamina)
	}

	brut, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	// Les clés de la vue elle-même, et non le texte entier : « resistance »
	// existe aussi dans les paramètres de partie, où il désign la jauge de
	// départ — publique, et sans rapport avec ce qu'il en reste.
	var champs map[string]json.RawMessage
	if err := json.Unmarshal(brut, &champs); err != nil {
		t.Fatal(err)
	}
	for nom, cle := range map[string]string{
		"la zone scellée":        "zone_scellee",
		"la résistance restante": "resistance",
		"la position du fugitif": "position_fugitif",
	} {
		if _, present := champs[cle]; present {
			t.Errorf("%s est dans la vue sérialisée", nom)
		}
	}

	// Une trace hors de portée ne doit apparaître nulle part, y compris sous
	// une forme que les champs nommés ne montreraient pas.
	if strings.Contains(string(brut), `"2,1"`) {
		t.Error("une trace hors de portée apparaît dans la vue sérialisée")
	}
}

// TestFugitiveViewSeesAll vérifie l'autre côté : rien ne lui est caché de
// lui-même.
func TestFugitiveViewSeesAll(t *testing.T) {
	p := partieCachee()
	v := p.ViewFor(SideFugitive)

	if v.PositionFugitif == nil || *v.PositionFugitif != p.Fugitive.Position {
		t.Error("le fugitif ne voit pas sa propre position")
	}
	if v.SealedZone == nil || *v.SealedZone != 3 {
		t.Error("le fugitif ne voit pas sa zone scellée")
	}
	if v.Stamina == nil || *v.Stamina != 7 {
		t.Error("le fugitif ne voit pas sa résistance")
	}
	if len(v.TracesConnues) != len(p.Trails) {
		t.Errorf("%d traces vues sur %d : il voit les siennes", len(v.TracesConnues), len(p.Trails))
	}
}

// TestUnsealedZoneStaysAbsent vérifie que la sentinelle du noyau ne franchit
// pas la vue.
//
// SealedZone vaut -1 tant que le fugitif n'a pas choisi. Le champ étant un
// pointeur, l'omettre dit « pas encore choisie » ; le renseigner à -1 forcerait
// chaque lecteur du JSON à connaître une valeur magique pour comprendre la
// même chose.
func TestUnsealedZoneStaysAbsent(t *testing.T) {
	p := partieCachee()
	p.Fugitive.SealedZone = -1

	if v := p.ViewFor(SideFugitive); v.SealedZone != nil {
		t.Errorf("zone scellée %d alors qu'aucune ne l'est", *v.SealedZone)
	}
}

// TestViewShowsSpottedFugitive vérifie que la position sort quand il est vu.
//
// C'est ce qui distingue « caché » de « invisible » : le repérage a un effet
// dans la vue, sinon voir le fugitif ne servirait à rien.
func TestViewShowsSpottedFugitive(t *testing.T) {
	p := partieCachee()
	p.Fugitive.Visible = true

	v := p.ViewFor(SideInspectors)
	if v.PositionFugitif == nil {
		t.Fatal("un fugitif repéré reste caché dans la vue")
	}
	if *v.PositionFugitif != p.Fugitive.Position {
		t.Errorf("position %v, attendu %v", *v.PositionFugitif, p.Fugitive.Position)
	}

	// Repéré ne veut pas dire déshabillé : sa zone et sa jauge restent à lui.
	if v.SealedZone != nil || v.Stamina != nil {
		t.Error("un fugitif repéré livre aussi sa zone ou sa résistance")
	}
}

// TestTrailsFilteredByRange vérifie qu'un inspecteur ne découvre que ce qu'il
// touche, et en distance de Manhattan.
//
// La règle dit « en occupant la case ou une case orthogonalement adjacente ».
// En Tchebychev, les quatre diagonales entreraient aussi, ce qui doublerait
// presque la couverture — c'est le défaut sur lequel un prototype antérieur
// s'est fait prendre.
func TestTrailsFilteredByRange(t *testing.T) {
	p := partieCachee()
	v := p.ViewFor(SideInspectors)

	proche := Position{Column: 1, Row: 4}
	loin := Position{Column: 2, Row: 1}

	if _, vue := v.TracesConnues[proche.Key()]; !vue {
		t.Error("une trace adjacente à un inspecteur n'est pas découverte")
	}
	if _, vue := v.TracesConnues[loin.Key()]; vue {
		t.Error("une trace hors de portée est découverte")
	}
}

// TestDiagonalTrailStaysHidden éprouve précisément Manhattan contre
// Tchebychev.
func TestDiagonalTrailStaysHidden(t *testing.T) {
	p := partieCachee()
	diagonale := Position{Column: 1, Row: 3} // en diagonale de l'inspecteur 0
	p.Trails = map[Position]Trail{diagonale: {Turn: 2}}

	v := p.ViewFor(SideInspectors)
	if _, vue := v.TracesConnues[diagonale.Key()]; vue {
		t.Error("une trace en diagonale est découverte : la portée est en Tchebychev")
	}
}

// TestTrackerExtendsRange vérifie que la capacité passive élargit la
// détection, sans toucher aux autres pions.
func TestTrackerExtendsRange(t *testing.T) {
	p := partieCachee()
	loin := Position{Column: 2, Row: 4} // à deux pas de l'inspecteur 0
	p.Trails = map[Position]Trail{loin: {Turn: 2}}

	if _, vue := p.ViewFor(SideInspectors).TracesConnues[loin.Key()]; vue {
		t.Fatal("la trace est déjà vue sans le Traqueur")
	}

	p.ActiveEffects = []ActiveEffect{{
		Effect:        Effect{Type: EffectRevealTrails, Target: TargetCurrentPiece, Radius: 2},
		EffectContext: EffectContext{Side: SideInspectors, Piece: 0},
	}}
	if _, vue := p.ViewFor(SideInspectors).TracesConnues[loin.Key()]; !vue {
		t.Error("le Traqueur ne découvre pas une trace à deux pas")
	}
}

// TestViewCarriesPublicInformation vérifie que ce qui doit être partagé
// l'est, dans les deux vues.
func TestViewCarriesPublicInformation(t *testing.T) {
	p := partieCachee()
	p.CrimeScenes = []CrimeScene{{Position: Position{Column: 1, Row: 1}, Turn: 2}}
	p.Roadblocks = map[Position]int{{Column: 3, Row: 3}: 5}
	p.ClosedZones = []int{0}

	for _, sideName := range []Side{SideFugitive, SideInspectors} {
		v := p.ViewFor(sideName)

		if len(v.CrimeScenes) != 1 {
			t.Errorf("%s : %d scènes, attendu 1 — un meurtre est public", sideName, len(v.CrimeScenes))
		}
		if len(v.Roadblocks) != 1 {
			t.Errorf("%s : %d barrages, attendu 1", sideName, len(v.Roadblocks))
		}
		if len(v.Zones) != 2 {
			t.Errorf("%s : %d zones, attendu 2", sideName, len(v.Zones))
		}
		if len(v.Streets) == 0 {
			t.Errorf("%s : aucune rue, le plateau serait invisible", sideName)
		}

		ferme := false
		for _, z := range v.Zones {
			if z.Number == 0 && z.Closed {
				ferme = true
			}
		}
		if !ferme {
			t.Errorf("%s : la zone fermée n'est pas marquée", sideName)
		}
	}
}

// TestViewGivesOnlyItsLegalMoves vérifie qu'un sideName ne lit pas les
// possibilités de l'autre.
func TestViewGivesOnlyItsLegalMoves(t *testing.T) {
	p := partieCachee()
	p.Phase = PhaseFugitive

	if len(p.ViewFor(SideFugitive).LegalMoves) == 0 {
		t.Error("le fugitif n'a aucun coup pendant sa phase")
	}
	if got := len(p.ViewFor(SideInspectors).LegalMoves); got != 0 {
		t.Errorf("%d coups offerts aux inspecteurs pendant la phase du fugitif", got)
	}
}

// TestOnlyAnnouncedEffects vérifie qu'un différé sans annonce reste invisible.
//
// C'est le choix de son auteur de ne pas prévenir : le champ le trahirait.
func TestOnlyAnnouncedEffects(t *testing.T) {
	p := partieCachee()
	p.PendingEffects = []PendingEffect{
		{Effets: []Effect{{Type: EffectCloseZone}}, Turn: 9, Announced: true,
			EffectContext: EffectContext{Zone: 3}},
		{Effets: []Effect{{Type: EffectBlockCell}}, Turn: 9, Announced: false,
			EffectContext: EffectContext{Case: Position{Column: 1, Row: 1}}},
	}

	v := p.ViewFor(SideInspectors)
	if len(v.EffetsAnnonces) != 1 {
		t.Fatalf("%d effets annoncés, attendu 1", len(v.EffetsAnnonces))
	}
	if !reflect.DeepEqual(v.ZonesAnnoncees, []int{3}) {
		t.Errorf("zones annoncées %v, attendu [3]", v.ZonesAnnoncees)
	}
}

// TestViewIsStable vérifie que deux projections du même état sont identiques.
//
// Les cases et les traces sortent de maps : sans tri, la vue changerait d'un
// appel à l'autre, et deux clients du même état afficheraient des choses
// différentes.
func TestViewIsStable(t *testing.T) {
	p := partieCachee()
	p.Roadblocks = map[Position]int{
		{Column: 1, Row: 1}: 5,
		{Column: 3, Row: 3}: 5,
		{Column: 0, Row: 2}: 5,
	}

	premiere, err := json.Marshal(p.ViewFor(SideInspectors))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		suivante, err := json.Marshal(p.ViewFor(SideInspectors))
		if err != nil {
			t.Fatal(err)
		}
		if string(premiere) != string(suivante) {
			t.Fatalf("la vue a changé à l'appel %d", i)
		}
	}
}

// TestViewSerialises vérifie qu'elle passe le réseau sans perte.
//
// Un bot reçoit exactement cette structure : si elle ne se sérialise pas, le
// mode réseau et le protocole de bot tombent ensemble.
func TestViewSerialises(t *testing.T) {
	p := partieCachee()
	brut, err := json.Marshal(p.ViewFor(SideFugitive))
	if err != nil {
		t.Fatal(err)
	}

	var relue View
	if err := json.Unmarshal(brut, &relue); err != nil {
		t.Fatalf("la vue ne se relit pas : %v", err)
	}
	if relue.PositionFugitif == nil || *relue.PositionFugitif != p.Fugitive.Position {
		t.Error("la position ne survit pas à l'aller-retour")
	}
	if len(relue.TracesConnues) != len(p.Trails) {
		t.Error("les traces ne survivent pas à l'aller-retour")
	}
}

// TestOutcomeInTheView vérifie qu'une partie finie le dit aux deux camps.
func TestOutcomeInTheView(t *testing.T) {
	p := partieCachee()
	if p.ViewFor(SideFugitive).Outcome != nil {
		t.Fatal("une partie en cours porte un résultat")
	}

	p.Fugitive.Stamina = 0
	for _, sideName := range []Side{SideFugitive, SideInspectors} {
		r := p.ViewFor(sideName).Outcome
		if r == nil {
			t.Fatalf("%s : la fin de partie n'est pas dans la vue", sideName)
		}
		if r.Reason != OutcomeStaminaSpent {
			t.Errorf("%s : motif %s", sideName, r.Reason)
		}
	}
}
