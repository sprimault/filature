// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package text

import (
	"errors"
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/core"
)

// threeMoves rend une liste courte mais variée, de quoi exercer la description
// comme la sélection.
func threeMoves() []core.Move {
	return []core.Move{
		{Turn: 2, Side: core.SideFugitive, Type: core.MoveStep,
			From: core.Position{Column: 3, Row: 3}, To: core.Position{Column: 3, Row: 2}},
		{Turn: 2, Side: core.SideFugitive, Type: core.MoveExpense, Expense: core.ExpenseSilence},
		{Turn: 2, Side: core.SideFugitive, Type: core.MoveEndPhase},
	}
}

// TestReadMoveReturnsTheChosenOne vérifie la sélection par numéro.
func TestReadMoveReturnsTheChosenOne(t *testing.T) {
	coups := threeMoves()

	c, err := ReadMove(strings.NewReader("2\n"), &strings.Builder{}, coups)
	if err != nil {
		t.Fatalf("saisie refusée : %v", err)
	}
	if c != coups[1] {
		t.Errorf("coup %+v, attendu %+v", c, coups[1])
	}
}

// TestFaultyInputAskedAgain vérifie qu'une faute de frappe ne coûte pas le
// tour.
//
// Le noyau ne doit jamais voir un coup illégal : le rattrapage se fait ici,
// avant, et non par un refus qui remonterait jusqu'à lui.
func TestFaultyInputAskedAgain(t *testing.T) {
	coups := threeMoves()
	var sortie strings.Builder

	c, err := ReadMove(strings.NewReader("zéro\n99\n0\n1\n"), &sortie, coups)
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

// TestQuitAndEndOfInput vérifie les deux façons de sortir.
//
// La fin d'entrée vaut abandon plutôt qu'erreur : c'est ce qui rend le mode
// texte pilotable depuis un fichier, donc utilisable en test d'intégration.
func TestQuitAndEndOfInput(t *testing.T) {
	for nom, entree := range map[string]string{
		"q tapé":         "q\n",
		"entrée épuisée": "",
	} {
		t.Run(nom, func(t *testing.T) {
			_, err := ReadMove(strings.NewReader(entree), &strings.Builder{}, threeMoves())
			if !errors.Is(err, ErrQuit) {
				t.Errorf("erreur %v, attendu ErrQuit", err)
			}
		})
	}
}

// TestDescribeEveryMoveType vérifie qu'aucun coup ne s'affiche en jargon.
//
// La list est ce sur quoi le joueur choisit : un type brut y serait illisible,
// et un coup mal décrit se joue par erreur.
func TestDescribeEveryMoveType(t *testing.T) {
	cas := []struct {
		coup    core.Move
		attendu string
	}{
		{core.Move{Type: core.MovePlace, Side: core.SideFugitive, Zone: 4}, "sceller la zone 4"},
		{core.Move{Type: core.MovePlace, Side: core.SideInspectors, Piece: 1,
			To: core.Position{Column: 2, Row: 6}}, "placer B en (2,6)"},
		{core.Move{Type: core.MoveStep, Side: core.SideInspectors, Piece: 0,
			From: core.Position{Column: 1, Row: 1}, To: core.Position{Column: 2, Row: 1}}, "déplacer A"},
		{core.Move{Type: core.MoveAbility, Piece: 2, Ability: "tracker"}, "capacité tracker de C"},
		{core.Move{Type: core.MoveExpense, Expense: core.ExpenseDecoy}, "dépenser decoy"},
		{core.Move{Type: core.MoveChangeZone, Zone: 1}, "resceller vers la zone 1"},
		{core.Move{Type: core.MovePass}, "passer"},
		{core.Move{Type: core.MoveEndPhase}, "rendre la main"},
	}

	for _, c := range cas {
		if rendu := Describe(c.coup); !strings.Contains(rendu, c.attendu) {
			t.Errorf("%s rendu %q, attendu qu'il contienne %q", c.coup.Type, rendu, c.attendu)
		}
	}
}

// TestDirectionNamed vérifie que les huit sens se lisent.
//
// Choisir parmi huit voisines qui ne diffèrent que d'une unité se fait au nom
// de la direction, pas en comparant des coordonnées.
func TestDirectionNamed(t *testing.T) {
	depart := core.Position{Column: 5, Row: 5}

	for d, attendu := range map[core.Direction]string{
		core.Nord:      "nord",
		core.Est:       "est",
		core.SudOuest:  "sud-ouest",
		core.NordOuest: "nord-ouest",
	} {
		coup := core.Move{Type: core.MoveStep, From: depart, To: depart.Step(d)}
		if rendu := Describe(coup); !strings.HasSuffix(rendu, attendu) {
			t.Errorf("direction %d rendue %q, attendu qu'elle finisse par %q", d, rendu, attendu)
		}
	}
}

// TestMovesNumbered vérifie que la liste affichée et la saisie s'accordent sur
// le premier numéro.
func TestMovesNumbered(t *testing.T) {
	list := Moves(threeMoves())

	if !strings.Contains(list, "  1. ") || !strings.Contains(list, "  3. ") {
		t.Errorf("numérotation absente :\n%s", list)
	}
	if strings.Contains(list, "  0. ") {
		t.Error("la liste commence à zéro, la saisie attend un")
	}
}
