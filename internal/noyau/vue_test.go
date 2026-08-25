// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// partieCachee monte une partie où tout ce qui est secret a une valeur
// reconnaissable, de quoi repérer une fuite dans la vue quelle qu'en soit la
// forme.
func partieCachee() *Partie {
	b := trame(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{
		{Numero: 0, Cases: []Position{{Colonne: 0, Ligne: 0}}},
		{Numero: 3, Cases: []Position{{Colonne: 4, Ligne: 4}}},
	}
	p := partieSur(b, Position{Colonne: 2, Ligne: 2},
		Position{Colonne: 0, Ligne: 4}, Position{Colonne: 4, Ligne: 0})
	p.Extensions = registreDEssai()
	p.Fugitif.ZoneScellee = 3
	p.Fugitif.Resistance = 7
	p.Traces = map[Position]Trace{
		// Hors de portée des deux inspecteurs.
		{Colonne: 2, Ligne: 1}: {Tour: 2, Direction: Sud},
		// Adjacente à l'inspecteur 0, donc découverte.
		{Colonne: 1, Ligne: 4}: {Tour: 2, Direction: Est},
	}
	return p
}

// TestVueInspecteursNeFuitRien est le test que la feuille de route exige à
// l'étape 2.
//
// Il ne regarde pas les champs un par un : il sérialise la vue et cherche les
// valeurs secrètes dans le JSON produit. Un champ ajouté à Partie et recopié
// par mégarde serait attrapé, ce qu'une liste de vérifications nommées ne
// ferait pas.
func TestVueInspecteursNeFuitRien(t *testing.T) {
	p := partieCachee()
	p.Fugitif.Visible = false

	v := p.VuePour(CampInspecteurs)

	if v.PositionFugitif != nil {
		t.Errorf("la position du fugitif est dans la vue : %v", *v.PositionFugitif)
	}
	if v.ZoneScellee != nil {
		t.Errorf("la zone scellée est dans la vue : %d", *v.ZoneScellee)
	}
	if v.Resistance != nil {
		t.Errorf("la résistance est dans la vue : %d", *v.Resistance)
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

// TestVueFugitifVoitTout vérifie l'autre côté : rien ne lui est caché de
// lui-même.
func TestVueFugitifVoitTout(t *testing.T) {
	p := partieCachee()
	v := p.VuePour(CampFugitif)

	if v.PositionFugitif == nil || *v.PositionFugitif != p.Fugitif.Position {
		t.Error("le fugitif ne voit pas sa propre position")
	}
	if v.ZoneScellee == nil || *v.ZoneScellee != 3 {
		t.Error("le fugitif ne voit pas sa zone scellée")
	}
	if v.Resistance == nil || *v.Resistance != 7 {
		t.Error("le fugitif ne voit pas sa résistance")
	}
	if len(v.TracesConnues) != len(p.Traces) {
		t.Errorf("%d traces vues sur %d : il voit les siennes", len(v.TracesConnues), len(p.Traces))
	}
}

// TestVueMontreLeFugitifRepere vérifie que la position sort quand il est vu.
//
// C'est ce qui distingue « caché » de « invisible » : le repérage a un effet
// dans la vue, sinon voir le fugitif ne servirait à rien.
func TestVueMontreLeFugitifRepere(t *testing.T) {
	p := partieCachee()
	p.Fugitif.Visible = true

	v := p.VuePour(CampInspecteurs)
	if v.PositionFugitif == nil {
		t.Fatal("un fugitif repéré reste caché dans la vue")
	}
	if *v.PositionFugitif != p.Fugitif.Position {
		t.Errorf("position %v, attendu %v", *v.PositionFugitif, p.Fugitif.Position)
	}

	// Repéré ne veut pas dire déshabillé : sa zone et sa jauge restent à lui.
	if v.ZoneScellee != nil || v.Resistance != nil {
		t.Error("un fugitif repéré livre aussi sa zone ou sa résistance")
	}
}

// TestTracesFiltreesParPortee vérifie qu'un inspecteur ne découvre que ce qu'il
// touche, et en distance de Manhattan.
//
// La règle dit « en occupant la case ou une case orthogonalement adjacente ».
// En Tchebychev, les quatre diagonales entreraient aussi, ce qui doublerait
// presque la couverture — c'est le défaut sur lequel un prototype antérieur
// s'est fait prendre.
func TestTracesFiltreesParPortee(t *testing.T) {
	p := partieCachee()
	v := p.VuePour(CampInspecteurs)

	proche := Position{Colonne: 1, Ligne: 4}
	loin := Position{Colonne: 2, Ligne: 1}

	if _, vue := v.TracesConnues[proche.Cle()]; !vue {
		t.Error("une trace adjacente à un inspecteur n'est pas découverte")
	}
	if _, vue := v.TracesConnues[loin.Cle()]; vue {
		t.Error("une trace hors de portée est découverte")
	}
}

// TestTraceEnDiagonaleResteCachee éprouve précisément Manhattan contre
// Tchebychev.
func TestTraceEnDiagonaleResteCachee(t *testing.T) {
	p := partieCachee()
	diagonale := Position{Colonne: 1, Ligne: 3} // en diagonale de l'inspecteur 0
	p.Traces = map[Position]Trace{diagonale: {Tour: 2}}

	v := p.VuePour(CampInspecteurs)
	if _, vue := v.TracesConnues[diagonale.Cle()]; vue {
		t.Error("une trace en diagonale est découverte : la portée est en Tchebychev")
	}
}

// TestTraqueurEtendLaPortee vérifie que la capacité passive élargit la
// détection, sans toucher aux autres pions.
func TestTraqueurEtendLaPortee(t *testing.T) {
	p := partieCachee()
	loin := Position{Colonne: 2, Ligne: 4} // à deux pas de l'inspecteur 0
	p.Traces = map[Position]Trace{loin: {Tour: 2}}

	if _, vue := p.VuePour(CampInspecteurs).TracesConnues[loin.Cle()]; vue {
		t.Fatal("la trace est déjà vue sans le Traqueur")
	}

	p.EffetsActifs = []EffetActif{{
		Effet:    Effet{Type: EffetRevelerTraces, Cible: CiblePionCourant, Rayon: 2},
		Contexte: Contexte{Acteur: CampInspecteurs, Pion: 0},
	}}
	if _, vue := p.VuePour(CampInspecteurs).TracesConnues[loin.Cle()]; !vue {
		t.Error("le Traqueur ne découvre pas une trace à deux pas")
	}
}

// TestVuePorteLesInformationsPubliques vérifie que ce qui doit être partagé
// l'est, dans les deux vues.
func TestVuePorteLesInformationsPubliques(t *testing.T) {
	p := partieCachee()
	p.Scenes = []Scene{{Position: Position{Colonne: 1, Ligne: 1}, Tour: 2}}
	p.Barrages = map[Position]int{{Colonne: 3, Ligne: 3}: 5}
	p.ZonesFermees = []int{0}

	for _, camp := range []Acteur{CampFugitif, CampInspecteurs} {
		v := p.VuePour(camp)

		if len(v.Scenes) != 1 {
			t.Errorf("%s : %d scènes, attendu 1 — un meurtre est public", camp, len(v.Scenes))
		}
		if len(v.Barrages) != 1 {
			t.Errorf("%s : %d barrages, attendu 1", camp, len(v.Barrages))
		}
		if len(v.Zones) != 2 {
			t.Errorf("%s : %d zones, attendu 2", camp, len(v.Zones))
		}
		if len(v.Rues) == 0 {
			t.Errorf("%s : aucune rue, le plateau serait invisible", camp)
		}

		ferme := false
		for _, z := range v.Zones {
			if z.Numero == 0 && z.Fermee {
				ferme = true
			}
		}
		if !ferme {
			t.Errorf("%s : la zone fermée n'est pas marquée", camp)
		}
	}
}

// TestVueNeDonneQueSesCoupsLegaux vérifie qu'un camp ne lit pas les
// possibilités de l'autre.
func TestVueNeDonneQueSesCoupsLegaux(t *testing.T) {
	p := partieCachee()
	p.Phase = PhaseFugitif

	if len(p.VuePour(CampFugitif).CoupsLegaux) == 0 {
		t.Error("le fugitif n'a aucun coup pendant sa phase")
	}
	if got := len(p.VuePour(CampInspecteurs).CoupsLegaux); got != 0 {
		t.Errorf("%d coups offerts aux inspecteurs pendant la phase du fugitif", got)
	}
}

// TestEffetsAnnoncesSeulement vérifie qu'un différé sans annonce reste invisible.
//
// C'est le choix de son auteur de ne pas prévenir : le champ le trahirait.
func TestEffetsAnnoncesSeulement(t *testing.T) {
	p := partieCachee()
	p.EffetsEnAttente = []EffetEnAttente{
		{Effets: []Effet{{Type: EffetFermerZone}}, Tour: 9, Annonce: true,
			Contexte: Contexte{Zone: 3}},
		{Effets: []Effet{{Type: EffetBloquerCase}}, Tour: 9, Annonce: false,
			Contexte: Contexte{Case: Position{Colonne: 1, Ligne: 1}}},
	}

	v := p.VuePour(CampInspecteurs)
	if len(v.EffetsAnnonces) != 1 {
		t.Fatalf("%d effets annoncés, attendu 1", len(v.EffetsAnnonces))
	}
	if !reflect.DeepEqual(v.ZonesAnnoncees, []int{3}) {
		t.Errorf("zones annoncées %v, attendu [3]", v.ZonesAnnoncees)
	}
}

// TestVueEstStable vérifie que deux projections du même état sont identiques.
//
// Les cases et les traces sortent de maps : sans tri, la vue changerait d'un
// appel à l'autre, et deux clients du même état afficheraient des choses
// différentes.
func TestVueEstStable(t *testing.T) {
	p := partieCachee()
	p.Barrages = map[Position]int{
		{Colonne: 1, Ligne: 1}: 5,
		{Colonne: 3, Ligne: 3}: 5,
		{Colonne: 0, Ligne: 2}: 5,
	}

	premiere, err := json.Marshal(p.VuePour(CampInspecteurs))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		suivante, err := json.Marshal(p.VuePour(CampInspecteurs))
		if err != nil {
			t.Fatal(err)
		}
		if string(premiere) != string(suivante) {
			t.Fatalf("la vue a changé à l'appel %d", i)
		}
	}
}

// TestVueSeSerialise vérifie qu'elle passe le réseau sans perte.
//
// Un bot reçoit exactement cette structure : si elle ne se sérialise pas, le
// mode réseau et le protocole de bot tombent ensemble.
func TestVueSeSerialise(t *testing.T) {
	p := partieCachee()
	brut, err := json.Marshal(p.VuePour(CampFugitif))
	if err != nil {
		t.Fatal(err)
	}

	var relue Vue
	if err := json.Unmarshal(brut, &relue); err != nil {
		t.Fatalf("la vue ne se relit pas : %v", err)
	}
	if relue.PositionFugitif == nil || *relue.PositionFugitif != p.Fugitif.Position {
		t.Error("la position ne survit pas à l'aller-retour")
	}
	if len(relue.TracesConnues) != len(p.Traces) {
		t.Error("les traces ne survivent pas à l'aller-retour")
	}
}

// TestResultatDansLaVue vérifie qu'une partie finie le dit aux deux camps.
func TestResultatDansLaVue(t *testing.T) {
	p := partieCachee()
	if p.VuePour(CampFugitif).Resultat != nil {
		t.Fatal("une partie en cours porte un résultat")
	}

	p.Fugitif.Resistance = 0
	for _, camp := range []Acteur{CampFugitif, CampInspecteurs} {
		r := p.VuePour(camp).Resultat
		if r == nil {
			t.Fatalf("%s : la fin de partie n'est pas dans la vue", camp)
		}
		if r.Motif != MotifResistance {
			t.Errorf("%s : motif %s", camp, r.Motif)
		}
	}
}
