// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import (
	"strings"
	"testing"
)

// plateauDessine construit un plateau depuis un croquis : un point par rue,
// tout le reste en bâtiment.
//
// Le symétrique de dessinerEnTexte, et c'est ce qui rend les cas de vision
// relisibles — un angle en équerre se voit dans le croquis, jamais dans une
// liste de coordonnées.
func plateauDessine(t *testing.T, croquis string) *PlateauBorne {
	t.Helper()

	var lignes []string
	for _, l := range strings.Split(croquis, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lignes = append(lignes, l)
		}
	}

	cote := len(lignes)
	b := &PlateauBorne{cote: cote, rues: make([]bool, cote*cote)}
	for ligne, texte := range lignes {
		if len(texte) != cote {
			t.Fatalf("ligne %d : %d colonnes pour %d lignes, le croquis n'est pas carré",
				ligne, len(texte), cote)
		}
		for colonne, c := range texte {
			if c == '.' {
				b.ouvrir(Position{Colonne: colonne, Ligne: ligne})
			}
		}
	}

	b.vues = precalculerVision(b, b.cote)
	return b
}

// TestVisionEtEstVisibleSAccordent est le test que ROADMAP.md §4 réclame :
// deux chemins indépendants pour la même règle, comparés sur les mêmes
// positions.
//
// EstVisible déroule la ligne à la demande, Vision lit la table précalculée et
// la tronque à la portée. Ils ne partagent que ligneDeVue ; tout le reste — le
// filtrage par distance d'un côté, le test d'alignement de l'autre — est écrit
// deux fois. C'est exactement là qu'un désaccord se glisserait, et c'est ce
// désaccord qui a coûté cher au prototype antérieur.
func TestVisionEtEstVisibleSAccordent(t *testing.T) {
	params := parametresPour(21)
	b, _, err := Generer(1, params)
	if err != nil {
		t.Fatal(err)
	}

	for ligne := 0; ligne < b.cote; ligne++ {
		for colonne := 0; colonne < b.cote; colonne++ {
			depuis := Position{Colonne: colonne, Ligne: ligne}
			if !b.EstRue(depuis) {
				continue
			}

			table := map[Position]bool{}
			for _, c := range b.Vision(depuis, params.Portee) {
				table[c] = true
			}

			for l := 0; l < b.cote; l++ {
				for c := 0; c < b.cote; c++ {
					cible := Position{Colonne: c, Ligne: l}
					vu := EstVisible(b, depuis, cible, params.Portee, nil)
					if vu != table[cible] {
						t.Fatalf("%v vers %v : EstVisible dit %v, la table dit %v",
							depuis, cible, vu, table[cible])
					}
				}
			}
		}
	}
}

// TestPorteeEstEnTchebychev est le piège où un prototype antérieur s'est fait
// prendre.
//
// À portée 8, huit pas en diagonale sont vus. En Manhattan cette case est à
// seize, donc hors de portée : mesurer dans la mauvaise unité ne se voit pas en
// jouant, seulement en perdant sans comprendre pourquoi.
func TestPorteeEstEnTchebychev(t *testing.T) {
	b := ouvert(21)
	centre := Position{Colonne: 10, Ligne: 10}

	cas := []struct {
		nom    string
		cible  Position
		portee int
		vue    bool
	}{
		{"huit en diagonale, portée 8", Position{Colonne: 18, Ligne: 18}, 8, true},
		{"neuf en diagonale, portée 8", Position{Colonne: 19, Ligne: 19}, 8, false},
		{"huit tout droit, portée 8", Position{Colonne: 18, Ligne: 10}, 8, true},
		{"neuf tout droit, portée 8", Position{Colonne: 19, Ligne: 10}, 8, false},
		{"quatre en diagonale, portée 4", Position{Colonne: 14, Ligne: 14}, 4, true},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if vu := EstVisible(b, centre, c.cible, c.portee, nil); vu != c.vue {
				t.Errorf("%v depuis %v à portée %d : %v, attendu %v",
					c.cible, centre, c.portee, vu, c.vue)
			}
		})
	}
}

