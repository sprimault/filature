// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sprimault/evasion/internal/core"
)

// TestFugitiveSeesWhatOnlyHeCanKnow vérifie que la vue du fugitif porte le
// compte de ses dépenses plafonnées et le leurre qu'il vient d'armer.
//
// Les deux vivaient dans Game sans sortir par ViewFor, alors que la godoc de
// celle-ci affirme deux fois que rien n'est caché au fugitif. Aucun des deux ne
// se recalcule depuis la vue : LegalMoves cesse de proposer une dépense épuisée
// comme il cesse de proposer une dépense trop chère, et deux leurres armés sur
// des cases différentes donnaient des vues identiques octet pour octet.
func TestFugitiveSeesWhatOnlyHeCanKnow(t *testing.T) {
	partie := partieLivree(t, "district")

	vue := partie.ViewFor(core.SideFugitive)
	if len(vue.ExpenseUses) == 0 {
		t.Fatal("aucune dépense plafonnée dans la vue du fugitif, alors que le " +
			"contenu livré en déclare une")
	}
	for _, u := range vue.ExpenseUses {
		if u.Limit <= 0 {
			t.Errorf("%s: plafond de %d — un compte sans son échelle ne se planifie pas",
				u.Expense, u.Limit)
		}
	}

	// Le leurre armé : deux cases différentes doivent donner deux vues
	// différentes, sans quoi le champ ne porte rien.
	sans := serialiser(t, partie.ViewFor(core.SideFugitive))

	partie.Fugitive.Decoy = &core.Decoy{
		At:     core.Position{Column: 3, Row: 4},
		Toward: core.Position{Column: 3, Row: 5},
	}
	avec := serialiser(t, partie.ViewFor(core.SideFugitive))

	partie.Fugitive.Decoy = &core.Decoy{
		At:     core.Position{Column: 7, Row: 2},
		Toward: core.Position{Column: 6, Row: 2},
	}
	ailleurs := serialiser(t, partie.ViewFor(core.SideFugitive))

	if sans == avec {
		t.Error("un leurre armé ne change pas la vue du fugitif")
	}
	if avec == ailleurs {
		t.Error("deux leurres armés sur des cases différentes donnent la même vue")
	}
}

// TestInspectorsSeeNeitherCountNorDecoy vérifie que les deux champs ajoutés à la
// vue du fugitif n'atteignent jamais celle des inspecteurs.
//
// C'est la moitié qui compte d'un ajout à ViewFor : une omission n'affiche que
// moins, une fuite donne la partie. Le leurre porte une case que le fugitif a
// touchée, et le compte de ses dépenses dit ce qu'il a joué.
func TestInspectorsSeeNeitherCountNorDecoy(t *testing.T) {
	partie := partieLivree(t, "district")
	partie.Fugitive.Decoy = &core.Decoy{
		At:     core.Position{Column: 3, Row: 4},
		Toward: core.Position{Column: 3, Row: 5},
	}
	partie.ExpenseUses = map[core.Expense]int{"decoy": 2}

	vue := partie.ViewFor(core.SideInspectors)
	if vue.Decoy != nil {
		t.Error("la vue des inspecteurs porte le leurre armé du fugitif")
	}
	if len(vue.ExpenseUses) != 0 {
		t.Error("la vue des inspecteurs porte le compte des dépenses du fugitif")
	}

	// Le JSON plutôt que les champs : c'est lui qui part sur le réseau, et un
	// champ recopié ailleurs par mégarde s'y verrait.
	brut := serialiser(t, vue)
	for _, champ := range []string{`"decoy"`, `"expense_uses"`} {
		if strings.Contains(brut, champ) {
			t.Errorf("la vue sérialisée des inspecteurs porte %s", champ)
		}
	}
}

// serialiser rend la forme qui part sur le réseau.
func serialiser(t *testing.T, v core.View) string {
	t.Helper()

	brut, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(brut)
}
