// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

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

// trame construit un plateau depuis un dessin, « # » pour un bâtiment.
//
// Un dessin vaut mieux qu'une liste de coordonnées : le cas testé se lit d'un
// coup d'œil, et une erreur d'index ne passe pas inaperçue.
func trame(lignes ...string) *plateauTrame {
	b := &plateauTrame{batiments: map[Position]bool{}}
	for l, ligne := range lignes {
		for c, r := range ligne {
			if r == '#' {
				b.batiments[Position{Colonne: c, Ligne: l}] = true
			}
		}
	}
	b.cote = lignes[0]
	return b
}

// EstRue traite le hors-limites comme du bâti, comme le plateau borné.
func (b *plateauTrame) EstRue(p Position) bool {
	if p.Colonne < 0 || p.Ligne < 0 || p.Colonne >= len(b.cote) || p.Ligne >= len(b.cote) {
		return false
	}
	return !b.batiments[p]
}

// Zones renvoie les zones déclarées par le cas de test.
func (b *plateauTrame) Zones() []Zone { return b.zones }

// Graine est figée : aucun test de ce fichier ne tire au sort.
func (b *plateauTrame) Graine() int64 { return 1 }

// Vision reste vide : la légalité d'un coup ne dépend pas de ce qu'on voit.
func (b *plateauTrame) Vision(p Position, portee int) []Position { return nil }

// CasesDans énumère les rues du carré, ligne par ligne.
func (b *plateauTrame) CasesDans(centre Position, rayon int) []Position {
	var cases []Position
	for ligne := centre.Ligne - rayon; ligne <= centre.Ligne+rayon; ligne++ {
		for colonne := centre.Colonne - rayon; colonne <= centre.Colonne+rayon; colonne++ {
			if p := (Position{Colonne: colonne, Ligne: ligne}); b.EstRue(p) {
				cases = append(cases, p)
			}
		}
	}
	return cases
}

// partieSur monte une partie en phase fugitif sur un plateau dessiné.
func partieSur(b *plateauTrame, fugitif Position, inspecteurs ...Position) *Partie {
	p := &Partie{
		Parametres: ParametresDefaut(),
		Plateau:    b,
		Tour:       3,
		Phase:      PhaseFugitif,
		Fugitif:    Fugitif{Position: fugitif, Resistance: 10},
	}
	p.Parametres.Cote = len(b.cote)
	for _, pos := range inspecteurs {
		p.Inspecteurs = append(p.Inspecteurs, Inspecteur{Position: pos})
	}
	return p
}

// arrivees extrait les destinations des déplacements, pour comparer sans se
// soucier des autres champs.
func arrivees(coups []Coup) []Position {
	var positions []Position
	for _, c := range coups {
		if c.Type == CoupDeplacer {
			positions = append(positions, c.Arrivee)
		}
	}
	return positions
}

// TestFugitifHuitDirections vérifie qu'en terrain dégagé le fugitif dispose de
// ses huit directions, là où un inspecteur n'en a que quatre.
func TestFugitifHuitDirections(t *testing.T) {
	p := partieSur(trame(
		".....",
		".....",
		".....",
		".....",
		".....",
	), Position{Colonne: 2, Ligne: 2})

	if got := len(arrivees(p.CoupsLegaux(CampFugitif))); got != 8 {
		t.Errorf("%d déplacements, attendu 8", got)
	}
}

// TestDiagonaleBloqueeParAngleFerme est le cas limite de docs/regles.md §2.
//
// Une diagonale exige qu'au moins une des deux cases orthogonales
// intermédiaires soit une rue. Sans cette règle, on traverse les angles de
// bâtiments et le bâti ne bloque plus rien.
func TestDiagonaleBloqueeParAngleFerme(t *testing.T) {
	// Le coin nord-est du fugitif est fermé par deux bâtiments en équerre, en
	// (2,1) et (3,2). La case visée (3,1) est libre, et c'est tout l'intérêt :
	// elle serait atteignable si seule la praticabilité comptait.
	p := partieSur(trame(
		".....",
		"..#..",
		"...#.",
		".....",
		".....",
	), Position{Colonne: 2, Ligne: 2})

	if !p.EstPraticable(Position{Colonne: 3, Ligne: 1}) {
		t.Fatal("la case visée doit être libre, sinon le test ne prouve rien")
	}

	for _, a := range arrivees(p.CoupsLegaux(CampFugitif)) {
		if a == (Position{Colonne: 3, Ligne: 1}) {
			t.Error("la diagonale traverse un angle fermé")
		}
	}
}