// TestBatimentCoupeLaLigne vérifie que la vue s'arrête au premier bâtiment et
// ne reprend pas derrière.
func TestBatimentCoupeLaLigne(t *testing.T) {
	b := plateauDessine(t, `
		.....
		.....
		..#..
		.....
		.....`)

	depuis := Position{Colonne: 2, Ligne: 0}
	if EstVisible(b, depuis, Position{Colonne: 2, Ligne: 1}, 5, nil) != true {
		t.Error("la case devant le bâtiment devrait être vue")
	}
	if EstVisible(b, depuis, Position{Colonne: 2, Ligne: 2}, 5, nil) != false {
		t.Error("le bâtiment lui-même n'est pas une case visible")
	}
	if EstVisible(b, depuis, Position{Colonne: 2, Ligne: 4}, 5, nil) != false {
		t.Error("la vue reprend derrière le bâtiment")
	}
}

// TestAngleFermeArreteLeRegard applique aux lignes de vue la règle d'angle des
// déplacements.
//
// Deux bâtiments en équerre ferment le passage : sans cette règle, un regard se
// faufile entre leurs coins et le bâti ne bloque plus rien en diagonale.
func TestAngleFermeArreteLeRegard(t *testing.T) {
	// Les deux bâtiments encadrent la diagonale sans être dessus : c'est ce qui
	// distingue un angle fermé d'un mur, et le croquis doit le montrer.
	ferme := plateauDessine(t, `
		.#.
		..#
		...`)
	ouvertDUnCote := plateauDessine(t, `
		...
		..#
		...`)

	depuis := Position{Colonne: 2, Ligne: 0}
	cible := Position{Colonne: 0, Ligne: 2}

	if EstVisible(ferme, depuis, cible, 5, nil) {
		t.Error("le regard passe entre deux bâtiments en équerre")
	}
	if !EstVisible(ouvertDUnCote, depuis, cible, 5, nil) {
		t.Error("une seule case libre suffit à contourner, la vue devrait passer")
	}
}

// TestInspecteurCoupeLaVue vérifie la règle de docs/regles.md §13 : cinq
// inspecteurs en file indienne ne voient qu'avec le premier.
func TestInspecteurCoupeLaVue(t *testing.T) {
	b := ouvert(21)
	depuis := Position{Colonne: 0, Ligne: 0}
	cible := Position{Colonne: 5, Ligne: 0}

	if !EstVisible(b, depuis, cible, 8, nil) {
		t.Fatal("sans obstacle, la case devrait être vue")
	}

	devant := map[Position]bool{{Colonne: 3, Ligne: 0}: true}
	if EstVisible(b, depuis, cible, 8, devant) {
		t.Error("un pion sur la ligne ne la coupe pas")
	}

	derriere := map[Position]bool{{Colonne: 7, Ligne: 0}: true}
	if !EstVisible(b, depuis, cible, 8, derriere) {
		t.Error("un pion au-delà de la cible coupe la vue")
	}
}

// TestCibleOccupeeResteVue distingue ce qui se tient devant de ce qu'on
// regarde.
//
// Le cas se produit dès qu'un inspecteur en observe un autre, et l'oublier
// rendrait un pion invisible à ses collègues au motif qu'il est un pion.
func TestCibleOccupeeResteVue(t *testing.T) {
	b := ouvert(21)
	depuis := Position{Colonne: 0, Ligne: 0}
	cible := Position{Colonne: 4, Ligne: 0}

	if !EstVisible(b, depuis, cible, 8, map[Position]bool{cible: true}) {
		t.Error("une case occupée cesse d'être vue quand elle est la cible")
	}
}

