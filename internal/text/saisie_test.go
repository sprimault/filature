// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package text

import (
	"errors"
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/core"
)

// troisCoups rend une liste courte mais variée, de quoi exercer la description
// comme la sélection.
func troisCoups() []core.Coup {
	return []core.Coup{
		{Tour: 2, Acteur: core.CampFugitif, Type: core.CoupDeplacer,
			Depart: core.Position{Colonne: 3, Ligne: 3}, Arrivee: core.Position{Colonne: 3, Ligne: 2}},
		{Tour: 2, Acteur: core.CampFugitif, Type: core.CoupDepense, Depense: core.DepenseSilence},
		{Tour: 2, Acteur: core.CampFugitif, Type: core.CoupFinDePhase},
	}
}

// TestLireCoupRendCeluiQuOnDesigne vérifie la sélection par numéro.
func TestLireCoupRendCeluiQuOnDesigne(t *testing.T) {
	coups := troisCoups()

	c, err := LireCoup(strings.NewReader("2\n"), &strings.Builder{}, coups)
	if err != nil {
		t.Fatalf("saisie refusée : %v", err)
	}
	if c != coups[1] {
		t.Errorf("coup %+v, attendu %+v", c, coups[1])
	}
}

// TestSaisieFautiveRedemandee vérifie qu'une faute de frappe ne coûte pas le
// tour.
//
// Le noyau ne doit jamais voir un coup illégal : le rattrapage se fait ici,
// avant, et non par un refus qui remonterait jusqu'à lui.
func TestSaisieFautiveRedemandee(t *testing.T) {
	coups := troisCoups()
	var sortie strings.Builder

	c, err := LireCoup(strings.NewReader("zéro\n99\n0\n1\n"), &sortie, coups)
	if err != nil {
		t.Fatalf("saisie refusée : %v", err)
	}
	if c != coups[0] {
		t.Errorf("coup %+v, attendu le premier", c)
	}
	if n := strings.Count(sortie.String(), "n'est pas un numéro"); n != 3 {
		t.Errorf("%d rappels, attendu 3", n)
	}
}

// TestAbandonEtFinDEntree vérifie les deux façons de sortir.
//
// La fin d'entrée vaut abandon plutôt qu'erreur : c'est ce qui rend le mode
// texte pilotable depuis un fichier, donc utilisable en test d'intégration.
func TestAbandonEtFinDEntree(t *testing.T) {
	for nom, entree := range map[string]string{
		"q tapé":         "q\n",
		"entrée épuisée": "",
	} {
		t.Run(nom, func(t *testing.T) {
			_, err := LireCoup(strings.NewReader(entree), &strings.Builder{}, troisCoups())
			if !errors.Is(err, ErrQuitter) {
				t.Errorf("erreur %v, attendu ErrQuitter", err)
			}
		})
	}
}

// TestDecrireChaqueTypeDeCoup vérifie qu'aucun coup ne s'affiche en jargon.
//
// La liste est ce sur quoi le joueur choisit : un type brut y serait illisible,
// et un coup mal décrit se joue par erreur.
func TestDecrireChaqueTypeDeCoup(t *testing.T) {
	cas := []struct {
		coup    core.Coup
		attendu string
	}{
		{core.Coup{Type: core.CoupPlacer, Acteur: core.CampFugitif, Zone: 4}, "sceller la zone 4"},
		{core.Coup{Type: core.CoupPlacer, Acteur: core.CampInspecteurs, Pion: 1,
			Arrivee: core.Position{Colonne: 2, Ligne: 6}}, "placer B en (2,6)"},
		{core.Coup{Type: core.CoupDeplacer, Acteur: core.CampInspecteurs, Pion: 0,
			Depart: core.Position{Colonne: 1, Ligne: 1}, Arrivee: core.Position{Colonne: 2, Ligne: 1}}, "déplacer A"},
		{core.Coup{Type: core.CoupCapacite, Pion: 2, Capacite: "traqueur"}, "capacité traqueur de C"},
		{core.Coup{Type: core.CoupDepense, Depense: core.DepenseMeurtre}, "dépenser meurtre"},
		{core.Coup{Type: core.CoupChangerZone, Zone: 1}, "resceller vers la zone 1"},
		{core.Coup{Type: core.CoupPasser}, "passer"},
		{core.Coup{Type: core.CoupFinDePhase}, "rendre la main"},
	}

	for _, c := range cas {
		if rendu := Decrire(c.coup); !strings.Contains(rendu, c.attendu) {
			t.Errorf("%s rendu %q, attendu qu'il contienne %q", c.coup.Type, rendu, c.attendu)
		}
	}
}

// TestDirectionNommee vérifie que les huit sens se lisent.
//
// Choisir parmi huit voisines qui ne diffèrent que d'une unité se fait au nom
// de la direction, pas en comparant des coordonnées.
func TestDirectionNommee(t *testing.T) {
	depart := core.Position{Colonne: 5, Ligne: 5}

	for d, attendu := range map[core.Direction]string{
		core.Nord:      "nord",
		core.Est:       "est",
		core.SudOuest:  "sud-ouest",
		core.NordOuest: "nord-ouest",
	} {
		coup := core.Coup{Type: core.CoupDeplacer, Depart: depart, Arrivee: depart.Avance(d)}
		if rendu := Decrire(coup); !strings.HasSuffix(rendu, attendu) {
			t.Errorf("direction %d rendue %q, attendu qu'elle finisse par %q", d, rendu, attendu)
		}
	}
}

// TestCoupsNumerotes vérifie que la liste affichée et la saisie s'accordent sur
// le premier numéro.
func TestCoupsNumerotes(t *testing.T) {
	liste := Coups(troisCoups())

	if !strings.Contains(liste, "  1. ") || !strings.Contains(liste, "  3. ") {
		t.Errorf("numérotation absente :\n%s", liste)
	}
	if strings.Contains(liste, "  0. ") {
		t.Error("la liste commence à zéro, la saisie attend un")
	}
}
