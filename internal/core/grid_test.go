// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// majAttendus réécrit les plateaux de référence au lieu de les compare.
//
// Jamais automatique : un attendu régénéré sans être relu ne teste plus rien.
// Le diff d'un plateau se lit à l'œil — c'est pour cela qu'ils sont stockés en
// texte et non sous une forme compacte.
var majAttendus = flag.Bool("maj-attendus", false, "réécrire les plateaux de référence")

// casPlateau est une graine figée et la taille qui l'accompagne.
type casPlateau struct {
	nom    string
	graine int64
	cote   int
}

// referenceBoards couvre les trois préréglages de taille et, pour chacun,
// les tirages les plus extrêmes trouvés sur quatre cents graines.
//
// Les cas pénibles importent plus que les cas moyens : un plateau très ouvert
// ne piège personne, un plateau fermé enferme le fugitif. Ce sont eux qui
// diront qu'un changement de génération a dérivé.
func referenceBoards() []casPlateau {
	return []casPlateau{
		{"petit-ouvert", 1, 21},
		{"petit-ferme", 23, 21},
		{"moyen-ouvert", 1, 31},
		{"moyen-ferme", 148, 31},
		{"grand-ouvert", 97, 41},
		{"grand-ferme", 73, 41},
		{"grand-ordinaire", 178342119, 41},
	}
}

// settingsFor rend des paramètres valides pour une taille donnée.
func settingsFor(cote int) Settings {
	return SettingsForSize(cote)
}

// TestReferenceBoards fige ce que la génération produit.
//
// C'est le critère de livraison de l'étape 3 : une graine donnée produit
// toujours le même plateau. Les autres tests vérifient qu'il est jouable ; seul
// celui-ci s'oppose à ce qu'il change, et c'est ce qu'on veut — une génération
// qui dérive périme les plateaux enregistrés, les parties rejouables et toute
// comparaison d'équilibrage.
func TestReferenceBoards(t *testing.T) {
	for _, cas := range referenceBoards() {
		t.Run(cas.nom, func(t *testing.T) {
			b, retenue, err := Generate(cas.graine, settingsFor(cas.cote))
			if err != nil {
				t.Fatalf("génération : %v", err)
			}

			rendu := drawAsText(b, cas, retenue)
			chemin := filepath.Join("testdata", "plateaux", cas.nom+".txt")

			if *majAttendus {
				if err := os.MkdirAll(filepath.Dir(chemin), 0o750); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(chemin, []byte(rendu), 0o600); err != nil {
					t.Fatal(err)
				}
				t.Log("plateau réécrit : relire le diff avant de le livrer")
				return
			}

			attendu, err := os.ReadFile(chemin)
			if err != nil {
				t.Fatalf("%v — lancer « go test ./internal/noyau -maj-attendus »", err)
			}
			if string(attendu) != rendu {
				t.Error("le plateau a changé — relancer avec -maj-attendus et relire le diff")
			}
		})
	}
}

// drawAsText rend un plateau lisible : un point par rue, un dièse par
// bâtiment, le numéro d'une zone sur ses cases.
//
// Lisible et non compact, parce que c'est un diff qu'on relit : une génération
// qui dérive se voit alors dans sa forme, pas dans une somme de contrôle.
func drawAsText(b *BoundedBoard, cas casPlateau, retenue int64) string {
	zones := map[Position]int{}
	for _, z := range b.zones {
		for _, c := range z.Cells {
			zones[c] = z.Number
		}
	}

	var texte strings.Builder
	// La graine retenue est une sortie, pas un choix : c'est celle où la
	// validation a fini par aboutir, et elle rebouge à chaque changement de
	// génération. Seule la demandée est fixée par le cas de test.
	fmt.Fprintf(&texte, "# %s — graine demandée %d, retenue %d (sortie), côté %d\n\n",
		cas.nom, cas.graine, retenue, cas.cote)

	for ligne := 0; ligne < b.cote; ligne++ {
		for colonne := 0; colonne < b.cote; colonne++ {
			p := Position{Column: colonne, Row: ligne}
			switch {
			case inAZone(b, p):
				fmt.Fprint(&texte, zones[p])
			case b.IsStreet(p):
				texte.WriteString(".")
			default:
				texte.WriteString("#")
			}
		}
		texte.WriteString("\n")
	}
	return texte.String()
}