// TestCaseNonAligneeInvisible vérifie qu'un regard est rectiligne.
//
// Les huit directions ne couvrent pas le plan : une case en pas de cavalier
// n'est sur aucune d'elles, quelle que soit la portée.
func TestCaseNonAligneeInvisible(t *testing.T) {
	b := ouvert(21)
	depuis := Position{Colonne: 10, Ligne: 10}

	for _, cible := range []Position{
		{Colonne: 12, Ligne: 11},
		{Colonne: 11, Ligne: 13},
		{Colonne: 10, Ligne: 10},
	} {
		if EstVisible(b, depuis, cible, 20, nil) {
			t.Errorf("%v est vue depuis %v", cible, depuis)
		}
	}
}

// TestVisionSansPorteeEstVide vérifie qu'une portée nulle ou négative ne
// renvoie rien.
//
// MobiliteDe accepte les valeurs négatives, PorteeDe les ramène à zéro : un
// effet de greffon qui aveugle un pion doit lui retirer sa vue, pas planter la
// lecture de la table.
func TestVisionSansPorteeEstVide(t *testing.T) {
	b, _, err := Generer(1, parametresPour(21))
	if err != nil {
		t.Fatal(err)
	}

	depuis := Position{Colonne: 0, Ligne: 0}
	for cote := 0; cote < b.cote && !b.EstRue(depuis); cote++ {
		depuis = Position{Colonne: cote, Ligne: 0}
	}

	if vues := b.Vision(depuis, 0); vues != nil {
		t.Errorf("portée nulle : %d cases vues", len(vues))
	}
	if vues := b.Vision(depuis, -1); vues != nil {
		t.Errorf("portée négative : %d cases vues", len(vues))
	}
}

// TestVisionIgnoreLesBatiments vérifie qu'aucune case bâtie n'entre dans la
// table.
//
// La vue sert à afficher ce qu'un camp couvre : y faire figurer des murs
// gonflerait la vue filtrée d'un tiers de cases sans rien apprendre.
func TestVisionIgnoreLesBatiments(t *testing.T) {
	b, _, err := Generer(7, parametresPour(21))
	if err != nil {
		t.Fatal(err)
	}

	for depuis, vues := range b.vues {
		if !b.EstRue(depuis) {
			t.Fatalf("%v est un bâtiment et porte une ligne de vue", depuis)
		}
		for _, c := range vues {
			if !b.EstRue(c) {
				t.Fatalf("%v voit le bâtiment %v", depuis, c)
			}
		}
	}
}

// TestVisionEstSymetrique vérifie que se voir est réciproque.
//
// Le terrain seul ne connaît pas de sens : si un mur coupait dans un sens et
// pas dans l'autre, la règle d'angle des diagonales serait mal appliquée d'un
// côté.
func TestVisionEstSymetrique(t *testing.T) {
	b, _, err := Generer(3, parametresPour(21))
	if err != nil {
		t.Fatal(err)
	}

	for depuis, vues := range b.vues {
		for _, cible := range vues {
			if !EstVisible(b, cible, depuis, b.cote, nil) {
				t.Fatalf("%v voit %v, mais pas l'inverse", depuis, cible)
			}
		}
	}
}

