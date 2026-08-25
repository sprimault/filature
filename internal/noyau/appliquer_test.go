// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import (
	"errors"
	"reflect"
	"testing"
)

// registreDEssai reproduit ce que greffons/base déclare, en plus court : de
// quoi exercer capacités et dépenses sans dépendre du chargeur, qui relève de
// l'étape 8.
func registreDEssai() *Registre {
	return &Registre{
		Capacites: map[string]Capacite{
			"barreur": {
				Nom: "Barreur", Camp: CampInspecteurs, Usages: 1,
				Declenchement: SurPhaseInspecteurs,
				Effets:        []Effet{{Type: EffetBloquerCase, Cible: CibleCase, Duree: 3}},
			},
			"guetteur": {
				Nom: "Guetteur", Camp: CampInspecteurs, Usages: 1,
				Declenchement: SurPhaseInspecteurs,
				Effets:        []Effet{{Type: EffetModifierPortee, Cible: CiblePionCourant, Valeur: 8, Duree: 1}},
			},
		},
		Depenses: map[Depense]Capacite{
			DepenseSilence: {
				Nom: "Silence", Camp: CampFugitif, Cout: 3,
				Declenchement: SurPhaseFugitif,
				Effets:        []Effet{{Type: EffetAnnulerRevelation, Cible: CibleFugitif}},
			},
			DepenseMeurtre: {
				Nom: "Meurtre", Camp: CampFugitif, Cout: 3, Usages: 2,
				Declenchement: SurPhaseFugitif,
				Effets: []Effet{
					{Type: EffetRevelerPosition, Cible: CibleFugitif},
					{Type: EffetMarquerScene, Cible: CibleFugitif},
				},
			},
			DepenseChangerZone: {
				Nom: "Changement de zone", Camp: CampFugitif, Cout: 2,
				Declenchement: SurPhaseFugitif,
				Effets:        []Effet{{Type: EffetScellerZone, Cible: CibleZone}},
			},
		},
	}
}

// partieJouable monte une partie en phase fugitif, registre compris.
func partieJouable() *Partie {
	b := trame(".....", ".....", ".....", ".....", ".....")
	b.zones = []Zone{{Numero: 0}, {Numero: 1}, {Numero: 2}}
	p := partieSur(b, Position{Colonne: 2, Ligne: 2},
		Position{Colonne: 0, Ligne: 0}, Position{Colonne: 4, Ligne: 0})
	p.Extensions = registreDEssai()
	return p
}

// premierCoup rend le premier coup légal du type demandé.
func premierCoup(t *testing.T, p *Partie, a Acteur, typ TypeCoup) Coup {
	t.Helper()
	for _, c := range p.CoupsLegaux(a) {
		if c.Type == typ {
			return c
		}
	}
	t.Fatalf("aucun coup de type %s", typ)
	return Coup{}
}

// TestCoupIllegalRefuse vérifie qu'un coup absent de la liste est rejeté, et
// qu'il ne laisse aucune trace.
//
// C'est ce que le serveur oppose à un bot fautif : le jeu ne corrige ni
// n'interprète, il refuse.
func TestCoupIllegalRefuse(t *testing.T) {
	p := partieJouable()
	avant := *p

	faux := Coup{Tour: p.Tour, Acteur: CampFugitif, Type: CoupDeplacer,
		Depart: p.Fugitif.Position, Arrivee: Position{Colonne: 9, Ligne: 9}}

	if err := p.Appliquer(faux); !errors.Is(err, ErrCoupIllegal) {
		t.Fatalf("erreur %v, attendu ErrCoupIllegal", err)
	}
	if len(p.Journal) != 0 {
		t.Error("un coup refusé est entré au journal")
	}
	if p.Fugitif != avant.Fugitif {
		t.Error("un coup refusé a modifié l'état")
	}
}

// TestCoupPresqueLegalRefuse vérifie que la comparaison est stricte : un coup
// dont un seul champ diffère est un autre coup.
func TestCoupPresqueLegalRefuse(t *testing.T) {
	p := partieJouable()
	c := premierCoup(t, p, CampFugitif, CoupDeplacer)
	c.Tour++

	if err := p.Appliquer(c); !errors.Is(err, ErrCoupIllegal) {
		t.Fatalf("erreur %v, attendu ErrCoupIllegal", err)
	}
}

