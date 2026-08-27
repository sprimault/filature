// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/core"
)

// recopiesEnBloc énumère les types que ViewFor recopie entiers, avec les champs
// qu'ils avaient le jour où la recopie a été relue.
//
// Ajouter un champ à l'un d'eux le rend visible des deux camps sans que
// personne ne l'ait décidé : la recopie ne choisit pas, elle prend tout. C'est
// pourquoi ce qui dépend de la position du fugitif vit dans Game et non dans
// Inspector — voir LastContacts.
var recopiesEnBloc = map[string][]string{
	"Inspector":  {"Position", "Ability", "AbilityUsed", "StepsTaken"},
	"Zone":       {"Number", "Cells", "Closed"},
	"Shelter":    {"Number", "Cells"},
	"CrimeScene": {"Position", "Turn"},
}

// TestBulkCopiedTypesAreDeclared épingle les champs des types que ViewFor
// recopie sans les filtrer.
//
// L'invariant de la vue filtrée est tenu au niveau de Game, où chaque champ est
// copié explicitement. Un cran plus bas, il ne l'est pas : quatre structures
// partent entières, et le commentaire qui garde cette règle en annonçait une
// seule — il disait vrai le jour où il a été écrit, les lieux et les scènes de
// crime sont arrivés depuis.
//
// Un commentaire ne se relit pas au moment où on ajoute un champ ailleurs. Ce
// test, si : il échoue à l'ajout, ce qui force la décision quand elle se prend
// plutôt que de la laisser passer.
//
// Le remède quand il tombe n'est pas d'allonger la liste sans réfléchir. Deux
// questions : ce champ doit-il partir aux deux camps, et dépend-il de ce que
// l'un des deux ignore ? Si oui, il vit dans Game, hors de ce qui se recopie.
func TestBulkCopiedTypesAreDeclared(t *testing.T) {
	types := map[string]reflect.Type{
		"Inspector":  reflect.TypeOf(core.Inspector{}),
		"Zone":       reflect.TypeOf(core.Zone{}),
		"Shelter":    reflect.TypeOf(core.Shelter{}),
		"CrimeScene": reflect.TypeOf(core.CrimeScene{}),
	}

	for nom, attendus := range recopiesEnBloc {
		typ, connu := types[nom]
		if !connu {
			t.Errorf("%s est déclaré recopié en bloc mais n'a pas de type en face", nom)
			continue
		}

		var champs []string
		for i := 0; i < typ.NumField(); i++ {
			champs = append(champs, typ.Field(i).Name)
		}

		if strings.Join(champs, ", ") != strings.Join(attendus, ", ") {
			t.Errorf("%s porte les champs [%s], la recopie en bloc en déclare [%s] — "+
				"tout champ ajouté part aux deux camps, décider avant de compléter la liste",
				nom, strings.Join(champs, ", "), strings.Join(attendus, ", "))
		}
	}
}
