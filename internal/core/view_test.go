// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// hiddenGame monte une partie où tout ce qui est secret a une valeur
// reconnaissable, de quoi repérer une fuite dans la vue quelle qu'en soit la
// forme.
func hiddenGame() *Game {
	b := grid(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{
		{Number: 0, Cells: []Position{{Column: 0, Row: 0}}},
		{Number: 3, Cells: []Position{{Column: 4, Row: 4}}},
	}
	p := gameOn(b, Position{Column: 2, Row: 2},
		Position{Column: 0, Row: 4}, Position{Column: 4, Row: 0})
	p.Extensions = testRegistry()
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
// par mégarde serait attrapé, ce qu'une liste de vérifications nommées ne
// ferait pas.
func TestInspectorsViewLeaksNothing(t *testing.T) {
	p := hiddenGame()
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
	// existe aussi dans les paramètres de partie, où il désigne la jauge de
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
	p := hiddenGame()
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
	if len(v.KnownTrails) != len(p.Trails) {
		t.Errorf("%d traces vues sur %d : il voit les siennes", len(v.KnownTrails), len(p.Trails))
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
	p := hiddenGame()
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
	p := hiddenGame()
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
	p := hiddenGame()
	v := p.ViewFor(SideInspectors)

	proche := Position{Column: 1, Row: 4}
	loin := Position{Column: 2, Row: 1}

	if _, vue := v.KnownTrails[proche.Key()]; !vue {
		t.Error("une trace adjacente à un inspecteur n'est pas découverte")
	}
	if _, vue := v.KnownTrails[loin.Key()]; vue {
		t.Error("une trace hors de portée est découverte")
	}
}

// TestDiagonalTrailStaysHidden éprouve précisément Manhattan contre
// Tchebychev.
func TestDiagonalTrailStaysHidden(t *testing.T) {
	p := hiddenGame()
	diagonale := Position{Column: 1, Row: 3} // en diagonale de l'inspecteur 0
	p.Trails = map[Position]Trail{diagonale: {Turn: 2}}

	v := p.ViewFor(SideInspectors)
	if _, vue := v.KnownTrails[diagonale.Key()]; vue {
		t.Error("une trace en diagonale est découverte : la portée est en Tchebychev")
	}
}

// TestTrackerExtendsRange vérifie que la capacité passive élargit la
// détection, sans toucher aux autres pions.
func TestTrackerExtendsRange(t *testing.T) {
	p := hiddenGame()
	loin := Position{Column: 2, Row: 4} // à deux pas de l'inspecteur 0
	p.Trails = map[Position]Trail{loin: {Turn: 2}}

	if _, vue := p.ViewFor(SideInspectors).KnownTrails[loin.Key()]; vue {
		t.Fatal("la trace est déjà vue sans le Traqueur")
	}

	p.ActiveEffects = []ActiveEffect{{
		Effect:        Effect{Type: EffectRevealTrails, Target: TargetCurrentPiece, Radius: 2},
		EffectContext: EffectContext{Side: SideInspectors, Piece: 0},
	}}
	if _, vue := p.ViewFor(SideInspectors).KnownTrails[loin.Key()]; !vue {
		t.Error("le Traqueur ne découvre pas une trace à deux pas")
	}
}

// TestAnnouncedDeferHidesTheFugitiveContext vérifie qu'un differer annoncé par
// le fugitif ne livre pas sa case aux inspecteurs.
//
// spend pose la position du fugitif dans le contexte, pour que le leurre sache
// où poser sa trace. Servi tel quel, il donnerait par la bande ce que la vue
// tait partout ailleurs — la position exacte, dans le champ le plus discret du
// programme.
//
// La dépense est jouée, pas construite : c'est spend qui remplit ce contexte,
// et un test qui poserait la file lui-même n'exercerait pas le chemin fautif.
func TestAnnouncedDeferHidesTheFugitiveContext(t *testing.T) {
	p := hiddenGame()
	p.Extensions.Expenses["plot"] = Ability{
		Name: "Complot", Camp: SideFugitive, Cost: 1,
		Trigger: OnFugitivePhase,
		Effects: []Effect{{
			Type: EffectDefer, Duration: 2, Announced: true,
			Then: []Effect{{Type: EffectBlockCell, Target: TargetCell, Duration: 1}},
		}},
	}

	var achat Move
	for _, c := range p.LegalMoves(SideFugitive) {
		if c.Type == MoveExpense && c.Expense == "plot" {
			achat = c
			break
		}
	}
	if achat.Type != MoveExpense {
		t.Fatal("la dépense n'est proposée à aucun coup légal")
	}
	if err := p.Apply(achat); err != nil {
		t.Fatal(err)
	}

	vues := p.ViewFor(SideInspectors).AnnouncedEffects
	if len(vues) != 1 {
		t.Fatalf("%d effet(s) annoncé(s) chez les inspecteurs, un seul attendu", len(vues))
	}
	if got := vues[0].EffectContext.Case; got != (Position{}) {
		t.Errorf("le contexte livre la case %v, qui est celle du fugitif", got)
	}
	if got := vues[0].EffectContext; got != (EffectContext{Side: SideFugitive}) {
		t.Errorf("contexte servi %+v, attendu vide hormis le camp", got)
	}

	// Le fugitif, lui, garde le sien : c'est le sien.
	siennes := p.ViewFor(SideFugitive).AnnouncedEffects
	if len(siennes) != 1 {
		t.Fatalf("%d effet(s) annoncé(s) chez le fugitif, un seul attendu", len(siennes))
	}
	if got := siennes[0].EffectContext.Case; got != p.Fugitive.Position {
		t.Errorf("le fugitif lit la case %v, attendu la sienne %v", got, p.Fugitive.Position)
	}
}

// TestAnnouncedDeferKeepsTheInspectorContext vérifie que le masquage ne porte
// que sur le camp caché.
//
// Les positions des inspecteurs sont publiques : masquer leur contexte
// retirerait à l'annonce ce qu'elle apporte — un barrage annoncé sans sa case
// ne se contourne pas, et le fugitif ne pourrait plus rien planifier.
func TestAnnouncedDeferKeepsTheInspectorContext(t *testing.T) {
	p := hiddenGame()
	cible := Position{Column: 3, Row: 1}
	if _, err := p.ApplyOneEffect(
		Effect{Type: EffectDefer, Duration: 2, Announced: true,
			Then: []Effect{{Type: EffectBlockCell, Target: TargetCell, Duration: 3}}},
		EffectContext{Side: SideInspectors, Piece: 0, Case: cible}); err != nil {
		t.Fatal(err)
	}

	vues := p.ViewFor(SideFugitive).AnnouncedEffects
	if len(vues) != 1 {
		t.Fatalf("%d effet(s) annoncé(s), un seul attendu", len(vues))
	}
	if got := vues[0].EffectContext.Case; got != cible {
		t.Errorf("case annoncée %v, attendu %v", got, cible)
	}
}

// TestNextRevealHitsZeroOnItsTurn vérifie le compte à rebours de révélation,
// zéro compris.
//
// Elle se déclenche en fin de résolution, donc après la phase du fugitif :
// annoncer une période entière à ce moment serait faux à l'instant précis où il
// décide d'acheter le silence ou de se montrer. La sentinelle sépare « c'est
// maintenant » de « jamais », que zéro confondrait.
func TestNextRevealHitsZeroOnItsTurn(t *testing.T) {
	p := hiddenGame()
	p.Settings.RevealPeriod = 4

	for _, c := range []struct{ tour, attendu int }{
		{1, 3}, {2, 2}, {3, 1}, {4, 0}, {5, 3}, {8, 0},
	} {
		p.Turn = c.tour
		if got := p.ViewFor(SideInspectors).ProchaineReveal; got != c.attendu {
			t.Errorf("au tour %d, révélation dans %d, attendu %d", c.tour, got, c.attendu)
		}
	}

	p.Settings.RevealPeriod = 0
	if got := p.ViewFor(SideInspectors).ProchaineReveal; got != -1 {
		t.Errorf("sans période, révélation dans %d, attendu -1", got)
	}
}

// TestViewCarriesPublicInformation vérifie que ce qui doit être partagé
// l'est, dans les deux vues.
func TestViewCarriesPublicInformation(t *testing.T) {
	p := hiddenGame()
	p.Roadblocks = map[Position]int{{Column: 3, Row: 3}: 5}
	p.ClosedZones = []int{0}

	for _, sideName := range []Side{SideFugitive, SideInspectors} {
		v := p.ViewFor(sideName)

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

// TestViewGivesOnlyItsLegalMoves vérifie qu'un camp ne lit pas les
// possibilités de l'autre.
func TestViewGivesOnlyItsLegalMoves(t *testing.T) {
	p := hiddenGame()
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
	p := hiddenGame()
	p.PendingEffects = []PendingEffect{
		{Effects: []Effect{{Type: EffectCloseZone}}, Turn: 9, Announced: true,
			EffectContext: EffectContext{Zone: 3}},
		{Effects: []Effect{{Type: EffectBlockCell}}, Turn: 9, Announced: false,
			EffectContext: EffectContext{Case: Position{Column: 1, Row: 1}}},
	}

	v := p.ViewFor(SideInspectors)
	if len(v.AnnouncedEffects) != 1 {
		t.Fatalf("%d effets annoncés, attendu 1", len(v.AnnouncedEffects))
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
	p := hiddenGame()
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
	p := hiddenGame()
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
	if len(relue.KnownTrails) != len(p.Trails) {
		t.Error("les traces ne survivent pas à l'aller-retour")
	}
}

// TestOutcomeInTheView vérifie qu'une partie finie le dit aux deux camps.
func TestOutcomeInTheView(t *testing.T) {
	p := hiddenGame()
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

// TestWatchedCellsReachBothSides vérifie que les cases surveillées partent aux
// deux camps.
//
// docs/regles.md §2 et §6 : les positions des inspecteurs sont déjà publiques et
// leurs lignes de vue s'en déduisent, donc les cacher ferait reposer l'équilibre
// sur la fatigue du joueur — quand son adversaire machine refait ce calcul
// entier à chaque coup.
func TestWatchedCellsReachBothSides(t *testing.T) {
	p := hiddenGame()

	inspecteurs := p.ViewFor(SideInspectors).CasesVisibles
	if len(inspecteurs) == 0 {
		t.Fatal("les inspecteurs ne voient aucune case")
	}

	fugitif := p.ViewFor(SideFugitive).CasesVisibles
	if len(fugitif) != len(inspecteurs) {
		t.Errorf("%d cases surveillées côté fugitif, %d côté inspecteurs",
			len(fugitif), len(inspecteurs))
	}
	for i := range fugitif {
		if fugitif[i] != inspecteurs[i] {
			t.Fatalf("les deux vues divergent en %d : %v contre %v",
				i, fugitif[i], inspecteurs[i])
		}
	}
}
