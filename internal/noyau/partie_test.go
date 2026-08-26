// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import (
	"reflect"
	"testing"
)

// plateauOuvert est un terrain dégagé de la taille demandée, avec ses zones
// d'extraction en périphérie.
//
// Un vrai plateau viendra de Generer, à l'étape 3. Celui-ci suffit à jouer une
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
		{Colonne: 0, Ligne: 0},
		{Colonne: cote - 1, Ligne: 0},
		{Colonne: 0, Ligne: cote - 1},
		{Colonne: cote - 1, Ligne: cote - 1},
		{Colonne: cote / 2, Ligne: 0},
		{Colonne: cote / 2, Ligne: cote - 1},
	}
	for i, c := range coins {
		b.zones = append(b.zones, Zone{Numero: i, Cases: []Position{c}})
	}
	return b
}

// EstRue accepte toute case dans les bornes.
func (b *plateauOuvert) EstRue(p Position) bool {
	return p.Colonne >= 0 && p.Ligne >= 0 && p.Colonne < b.cote && p.Ligne < b.cote
}

// Zones renvoie les six points d'extraction.
func (b *plateauOuvert) Zones() []Zone { return b.zones }

// Graine est figée : le tirage vient de la partie, pas du plateau.
func (b *plateauOuvert) Graine() int64 { return 1 }

// Vision déroule les huit directions à la demande.
//
// Pas de table précalculée ici : sur un terrain sans le moindre bâtiment, la
// ligne se calcule aussi vite qu'elle se lirait, et un plateau d'essai n'a pas
// à porter la mémoire d'un vrai.
func (b *plateauOuvert) Vision(p Position, portee int) []Position {
	if !b.EstRue(p) || portee <= 0 {
		return nil
	}

	var vues []Position
	for d := Nord; d <= NordOuest; d++ {
		vues = append(vues, ligneDeVue(b, p, d, portee)...)
	}
	return vues
}

// CasesDans énumère le carré autour du centre.
func (b *plateauOuvert) CasesDans(centre Position, rayon int) []Position {
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

// parametresDEssai réduit le plateau sans sortir de ce que Valider accepte.
func parametresDEssai() Parametres {
	p := ParametresDefaut()
	p.Cote = 15
	p.Portee = 7
	return p
}

// TestNouvelleRefuseCeQuiEstInjouable vérifie que les manques sont dits plutôt
// que découverts au premier coup.
func TestNouvelleRefuseCeQuiEstInjouable(t *testing.T) {
	bon := parametresDEssai()

	cas := []struct {
		nom     string
		plateau Plateau
		params  Parametres
		reg     *Registre
	}{
		{"sans plateau", nil, bon, registreDEssai()},
		{"sans registre", ouvert(15), bon, nil},
		{"paramètres refusés", ouvert(15), Parametres{}, registreDEssai()},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if _, err := Nouvelle(c.plateau, 1, c.params, c.reg); err == nil {
				t.Error("accepté alors que la partie serait injouable")
			}
		})
	}
}

// TestNouvellePlaceLeFugitifAuCentre vérifie que le tirage reste dans le noyau
// central : il doit avoir à traverser, sinon les zones ne servent à rien.
func TestNouvellePlaceLeFugitifAuCentre(t *testing.T) {
	params := parametresDEssai()
	milieu := params.Cote / 2

	for graine := int64(0); graine < 50; graine++ {
		p, err := Nouvelle(ouvert(params.Cote), graine, params, registreDEssai())
		if err != nil {
			t.Fatal(err)
		}
		centre := Position{Colonne: milieu, Ligne: milieu}
		if d := DistanceTchebychev(p.Fugitif.Position, centre); d > RayonNoyauCentral {
			t.Fatalf("graine %d : fugitif à %d du centre, rayon %d", graine, d, RayonNoyauCentral)
		}
	}
}

// TestNouvelleEstDeterministe vérifie que la graine décide seule du départ.
func TestNouvelleEstDeterministe(t *testing.T) {
	params := parametresDEssai()

	a, err := Nouvelle(ouvert(params.Cote), 178342119, params, registreDEssai())
	if err != nil {
		t.Fatal(err)
	}
	b, err := Nouvelle(ouvert(params.Cote), 178342119, params, registreDEssai())
	if err != nil {
		t.Fatal(err)
	}
	if a.Fugitif.Position != b.Fugitif.Position {
		t.Errorf("départs %v et %v pour la même graine", a.Fugitif.Position, b.Fugitif.Position)
	}

	autre, err := Nouvelle(ouvert(params.Cote), 1, params, registreDEssai())
	if err != nil {
		t.Fatal(err)
	}
	if autre.Fugitif.Position == a.Fugitif.Position {
		t.Log("deux graines donnent le même départ, ce qui arrive")
	}
}

// TestNouvelleNeScellePasDeZone vérifie que le fugitif choisit, et que rien
// n'est décidé pour lui.
func TestNouvelleNeScellePasDeZone(t *testing.T) {
	params := parametresDEssai()
	p, err := Nouvelle(ouvert(params.Cote), 7, params, registreDEssai())
	if err != nil {
		t.Fatal(err)
	}

	if p.Phase != PhasePlacementFugitif {
		t.Errorf("phase %s, attendu placement_fugitif", p.Phase)
	}
	if _, scellee := p.zoneScellee(); scellee {
		t.Error("une zone est déjà scellée avant que le fugitif ait choisi")
	}
	if p.Fugitif.Resistance != params.Resistance {
		t.Errorf("résistance %d, attendu %d", p.Fugitif.Resistance, params.Resistance)
	}
}

