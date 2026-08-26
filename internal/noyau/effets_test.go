// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import (
	"reflect"
	"testing"
)

// plateauNu est un terrain sans bâtiment, de quoi exercer les effets sans
// dépendre de la génération, qui relève de l'étape 3.
type plateauNu struct{ cote int }

// EstRue accepte toute case dans les bornes.
func (b plateauNu) EstRue(p Position) bool {
	return p.Colonne >= 0 && p.Ligne >= 0 && p.Colonne < b.cote && p.Ligne < b.cote
}

// Zones renvoie une unique zone, suffisante pour les bascules.
func (b plateauNu) Zones() []Zone {
	return []Zone{{Numero: 0, Cases: []Position{{Colonne: 1, Ligne: 1}}}}
}

// Graine est figée : aucun test de ce fichier ne tire au sort.
func (b plateauNu) Graine() int64 { return 1 }

// Vision reste vide : ce sont les effets qu'on applique ici, pas la vue.
func (b plateauNu) Vision(p Position, portee int) []Position { return nil }

// CasesDans énumère les cases du carré, dans l'ordre du plateau borné.
func (b plateauNu) CasesDans(centre Position, rayon int) []Position {
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

// partieDEssai monte une partie au tour 5, avec trois inspecteurs et un
// fugitif, de quoi appliquer n'importe quelle primitive.
func partieDEssai() *Partie {
	return &Partie{
		Graine:     1,
		Parametres: ParametresDefaut(),
		Plateau:    plateauNu{cote: 9},
		Tour:       5,
		Phase:      PhaseInspecteurs,
		Fugitif: Fugitif{
			Position:   Position{Colonne: 4, Ligne: 4},
			Resistance: 10,
		},
		Inspecteurs: []Inspecteur{
			{Position: Position{Colonne: 0, Ligne: 0}, Capacite: "guetteur"},
			{Position: Position{Colonne: 1, Ligne: 0}, Capacite: "coureur"},
			{Position: Position{Colonne: 2, Ligne: 0}, Capacite: "chef"},
		},
		Traces: map[Position]Trace{
			{Colonne: 3, Ligne: 4}: {Tour: 4, Direction: Est},
			{Colonne: 2, Ligne: 4}: {Tour: 1, Direction: Est},
		},
		Barrages:   map[Position]int{},
		Ouvertures: map[Position]int{},
	}
}

// casEffet décrit une primitive à appliquer et ce qu'elle doit produire.
type casEffet struct {
	nom string

	// prepare pose l'état que la primitive exige pour agir. Une primitive qui
	// n'a rien à défaire sort par une branche vide et ne prouve rien : c'est
	// exactement ce qui a laissé ouvrir_zone sans couverture.
	prepare func(*Partie)

	effet   Effet
	ctx     Contexte
	verifie func(*testing.T, *Partie)
}

// partiePour monte une partie d'essai dans l'état qu'un cas réclame.
func (c casEffet) partiePour() *Partie {
	p := partieDEssai()
	if c.prepare != nil {
		c.prepare(p)
	}
	return p
}

// partieAvecPreparations monte une partie qui satisfait tous les cas à la fois,
// pour ceux qui les enchaînent sur un même état.
func partieAvecPreparations() *Partie {
	p := partieDEssai()
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
	inspecteur := Contexte{Acteur: CampInspecteurs, Pion: 1}
	fugitif := Contexte{Acteur: CampFugitif}
	uneCase := Position{Colonne: 6, Ligne: 6}

	return []casEffet{
		{
			nom:   "deplacer",
			effet: Effet{Type: EffetDeplacer, Cible: CiblePionCourant},
			ctx:   Contexte{Acteur: CampInspecteurs, Pion: 1, Case: uneCase},
			verifie: func(t *testing.T, p *Partie) {
				if p.Inspecteurs[1].Position != uneCase {
					t.Errorf("pion en %v, attendu %v", p.Inspecteurs[1].Position, uneCase)
				}
			},
		},
		{
			nom:   "teleporter le fugitif",
			effet: Effet{Type: EffetTeleporter, Cible: CibleFugitif},
			ctx:   Contexte{Acteur: CampFugitif, Case: uneCase},
			verifie: func(t *testing.T, p *Partie) {
				if p.Fugitif.Position != uneCase {
					t.Errorf("fugitif en %v, attendu %v", p.Fugitif.Position, uneCase)
				}
			},
		},
		{
			nom:   "modifier_portee",
			effet: Effet{Type: EffetModifierPortee, Cible: CiblePionCourant, Valeur: 8, Duree: 1},
			ctx:   inspecteur,
			verifie: func(t *testing.T, p *Partie) {
				if got := p.PorteeDe(1); got != p.Parametres.Portee+8 {
					t.Errorf("portée du pion visé %d, attendu %d", got, p.Parametres.Portee+8)
				}
				if got := p.PorteeDe(0); got != p.Parametres.Portee {
					t.Errorf("portée d'un autre pion %d, attendu %d", got, p.Parametres.Portee)
				}
			},
		},
		{
			nom:   "modifier_mobilite",
			effet: Effet{Type: EffetModifierMobilite, Cible: CibleFugitif, Valeur: 1, Duree: 1},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Partie) {
				if got := p.MobiliteDe(CampFugitif, 0); got != 2 {
					t.Errorf("mobilité du fugitif %d, attendu 2", got)
				}
			},
		},
		{
			nom:   "bloquer_case",
			effet: Effet{Type: EffetBloquerCase, Cible: CibleCase, Duree: 3},
			ctx:   Contexte{Acteur: CampInspecteurs, Pion: 1, Case: uneCase},
			verifie: func(t *testing.T, p *Partie) {
				if p.EstPraticable(uneCase) {
					t.Error("la case barrée reste praticable")
				}
			},
		},
		{
			nom:   "ouvrir_case",
			effet: Effet{Type: EffetOuvrirCase, Cible: CibleCase, Duree: 3},
			ctx:   Contexte{Acteur: CampInspecteurs, Pion: 1, Case: Position{Colonne: 99, Ligne: 99}},
			verifie: func(t *testing.T, p *Partie) {
				if !p.EstPraticable(Position{Colonne: 99, Ligne: 99}) {
					t.Error("la case percée reste impraticable")
				}
			},
		},
		{
			nom:   "reveler_traces",
			effet: Effet{Type: EffetRevelerTraces, Cible: CiblePionCourant, Rayon: 2},
			ctx:   inspecteur,
			verifie: func(t *testing.T, p *Partie) {
				if got := p.RayonTracesDe(1); got != 2 {
					t.Errorf("rayon du Traqueur %d, attendu 2", got)
				}
				if got := p.RayonTracesDe(0); got != 1 {
					t.Errorf("rayon d'un autre pion %d, attendu 1", got)
				}
			},
		},
		{
			nom:   "reveler_position",
			effet: Effet{Type: EffetRevelerPosition, Cible: CibleFugitif},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Partie) {
				if !p.Fugitif.Visible {
					t.Error("le fugitif reste caché après révélation")
				}
			},
		},
		{
			nom:   "marquer_scene",
			effet: Effet{Type: EffetMarquerScene, Cible: CibleFugitif},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Partie) {
				attendue := Scene{Position: Position{Colonne: 4, Ligne: 4}, Tour: 5}
				if !reflect.DeepEqual(p.Scenes, []Scene{attendue}) {
					t.Errorf("scènes %v, attendu [%v]", p.Scenes, attendue)
				}
			},
		},
		{
			nom:   "annuler_revelation",
			effet: Effet{Type: EffetAnnulerRevelation, Cible: CibleFugitif},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Partie) {
				if !p.Fugitif.SilenceAchete {
					t.Error("le silence n'est pas enregistré")
				}
			},
		},
		{
			nom:   "partager_vue",
			effet: Effet{Type: EffetPartagerVue, Cible: CibleAutrePion, Duree: 2},
			ctx:   Contexte{Acteur: CampInspecteurs, Pion: 2, AutrePion: 0},
			verifie: func(t *testing.T, p *Partie) {
				if got := p.VuePartageeDe(2); !reflect.DeepEqual(got, []int{0}) {
					t.Errorf("vue partagée %v, attendu [0]", got)
				}
			},
		},
		{
			nom:   "couter_resistance",
			effet: Effet{Type: EffetCouterResistance, Cible: CibleFugitif, Valeur: 3},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Partie) {
				if p.Fugitif.Resistance != 7 {
					t.Errorf("résistance %d, attendu 7", p.Fugitif.Resistance)
				}
			},
		},
		{
			nom:   "rendre_resistance",
			effet: Effet{Type: EffetRendreResistance, Cible: CibleFugitif, Valeur: 2},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Partie) {
				if p.Fugitif.Resistance != 12 {
					t.Errorf("résistance %d, attendu 12", p.Fugitif.Resistance)
				}
			},
		},
		{
			nom:   "effacer_traces",
			effet: Effet{Type: EffetEffacerTraces, Cible: CibleFugitif, Duree: 3},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Partie) {
				if _, reste := p.Traces[Position{Colonne: 3, Ligne: 4}]; reste {
					t.Error("la trace du tour 4 devait être effacée")
				}
				if _, reste := p.Traces[Position{Colonne: 2, Ligne: 4}]; !reste {
					t.Error("la trace du tour 1 est trop vieille pour être effacée")
				}
			},
		},
		{
			nom:   "fermer_zone",
			effet: Effet{Type: EffetFermerZone, Cible: CibleZone},
			ctx:   Contexte{Acteur: CampInspecteurs, Pion: 1, Zone: 3},
			verifie: func(t *testing.T, p *Partie) {
				if !reflect.DeepEqual(p.ZonesFermees, []int{3}) {
					t.Errorf("zones fermées %v, attendu [3]", p.ZonesFermees)
				}
			},
		},
		{
			nom: "ouvrir_zone",
			// Trois zones fermées et celle qu'on rouvre au milieu : c'est son
			// rang qui compte. L'annulation doit la remettre entre 5 et 2, pas
			// au bout — cet ordre part dans le journal, et un rejeu qui le
			// retrouve différent n'est plus le même octet pour octet.
			prepare: func(p *Partie) { p.ZonesFermees = []int{5, 1, 2} },
			effet:   Effet{Type: EffetOuvrirZone, Cible: CibleZone},
			ctx:     Contexte{Acteur: CampInspecteurs, Pion: 1, Zone: 1},
			verifie: func(t *testing.T, p *Partie) {
				if !reflect.DeepEqual(p.ZonesFermees, []int{5, 2}) {
					t.Errorf("zones fermées %v, attendu [5 2]", p.ZonesFermees)
				}
			},
		},
		{
			nom:   "sceller_zone",
			effet: Effet{Type: EffetScellerZone, Cible: CibleZone},
			ctx:   Contexte{Acteur: CampFugitif, Zone: 4},
			verifie: func(t *testing.T, p *Partie) {
				if p.Fugitif.ZoneScellee != 4 {
					t.Errorf("zone scellée %d, attendu 4", p.Fugitif.ZoneScellee)
				}
			},
		},
		{
			nom:   "differer",
			effet: Effet{Type: EffetDifferer, Duree: 2, Annonce: true, Puis: []Effet{{Type: EffetFermerZone, Cible: CibleZone}}},
			ctx:   Contexte{Acteur: CampInspecteurs, Pion: 1, Zone: 2},
			verifie: func(t *testing.T, p *Partie) {
				if len(p.EffetsEnAttente) != 1 {
					t.Fatalf("%d effets en attente, attendu 1", len(p.EffetsEnAttente))
				}
				if got := p.EffetsEnAttente[0].Tour; got != 7 {
					t.Errorf("échéance au tour %d, attendu 7", got)
				}
				if !p.EffetsEnAttente[0].Annonce {
					t.Error("l'annonce n'est pas conservée")
				}
			},
		},
		{
			nom:   "fin_partie",
			effet: Effet{Type: EffetFinPartie, Cible: CibleFugitif},
			ctx:   fugitif,
			verifie: func(t *testing.T, p *Partie) {
				r, fini := p.Resultat()
				if !fini {
					t.Fatal("la partie devait être terminée")
				}
				if r.Vainqueur != CampFugitif || r.Motif != MotifGreffon {
					t.Errorf("résultat %+v, attendu fugitif/greffon", r)
				}
			},
		},
	}
}

