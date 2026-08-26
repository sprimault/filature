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

// majAttendus réécrit les plateaux de référence au lieu de les comparer.
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

// plateauxDeReference couvre les trois préréglages de taille et, pour chacun,
// les tirages les plus extrêmes trouvés sur quatre cents graines.
//
// Les cas pénibles importent plus que les cas moyens : un plateau très ouvert
// ne piège personne, un plateau fermé enferme le fugitif. Ce sont eux qui
// diront qu'un changement de génération a dérivé.
func plateauxDeReference() []casPlateau {
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

// parametresPour rend des paramètres valides pour une taille donnée.
func parametresPour(cote int) Settings {
	p := DefaultSettings()
	p.Size = cote
	p.Range = max(3, cote/5)
	return p
}

// TestReferenceBoards fige ce que la génération produit.
//
// C'est le critère de livraison de l'étape 3 : une graine donnée produit
// toujours le même plateau. Les autres tests vérifient qu'il est jouable ; seul
// celui-ci s'oppose à ce qu'il change, et c'est ce qu'on veut — une génération
// qui dérive périme les plateaux enregistrés, les parties rejouables et toute
// comparaison d'équilibrage.
func TestReferenceBoards(t *testing.T) {
	for _, cas := range plateauxDeReference() {
		t.Run(cas.nom, func(t *testing.T) {
			b, retenue, err := Generate(cas.graine, parametresPour(cas.cote))
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
	fmt.Fprintf(&texte, "# %s — graine demandée %d, retenue %d, côté %d\n\n",
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
		b, _, err := Generate(graine, p)
		if err != nil {
			t.Fatalf("graine %d : %v", graine, err)
		}
		if err := b.validate(p); err != nil {
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

// TestValidationRejects vérifie que les trois critères mordent réellement.
//
// Un validateur qui accepte tout laisserait passer les plateaux injouables, et
// aucun autre test ne s'en apercevrait puisqu'ils partent tous de plateaux
// validés.
func TestValidationRejects(t *testing.T) {
	p := parametresPour(21)

	t.Run("plateau vide", func(t *testing.T) {
		b := &BoundedBoard{cote: p.Size, rues: make([]bool, p.Size*p.Size)}
		if err := b.validate(p); err == nil {
			t.Error("un plateau sans rue est accepté")
		}
	})

	t.Run("plateau entièrement ouvert", func(t *testing.T) {
		b := &BoundedBoard{cote: p.Size, rues: make([]bool, p.Size*p.Size)}
		for i := range b.rues {
			b.rues[i] = true
		}
		if err := b.validate(p); err == nil {
			t.Error("un plateau à 100 % de rues est accepté")
		}
	})

	t.Run("rue isolée", func(t *testing.T) {
		b, _, err := Generate(3, p)
		if err != nil {
			t.Fatal(err)
		}
		// Une case ouverte au milieu d'un îlot, sans voisine praticable.
		for ligne := 1; ligne < b.cote-1; ligne++ {
			for colonne := 1; colonne < b.cote-1; colonne++ {
				c := Position{Column: colonne, Row: ligne}
				if !b.IsStreet(c) && !b.touchesStreet(c) {
					b.open(c)
					if err := b.validate(p); err == nil {
						t.Error("une rue isolée du reste du plateau est acceptée")
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
// Le seuil est un plancher, pas une cible : combien il en faut ne se saura
// qu'en jouant. Il est placé sous le minimum mesuré et au-dessus de ce que les
// seules cours percées produisent, ce qui est tout ce qu'on lui demande — une
// version de carveDeadEnds sans effet échoue ici sur les trois préréglages.
func TestBoardCarriesDeadEnds(t *testing.T) {
	for _, pre := range Presets() {
		t.Run(pre.Key, func(t *testing.T) {
			plancher := pre.Settings.Size / 4

			for graine := int64(1); graine <= 60; graine++ {
				b, _, err := Generate(graine, pre.Settings)
				if err != nil {
					t.Fatalf("graine %d : %v", graine, err)
				}
				if n := deadEnds(b); n < plancher {
					t.Fatalf("graine %d : %d impasses, attendu au moins %d", graine, n, plancher)
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

				partie, err := NewGame(b, retenue, pre.Settings, registreDEssai())
				if err != nil {
					t.Fatalf("graine %d : %v", graine, err)
				}

				depart := partie.Fugitive.Position
				if !b.IsStreet(depart) {
					t.Fatalf("graine %d : le fugitif part d'un bâtiment en %v", graine, depart)
				}
				if abs(depart.Column-milieu) > CentreRadius ||
					abs(depart.Row-milieu) > CentreRadius {
					t.Fatalf("graine %d : départ en %v, hors du noyau central autour de (%d, %d)",
						graine, depart, milieu, milieu)
				}
			}
		})
	}
}
