// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"strings"
	"testing"
)

// TestPresetsArePlayable vérifie que chaque préréglage livré produit un
// plateau.
//
// Un préréglage qu'on propose dans un sélecteur et qui échoue à la génération
// serait pire qu'un préréglage absent : le joueur le choisit, et la partie ne
// démarre pas.
func TestPresetsArePlayable(t *testing.T) {
	for _, p := range Presets() {
		t.Run(p.Key, func(t *testing.T) {
			if err := p.Settings.Validate(); err != nil {
				t.Fatalf("paramètres refusés : %v", err)
			}

			// Plusieurs graines : un préréglage qui ne marcherait qu'une fois
			// sur deux n'est pas jouable.
			for graine := int64(1); graine <= 30; graine++ {
				b, _, err := Generate(graine, p.Settings)
				if err != nil {
					t.Fatalf("graine %d : %v", graine, err)
				}
				if len(b.Zones()) != p.Settings.Zones {
					t.Fatalf("graine %d : %d zones, attendu %d",
						graine, len(b.Zones()), p.Settings.Zones)
				}
			}
		})
	}
}

// TestPresetsCoverThreeSizes vérifie ce que docs/regles.md §11
// annonce : des préréglages 21, 31 et 41.
func TestPresetsCoverThreeSizes(t *testing.T) {
	vus := map[int]bool{}
	for _, p := range Presets() {
		vus[p.Settings.Size] = true
	}

	for _, cote := range []int{21, 31, 41} {
		if !vus[cote] {
			t.Errorf("aucun préréglage en %d×%d", cote, cote)
		}
	}
}

// TestPresetsOrderedAndDistinct vérifie qu'un sélecteur les affiche du
// plus petit au plus grand, sans doublon.
func TestPresetsOrderedAndDistinct(t *testing.T) {
	list := Presets()
	cles := map[string]bool{}

	for i, p := range list {
		if p.Key == "" {
			t.Errorf("préréglage %d sans clé", i)
		}
		if cles[p.Key] {
			t.Errorf("clé %s en double", p.Key)
		}
		cles[p.Key] = true

		if i > 0 && p.Settings.Size <= list[i-1].Settings.Size {
			t.Errorf("%s n'est pas plus grand que %s", p.Key, list[i-1].Key)
		}
	}
}

// TestPresetRangeAndLengthFollowSize vérifie que les deux valeurs
// dérivées le restent.
//
// Sur un petit plateau, une portée de 8 verrait presque tout le terrain et
// quarante tours laisseraient au fugitif le temps d'en faire trois fois le
// tour : les figer viderait les petits préréglages de leur intérêt.
func TestPresetRangeAndLengthFollowSize(t *testing.T) {
	for _, p := range Presets() {
		cote := p.Settings.Size

		if attendue := max(MinRange, cote/5); p.Settings.Range != attendue {
			t.Errorf("%s : portée %d, attendu %d", p.Key, p.Settings.Range, attendue)
		}
		// « Environ le côté », dit la règle, et non exactement : la Ville garde
		// ses quarante tours historiques sur un plateau de quarante et un.
		if ecart := abs(p.Settings.Turns - cote); ecart > 2 {
			t.Errorf("%s : %d tours pour un côté de %d", p.Key, p.Settings.Turns, cote)
		}
		if p.Settings.StranglingStart >= p.Settings.Turns {
			t.Errorf("%s : étranglement au tour %d sur %d",
				p.Key, p.Settings.StranglingStart, p.Settings.Turns)
		}
		if attendu := cote / 4; p.Settings.CentreRadius != attendu {
			t.Errorf("%s : noyau de rayon %d, attendu %d",
				p.Key, p.Settings.CentreRadius, attendu)
		}
	}
}

// TestStranglingFitsInTheEndgame vérifie que les fermetures tiennent dans ce
// qui reste de la partie, sur chaque préréglage.
//
// Une période figée épuiserait la pression à mi-chemin sur une longue partie et
// déborderait la fin sur une courte : l'entonnoir doit pousser jusqu'au bout
// sans mordre sur les tours qui restent pour conclure.
func TestStranglingFitsInTheEndgame(t *testing.T) {
	for _, p := range Presets() {
		s := p.Settings
		fermetures := s.Zones - s.ZonesLeftOpen
		if fermetures < 1 {
			t.Fatalf("%s : %d zones pour %d laissées ouvertes", p.Key, s.Zones, s.ZonesLeftOpen)
		}

		derniere := s.StranglingStart + (fermetures-1)*s.StranglingPeriod
		if derniere > s.Turns-2 {
			t.Errorf("%s : dernière fermeture au tour %d pour %d tours de jeu, "+
				"il n'en reste pas assez pour conclure", p.Key, derniere, s.Turns)
		}
		if derniere <= s.StranglingStart {
			t.Errorf("%s : les %d fermetures tombent toutes au tour %d",
				p.Key, fermetures, derniere)
		}
	}
}

// TestValidateRejectsOversizedCentre vérifie qu'un noyau trop large est refusé
// à la lecture du réglage plutôt qu'au premier placement impossible.
//
// Le cas n'arrive pas des préréglages, qui dérivent le rayon du côté, mais d'un
// réglage écrit à la main — ou d'un appelant qui part de DefaultSettings et n'y
// change que la taille.
func TestValidateRejectsOversizedCentre(t *testing.T) {
	p := SettingsForSize(21)
	p.CentreRadius = p.Size/2 - ZoneSize + 1

	err := p.Validate()
	if err == nil {
		t.Fatal("noyau plus large que la couronne accepté")
	}
	if !strings.Contains(err.Error(), "noyau") {
		t.Errorf("message %q, attendu qu'il nomme le noyau", err)
	}
}

// TestCentreLeavesRoomForInspectors vérifie que la couronne hors noyau existe
// dans chaque préréglage.
//
// C'est devenu une condition de jouabilité et non d'esthétique : les
// inspecteurs s'y placent, et un noyau qui mange le plateau ne leur laisserait
// nulle part où se poser.
func TestCentreLeavesRoomForInspectors(t *testing.T) {
	for _, p := range Presets() {
		couronne := p.Settings.Size/2 - p.Settings.CentreRadius
		if couronne < ZoneSize {
			t.Errorf("%s : couronne de %d cases hors noyau, une zone en demande %d",
				p.Key, couronne, ZoneSize)
		}
	}
}

// TestPresetByKey vérifie la recherche, et qu'une clé inconnue le dit.
func TestPresetByKey(t *testing.T) {
	p, trouve := PresetByKey("city")
	if !trouve {
		t.Fatal("le préréglage par défaut est introuvable")
	}
	if p.Settings.Size != DefaultSettings().Size {
		t.Error("« ville » ne correspond pas aux paramètres par défaut")
	}

	if _, trouve := PresetByKey("inexistant"); trouve {
		t.Error("une clé inconnue est acceptée")
	}
}

// TestDefaultPresetIsTheCity vérifie que le défaut du jeu figure bien parmi
// les préréglages proposés, plutôt que d'être un quatrième jeu de valeurs.
func TestDefaultPresetIsTheCity(t *testing.T) {
	defaut := DefaultSettings()

	for _, p := range Presets() {
		if p.Settings == defaut {
			return
		}
	}
	t.Error("les paramètres par défaut ne correspondent à aucun préréglage")
}
