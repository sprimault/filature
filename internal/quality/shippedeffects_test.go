// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/sprimault/filature/internal/core"
)

// manifesteLivre est ce que plugins/base déclare, dans les types du noyau.
type manifesteLivre struct {
	Ability map[string]core.Ability `toml:"ability"`
	Expense map[string]core.Ability `toml:"expense"`
	Mode    map[string]core.Mode    `toml:"mode"`
}

// plateauPlat est un terrain dégagé, de quoi appliquer un effet sans dépendre
// de la génération.
//
// Le test vit ici et non dans internal/noyau parce que celui-ci n'a aucune
// dépendance disque : c'est sa définition, et lire un TOML l'enfreindrait.
type plateauPlat struct{}

// IsStreet accepte tout le carré de vingt et une cases de côté.
func (plateauPlat) IsStreet(p core.Position) bool {
	return p.Column >= 0 && p.Row >= 0 && p.Column < 21 && p.Row < 21
}

// Zones renvoie six zones, comme la règle standard.
func (plateauPlat) Zones() []core.Zone {
	zones := make([]core.Zone, 6)
	for i := range zones {
		zones[i] = core.Zone{Number: i}
	}
	return zones
}

// Shelters reste vide : ce sont les effets livrés qu'on exerce ici.
func (plateauPlat) Shelters() []core.Shelter { return nil }

// Seed est figée : ce test ne tire rien au sort.
func (plateauPlat) Seed() int64 { return 1 }

// Sight reste vide : ce test applique les effets livrés, il ne regarde rien.
func (plateauPlat) Sight(p core.Position, portee int) []core.Position { return nil }

// CellsWithin énumère le carré autour du centre.
func (b plateauPlat) CellsWithin(centre core.Position, rayon int) []core.Position {
	var cases []core.Position
	for ligne := centre.Row - rayon; ligne <= centre.Row+rayon; ligne++ {
		for colonne := centre.Column - rayon; colonne <= centre.Column+rayon; colonne++ {
			if p := (core.Position{Column: colonne, Row: ligne}); b.IsStreet(p) {
				cases = append(cases, p)
			}
		}
	}
	return cases
}

// TestShippedEffectsApply vérifie que tout effet déclaré par le contenu
// livré s'applique sans erreur, dans le contexte où le jeu le déclenche.
//
// La résolution de fin de tour avale l'erreur d'un effet différé qui échoue :
// interrompre la partie de quelqu'un sur un plugin fautif serait pire. Ce test
// est ce qui rend ce silence supportable pour le contenu livré — sans lui,
// ajouter une capacité au manifeste et se tromper de cible ne se verrait qu'en
// jouant, et seulement si quelqu'un remarquait que rien ne s'était passé.
//
// Sa portée s'arrête là : il éprouve les effets livrés dans un contexte
// plausible, pas toutes les combinaisons qu'un plugin tiers pourrait former.
// C'est la validation au chargement, à l'étape 8, qui couvrira celles-là.
func TestShippedEffectsApply(t *testing.T) {
	var livre manifesteLivre
	chemin := filepath.Join(racine, "plugins", "base", "manifest.toml")
	if _, err := toml.DecodeFile(chemin, &livre); err != nil {
		t.Fatalf("lecture du manifeste : %v", err)
	}
	if len(livre.Ability) == 0 || len(livre.Expense) == 0 || len(livre.Mode) == 0 {
		t.Fatal("le manifeste livré ne déclare plus capacités, dépenses et modes")
	}

	for cle, c := range livre.Ability {
		t.Run("capacite/"+cle, func(t *testing.T) {
			checkEffects(t, c.Effects, inspectorContext())
		})
	}
	for cle, d := range livre.Expense {
		t.Run("depense/"+cle, func(t *testing.T) {
			checkEffects(t, d.Effects, fugitiveContext())
		})
	}
	for cle, m := range livre.Mode {
		t.Run("mode/"+cle, func(t *testing.T) {
			// Un mode est déclenché par le jeu : son contexte ne désigne
			// aucun pion, seulement la zone visée.
			checkEffects(t, m.Effects, core.EffectContext{Side: core.SideInspectors, Zone: 3})
		})
	}
}

// inspectorContext est ce dont dispose une capacité au déclenchement.
func inspectorContext() core.EffectContext {
	return core.EffectContext{
		Side:      core.SideInspectors,
		Piece:     0,
		AutrePion: 1,
		Case:      core.Position{Column: 5, Row: 5},
		Zone:      2,
	}
}

// fugitiveContext est ce dont dispose une dépense.
func fugitiveContext() core.EffectContext {
	return core.EffectContext{
		Side: core.SideFugitive,
		Case: core.Position{Column: 10, Row: 10},
		Zone: 4,
	}
}

// checkEffects applique une suite d'effets et défait tout, en descendant dans
// les effets différés que le jeu appliquera à l'échéance.
func checkEffects(t *testing.T, effets []core.Effect, ctx core.EffectContext) {
	t.Helper()
	for _, e := range effets {
		p := testGame()

		defaire, err := p.ApplyOneEffect(e, ctx)
		if err != nil {
			t.Errorf("%s refusé : %v", e.Type, err)
			continue
		}
		if defaire == nil {
			t.Errorf("%s ne renvoie pas d'annulation", e.Type)
			continue
		}
		defaire()

		// Les effets d'un differer ne s'appliquent pas tout de suite : ils
		// partent en file, et c'est à l'échéance qu'ils échoueraient.
		if len(e.Then) > 0 {
			checkEffects(t, e.Then, ctx)
		}
	}
}

// testGame monte une partie plausible : cinq inspecteurs posés, un fugitif
// au centre, de quoi qu'aucun effet ne manque de cible.
func testGame() *core.Game {
	p := &core.Game{
		Settings: core.DefaultSettings(),
		Board:    plateauPlat{},
		Turn:     5,
		Phase:    core.PhaseInspectors,
		Fugitive: core.Fugitive{
			Position: core.Position{Column: 10, Row: 10},
			Stamina:  10,
		},
	}
	p.Settings.Size = 21
	for i := 0; i < p.Settings.Inspectors; i++ {
		p.Inspectors = append(p.Inspectors, core.Inspector{
			Position: core.Position{Column: i, Row: 0},
		})
	}
	return p
}
