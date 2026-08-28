// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/core"
)

// TestVocabularyTablesMatchTheCode rapproche les tableaux de primitives du
// vocabulaire d'effets des types que le noyau applique.
//
// C'est un contrat public de dix-huit primitives que rien ne lisait : les
// énumérations du schéma étaient gardées, le document non. Il a annoncé un
// paramètre « zone » sur trois primitives, quand la zone vient du contexte du
// coup et qu'un manifeste qui l'écrirait serait refusé pour champ inconnu ; et
// un « value » sur step, que place ignore, sa signature ne prenant même pas
// l'effet. Dans un cas l'auteur voit un refus, dans l'autre rien du tout.
//
// Le contrôle porte sur ce qui est vérifiable — les noms de primitives et
// l'existence des champs cités —, jamais sur la colonne d'effet, qui est de la
// prose.
//
// **Il ne voit pas le second cas.** « value » est un champ réel, simplement
// ignoré par step : dire qu'une primitive lit un champ qu'elle ne consulte pas
// demanderait de suivre le flot dans ApplyOneEffect, et ce serait réimplémenter
// le noyau pour le relire. Ce reste-là se vérifie à la lecture, comme la
// colonne d'effet.
func TestVocabularyTablesMatchTheCode(t *testing.T) {
	lignes := tableauxDuVocabulaire(t)
	if len(lignes) == 0 {
		t.Fatal("aucun tableau de primitives trouvé : le contrôle ne vérifie plus rien")
	}

	var nommees []string
	champs := champsDUnEffet(t)

	for primitive, parametres := range lignes {
		nommees = append(nommees, primitive)
		for _, p := range parametres {
			if !slices.Contains(champs, p) {
				t.Errorf("%s annonce un paramètre %q, qui n'est pas un champ d'un effet : "+
					"un manifeste qui l'écrirait serait refusé pour champ inconnu",
					primitive, p)
			}
		}
	}

	var declarees []string
	for _, e := range core.EffectTypes() {
		declarees = append(declarees, string(e))
	}
	slices.Sort(declarees)
	slices.Sort(nommees)
	comparerListes(t, "les primitives du vocabulaire", declarees, nommees)
}

// tableauxDuVocabulaire rend, par primitive, les paramètres que le document lui
// prête.
//
// Seuls les tableaux « Type | Paramètres | Effet » sont lus : celui des champs
// d'un differer et celui des cibles décrivent autre chose et ont leurs propres
// colonnes.
func tableauxDuVocabulaire(t *testing.T) map[string][]string {
	t.Helper()

	contenu, err := os.ReadFile(filepath.Join(racine, "docs", "vocabulaire-effets.md"))
	if err != nil {
		t.Fatal(err)
	}

	lignes := map[string][]string{}
	dansUnTableau := false
	for _, ligne := range strings.Split(string(contenu), "\n") {
		switch {
		case strings.HasPrefix(ligne, "| Type | Paramètres | Effet |"):
			dansUnTableau = true
			continue
		case !strings.HasPrefix(ligne, "|"):
			dansUnTableau = false
			continue
		case !dansUnTableau || strings.HasPrefix(ligne, "|---"):
			continue
		}

		cellules := strings.Split(ligne, "|")
		if len(cellules) < 4 {
			continue
		}
		primitive := entreAccents(cellules[1])
		if len(primitive) != 1 {
			t.Errorf("ligne de tableau sans primitive unique : %s", ligne)
			continue
		}
		lignes[primitive[0]] = entreAccents(cellules[2])
	}
	return lignes
}

// entreAccents extrait les identifiants entre accents graves d'une cellule.
func entreAccents(cellule string) []string {
	var noms []string
	for i, part := range strings.Split(cellule, "`") {
		if i%2 == 1 {
			noms = append(noms, part)
		}
	}
	return noms
}

// champsDUnEffet rend les tags TOML de core.Effect, tels qu'un manifeste les
// écrit.
//
// Lus par réflexion et non recopiés : une liste écrite à la main serait un
// second contrat à tenir d'accord avec le premier, ce que ce test existe
// précisément pour éviter.
func champsDUnEffet(t *testing.T) []string {
	t.Helper()

	typ := reflect.TypeOf(core.Effect{})
	champs := make([]string, 0, typ.NumField())
	for i := range typ.NumField() {
		if tag := typ.Field(i).Tag.Get("toml"); tag != "" && tag != "-" {
			champs = append(champs, tag)
		}
	}

	// Un garde-fou contre un tag renommé sans que personne le remarque : sans
	// lui, une liste vide rendrait le contrôle muet.
	if len(champs) < 4 {
		t.Fatalf("%d champs trouvés sur core.Effect : la lecture par réflexion a cessé de marcher", len(champs))
	}
	return champs
}