// TestDeplacementCompteEtSeDefait vérifie qu'un déplacement avance le pion,
// consomme sa mobilité, et se défait entièrement.
func TestDeplacementCompteEtSeDefait(t *testing.T) {
	p := partieJouable()
	depart := p.Fugitif.Position

	c := premierCoup(t, p, CampFugitif, CoupDeplacer)
	if err := p.Appliquer(c); err != nil {
		t.Fatal(err)
	}
	if p.Fugitif.Position != c.Arrivee {
		t.Errorf("fugitif en %v, attendu %v", p.Fugitif.Position, c.Arrivee)
	}
	if p.Fugitif.DeplacementsFaits != 1 {
		t.Errorf("%d déplacements comptés, attendu 1", p.Fugitif.DeplacementsFaits)
	}

	if err := p.Annuler(); err != nil {
		t.Fatal(err)
	}
	if p.Fugitif.Position != depart || p.Fugitif.DeplacementsFaits != 0 {
		t.Error("l'annulation n'a pas rendu la position ou le compteur")
	}
	if len(p.Journal) != 0 {
		t.Error("le journal garde un coup annulé")
	}
}

// TestMobiliteEpuiseeApresUnPas vérifie qu'un fugitif sans bonus ne joue qu'un
// déplacement par tour.
func TestMobiliteEpuiseeApresUnPas(t *testing.T) {
	p := partieJouable()
	if err := p.Appliquer(premierCoup(t, p, CampFugitif, CoupDeplacer)); err != nil {
		t.Fatal(err)
	}
	for _, c := range p.CoupsLegaux(CampFugitif) {
		if c.Type == CoupDeplacer {
			t.Fatal("un second déplacement est proposé sans bonus de mobilité")
		}
	}
}

// TestDepenseCouteEtSeDefait vérifie le prélèvement, l'effet, et le retour en
// arrière complet.
func TestDepenseCouteEtSeDefait(t *testing.T) {
	p := partieJouable()

	var silence Coup
	for _, c := range p.CoupsLegaux(CampFugitif) {
		if c.Type == CoupDepense && c.Depense == DepenseSilence {
			silence = c
		}
	}
	if silence.Type == "" {
		t.Fatal("le silence n'est pas proposé")
	}

	if err := p.Appliquer(silence); err != nil {
		t.Fatal(err)
	}
	if p.Fugitif.Resistance != 7 {
		t.Errorf("résistance %d, attendu 7", p.Fugitif.Resistance)
	}
	if !p.Fugitif.SilenceAchete {
		t.Error("le silence n'a pas pris effet")
	}

	if err := p.Annuler(); err != nil {
		t.Fatal(err)
	}
	if p.Fugitif.Resistance != 10 || p.Fugitif.SilenceAchete {
		t.Error("l'annulation n'a pas rendu la résistance ou défait l'effet")
	}
}

// TestUsagesPlafonnes vérifie que le compteur d'emplois est générique : le
// noyau n'a pas à savoir que la dépense plafonnée s'appelle meurtre.
func TestUsagesPlafonnes(t *testing.T) {
	p := partieJouable()
	p.Fugitif.Resistance = 20

	for i := 0; i < 2; i++ {
		var meurtre Coup
		for _, c := range p.CoupsLegaux(CampFugitif) {
			if c.Type == CoupDepense && c.Depense == DepenseMeurtre {
				meurtre = c
			}
		}
		if meurtre.Type == "" {
			t.Fatalf("le meurtre n'est plus proposé au %dᵉ emploi", i+1)
		}
		if err := p.Appliquer(meurtre); err != nil {
			t.Fatal(err)
		}
	}

	if p.UsagesDepense[DepenseMeurtre] != 2 {
		t.Errorf("%d emplois comptés, attendu 2", p.UsagesDepense[DepenseMeurtre])
	}
	for _, c := range p.CoupsLegaux(CampFugitif) {
		if c.Type == CoupDepense && c.Depense == DepenseMeurtre {
			t.Error("un troisième meurtre est proposé")
		}
	}
	if len(p.Scenes) != 2 {
		t.Errorf("%d scènes, attendu 2", len(p.Scenes))
	}
}