// TestDiagonaleOuverteParUnSeulCote vérifie l'autre moitié de la règle : une
// seule des deux cases intermédiaires suffit.
func TestDiagonaleOuverteParUnSeulCote(t *testing.T) {
	// Seul (3,2) est bâti ; (2,1) reste libre, et suffit.
	p := partieSur(trame(
		".....",
		".....",
		"...#.",
		".....",
		".....",
	), Position{Colonne: 2, Ligne: 2})

	trouve := false
	for _, a := range arrivees(p.CoupsLegaux(CampFugitif)) {
		if a == (Position{Colonne: 3, Ligne: 1}) {
			trouve = true
		}
	}
	if !trouve {
		t.Error("la diagonale devrait passer par la case orthogonale libre")
	}
}

// TestFugitifNEntrePasSurUnInspecteur applique la décision de docs/regles.md
// §5 : il y serait à l'abri de tout contact, l'adjacence n'incluant pas la
// superposition.
func TestFugitifNEntrePasSurUnInspecteur(t *testing.T) {
	p := partieSur(trame(
		".....",
		".....",
		".....",
		".....",
		".....",
	), Position{Colonne: 2, Ligne: 2}, Position{Colonne: 3, Ligne: 2})

	for _, a := range arrivees(p.CoupsLegaux(CampFugitif)) {
		if a == (Position{Colonne: 3, Ligne: 2}) {
			t.Error("le fugitif peut se poser sous un inspecteur")
		}
	}
}

// TestFugitifSansDeplacementLegal est le cas limite qui décide d'une fin de
// partie : le fugitif muré ne peut plus bouger.
//
// Il lui reste ses dépenses et le passage, mais aucun déplacement — c'est ce
// que l'arbitre lira pour conclure.
func TestFugitifSansDeplacementLegal(t *testing.T) {
	p := partieSur(trame(
		"#####",
		"#####",
		"##.##",
		"#####",
		"#####",
	), Position{Colonne: 2, Ligne: 2})

	if got := arrivees(p.CoupsLegaux(CampFugitif)); got != nil {
		t.Errorf("%v, attendu aucun déplacement", got)
	}
}

// TestQuotaPorteSurDesPionsDistincts vérifie que le quota compte les pions et
// non les déplacements — ce qu'un simple compteur ne savait pas faire.
func TestQuotaPorteSurDesPionsDistincts(t *testing.T) {
	p := partieSur(trame(
		".....",
		".....",
		".....",
		".....",
		".....",
	), Position{Colonne: 4, Ligne: 4},
		Position{Colonne: 0, Ligne: 0},
		Position{Colonne: 1, Ligne: 0},
		Position{Colonne: 2, Ligne: 0},
		Position{Colonne: 3, Ligne: 0})
	p.Phase = PhaseInspecteurs
	p.Parametres.PionsParTour = 2

	// Deux pions distincts ont bougé : le quota est atteint, et les deux
	// autres n'ont plus de coup.
	p.Inspecteurs[0].DeplacementsFaits = 1
	p.Inspecteurs[1].DeplacementsFaits = 1

	for _, c := range p.CoupsLegaux(CampInspecteurs) {
		if c.Type == CoupDeplacer && c.Pion >= 2 {
			t.Errorf("le pion %d bouge alors que le quota est atteint", c.Pion)
		}
	}
}

