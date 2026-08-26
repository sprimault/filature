// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/BurntSushi/toml"
)

// dictionnaire est la forme d'un langue.toml.
type dictionnaire struct {
	Libelle map[string]string `toml:"libelle"`
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
		chemin := filepath.Join(racine, "plugins", d.Name(), "langue.toml")
		if _, err := os.Stat(chemin); err != nil {
			continue
		}

		t.Run(d.Name(), func(t *testing.T) {
			traduction := lireDictionnaire(t, d.Name())
			for _, cle := range manquantes(base, traduction) {
				t.Errorf("clé non traduite : %s", cle)
			}
			for _, cle := range manquantes(traduction, base) {
				t.Errorf("clé inconnue du français : %s", cle)
			}
		})
	}
}

// lireDictionnaire charge les libellés d'un plugin livré.
func lireDictionnaire(t *testing.T, plugin string) map[string]string {
	t.Helper()
	var d dictionnaire
	if _, err := toml.DecodeFile(filepath.Join(racine, "plugins", plugin, "langue.toml"), &d); err != nil {
		t.Fatalf("lecture de %s : %v", plugin, err)
	}
	return d.Libelle
}

// manquantes renvoie les clés du premier dictionnaire absentes du second,
// triées pour que deux exécutions signalent les mêmes défauts dans le même
// ordre.
func manquantes(attendu, obtenu map[string]string) []string {
	var cles []string
	for cle := range attendu {
		if _, ok := obtenu[cle]; !ok {
			cles = append(cles, cle)
		}
	}
	sort.Strings(cles)
	return cles
}