// inAZone dit si la case appartient à un point d'extraction.
func inAZone(b *BoundedBoard, p Position) bool {
	for _, z := range b.zones {
		if z.Contains(p) {
			return true
		}
	}
	return false
}

// TestBoardAlwaysPlayable vérifie les trois critères sur beaucoup de
// graines, et non sur les seules qu'on a figées.
//
// Un plateau qui passe la validation est jouable par construction ; ce test
// vérifie surtout qu'aucune graine ne met le générateur en échec, ce qui
// arriverait si les perçages ou les impasses dérivaient.
func TestBoardAlwaysPlayable(t *testing.T) {
	p := DefaultSettings()

	for graine := int64(1); graine <= 150; graine++ {
		b, retenue, err := Generate(graine, p)
		if err != nil {
			t.Fatalf("graine %d : %v", graine, err)
		}
		// La graine retenue redonne le même tirage, donc la même trame : c'est
		// le seul moyen de retrouver le compte que Generate a validé.
		_, trame := draw(retenue, p)
		if err := b.validate(p, trame); err != nil {
			t.Fatalf("graine %d : un plateau retenu ne valide pas — %v", graine, err)
		}
		if len(b.Zones()) != p.Zones {
			t.Fatalf("graine %d : %d zones, attendu %d", graine, len(b.Zones()), p.Zones)
		}
	}
}

// TestSameSeedSameBoard est l'invariant de déterminisme appliqué au
// terrain.
func TestSameSeedSameBoard(t *testing.T) {
	p := DefaultSettings()

	for graine := int64(1); graine <= 20; graine++ {
		a, ga, err := Generate(graine, p)
		if err != nil {
			t.Fatal(err)
		}
		b, gb, err := Generate(graine, p)
		if err != nil {
			t.Fatal(err)
		}

		if ga != gb {
			t.Fatalf("graine %d : deux graines retenues différentes, %d et %d", graine, ga, gb)
		}
		for i := range a.rues {
			if a.rues[i] != b.rues[i] {
				t.Fatalf("graine %d : les plateaux diffèrent à la case %d", graine, i)
			}
		}
	}
}

// TestZonesOnThePerimeter vérifie qu'elles sont en périphérie et séparées.
//
// Deux zones voisines seraient couvrables par un seul inspecteur, ce qui
// viderait de son sens la règle qui en exige plus qu'il n'y a d'inspecteurs.
func TestZonesOnThePerimeter(t *testing.T) {
	p := DefaultSettings()
	b, _, err := Generate(7, p)
	if err != nil {
		t.Fatal(err)
	}

	centres := make([]Position, 0, len(b.Zones()))
	for _, z := range b.Zones() {
		centre := z.Cells[len(z.Cells)/2]
		centres = append(centres, centre)

		// En périphérie : à moins d'un tiers du côté depuis un bord.
		bord := min(min(centre.Column, centre.Row),
			min(p.Size-1-centre.Column, p.Size-1-centre.Row))
		if bord > p.Size/3 {
			t.Errorf("zone %d à %d cases du bord le plus proche", z.Number, bord)
		}
	}

	for i := range centres {
		for j := i + 1; j < len(centres); j++ {
			if d := ChebyshevDistance(centres[i], centres[j]); d < ZoneSize {
				t.Errorf("zones %d et %d distantes de %d, elles se chevauchent", i, j, d)
			}
		}
	}
}