// TestCapaciteUneSeuleParTour vérifie les deux limites de la règle : une par
// pion et par partie, et une seule par tour tous pions confondus.
func TestCapaciteUneSeuleParTour(t *testing.T) {
	p := partieJouable()
	p.Phase = PhaseInspecteurs
	p.Inspecteurs[0].Capacite = "guetteur"
	p.Inspecteurs[1].Capacite = "barreur"

	c := premierCoup(t, p, CampInspecteurs, CoupCapacite)
	if err := p.Appliquer(c); err != nil {
		t.Fatal(err)
	}
	if !p.CapaciteJouee {
		t.Error("la capacité du tour n'est pas marquée")
	}
	for _, l := range p.CoupsLegaux(CampInspecteurs) {
		if l.Type == CoupCapacite {
			t.Error("une seconde capacité est proposée dans le même tour")
		}
	}

	if err := p.Annuler(); err != nil {
		t.Fatal(err)
	}
	if p.CapaciteJouee || p.Inspecteurs[c.Pion].CapaciteUtilisee {
		t.Error("l'annulation n'a pas rendu les marques de capacité")
	}
}

// TestFinDeTourRouvreLesQuotas vérifie que la résolution remet les compteurs à
// zéro, et que l'annulation les rétablit.
func TestFinDeTourRouvreLesQuotas(t *testing.T) {
	p := partieJouable()
	p.Phase = PhaseInspecteurs
	p.Inspecteurs[0].DeplacementsFaits = 1
	p.CapaciteJouee = true
	tour := p.Tour

	// Les inspecteurs rendent la main, puis le fugitif : le tour se résout.
	if err := p.Appliquer(premierCoup(t, p, CampInspecteurs, CoupFinDePhase)); err != nil {
		t.Fatal(err)
	}
	if p.Phase != PhaseFugitif {
		t.Fatalf("phase %s, attendu fugitif", p.Phase)
	}
	if p.Tour != tour {
		t.Error("le tour a avancé entre les deux phases")
	}

	if err := p.Appliquer(premierCoup(t, p, CampFugitif, CoupFinDePhase)); err != nil {
		t.Fatal(err)
	}
	if p.Tour != tour+1 {
		t.Errorf("tour %d, attendu %d", p.Tour, tour+1)
	}
	if p.Inspecteurs[0].DeplacementsFaits != 0 || p.CapaciteJouee {
		t.Error("les quotas n'ont pas été rouverts")
	}

	if err := p.Annuler(); err != nil {
		t.Fatal(err)
	}
	if p.Tour != tour || p.Inspecteurs[0].DeplacementsFaits != 1 || !p.CapaciteJouee {
		t.Error("l'annulation de la fin de tour n'a pas tout rendu")
	}
}

// TestAnnulerSansCoup vérifie qu'annuler sur une partie vierge le dit.
func TestAnnulerSansCoup(t *testing.T) {
	if err := partieJouable().Annuler(); !errors.Is(err, ErrRienAAnnuler) {
		t.Fatalf("erreur %v, attendu ErrRienAAnnuler", err)
	}
}

// TestPartieEntiereSeDefait est l'invariant de réversibilité au niveau du coup.
//
// Une suite de coups quelconques, puis autant d'annulations, doit rendre un
// état identique à l'original. C'est ce dont l'IA dépend pour explorer sans
// copier l'état, et ce qui casse en silence dès qu'un coup oublie une
// annulation.
func TestPartieEntiereSeDefait(t *testing.T) {
	p := partieJouable()
	avant := partieJouable()

	// Une partie qui traverse les deux phases, un déplacement, une dépense et
	// une résolution de tour.
	joues := 0
	for _, choix := range []struct {
		acteur Acteur
		typ    TypeCoup
	}{
		{CampFugitif, CoupDeplacer},
		{CampFugitif, CoupDepense},
		{CampFugitif, CoupFinDePhase},
		{CampInspecteurs, CoupDeplacer},
		{CampInspecteurs, CoupFinDePhase},
		{CampFugitif, CoupDeplacer},
		{CampFugitif, CoupFinDePhase},
	} {
		c := premierCoup(t, p, choix.acteur, choix.typ)
		if err := p.Appliquer(c); err != nil {
			t.Fatalf("coup %d refusé : %v", joues, err)
		}
		joues++
	}

	for i := 0; i < joues; i++ {
		if err := p.Annuler(); err != nil {
			t.Fatalf("annulation %d refusée : %v", i, err)
		}
	}

	// Les fermetures ne se comparent pas : la pile doit être vide des deux
	// côtés, et le reste identique.
	if len(p.annulations) != 0 {
		t.Errorf("%d annulations restantes", len(p.annulations))
	}
	p.annulations, avant.annulations = nil, nil
	if !reflect.DeepEqual(p, avant) {
		t.Errorf("l'état diffère après annulation complète\n  obtenu : %+v\n  attendu: %+v", p, avant)
	}
}