// jouerUnePartie déroule une partie entière en choisissant au hasard parmi les
// coups légaux, et rend le journal.
//
// Le hasard vient d'Alea : deux appels sur la même graine jouent la même
// partie, ce qui rend l'échec reproductible.
func jouerUnePartie(t *testing.T, graine int64) *Partie {
	t.Helper()

	params := parametresDEssai()
	p, err := Nouvelle(ouvert(params.Cote), graine, params, registreDEssai())
	if err != nil {
		t.Fatal(err)
	}

	des := NouvelAlea(graine, "joueur")
	for coups := 0; coups < 20000; coups++ {
		if _, fini := p.Resultat(); fini || p.Phase == PhaseTerminee {
			return p
		}

		acteur := CampFugitif
		if p.Phase == PhasePlacementInspecteurs || p.Phase == PhaseInspecteurs {
			acteur = CampInspecteurs
		}

		legaux := p.CoupsLegaux(acteur)
		if len(legaux) == 0 {
			t.Fatalf("aucun coup légal au tour %d, phase %s", p.Tour, p.Phase)
		}
		if err := p.Appliquer(legaux[des.Entier(len(legaux))]); err != nil {
			t.Fatalf("coup refusé au tour %d : %v", p.Tour, err)
		}
	}

	t.Fatal("la partie ne se termine pas")
	return nil
}

// TestPartieCompleteSeJoue est le critère de livraison de l'étape 1 : une
// partie entière se joue depuis des appels Go, sans interface.
//
// Elle est jouée au hasard sur vingt graines figées. Ce qui est vérifié n'est
// pas qui gagne, mais qu'aucune ne s'enlise, qu'aucun coup légal n'est refusé,
// et que chaque fin porte un motif connu.
func TestPartieCompleteSeJoue(t *testing.T) {
	motifs := map[string]int{}

	for graine := int64(1); graine <= 20; graine++ {
		p := jouerUnePartie(t, graine)

		r, fini := p.Resultat()
		if !fini {
			t.Fatalf("graine %d : la partie s'arrête sans résultat", graine)
		}
		switch r.Motif {
		case MotifExtraction, MotifResistance, MotifBlocage, MotifTempsEcoule:
			motifs[r.Motif]++
		default:
			t.Errorf("graine %d : motif inattendu %q", graine, r.Motif)
		}
		if r.Vainqueur != CampFugitif && r.Vainqueur != CampInspecteurs {
			t.Errorf("graine %d : vainqueur %q", graine, r.Vainqueur)
		}
		if len(p.Journal) == 0 {
			t.Errorf("graine %d : journal vide", graine)
		}
	}

	t.Logf("motifs de fin sur vingt parties : %v", motifs)
}

// TestDeuxPartiesMemeGraineMemeJournal est le filet du déterminisme.
//
// Deux parties menées de la même façon sur la même graine produisent la même
// suite de coups. Sans cela, rejouer un journal ne reconstruit pas la partie, et
// la reprise comme l'entraînement de l'IA tombent ensemble.
func TestDeuxPartiesMemeGraineMemeJournal(t *testing.T) {
	for graine := int64(1); graine <= 5; graine++ {
		a := jouerUnePartie(t, graine)
		b := jouerUnePartie(t, graine)

		if !reflect.DeepEqual(a.Journal, b.Journal) {
			t.Fatalf("graine %d : les journaux divergent après %d et %d coups",
				graine, len(a.Journal), len(b.Journal))
		}
		if a.Fugitif != b.Fugitif || a.Tour != b.Tour {
			t.Errorf("graine %d : états finaux différents", graine)
		}
	}
}

// TestPartieCompleteSeDefait vérifie que la réversibilité tient sur une partie
// entière, et pas seulement sur un coup.
//
// C'est ce dont l'IA dépendra pour explorer : des milliers de coups appliqués
// puis défaits, sans jamais copier l'état.
func TestPartieCompleteSeDefait(t *testing.T) {
	params := parametresDEssai()
	depart, err := Nouvelle(ouvert(params.Cote), 3, params, registreDEssai())
	if err != nil {
		t.Fatal(err)
	}
	avant := *depart

	des := NouvelAlea(3, "joueur")
	joues := 0
	for ; joues < 60; joues++ {
		if _, fini := depart.Resultat(); fini {
			break
		}
		acteur := CampFugitif
		if depart.Phase == PhasePlacementInspecteurs || depart.Phase == PhaseInspecteurs {
			acteur = CampInspecteurs
		}
		legaux := depart.CoupsLegaux(acteur)
		if len(legaux) == 0 {
			break
		}
		if err := depart.Appliquer(legaux[des.Entier(len(legaux))]); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < joues; i++ {
		if err := depart.Annuler(); err != nil {
			t.Fatalf("annulation %d : %v", i, err)
		}
	}

	depart.annulations = nil
	avant.annulations = nil
	if !reflect.DeepEqual(*depart, avant) {
		t.Error("l'état diffère après avoir défait toute la partie")
	}
}