// TestVueEtVisibiliteSAccordent vérifie que la vue envoyée aux inspecteurs dit
// la même chose qu'EstVisible, occlusion comprise.
//
// Deux réponses à la même question partent d'ici, et l'une d'elles va sur le
// réseau : si la vue annonçait une case derrière un barrage, un bot y chercherait
// un fugitif que le noyau n'y verrait pas.
func TestVueEtVisibiliteSAccordent(t *testing.T) {
	params := parametresPour(21)
	p, err := Nouvelle(ouvert(params.Cote), 1, params, registreDEssai())
	if err != nil {
		t.Fatal(err)
	}

	// Deux inspecteurs alignés, le second masquant tout ce qui le suit, plus un
	// barrage sur une autre ligne.
	p.Inspecteurs = []Inspecteur{
		{Position: Position{Colonne: 5, Ligne: 5}},
		{Position: Position{Colonne: 7, Ligne: 5}},
	}
	p.Barrages = map[Position]int{{Colonne: 5, Ligne: 8}: 3}

	vue := map[Position]bool{}
	for _, c := range p.casesVisiblesPour(CampInspecteurs) {
		vue[c] = true
	}
	if len(vue) == 0 {
		t.Fatal("les inspecteurs ne voient rien")
	}
	for i := range p.Inspecteurs {
		if !vue[p.Inspecteurs[i].Position] {
			t.Errorf("l'inspecteur %d ne voit pas la case où il se tient", i)
		}
	}

	occupees := p.casesOccupees()
	for ligne := 0; ligne < params.Cote; ligne++ {
		for colonne := 0; colonne < params.Cote; colonne++ {
			cible := Position{Colonne: colonne, Ligne: ligne}

			atteinte := false
			for i := range p.Inspecteurs {
				depuis := p.Inspecteurs[i].Position
				if cible == depuis || EstVisible(p.Plateau, depuis, cible, p.PorteeDe(i), occupees) {
					atteinte = true
					break
				}
			}
			if atteinte != vue[cible] {
				t.Fatalf("%v : la vue dit %v, EstVisible dit %v", cible, vue[cible], atteinte)
			}
		}
	}
}

// TestBarrageMasqueLaVue vérifie que la capacité du Barreur porte sur
// l'information et pas seulement sur les déplacements.
//
// C'est ce que docs/regles.md §13 exige : un mur qui n'aveugle pas ne servirait
// à rien dans un jeu où tout se joue sur ce que l'autre ignore.
func TestBarrageMasqueLaVue(t *testing.T) {
	params := parametresPour(21)
	p, err := Nouvelle(ouvert(params.Cote), 1, params, registreDEssai())
	if err != nil {
		t.Fatal(err)
	}

	// À portée de vue : parametresPour donne 4 sur un plateau de 21.
	p.Inspecteurs = []Inspecteur{{Position: Position{Colonne: 0, Ligne: 0}}}
	p.Fugitif.Position = Position{Colonne: 4, Ligne: 0}

	p.recalculerVisibilite()
	if !p.Fugitif.Visible {
		t.Fatal("sans barrage, le fugitif devrait être vu")
	}

	p.Barrages = map[Position]int{{Colonne: 2, Ligne: 0}: 3}
	p.recalculerVisibilite()
	if p.Fugitif.Visible {
		t.Error("le barrage ne coupe pas la ligne de vue")
	}
}

// TestFugitifRepereEtPerdu exerce le câblage complet : la visibilité se
// recalcule à chaque fin de tour, dans les deux sens.
//
// C'est ce que l'étape 4 débloque réellement — jusqu'ici EstVisible renvoyait
// toujours false, et le fugitif n'était jamais repéré quoi qu'il fasse.
func TestFugitifRepereEtPerdu(t *testing.T) {
	params := parametresPour(21)
	p, err := Nouvelle(ouvert(params.Cote), 1, params, registreDEssai())
	if err != nil {
		t.Fatal(err)
	}

	p.Inspecteurs = []Inspecteur{{Position: Position{Colonne: 0, Ligne: 0}}}
	p.Fugitif.Position = Position{Colonne: 4, Ligne: 0}

	if defaire := p.recalculerVisibilite(); len(defaire) == 0 {
		t.Fatal("le fugitif en vue n'est pas repéré")
	}
	if !p.Fugitif.Visible {
		t.Fatal("le fugitif est en ligne de vue et reste invisible")
	}

	// Hors de portée : il redevient invisible, ce qu'un simple « devient
	// visible » ne rendrait pas.
	p.Fugitif.Position = Position{Colonne: 20, Ligne: 20}
	p.recalculerVisibilite()
	if p.Fugitif.Visible {
		t.Error("le fugitif sorti de la ligne de vue reste repéré")
	}
}
