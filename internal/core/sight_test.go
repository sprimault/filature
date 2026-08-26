// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"strings"
	"testing"
)

// drawnBoard construit un plateau depuis un croquis : un point par rue,
// tout le reste en bâtiment.
//
// Le symétrique de drawAsText, et c'est ce qui rend les cas de vision
// relisibles — un angle en équerre se voit dans le croquis, jamais dans une
// list de coordonnées.
func drawnBoard(t *testing.T, croquis string) *BoundedBoard {
	t.Helper()

	var lignes []string
	for _, l := range strings.Split(croquis, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lignes = append(lignes, l)
		}
	}

	cote := len(lignes)
	b := &BoundedBoard{cote: cote, rues: make([]bool, cote*cote)}
	for ligne, texte := range lignes {
		if len(texte) != cote {
			t.Fatalf("ligne %d : %d colonnes pour %d lignes, le croquis n'est pas carré",
				ligne, len(texte), cote)
		}
		for colonne, c := range texte {
			if c == '.' {
				b.open(Position{Column: colonne, Row: ligne})
			}
		}
	}

	b.vues = precomputeSight(b, b.cote)
	return b
}

// TestSightAndIsVisibleAgree est le test que ROADMAP.md §4 réclame :
// deux chemins indépendants pour la même règle, comparés sur les mêmes
// positions.
//
// IsVisible déroule la ligne à la demande, Sight lit la table précalculée et
// la tronque à la portée. Ils ne partagent que lineOfSight ; tout le reste — le
// filtrage par distance d'un côté, le test d'alignment de l'autre — est écrit
// deux fois. C'est exactement là qu'un désaccord se glisserait, et c'est ce
// désaccord qui a coûté cher au prototype antérieur.
func TestSightAndIsVisibleAgree(t *testing.T) {
	params := settingsFor(21)
	b, _, err := Generate(1, params)
	if err != nil {
		t.Fatal(err)
	}

	for ligne := 0; ligne < b.cote; ligne++ {
		for colonne := 0; colonne < b.cote; colonne++ {
			depuis := Position{Column: colonne, Row: ligne}
			if !b.IsStreet(depuis) {
				continue
			}

			table := map[Position]bool{}
			for _, c := range b.Sight(depuis, params.Range) {
				table[c] = true
			}

			for l := 0; l < b.cote; l++ {
				for c := 0; c < b.cote; c++ {
					cible := Position{Column: c, Row: l}
					vu := IsVisible(b, depuis, cible, params.Range, nil)
					if vu != table[cible] {
						t.Fatalf("%v vers %v : IsVisible dit %v, la table dit %v",
							depuis, cible, vu, table[cible])
					}
				}
			}
		}
	}
}

// TestRangeIsChebyshev est le piège où un prototype antérieur s'est fait
// prendre.
//
// À portée 8, huit pas en diagonale sont vus. En Manhattan cette case est à
// seize, donc hors de portée : mesurer dans la mauvaise unité ne se voit pas en
// jouant, seulement en perdant sans comprendre pourquoi.
func TestRangeIsChebyshev(t *testing.T) {
	b := ouvert(21)
	centre := Position{Column: 10, Row: 10}

	cas := []struct {
		nom    string
		cible  Position
		portee int
		vue    bool
	}{
		{"huit en diagonale, portée 8", Position{Column: 18, Row: 18}, 8, true},
		{"neuf en diagonale, portée 8", Position{Column: 19, Row: 19}, 8, false},
		{"huit tout droit, portée 8", Position{Column: 18, Row: 10}, 8, true},
		{"neuf tout droit, portée 8", Position{Column: 19, Row: 10}, 8, false},
		{"quatre en diagonale, portée 4", Position{Column: 14, Row: 14}, 4, true},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if vu := IsVisible(b, centre, c.cible, c.portee, nil); vu != c.vue {
				t.Errorf("%v depuis %v à portée %d : %v, attendu %v",
					c.cible, centre, c.portee, vu, c.vue)
			}
		})
	}
}

// TestBuildingCutsTheLine vérifie que la vue s'arrête au premier bâtiment et
// ne reprend pas derrière.
func TestBuildingCutsTheLine(t *testing.T) {
	b := drawnBoard(t, `
		.....
		.....
		..#..
		.....
		.....`)

	depuis := Position{Column: 2, Row: 0}
	if IsVisible(b, depuis, Position{Column: 2, Row: 1}, 5, nil) != true {
		t.Error("la case devant le bâtiment devrait être vue")
	}
	if IsVisible(b, depuis, Position{Column: 2, Row: 2}, 5, nil) != false {
		t.Error("le bâtiment lui-même n'est pas une case visible")
	}
	if IsVisible(b, depuis, Position{Column: 2, Row: 4}, 5, nil) != false {
		t.Error("la vue reprend derrière le bâtiment")
	}
}

