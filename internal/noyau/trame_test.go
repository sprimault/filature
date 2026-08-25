// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

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
func parametresPour(cote int) Parametres {
	p := ParametresDefaut()
	p.Cote = cote
	p.Portee = max(3, cote/5)
	return p
}

// TestPlateauxDeReference fige ce que la génération produit.
//
// C'est le critère de livraison de l'étape 3 : une graine donnée produit
// toujours le même plateau. Les autres tests vérifient qu'il est jouable ; seul
// celui-ci s'oppose à ce qu'il change, et c'est ce qu'on veut — une génération
// qui dérive périme les plateaux enregistrés, les parties rejouables et toute
// comparaison d'équilibrage.
func TestPlateauxDeReference(t *testing.T) {
	for _, cas := range plateauxDeReference() {
		t.Run(cas.nom, func(t *testing.T) {
			b, retenue, err := Generer(cas.graine, parametresPour(cas.cote))
			if err != nil {
				t.Fatalf("génération : %v", err)
			}

			rendu := dessinerEnTexte(b, cas, retenue)
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

// dessinerEnTexte rend un plateau lisible : un point par rue, un dièse par
// bâtiment, le numéro d'une zone sur ses cases.
//
// Lisible et non compact, parce que c'est un diff qu'on relit : une génération
// qui dérive se voit alors dans sa forme, pas dans une somme de contrôle.
func dessinerEnTexte(b *PlateauBorne, cas casPlateau, retenue int64) string {
	zones := map[Position]int{}
	for _, z := range b.zones {
		for _, c := range z.Cases {
			zones[c] = z.Numero
		}
	}

	var texte strings.Builder
	fmt.Fprintf(&texte, "# %s — graine demandée %d, retenue %d, côté %d\n\n",
		cas.nom, cas.graine, retenue, cas.cote)

	for ligne := 0; ligne < b.cote; ligne++ {
		for colonne := 0; colonne < b.cote; colonne++ {
			p := Position{Colonne: colonne, Ligne: ligne}
			switch {
			case dansUneZone(b, p):
				fmt.Fprint(&texte, zones[p])
			case b.EstRue(p):
				texte.WriteString(".")
			default:
				texte.WriteString("#")
			}
		}
		texte.WriteString("\n")
	}
	return texte.String()
}

// dansUneZone dit si la case appartient à un point d'extraction.
func dansUneZone(b *PlateauBorne, p Position) bool {
	for _, z := range b.zones {
		if z.Contient(p) {
			return true
		}
	}
	return false
}

// TestPlateauToujoursJouable vérifie les trois critères sur beaucoup de
// graines, et non sur les seules qu'on a figées.
//
// Un plateau qui passe la validation est jouable par construction ; ce test
// vérifie surtout qu'aucune graine ne met le générateur en échec, ce qui
// arriverait si les perçages ou les impasses dérivaient.
func TestPlateauToujoursJouable(t *testing.T) {
	p := ParametresDefaut()

	for graine := int64(1); graine <= 150; graine++ {
		b, _, err := Generer(graine, p)
		if err != nil {
			t.Fatalf("graine %d : %v", graine, err)
		}
		if err := b.valider(p); err != nil {
			t.Fatalf("graine %d : un plateau retenu ne valide pas — %v", graine, err)
		}
		if len(b.Zones()) != p.Zones {
			t.Fatalf("graine %d : %d zones, attendu %d", graine, len(b.Zones()), p.Zones)
		}
	}
}

// TestMemeGraineMemePlateau est l'invariant de déterminisme appliqué au
// terrain.
func TestMemeGraineMemePlateau(t *testing.T) {
	p := ParametresDefaut()

	for graine := int64(1); graine <= 20; graine++ {
		a, ga, err := Generer(graine, p)
		if err != nil {
			t.Fatal(err)
		}
		b, gb, err := Generer(graine, p)
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

// TestZonesSurLePourtour vérifie qu'elles sont en périphérie et séparées.
//
// Deux zones voisines seraient couvrables par un seul inspecteur, ce qui
// viderait de son sens la règle qui en exige plus qu'il n'y a d'inspecteurs.
func TestZonesSurLePourtour(t *testing.T) {
	p := ParametresDefaut()
	b, _, err := Generer(7, p)
	if err != nil {
		t.Fatal(err)
	}

	centres := make([]Position, 0, len(b.Zones()))
	for _, z := range b.Zones() {
		centre := z.Cases[len(z.Cases)/2]
		centres = append(centres, centre)

		// En périphérie : à moins d'un tiers du côté depuis un bord.
		bord := min(min(centre.Colonne, centre.Ligne),
			min(p.Cote-1-centre.Colonne, p.Cote-1-centre.Ligne))
		if bord > p.Cote/3 {
			t.Errorf("zone %d à %d cases du bord le plus proche", z.Numero, bord)
		}
	}

	for i := range centres {
		for j := i + 1; j < len(centres); j++ {
			if d := DistanceTchebychev(centres[i], centres[j]); d < CoteZone {
				t.Errorf("zones %d et %d distantes de %d, elles se chevauchent", i, j, d)
			}
		}
	}
}

// TestZonesPraticables vérifie qu'on peut se tenir dans chacune.
//
// Une zone taillée dans un îlot serait un point d'extraction où le fugitif ne
// peut pas entrer : la partie deviendrait ingagnable pour lui sans que rien ne
// le dise.
func TestZonesPraticables(t *testing.T) {
	p := ParametresDefaut()

	for graine := int64(1); graine <= 50; graine++ {
		b, _, err := Generer(graine, p)
		if err != nil {
			t.Fatal(err)
		}
		for _, z := range b.Zones() {
			rues := 0
			for _, c := range z.Cases {
				if b.EstRue(c) {
					rues++
				}
			}
			if rues < RuesParZone {
				t.Errorf("graine %d, zone %d : %d cases praticables, attendu %d",
					graine, z.Numero, rues, RuesParZone)
			}
		}
	}
}

// TestValidationRejette vérifie que les trois critères mordent réellement.
//
// Un validateur qui accepte tout laisserait passer les plateaux injouables, et
// aucun autre test ne s'en apercevrait puisqu'ils partent tous de plateaux
// validés.
func TestValidationRejette(t *testing.T) {
	p := parametresPour(21)

	t.Run("plateau vide", func(t *testing.T) {
		b := &PlateauBorne{cote: p.Cote, rues: make([]bool, p.Cote*p.Cote)}
		if err := b.valider(p); err == nil {
			t.Error("un plateau sans rue est accepté")
		}
	})

	t.Run("plateau entièrement ouvert", func(t *testing.T) {
		b := &PlateauBorne{cote: p.Cote, rues: make([]bool, p.Cote*p.Cote)}
		for i := range b.rues {
			b.rues[i] = true
		}
		if err := b.valider(p); err == nil {
			t.Error("un plateau à 100 % de rues est accepté")
		}
	})

	t.Run("rue isolée", func(t *testing.T) {
		b, _, err := Generer(3, p)
		if err != nil {
			t.Fatal(err)
		}
		// Une case ouverte au milieu d'un îlot, sans voisine praticable.
		for ligne := 1; ligne < b.cote-1; ligne++ {
			for colonne := 1; colonne < b.cote-1; colonne++ {
				c := Position{Colonne: colonne, Ligne: ligne}
				if !b.EstRue(c) && !b.toucheUneRue(c) {
					b.ouvrir(c)
					if err := b.valider(p); err == nil {
						t.Error("une rue isolée du reste du plateau est acceptée")
					}
					return
				}
			}
		}
		t.Skip("aucune case assez enclavée sur ce plateau")
	})
}
