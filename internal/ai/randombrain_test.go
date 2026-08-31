// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"errors"
	"testing"

	"github.com/sprimault/evasion/internal/core"
)

// viewWithMoves monte une vue qui n'offre que les coups donnés.
//
// Rien d'autre n'est renseigné : un cerveau qui choisirait sur autre chose que
// LegalMoves échouerait ici, ce qui est le but.
func viewWithMoves(moves ...core.Move) core.View {
	return core.View{LegalMoves: moves}
}

// TestRandomBrainStaysInTheList vérifie qu'il ne propose jamais un coup
// absent des coups légaux.
//
// C'est tout ce que le noyau attend d'un cerveau : un coup reçu se compare par
// égalité à une entrée de LegalMoves, et un coup illégal interrompt la partie.
func TestRandomBrainStaysInTheList(t *testing.T) {
	moves := []core.Move{
		{Type: core.MoveStep, To: core.Position{Column: 1}},
		{Type: core.MoveStep, To: core.Position{Column: 2}},
		{Type: core.MoveEndPhase},
	}
	v := viewWithMoves(moves...)

	a := core.NewRandom(7, "test")
	for i := 0; i < 200; i++ {
		chosen, err := RandomBrain{}.Play(v, a)
		if err != nil {
			t.Fatalf("tirage %d : %v", i, err)
		}

		found := false
		for _, c := range moves {
			if c == chosen {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("tirage %d : %+v n'est pas dans la liste", i, chosen)
		}
	}
}

// TestRandomBrainIsDeterministic est l'invariant qui compte pour ce cerveau.
//
// Deux parties lancées sur la même graine doivent produire la même suite de
// coups : sans cela, ni le rejeu du journal ni la comparaison de deux versions
// d'IA ne veulent dire quoi que ce soit.
func TestRandomBrainIsDeterministic(t *testing.T) {
	v := viewWithMoves(
		core.Move{Type: core.MoveStep, To: core.Position{Column: 1}},
		core.Move{Type: core.MoveStep, To: core.Position{Column: 2}},
		core.Move{Type: core.MovePass},
		core.Move{Type: core.MoveEndPhase},
	)

	sequence := func(seed int64) []core.Move {
		a := core.NewRandom(seed, "cerveau")
		var played []core.Move
		for i := 0; i < 30; i++ {
			c, err := RandomBrain{}.Play(v, a)
			if err != nil {
				t.Fatal(err)
			}
			played = append(played, c)
		}
		return played
	}

	if a, b := sequence(42), sequence(42); !sameSequence(a, b) {
		t.Error("deux flux de même graine donnent des suites différentes")
	}
	if a, b := sequence(42), sequence(43); sameSequence(a, b) {
		t.Error("deux graines différentes donnent la même suite")
	}
}

// sameSequence compare deux suites de coups.
func sameSequence(a, b []core.Move) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestRandomBrainInventsNothing vérifie qu'une position sans coup remonte une
// erreur au lieu d'un coup vide.
//
// Un Move nul serait accepté par la comparaison d'égalité nulle part, mais il
// traverserait la boucle jusqu'à Apply, qui le refuserait avec un message
// parlant de légalité — alors que le vrai fait est qu'il n'y avait rien à
// jouer.
func TestRandomBrainInventsNothing(t *testing.T) {
	_, err := RandomBrain{}.Play(core.View{}, core.NewRandom(1, "test"))
	if !errors.Is(err, ErrNoLegalMove) {
		t.Errorf("erreur %v, attendu ErrNoLegalMove", err)
	}
}

// TestRandomBrainCoversEveryEntry vérifie qu'aucun coup n'est
// structurellement hors d'atteinte.
//
// Un tirage biaisé — le premier toujours écarté, le dernier jamais choisi —
// ferait un adversaire prévisible, et fausserait la passe d'équilibrage qui
// s'appuiera sur lui.
func TestRandomBrainCoversEveryEntry(t *testing.T) {
	const count = 6
	var moves []core.Move
	for i := 0; i < count; i++ {
		moves = append(moves, core.Move{Type: core.MoveStep, To: core.Position{Column: i}})
	}

	seen := map[core.Move]int{}
	a := core.NewRandom(3, "couverture")
	for i := 0; i < 600; i++ {
		c, err := RandomBrain{}.Play(viewWithMoves(moves...), a)
		if err != nil {
			t.Fatal(err)
		}
		seen[c]++
	}

	for _, c := range moves {
		if seen[c] == 0 {
			t.Errorf("%+v n'a jamais été choisi sur six cents tirages", c)
		}
	}
}

// TestRandomBrainHonoursTheSignature vérifie qu'il se branche là où l'IA
// véritable se branchera.
//
// Le registre expose BrainFactory : si ce cerveau ne s'y assigne pas, la boucle
// de jeu devra le traiter à part, et l'étape 9 la fera changer.
func TestRandomBrainHonoursTheSignature(t *testing.T) {
	var factory core.BrainFactory = RandomBrain{}.Play

	c, err := factory(viewWithMoves(core.Move{Type: core.MovePass}), core.NewRandom(1, "test"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Type != core.MovePass {
		t.Errorf("coup %s, attendu %s", c.Type, core.MovePass)
	}
}