// TestAppliquerEffets vérifie que chaque primitive produit ce qu'elle annonce.
func TestAppliquerEffets(t *testing.T) {
	for _, cas := range tousLesCas() {
		t.Run(cas.nom, func(t *testing.T) {
			p := cas.partiePour()
			annuler, err := p.Appliquer1Effet(cas.effet, cas.ctx)
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

// TestAnnulerRendLEtatIdentique est l'invariant de réversibilité.
//
// Ce n'est pas un confort d'interface : l'IA explore des milliers de positions
// en appliquant puis défaisant, sans copier l'état. Un effet qui laisse la
// moindre trace après annulation fait diverger l'exploration, puis le rejeu du
// journal.
func TestAnnulerRendLEtatIdentique(t *testing.T) {
	for _, cas := range tousLesCas() {
		t.Run(cas.nom, func(t *testing.T) {
			p := cas.partiePour()
			avant := cas.partiePour()

			annuler, err := p.Appliquer1Effet(cas.effet, cas.ctx)
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

// TestAnnulerEnChaine vérifie que plusieurs effets se défont dans l'ordre
// inverse, ce dont les annulations qui tronquent une tranche dépendent.
func TestAnnulerEnChaine(t *testing.T) {
	// Les préparations sont posées toutes ensemble avant la chaîne : appliquée
	// entre deux effets, une préparation modifierait l'état sans annulation
	// correspondante, et la comparaison finale échouerait sans qu'aucun effet
	// soit en cause.
	p := partieAvecPreparations()
	avant := partieAvecPreparations()

	var annulations []func()
	for _, cas := range tousLesCas() {
		annuler, err := p.Appliquer1Effet(cas.effet, cas.ctx)
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

// TestEffetInconnuEchoue vérifie qu'un type hors vocabulaire est refusé plutôt
// qu'ignoré : un greffon entré sans validation doit s'entendre dire non.
func TestEffetInconnuEchoue(t *testing.T) {
	p := partieDEssai()
	if _, err := p.Appliquer1Effet(Effet{Type: "voler"}, Contexte{Acteur: CampFugitif}); err == nil {
		t.Fatal("un effet inconnu a été accepté")
	}
}

// TestPionHorsBornesEchoue vérifie qu'un index de pion invalide est refusé.
// Sans ce contrôle, un contexte mal formé ferait paniquer le noyau.
func TestPionHorsBornesEchoue(t *testing.T) {
	p := partieDEssai()
	ctx := Contexte{Acteur: CampInspecteurs, Pion: 9}
	if _, err := p.Appliquer1Effet(Effet{Type: EffetDeplacer}, ctx); err == nil {
		t.Fatal("un pion hors bornes a été accepté")
	}
}

// TestEffetExpire vérifie qu'un effet à durée cesse de compter passé son
// échéance, et qu'un effet sans durée ne cesse jamais.
func TestEffetExpire(t *testing.T) {
	p := partieDEssai()
	ctx := Contexte{Acteur: CampInspecteurs, Pion: 0}

	if _, err := p.Appliquer1Effet(Effet{Type: EffetModifierPortee, Cible: CiblePionCourant, Valeur: 8, Duree: 1}, ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Appliquer1Effet(Effet{Type: EffetRevelerTraces, Cible: CiblePionCourant, Rayon: 2}, ctx); err != nil {
		t.Fatal(err)
	}

	if got := p.PorteeDe(0); got != p.Parametres.Portee+8 {
		t.Errorf("au tour du déclenchement, portée %d", got)
	}

	p.Tour++
	if got := p.PorteeDe(0); got != p.Parametres.Portee {
		t.Errorf("au tour suivant, portée %d, attendu %d", got, p.Parametres.Portee)
	}
	if got := p.RayonTracesDe(0); got != 2 {
		t.Errorf("un effet sans durée a expiré : rayon %d, attendu 2", got)
	}
}

// TestMobiliteNegativeImmobilise vérifie que le vocabulaire tient sa promesse :
// une valeur négative est légale, et modifier_mobilite à -1 cloue le pion.
func TestMobiliteNegativeImmobilise(t *testing.T) {
	p := partieDEssai()
	ctx := Contexte{Acteur: CampInspecteurs, Pion: 0}

	if _, err := p.Appliquer1Effet(Effet{Type: EffetModifierMobilite, Cible: CiblePionCourant, Valeur: -1, Duree: 1}, ctx); err != nil {
		t.Fatal(err)
	}
	if got := p.MobiliteDe(CampInspecteurs, 0); got != 0 {
		t.Errorf("mobilité %d, attendu 0", got)
	}
}

// TestResistancePlancher vérifie que la résistance ne passe pas sous zéro, et
// que l'annulation rend la valeur d'origine malgré ce plafonnement.
func TestResistancePlancher(t *testing.T) {
	p := partieDEssai()
	annuler, err := p.Appliquer1Effet(
		Effet{Type: EffetCouterResistance, Cible: CibleFugitif, Valeur: 99},
		Contexte{Acteur: CampFugitif})
	if err != nil {
		t.Fatal(err)
	}
	if p.Fugitif.Resistance != 0 {
		t.Errorf("résistance %d, attendu 0", p.Fugitif.Resistance)
	}
	annuler()
	if p.Fugitif.Resistance != 10 {
		t.Errorf("après annulation, résistance %d, attendu 10", p.Fugitif.Resistance)
	}
}

// TestBarrageLEmporteSurOuverture fixe l'ordre des couches de terrain. Sans
// priorité déclarée, le résultat dépendrait de l'ordre d'application et le
// rejeu du journal cesserait d'être reproductible.
func TestBarrageLEmporteSurOuverture(t *testing.T) {
	p := partieDEssai()
	pos := Position{Colonne: 3, Ligne: 3}
	ctx := Contexte{Acteur: CampInspecteurs, Pion: 0, Case: pos}

	if _, err := p.Appliquer1Effet(Effet{Type: EffetOuvrirCase, Cible: CibleCase, Duree: 3}, ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Appliquer1Effet(Effet{Type: EffetBloquerCase, Cible: CibleCase, Duree: 3}, ctx); err != nil {
		t.Fatal(err)
	}
	if p.EstPraticable(pos) {
		t.Error("le barrage doit l'emporter sur le percement")
	}
}
