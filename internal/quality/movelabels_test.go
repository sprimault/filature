// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"testing"

	"github.com/sprimault/evasion/internal/core"
	"github.com/sprimault/evasion/internal/text"
)

// TestEveryLegalMoveReadsApart vérifie que deux coups légaux distincts ne se
// lisent jamais pareil.
//
// La liste affichée est ce sur quoi le joueur pointe : deux lignes identiques
// pour deux coups différents lui font choisir à l'aveugle. Le cas s'est présenté
// deux fois — la capacité qui désigne une voisine, le leurre qui désigne une
// case et un sens — parce que les énumérateurs ont gagné leurs cases sans que
// l'affichage suive.
//
// Les coups viennent de LegalMoves et non d'une table : une description écrite
// à la main porte les champs que son auteur a en tête, et c'est précisément
// ceux qu'il oublie qui manquent.
func TestEveryLegalMoveReadsApart(t *testing.T) {
	partie := partieLivree(t, "district")

	vus := map[core.MoveType]int{}
	for coup := 0; coup < 200; coup++ {
		if _, fini := partie.Outcome(); fini {
			break
		}

		coups := partie.LegalMoves(campDe(partie.Phase))
		if len(coups) == 0 {
			break
		}

		libelles := make(map[string]core.Move, len(coups))
		for _, c := range coups {
			libelle := text.Describe(c)
			if autre, deja := libelles[libelle]; deja {
				t.Fatalf("deux coups distincts se lisent tous deux %q :\n  %+v\n  %+v",
					libelle, autre, c)
			}
			libelles[libelle] = c
			vus[c.Type]++
		}

		// Le coup du milieu plutôt que le premier : la liste est triée, et son
		// premier élément est souvent le même geste tour après tour.
		if err := partie.Apply(coups[len(coups)/2]); err != nil {
			t.Fatalf("coup refusé : %v", err)
		}
	}

	// Sans ce contrôle, le test resterait vert sur une partie qui n'aurait
	// rencontré ni capacité ni dépense — les deux types qui portent le défaut.
	for _, sorte := range []core.MoveType{core.MoveAbility, core.MoveExpense} {
		if vus[sorte] == 0 {
			t.Errorf("aucun coup de type %s rencontré : le test ne dit rien de lui", sorte)
		}
	}
}