// TestPionEntameContinueHorsQuota vérifie qu'un pion déjà déplacé poursuit sur
// sa mobilité propre sans prendre une place de plus.
func TestPionEntameContinueHorsQuota(t *testing.T) {
	p := partieSur(trame(
		".....",
		".....",
		".....",
		".....",
		".....",
	), Position{Colonne: 4, Ligne: 4},
		Position{Colonne: 1, Ligne: 1},
		Position{Colonne: 3, Ligne: 1})
	p.Phase = PhaseInspecteurs
	p.Parametres.PionsParTour = 1

	// Le pion 0 a bougé et porte un bonus de mobilité : il lui reste un pas,
	// alors que le quota d'un seul pion est déjà consommé.
	p.Inspecteurs[0].DeplacementsFaits = 1
	p.EffetsActifs = []EffetActif{{
		Effet:    Effet{Type: EffetModifierMobilite, Cible: CiblePionCourant, Valeur: 1, Duree: 1},
		Contexte: Contexte{Acteur: CampInspecteurs, Pion: 0},
		Echeance: p.Tour,
	}}

	var pions []int
	for _, c := range p.CoupsLegaux(CampInspecteurs) {
		if c.Type == CoupDeplacer {
			pions = append(pions, c.Pion)
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

// TestInspecteursOrthogonalesSeulement vérifie qu'ils n'ont pas les diagonales.
func TestInspecteursOrthogonalesSeulement(t *testing.T) {
	p := partieSur(trame(
		".....",
		".....",
		".....",
		".....",
		".....",
	), Position{Colonne: 4, Ligne: 4}, Position{Colonne: 2, Ligne: 2})
	p.Phase = PhaseInspecteurs

	if got := len(arrivees(p.CoupsLegaux(CampInspecteurs))); got != 4 {
		t.Errorf("%d déplacements, attendu 4", got)
	}
}

// TestInspecteursPeuventSEmpiler applique l'autre moitié de la décision : eux
// n'ont aucune raison d'être séparés, et ils n'y gagnent rien.
func TestInspecteursPeuventSEmpiler(t *testing.T) {
	p := partieSur(trame(
		".....",
		".....",
		".....",
		".....",
		".....",
	), Position{Colonne: 4, Ligne: 4},
		Position{Colonne: 2, Ligne: 2},
		Position{Colonne: 3, Ligne: 2})
	p.Phase = PhaseInspecteurs

	trouve := false
	for _, c := range p.CoupsLegaux(CampInspecteurs) {
		if c.Type == CoupDeplacer && c.Pion == 0 && c.Arrivee == (Position{Colonne: 3, Ligne: 2}) {
			trouve = true
		}
	}
	if !trouve {
		t.Error("un inspecteur devrait pouvoir rejoindre la case d'un autre")
	}
}

// TestPasDeCoupHorsDeSonTour vérifie qu'un camp interrogé pendant la phase de
// l'autre n'obtient rien. C'est le serveur qui s'en sert pour refuser un coup
// arrivé trop tôt.
func TestPasDeCoupHorsDeSonTour(t *testing.T) {
	p := partieSur(trame(".....", ".....", ".....", ".....", "....."),
		Position{Colonne: 2, Ligne: 2}, Position{Colonne: 0, Ligne: 0})

	if got := p.CoupsLegaux(CampInspecteurs); got != nil {
		t.Errorf("%d coups pour les inspecteurs pendant la phase du fugitif", len(got))
	}

	p.Phase = PhaseInspecteurs
	if got := p.CoupsLegaux(CampFugitif); got != nil {
		t.Errorf("%d coups pour le fugitif pendant la phase des inspecteurs", len(got))
	}

	p.Phase = PhaseTerminee
	if got := p.CoupsLegaux(CampFugitif); got != nil {
		t.Errorf("%d coups sur une partie terminée", len(got))
	}
}

// TestScellerUneZone vérifie que la mise en place propose les zones, et non des
// cases : la position de départ du fugitif est tirée au sort, il ne la choisit
// jamais.
func TestScellerUneZone(t *testing.T) {
	b := trame(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{{Numero: 0}, {Numero: 1}, {Numero: 2}}
	p := partieSur(b, Position{Colonne: 2, Ligne: 2})
	p.Phase = PhasePlacementFugitif

	coups := p.CoupsLegaux(CampFugitif)
	if len(coups) != 3 {
		t.Fatalf("%d coups, attendu une par zone", len(coups))
	}
	for _, c := range coups {
		if c.Type != CoupPlacer {
			t.Errorf("type %s, attendu placer", c.Type)
		}
		if c.Arrivee != (Position{}) {
			t.Error("sceller une zone ne désigne pas de case")
		}
	}
}

// TestPlacementNExclutPasLaCaseDuFugitif protège l'invariant de la vue filtrée
// à la mise en place.
//
// Retirer sa case de la liste dirait aux inspecteurs où il n'est pas, ce qui
// est une fuite exactement comme dire où il est.
func TestPlacementNExclutPasLaCaseDuFugitif(t *testing.T) {
	cache := Position{Colonne: 2, Ligne: 2}
	p := partieSur(trame(".....", ".....", ".....", ".....", "....."), cache)
	p.Phase = PhasePlacementInspecteurs
	p.Parametres.Inspecteurs = 2

	trouve := false
	for _, c := range p.CoupsLegaux(CampInspecteurs) {
		if c.Arrivee == cache {
			trouve = true
		}
	}
	if !trouve {
		t.Error("la case du fugitif est absente des placements, ce qui la trahit")
	}
}

// TestPlacementSArreteAuCompte vérifie qu'on ne pose pas un sixième pion.
func TestPlacementSArreteAuCompte(t *testing.T) {
	p := partieSur(trame(".....", ".....", ".....", ".....", "....."),
		Position{Colonne: 2, Ligne: 2},
		Position{Colonne: 0, Ligne: 0}, Position{Colonne: 1, Ligne: 0})
	p.Phase = PhasePlacementInspecteurs
	p.Parametres.Inspecteurs = 2

	if got := p.CoupsLegaux(CampInspecteurs); got != nil {
		t.Errorf("%d coups alors que les deux pions sont placés", len(got))
	}
}

// TestChangerDeZone vérifie que le fugitif peut resceller ailleurs, et
// seulement ailleurs.
//
// La règle le facture 2 points et l'étranglement peut l'y contraindre en
// fermant sa zone : sans ce coup, une partie se perd sur une règle documentée
// mais injouable.
func TestChangerDeZone(t *testing.T) {
	b := trame(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{{Numero: 0}, {Numero: 1}, {Numero: 2}}
	p := partieSur(b, Position{Colonne: 2, Ligne: 2})
	p.Fugitif.ZoneScellee = 1
	p.Extensions = &Registre{Depenses: map[Depense]Capacite{
		DepenseChangerZone: {Nom: "Changement de zone", Camp: CampFugitif, Cout: 2},
	}}

	var zones []int
	for _, c := range p.CoupsLegaux(CampFugitif) {
		if c.Type == CoupChangerZone {
			zones = append(zones, c.Zone)
		}
	}
	if !reflect.DeepEqual(zones, []int{0, 2}) {
		t.Errorf("zones proposées %v, attendu [0 2]", zones)
	}
}

// TestChangerDeZoneHorsDePrix vérifie qu'une dépense inabordable n'est pas
// proposée. Le fugitif à bout de résistance doit voir ce qu'il ne peut plus
// faire disparaître de ses coups, pas échouer en le tentant.
func TestChangerDeZoneHorsDePrix(t *testing.T) {
	b := trame(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{{Numero: 0}, {Numero: 1}}
	p := partieSur(b, Position{Colonne: 2, Ligne: 2})
	p.Fugitif.Resistance = 1
	p.Extensions = &Registre{Depenses: map[Depense]Capacite{
		DepenseChangerZone: {Nom: "Changement de zone", Camp: CampFugitif, Cout: 2},
	}}

	for _, c := range p.CoupsLegaux(CampFugitif) {
		if c.Type == CoupChangerZone {
			t.Error("une dépense à 2 est proposée avec 1 point de résistance")
		}
	}
}

// TestOrdreStable vérifie que deux appels rendent la même liste dans le même
// ordre.
//
// L'ordre des coups légaux décide de ce qu'un rejeu doit retrouver et de ce que
// l'IA explore en premier. Un parcours de map le rendrait instable sans que
// rien ne le signale, puisque chaque exécution serait cohérente avec elle-même.
func TestOrdreStable(t *testing.T) {
	p := partieSur(trame(".....", ".....", ".....", ".....", "....."),
		Position{Colonne: 2, Ligne: 2}, Position{Colonne: 0, Ligne: 0})

	premier := p.CoupsLegaux(CampFugitif)
	for i := 0; i < 20; i++ {
		if got := p.CoupsLegaux(CampFugitif); !reflect.DeepEqual(got, premier) {
			t.Fatalf("l'ordre a changé à l'appel %d", i)
		}
	}
}

// TestFinDePhaseToujoursOfferte vérifie qu'un camp peut toujours rendre la
// main, même sans rien d'autre à jouer. Sans ce coup, une partie où plus rien
// n'est possible se figerait.
func TestFinDePhaseToujoursOfferte(t *testing.T) {
	p := partieSur(trame("#####", "#####", "##.##", "#####", "#####"),
		Position{Colonne: 2, Ligne: 2})

	fin := false
	for _, c := range p.CoupsLegaux(CampFugitif) {
		if c.Type == CoupFinDePhase {
			fin = true
		}
	}
	if !fin {
		t.Error("le fugitif muré ne peut pas rendre la main")
	}
}