// TestClosedCornerBlocksSight applique aux lignes de vue la règle d'angle des
// déplacements.
//
// Deux bâtiments en équerre ferment le passage : sans cette règle, un regard se
// faufile entre leurs coins et le bâti ne bloque plus rien en diagonale.
func TestClosedCornerBlocksSight(t *testing.T) {
	// Les deux bâtiments encadrent la diagonale sans être dessus : c'est ce qui
	// distingue un angle fermé d'un mur, et le croquis doit le montrer.
	ferme := drawnBoard(t, `
		.#.
		..#
		...`)
	ouvertDUnCote := drawnBoard(t, `
		...
		..#
		...`)

	depuis := Position{Column: 2, Row: 0}
	cible := Position{Column: 0, Row: 2}

	if IsVisible(ferme, depuis, cible, 5, nil) {
		t.Error("le regard passe entre deux bâtiments en équerre")
	}
	if !IsVisible(ouvertDUnCote, depuis, cible, 5, nil) {
		t.Error("une seule case libre suffit à contourner, la vue devrait passer")
	}
}

// TestInspectorBlocksSight vérifie la règle de docs/regles.md §13 : cinq
// inspecteurs en file indienne ne voient qu'avec le premier.
func TestInspectorBlocksSight(t *testing.T) {
	b := ouvert(21)
	depuis := Position{Column: 0, Row: 0}
	cible := Position{Column: 5, Row: 0}

	if !IsVisible(b, depuis, cible, 8, nil) {
		t.Fatal("sans obstacle, la case devrait être vue")
	}

	devant := map[Position]bool{{Column: 3, Row: 0}: true}
	if IsVisible(b, depuis, cible, 8, devant) {
		t.Error("un pion sur la ligne ne la coupe pas")
	}

	derriere := map[Position]bool{{Column: 7, Row: 0}: true}
	if !IsVisible(b, depuis, cible, 8, derriere) {
		t.Error("un pion au-delà de la cible coupe la vue")
	}
}

// TestOccupiedTargetStaysVisible distingue ce qui se tient devant de ce qu'on
// regarde.
//
// Le cas se produit dès qu'un inspecteur en observe un autre, et l'oublier
// rendrait un pion invisible à ses collègues au motif qu'il est un pion.
func TestOccupiedTargetStaysVisible(t *testing.T) {
	b := ouvert(21)
	depuis := Position{Column: 0, Row: 0}
	cible := Position{Column: 4, Row: 0}

	if !IsVisible(b, depuis, cible, 8, map[Position]bool{cible: true}) {
		t.Error("une case occupée cesse d'être vue quand elle est la cible")
	}
}

// TestUnalignedCellIsInvisible vérifie qu'un regard est rectiligne.
//
// Les huit directions ne couvrent pas le plan : une case en pas de cavalier
// n'est sur aucune d'elles, quelle que soit la portée.
func TestUnalignedCellIsInvisible(t *testing.T) {
	b := ouvert(21)
	depuis := Position{Column: 10, Row: 10}

	for _, cible := range []Position{
		{Column: 12, Row: 11},
		{Column: 11, Row: 13},
		{Column: 10, Row: 10},
	} {
		if IsVisible(b, depuis, cible, 20, nil) {
			t.Errorf("%v est vue depuis %v", cible, depuis)
		}
	}
}

// TestSightWithoutRangeIsEmpty vérifie qu'une portée nulle ou négative ne
// renvoie rien.
//
// MobilityOf accepte les valeurs négatives, RangeOf les ramène à zéro : un
// effet de plugin qui aveugle un pion doit lui retirer sa vue, pas planter la
// lecture de la table.
func TestSightWithoutRangeIsEmpty(t *testing.T) {
	b, _, err := Generate(1, settingsFor(21))
	if err != nil {
		t.Fatal(err)
	}

	depuis := Position{Column: 0, Row: 0}
	for cote := 0; cote < b.cote && !b.IsStreet(depuis); cote++ {
		depuis = Position{Column: cote, Row: 0}
	}

	if vues := b.Sight(depuis, 0); vues != nil {
		t.Errorf("portée nulle : %d cases vues", len(vues))
	}
	if vues := b.Sight(depuis, -1); vues != nil {
		t.Errorf("portée négative : %d cases vues", len(vues))
	}
}

// TestSightIgnoresBuildings vérifie qu'aucune case bâtie n'entre dans la
// table.
//
// La vue sert à afficher ce qu'un camp couvre : y faire figurer des murs
// gonflerait la vue filtrée d'un tiers de cases sans rien apprendre.
func TestSightIgnoresBuildings(t *testing.T) {
	b, _, err := Generate(7, settingsFor(21))
	if err != nil {
		t.Fatal(err)
	}

	for depuis, vues := range b.vues {
		if !b.IsStreet(depuis) {
			t.Fatalf("%v est un bâtiment et porte une ligne de vue", depuis)
		}
		for _, c := range vues {
			if !b.IsStreet(c) {
				t.Fatalf("%v voit le bâtiment %v", depuis, c)
			}
		}
	}
}