// TestZonesWalkable vérifie qu'on peut se tenir dans chacune.
//
// Une zone taillée dans un îlot serait un point d'extraction où le fugitif ne
// peut pas entrer : la partie deviendrait ingagnable pour lui sans que rien ne
// le dise.
func TestZonesWalkable(t *testing.T) {
	p := DefaultSettings()

	for graine := int64(1); graine <= 50; graine++ {
		b, _, err := Generate(graine, p)
		if err != nil {
			t.Fatal(err)
		}
		for _, z := range b.Zones() {
			rues := 0
			for _, c := range z.Cells {
				if b.IsStreet(c) {
					rues++
				}
			}
			if rues < StreetsPerZone {
				t.Errorf("graine %d, zone %d : %d cases praticables, attendu %d",
					graine, z.Number, rues, StreetsPerZone)
			}
		}
	}
}

// TestValidationRejects vérifie que chacun des six refus de validate mord, et
// mord pour la raison annoncée.
//
// Un validateur qui accepte tout laisserait passer les plateaux injouables, et
// aucun autre test ne s'en apercevrait puisqu'ils partent tous de plateaux
// validés.
//
// **Le message autant que le rejet.** Plusieurs critères attrapent les mêmes
// plateaux dégénérés — un plateau vide échoue aussi sur son taux —, si bien
// qu'un critère supprimé peut rester invisible. Trois l'étaient : les zones et
// les lieux atteignables n'avaient aucun cas, et « sans rue » était couvert par
// le taux qui vaut alors zéro.
//
// Quand le taux n'est pas le sujet, ruesTrame est passé plutôt que compté :
// c'est ce qui permet d'atteindre le critère suivant sans avoir à bâtir un
// plateau qui satisfasse tous les précédents par accident.
func TestValidationRejects(t *testing.T) {
	p := settingsFor(21)
	trameValide := p.Size * p.Size * (MinStreetRatio + MaxStreetRatio) / 200

	// connexe ouvre le bord haut et la colonne gauche : de quoi former une
	// seule composante et porter des rues hors du noyau, sans rien ouvrir
	// ailleurs — ce qui laisse la place aux blocs qu'on veut voir refusés.
	connexe := func() *BoundedBoard {
		b := &BoundedBoard{cote: p.Size, rues: make([]bool, p.Size*p.Size)}
		for i := 0; i < p.Size; i++ {
			b.open(Position{Column: i, Row: 0})
			b.open(Position{Column: 0, Row: i})
		}
		return b
	}

	// batiCentral rend les neuf cases d'un bloc laissé entièrement bâti, au
	// milieu du plateau où connexe n'a rien ouvert.
	batiCentral := func() []Position {
		milieu := p.Size / 2
		var cases []Position
		for ligne := milieu - 1; ligne <= milieu+1; ligne++ {
			for colonne := milieu - 1; colonne <= milieu+1; colonne++ {
				cases = append(cases, Position{Column: colonne, Row: ligne})
			}
		}
		return cases
	}

	t.Run("plateau vide", func(t *testing.T) {
		b := &BoundedBoard{cote: p.Size, rues: make([]bool, p.Size*p.Size)}
		err := b.validate(p, trameValide)
		if err == nil {
			t.Fatal("un plateau sans rue est accepté")
		}
		if !strings.Contains(err.Error(), "sans rue") {
			t.Errorf("rejeté pour %q, attendu le critère de la rue absente", err)
		}
	})

	t.Run("plateau entièrement ouvert", func(t *testing.T) {
		b := &BoundedBoard{cote: p.Size, rues: make([]bool, p.Size*p.Size)}
		for i := range b.rues {
			b.rues[i] = true
		}
		err := b.validate(p, b.countStreets())
		if err == nil {
			t.Fatal("un plateau à 100 % de rues est accepté")
		}
		if !strings.Contains(err.Error(), "taux de rues") {
			t.Errorf("rejeté pour %q, attendu le critère du taux", err)
		}
	})

	// Les deux cas suivants ne se produisent pas depuis draw, dont carveBlock
	// garantit cinq cases praticables par bloc : ils gardent le générateur de
	// plateau d'un plugin, qui pose ses zones comme il l'entend. Un bloc qui
	// porte au moins une rue est toujours atteignable, la connexité étant
	// vérifiée juste avant — c'est un bloc entièrement bâti qu'il faut.
	t.Run("zone inatteignable", func(t *testing.T) {
		b := connexe()
		b.zones = []Zone{{Number: 3, Cells: batiCentral()}}

		err := b.validate(p, trameValide)
		if err == nil {
			t.Fatal("une zone sans case praticable est acceptée")
		}
		if !strings.Contains(err.Error(), "zone 3") {
			t.Errorf("rejeté pour %q, attendu le critère des zones atteignables", err)
		}
	})

	t.Run("lieu inatteignable", func(t *testing.T) {
		b := connexe()
		b.abris = []Shelter{{Number: 2, Cells: batiCentral()}}

		err := b.validate(p, trameValide)
		if err == nil {
			t.Fatal("un lieu sans case praticable est accepté")
		}
		if !strings.Contains(err.Error(), "lieu 2") {
			t.Errorf("rejeté pour %q, attendu le critère des lieux atteignables", err)
		}
	})

	t.Run("aucune rue hors du noyau", func(t *testing.T) {
		// Un bloc plein au centre : le taux de rues tombe dans ses bornes et
		// tout se tient, mais les inspecteurs n'ont nulle part où se placer.
		// Le rayon est poussé à sa borne haute, seule façon d'atteindre le
		// critère — aux rayons dérivés du côté, la couronne porte des
		// centaines de rues et il ne se déclenche jamais.
		q := p
		q.CentreRadius = q.Size/2 - ZoneSize

		b := &BoundedBoard{cote: q.Size, rues: make([]bool, q.Size*q.Size)}
		milieu := q.Size / 2
		for ligne := milieu - 7; ligne <= milieu+6; ligne++ {
			for colonne := milieu - 7; colonne <= milieu+6; colonne++ {
				b.rues[ligne*b.cote+colonne] = true
			}
		}

		err := b.validate(q, b.countStreets())
		if err == nil {
			t.Fatal("un plateau sans rue hors du noyau est accepté")
		}
		// Le message autant que le rejet : ce plateau tient ses autres
		// critères, et un échec sur le taux de rues ferait croire que celui-ci
		// mord alors qu'il n'aurait pas été atteint.
		if !strings.Contains(err.Error(), "hors du noyau") {
			t.Errorf("rejeté pour %q, attendu le critère des rues hors du noyau", err)
		}
	})

	t.Run("rue isolée", func(t *testing.T) {
		b, retenue, err := Generate(3, p)
		if err != nil {
			t.Fatal(err)
		}
		_, trame := draw(retenue, p)
		// Une case ouverte au milieu d'un îlot, sans voisine praticable.
		for ligne := 1; ligne < b.cote-1; ligne++ {
			for colonne := 1; colonne < b.cote-1; colonne++ {
				c := Position{Column: colonne, Row: ligne}
				if !b.IsStreet(c) && !b.touchesStreet(c) {
					b.open(c)
					err := b.validate(p, trame)
					if err == nil {
						t.Fatal("une rue isolée du reste du plateau est acceptée")
					}
					if !strings.Contains(err.Error(), "isolees") {
						t.Errorf("rejeté pour %q, attendu le critère de la connexité", err)
					}
					return
				}
			}
		}
		t.Skip("aucune case assez enclavée sur ce plateau")
	})
}

