// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package qualite

import (
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/sprimault/filature/internal/noyau"
)

// manifesteLivre est ce que greffons/base déclare, dans les types du noyau.
type manifesteLivre struct {
	Capacite map[string]noyau.Capacite `toml:"capacite"`
	Depense  map[string]noyau.Capacite `toml:"depense"`
	Mode     map[string]noyau.Mode     `toml:"mode"`
}

// plateauPlat est un terrain dégagé, de quoi appliquer un effet sans dépendre
// de la génération.
//
// Le test vit ici et non dans internal/noyau parce que celui-ci n'a aucune
// dépendance disque : c'est sa définition, et lire un TOML l'enfreindrait.
type plateauPlat struct{}

// EstRue accepte tout le carré de vingt et une cases de côté.
func (plateauPlat) EstRue(p noyau.Position) bool {
	return p.Colonne >= 0 && p.Ligne >= 0 && p.Colonne < 21 && p.Ligne < 21
}

// Zones renvoie six zones, comme la règle standard.
func (plateauPlat) Zones() []noyau.Zone {
	zones := make([]noyau.Zone, 6)
	for i := range zones {
		zones[i] = noyau.Zone{Numero: i}
	}
	return zones
}

// Graine est figée : ce test ne tire rien au sort.
func (plateauPlat) Graine() int64 { return 1 }

// Vision n'est pas exercée : la table relève de l'étape 4.
func (plateauPlat) Vision(p noyau.Position, portee int) []noyau.Position { return nil }

// CasesDans énumère le carré autour du centre.
func (b plateauPlat) CasesDans(centre noyau.Position, rayon int) []noyau.Position {
	var cases []noyau.Position
	for ligne := centre.Ligne - rayon; ligne <= centre.Ligne+rayon; ligne++ {
		for colonne := centre.Colonne - rayon; colonne <= centre.Colonne+rayon; colonne++ {
			if p := (noyau.Position{Colonne: colonne, Ligne: ligne}); b.EstRue(p) {
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
// interrompre la partie de quelqu'un sur un greffon fautif serait pire. Ce test
// est ce qui rend ce silence supportable pour le contenu livré — sans lui,
// ajouter une capacité au manifeste et se tromper de cible ne se verrait qu'en
// jouant, et seulement si quelqu'un remarquait que rien ne s'était passé.
//
// Sa portée s'arrête là : il éprouve les effets livrés dans un contexte
// plausible, pas toutes les combinaisons qu'un greffon tiers pourrait former.
// C'est la validation au chargement, à l'étape 8, qui couvrira celles-là.
func TestEffetsLivresSAppliquent(t *testing.T) {
	var livre manifesteLivre
	chemin := filepath.Join(racine, "greffons", "base", "manifeste.toml")
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
			verifierEffets(t, m.Effets, noyau.Contexte{Acteur: noyau.CampInspecteurs, Zone: 3})
		})
	}
}

// contexteInspecteur est ce dont dispose une capacité au déclenchement.
func contexteInspecteur() noyau.Contexte {
	return noyau.Contexte{
		Acteur:    noyau.CampInspecteurs,
		Pion:      0,
		AutrePion: 1,
		Case:      noyau.Position{Colonne: 5, Ligne: 5},
		Zone:      2,
	}
}

// contexteFugitif est ce dont dispose une dépense.
func contexteFugitif() noyau.Contexte {
	return noyau.Contexte{
		Acteur: noyau.CampFugitif,
		Case:   noyau.Position{Colonne: 10, Ligne: 10},
		Zone:   4,
	}
}

// verifierEffets applique une suite d'effets et défait tout, en descendant dans
// les effets différés que le jeu appliquera à l'échéance.
func verifierEffets(t *testing.T, effets []noyau.Effet, ctx noyau.Contexte) {
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
func partieDEssai() *noyau.Partie {
	p := &noyau.Partie{
		Parametres: noyau.ParametresDefaut(),
		Plateau:    plateauPlat{},
		Tour:       5,
		Phase:      noyau.PhaseInspecteurs,
		Fugitif: noyau.Fugitif{
			Position:   noyau.Position{Colonne: 10, Ligne: 10},
			Resistance: 10,
		},
	}
	p.Parametres.Cote = 21
	for i := 0; i < p.Parametres.Inspecteurs; i++ {
		p.Inspecteurs = append(p.Inspecteurs, noyau.Inspecteur{
			Position: noyau.Position{Colonne: i, Ligne: 0},
		})
	}
	return p
}
