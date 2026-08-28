// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/sprimault/filature/internal/core"
	"github.com/sprimault/filature/internal/loader"
	"github.com/sprimault/filature/plugins"
)

// dictionnaire est la forme d'un language.toml.
type dictionnaire struct {
	Libelle map[string]string `toml:"label"`
}

// langueDeBase est le plugin qui porte le français, langue de repli.
const langueDeBase = "base"

// TestShippedLanguagesCoverKeys vérifie que chaque langue livrée traduit
// toutes les clés du français, et n'en invente aucune.
//
// Le repli sur le français rend une traduction partielle jouable, donc muette :
// rien à l'écran ne distingue un libellé oublié d'un libellé volontairement
// identique. C'est exactement ce qu'un test doit attraper, puisque personne ne
// le verra en jouant.
func TestShippedLanguagesCoverKeys(t *testing.T) {
	base := lireDictionnaire(t, langueDeBase)
	if len(base) == 0 {
		t.Fatal("le dictionnaire de repli est vide")
	}

	dossiers, err := os.ReadDir(filepath.Join(racine, "plugins"))
	if err != nil {
		t.Fatal(err)
	}

	for _, d := range dossiers {
		if !d.IsDir() || d.Name() == langueDeBase {
			continue
		}
		chemin := filepath.Join(racine, "plugins", d.Name(), "language.toml")
		if _, err := os.Stat(chemin); err != nil {
			continue
		}

		t.Run(d.Name(), func(t *testing.T) {
			traduction := lireDictionnaire(t, d.Name())
			for _, cle := range missing(base, traduction) {
				t.Errorf("clé non traduite : %s", cle)
			}
			for _, cle := range missing(traduction, base) {
				t.Errorf("clé inconnue du français : %s", cle)
			}
		})
	}
}

// lireDictionnaire charge les libellés d'un plugin livré.
func lireDictionnaire(t *testing.T, plugin string) map[string]string {
	t.Helper()
	var d dictionnaire
	if _, err := toml.DecodeFile(filepath.Join(racine, "plugins", plugin, "language.toml"), &d); err != nil {
		t.Fatalf("lecture de %s : %v", plugin, err)
	}
	return d.Libelle
}

// missing renvoie les clés du premier dictionnaire absentes du second,
// triées pour que deux exécutions signalent les mêmes défauts dans le même
// ordre.
func missing(attendu, obtenu map[string]string) []string {
	var cles []string
	for cle := range attendu {
		if _, ok := obtenu[cle]; !ok {
			cles = append(cles, cle)
		}
	}
	sort.Strings(cles)
	return cles
}

// TestPresetKeysHaveALabel vérifie que chaque préréglage a de quoi s'afficher.
//
// La clé d'un préréglage est un identifiant ; son libellé vient du
// dictionnaire, sous « preset_<cle> ». Les deux ne s'étaient jamais rencontrés :
// les dictionnaires déclaraient preset_district quand Presets rendait
// « quartier ». Rien ne le disait, l'interface étant l'étape 7 — et le repli sur
// le français n'aurait pas joué non plus, la clé manquant des deux côtés.
//
// Le contrôle porte sur la langue de repli seulement : TestShippedLanguagesCoverKeys
// tient les autres alignées sur elle.
func TestPresetKeysHaveALabel(t *testing.T) {
	base := lireDictionnaire(t, langueDeBase)

	for _, p := range core.Presets() {
		cle := "preset_" + p.Key
		if _, present := base[cle]; !present {
			t.Errorf("le préréglage %q n'a pas de libellé : %s manque du dictionnaire de repli",
				p.Key, cle)
		}
	}
}

// TestShippedKeysMatchWhatTheGameProduces rapproche le dictionnaire de repli de
// ce que le contenu livré déclare et de ce que le noyau produit.
//
// Dans les deux sens, et c'est ce qui compte : une clé manquante laisse une
// mécanique sans libellé, une clé de trop survit à la mécanique qu'elle
// nommait. Les deux étaient là — le leurre et la capture n'avaient rien à
// afficher, et « expense_murder » nommait une dépense retirée du jeu.
//
// TestShippedLanguagesCoverKeys ne pouvait pas les voir : il aligne les autres
// langues sur celle-ci, et l'anglais portait exactement le même trou. Un
// contrôle qui compare deux copies ne dit rien de ce qu'elles copient.
func TestShippedKeysMatchWhatTheGameProduces(t *testing.T) {
	base := lireDictionnaire(t, langueDeBase)
	registre, _, err := loader.Load(plugins.Shipped(), "")
	if err != nil {
		t.Fatalf("chargement du contenu livré : %v", err)
	}

	cas := []struct {
		quoi     string
		prefixe  string
		attendus []string
	}{
		{"dépense", "expense_", triees(registre.Expenses)},
		{"capacité", "piece_", triees(registre.Abilities)},
		{"motif de fin", "end_", motifsDeFin()},
	}

	for _, c := range cas {
		libelles := map[string]bool{}
		for cle := range base {
			if apres, coupe := strings.CutPrefix(cle, c.prefixe); coupe {
				libelles[apres] = true
			}
		}

		for _, attendu := range c.attendus {
			if !libelles[attendu] {
				t.Errorf("la %s %q n'a pas de libellé : %s%s manque du dictionnaire de repli",
					c.quoi, attendu, c.prefixe, attendu)
			}
			delete(libelles, attendu)
		}
		for reste := range libelles {
			t.Errorf("%s%s nomme une %s que le jeu ne produit plus",
				c.prefixe, reste, c.quoi)
		}
	}
}

// motifsDeFin rend les motifs que le jeu de base produit.
//
// OutcomePlugin en est écarté : il vient d'un plugin de règles, qui livre son
// libellé avec sa condition de victoire. Le lui donner ici reviendrait à
// nommer d'avance ce qu'on ne connaît pas.
func motifsDeFin() []string {
	motifs := []string{
		core.OutcomeExtraction, core.OutcomeCaptured, core.OutcomeStaminaSpent,
		core.OutcomeCornered, core.OutcomeTimeUp,
	}
	slices.Sort(motifs)
	return motifs
}

// triees rend les clés d'une table dans un ordre stable.
func triees[K ~string, V any](table map[K]V) []string {
	cles := make([]string, 0, len(table))
	for cle := range table {
		cles = append(cles, string(cle))
	}
	slices.Sort(cles)
	return cles
}