// deadEnds compte les rues qui n'ont qu'une seule voisine praticable.
//
// C'est la seule mesure d'une impasse qui ait un sens ici : carveDeadEnds
// ouvre des couloirs, mais un couloir qui rejoint une rue existante n'en est
// pas une, et rien ne distingue les deux au moment du tirage.
func deadEnds(b *BoundedBoard) int {
	n := 0
	for ligne := 0; ligne < b.cote; ligne++ {
		for colonne := 0; colonne < b.cote; colonne++ {
			c := Position{Column: colonne, Row: ligne}
			if !b.IsStreet(c) {
				continue
			}
			voisines := 0
			for _, d := range Orthogonales {
				if b.IsStreet(c.Step(d)) {
					voisines++
				}
			}
			if voisines == 1 {
				n++
			}
		}
	}
	return n
}

// TestBoardCarriesDeadEnds garde la quatrième étape de la génération, seule
// à n'avoir aucun critère dans validate.
//
// docs/regles.md §3 leur donne un rôle nommé : sans bords à exploiter, ce sont
// elles qui permettent le piégeage, et le Barreur y gagne sa raison d'être. Un
// plateau qui n'en aurait pas resterait connexe, praticable et accepté par la
// validation — la capacité deviendrait décorative sans que rien ne le dise.
//
// Le même tirage est comparé à lui-même privé de l'étape, et non à un plancher.
// Un seuil aurait été emprunté à la distribution que les autres critères
// laissent passer : quand le taux de rues a cessé de compter les blocs, la
// population s'est élargie et sa queue basse est apparue — quatre impasses sur
// un Quartier là où le plancher en demandait cinq, sans que l'étape ait bougé.
// Un seuil calibré ainsi ne dit pas si carveDeadEnds agit, il dit ce que le
// hasard a laissé passer. Ce qui doit être vrai est plus simple, et vrai quelle
// que soit la taille : après l'étape, il y en a davantage qu'avant.
func TestBoardCarriesDeadEnds(t *testing.T) {
	for _, pre := range Presets() {
		t.Run(pre.Key, func(t *testing.T) {
			for graine := int64(1); graine <= 60; graine++ {
				b, retenue, err := Generate(graine, pre.Settings)
				if err != nil {
					t.Fatalf("graine %d : %v", graine, err)
				}

				// Le même plateau sans la quatrième étape. La graine retenue
				// rejoue les trois premières à l'identique, donc l'écart ne
				// vient que d'elle.
				sans := &BoundedBoard{graine: retenue, cote: pre.Settings.Size,
					rues: make([]bool, pre.Settings.Size*pre.Settings.Size)}
				sans.traceAvenues(NewRandom(retenue, "grid"))
				sans.punchCourtyards(NewRandom(retenue, "courtyards"))
				sans.placeZones(pre.Settings.Zones)
				sans.placeShelters(pre.Settings)

				avec, avant := deadEnds(b), deadEnds(sans)
				if avec <= avant {
					t.Fatalf("graine %d : %d impasses avec l'etape, %d sans — elle n'ajoute rien",
						graine, avec, avant)
				}
			}
		})
	}
}

