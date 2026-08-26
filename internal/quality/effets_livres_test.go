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
	Capacite map[string]core.Capacite `toml:"capacite"`
	Depense  map[string]core.Capacite `toml:"depense"`
	Mode     map[string]core.Mode     `toml:"mode"`
}

// plateauPlat est un terrain dégagé, de quoi appliquer un effet sans dépendre
// de la génération.
//
// Le test vit ici et non dans internal/noyau parce que celui-ci n'a aucune
// dépendance disque : c'est sa définition, et lire un TOML l'enfreindrait.
type plateauPlat struct{}

// EstRue accepte tout le carré de vingt et une cases de côté.
func (plateauPlat) EstRue(p core.Position) bool {
	return p.Colonne >= 0 && p.Ligne >= 0 && p.Colonne < 21 && p.Ligne < 21
}

// Zones renvoie six zones, comme la règle standard.
func (plateauPlat) Zones() []core.Zone {
	zones := make([]core.Zone, 6)
	for i := range zones {
		zones[i] = core.Zone{Numero: i}
	}
	return zones
}

// Graine est figée : ce test ne tire rien au sort.
func (plateauPlat) Graine() int64 { return 1 }

// Vision reste vide : ce test applique les effets livrés, il ne regarde rien.
func (plateauPlat) Vision(p core.Position, portee int) []core.Position { return nil }

// CasesDans énumère le carré autour du centre.
func (b plateauPlat) CasesDans(centre core.Position, rayon int) []core.Position {
	var cases []core.Position
	for ligne := centre.Ligne - rayon; ligne <= centre.Ligne+rayon; ligne++ {
		for colonne := centre.Colonne - rayon; colonne <= centre.Colonne+rayon; colonne++ {
			if p := (core.Position{Colonne: colonne, Ligne: ligne}); b.EstRue(p) {
				cases = append(cases, p)
			}
		}
	}
	return cases
}

// TestEffetsLivresSAppliquent vérifie que tout effet déclaré par le contenu
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
func TestEffetsLivresSAppliquent(t *testing.T) {
	var livre manifesteLivre
	chemin := filepath.Join(racine, "plugins", "base", "manifeste.toml")
	if _, err := toml.DecodeFile(chemin, &livre); err != nil {
		t.Fatalf("lecture du manifeste : %v", err)
	}
	if len(livre.Capacite) == 0 || len(livre.Depense) == 0 || len(livre.Mode) == 0 {
		t.Fatal("le manifeste livré ne déclare plus capacités, dépenses et modes")
	}

	for cle, c := range livre.Capacite {
		t.Run("capacite/"+cle, func(t *testing.T) {
			verifierEffets(t, c.Effets, contexteInspecteur())
		})
	}
	for cle, d := range livre.Depense {
		t.Run("depense/"+cle, func(t *testing.T) {
			verifierEffets(t, d.Effets, contexteFugitif())
		})
	}
	for cle, m := range livre.Mode {
		t.Run("mode/"+cle, func(t *testing.T) {
			// Un mode est déclenché par le jeu : son contexte ne désigne
			// aucun pion, seulement la zone visée.
			verifierEffets(t, m.Effets, core.Contexte{Acteur: core.CampInspecteurs, Zone: 3})
		})
	}
}

// contexteInspecteur est ce dont dispose une capacité au déclenchement.
func contexteInspecteur() core.Contexte {
	return core.Contexte{
		Acteur:    core.CampInspecteurs,
		Pion:      0,
		AutrePion: 1,
		Case:      core.Position{Colonne: 5, Ligne: 5},
		Zone:      2,
	}
}

// contexteFugitif est ce dont dispose une dépense.
func contexteFugitif() core.Contexte {
	return core.Contexte{
		Acteur: core.CampFugitif,
		Case:   core.Position{Colonne: 10, Ligne: 10},
		Zone:   4,
	}
}

// verifierEffets applique une suite d'effets et défait tout, en descendant dans
// les effets différés que le jeu appliquera à l'échéance.
func verifierEffets(t *testing.T, effets []core.Effet, ctx core.Contexte) {
	t.Helper()
	for _, e := range effets {
		p := partieDEssai()

		defaire, err := p.Appliquer1Effet(e, ctx)
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
		if len(e.Puis) > 0 {
			verifierEffets(t, e.Puis, ctx)
		}
	}
}

// partieDEssai monte une partie plausible : cinq inspecteurs posés, un fugitif
// au centre, de quoi qu'aucun effet ne manque de cible.
func partieDEssai() *core.Partie {
	p := &core.Partie{
		Parametres: core.ParametresDefaut(),
		Plateau:    plateauPlat{},
		Tour:       5,
		Phase:      core.PhaseInspecteurs,
		Fugitif: core.Fugitif{
			Position:   core.Position{Colonne: 10, Ligne: 10},
			Resistance: 10,
		},
	}
	p.Parametres.Cote = 21
	for i := 0; i < p.Parametres.Inspecteurs; i++ {
		p.Inspecteurs = append(p.Inspecteurs, core.Inspecteur{
			Position: core.Position{Colonne: i, Ligne: 0},
		})
	}
	return p
}