// TestSightIsSymmetric vérifie que se voir est réciproque.
//
// Le terrain seul ne connaît pas de sens : si un mur coupait dans un sens et
// pas dans l'autre, la règle d'angle des diagonales serait mal appliquée d'un
// côté.
func TestSightIsSymmetric(t *testing.T) {
	b, _, err := Generate(3, settingsFor(21))
	if err != nil {
		t.Fatal(err)
	}

	for depuis, vues := range b.vues {
		for _, cible := range vues {
			if !IsVisible(b, cible, depuis, b.cote, nil) {
				t.Fatalf("%v voit %v, mais pas l'inverse", depuis, cible)
			}
		}
	}
}

// TestViewAndSpottingAgree vérifie que la vue envoyée aux inspecteurs dit
// la même chose qu'IsVisible, occlusion comprise.
//
// Deux réponses à la même question partent d'ici, et l'une d'elles va sur le
// réseau : si la vue annonçait une case derrière un barrage, un bot y chercherait
// un fugitif que le noyau n'y verrait pas.
func TestViewAndSpottingAgree(t *testing.T) {
	params := settingsFor(21)
	p, err := NewGame(ouvert(params.Size), 1, params, testRegistry())
	if err != nil {
		t.Fatal(err)
	}

	// Deux inspecteurs alignés, le second masquant tout ce qui le suit, plus un
	// barrage sur une autre ligne.
	p.Inspectors = []Inspector{
		{Position: Position{Column: 5, Row: 5}},
		{Position: Position{Column: 7, Row: 5}},
	}
	p.Roadblocks = map[Position]int{{Column: 5, Row: 8}: 3}

	vue := map[Position]bool{}
	for _, c := range p.visibleCellsFor(SideInspectors) {
		vue[c] = true
	}
	if len(vue) == 0 {
		t.Fatal("les inspecteurs ne voient rien")
	}
	for i := range p.Inspectors {
		if !vue[p.Inspectors[i].Position] {
			t.Errorf("l'inspecteur %d ne voit pas la case où il se tient", i)
		}
	}

	occupees := p.occupiedCells()
	for ligne := 0; ligne < params.Size; ligne++ {
		for colonne := 0; colonne < params.Size; colonne++ {
			cible := Position{Column: colonne, Row: ligne}

			atteinte := false
			for i := range p.Inspectors {
				depuis := p.Inspectors[i].Position
				if cible == depuis || IsVisible(p.Board, depuis, cible, p.RangeOf(i), occupees) {
					atteinte = true
					break
				}
			}
			if atteinte != vue[cible] {
				t.Fatalf("%v : la vue dit %v, IsVisible dit %v", cible, vue[cible], atteinte)
			}
		}
	}
}

// TestRoadblockBlocksSight vérifie que la capacité du Barreur porte sur
// l'information et pas seulement sur les déplacements.
//
// C'est ce que docs/regles.md §13 exige : un mur qui n'aveugle pas ne servirait
// à rien dans un jeu où tout se joue sur ce que l'autre ignore.
func TestRoadblockBlocksSight(t *testing.T) {
	params := settingsFor(21)
	p, err := NewGame(ouvert(params.Size), 1, params, testRegistry())
	if err != nil {
		t.Fatal(err)
	}

	// À portée de vue : settingsFor donne 4 sur un plateau de 21.
	p.Inspectors = []Inspector{{Position: Position{Column: 0, Row: 0}}}
	p.Fugitive.Position = Position{Column: 4, Row: 0}

	p.recomputeSpotting()
	if !p.Fugitive.Visible {
		t.Fatal("sans barrage, le fugitif devrait être vu")
	}

	p.Roadblocks = map[Position]int{{Column: 2, Row: 0}: 3}
	p.recomputeSpotting()
	if p.Fugitive.Visible {
		t.Error("le barrage ne coupe pas la ligne de vue")
	}
}

// TestFugitiveSpottedThenLost exerce le câblage complet : la visibilité se
// recalcule à chaque fin de tour, dans les deux sens.
//
// C'est ce que l'étape 4 débloque réellement — jusqu'ici IsVisible renvoyait
// toujours false, et le fugitif n'était jamais repéré quoi qu'il fasse.
func TestFugitiveSpottedThenLost(t *testing.T) {
	params := settingsFor(21)
	p, err := NewGame(ouvert(params.Size), 1, params, testRegistry())
	if err != nil {
		t.Fatal(err)
	}

	p.Inspectors = []Inspector{{Position: Position{Column: 0, Row: 0}}}
	p.Fugitive.Position = Position{Column: 4, Row: 0}

	if defaire := p.recomputeSpotting(); len(defaire) == 0 {
		t.Fatal("le fugitif en vue n'est pas repéré")
	}
	if !p.Fugitive.Visible {
		t.Fatal("le fugitif est en ligne de vue et reste invisible")
	}

	// Hors de portée : il redevient invisible, ce qu'un simple « devient
	// visible » ne rendrait pas.
	p.Fugitive.Position = Position{Column: 20, Row: 20}
	p.recomputeSpotting()
	if p.Fugitive.Visible {
		t.Error("le fugitif sorti de la ligne de vue reste repéré")
	}
}