// TestGeneratedBoardStartsAGame relie les deux moitiés de la mise en place :
// un plateau tiré au sort, et le noyau qui doit y place le fugitif.
//
// La validation ne regarde que la connexité, le taux de rues et les zones — le
// centre ne fait partie d'aucun des trois. Un tirage qui bâtirait tout le noyau
// central serait donc retenu, puis refusé par NewGame sur « aucune rue au
// centre du plateau » : à la première partie, et non à la génération.
func TestGeneratedBoardStartsAGame(t *testing.T) {
	for _, pre := range Presets() {
		t.Run(pre.Key, func(t *testing.T) {
			milieu := pre.Settings.Size / 2

			for graine := int64(1); graine <= 40; graine++ {
				b, retenue, err := Generate(graine, pre.Settings)
				if err != nil {
					t.Fatalf("graine %d : %v", graine, err)
				}

				partie, err := NewGame(b, retenue, pre.Settings, testRegistry())
				if err != nil {
					t.Fatalf("graine %d : %v", graine, err)
				}

				depart := partie.Fugitive.Position
				if !b.IsStreet(depart) {
					t.Fatalf("graine %d : le fugitif part d'un bâtiment en %v", graine, depart)
				}
				if abs(depart.Column-milieu) > pre.Settings.CentreRadius ||
					abs(depart.Row-milieu) > pre.Settings.CentreRadius {
					t.Fatalf("graine %d : départ en %v, hors du noyau central autour de (%d, %d)",
						graine, depart, milieu, milieu)
				}
			}
		})
	}
}
