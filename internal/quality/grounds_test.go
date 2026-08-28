// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/sprimault/filature/internal/render"
)

// luminance rend la clarté perçue d'une couleur, sur 255.
//
// Rec. 601 parce que c'est celle qui décrit la perception d'un écart de gris, et
// que c'est un écart de gris qui sépare deux sols pour un daltonien comme sur
// une capture en noir et blanc.
func luminance(t *testing.T, hex string) float64 {
	t.Helper()
	var r, v, b int
	if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &v, &b); err != nil {
		t.Fatalf("couleur %q illisible : %v", hex, err)
	}
	return 0.299*float64(r) + 0.587*float64(v) + 0.114*float64(b)
}

// TestGroundsStayApartUnderGrain vérifie qu'aucune paire de sols voisins sur
// l'échelle de luminance ne se recouvre une fois le grain appliqué.
//
// Le grain déplace chaque case de ±GroundGrain : deux sols séparés de moins du
// double peuvent donc échanger leur rang à l'affichage, et le joueur lit une
// zone fermée là où il y a un lieu en recharge. C'est une donnée de jeu que la
// décoration effacerait.
//
// Le contrôle compose la palette et l'amplitude, parce que l'écart se mesure
// après le grain et non avant. Sans lui, la règle serait une phrase de plus dans
// un document — les deux couleurs de lieu ont vécu des mois à neuf niveaux l'une
// de l'autre sans que rien ne le dise.
func TestGroundsStayApartUnderGrain(t *testing.T) {
	sols := solsLivres(t)

	minimal := float64(2 * render.GroundGrainAmplitude)
	for i := 1; i < len(sols); i++ {
		if ecart := sols[i-1].l - sols[i].l; ecart <= minimal {
			t.Errorf("%s et %s sont à %.1f niveaux, le grain en déplace %d : "+
				"les deux se recouvrent à l'affichage",
				sols[i-1].nom, sols[i].nom, ecart, render.GroundGrainAmplitude)
		}
	}
}

// TestGrainKeepsGroundsInOrder applique le grain tel que le rendu le pose et
// vérifie qu'aucun sol ne passe devant son voisin.
//
// Le test précédent compare des luminances nominales et le seuil qui les borne.
// Celui-ci compare ce qui s'affiche, et c'est la différence qui compte : le
// grain a longtemps été appliqué en pourcents, ce qui déplace d'autant moins
// qu'une couleur est sombre. Sur la palette livrée, la rue passait alors sous
// un lieu actif et un lieu actif sous une zone ouverte — deux inversions qu'un
// seuil de dix niveaux ne pouvait pas voir, les écarts nominaux valant
// dix-sept.
//
// Le pire cas suffit et se calcule : le plus clair au grain le plus bas, le
// plus sombre au plus haut.
func TestGrainKeepsGroundsInOrder(t *testing.T) {
	sols := solsLivres(t)

	for i := 1; i < len(sols); i++ {
		clair := luminance(t, render.ShiftLuminance(sols[i-1].hex, -render.GroundGrainAmplitude))
		sombre := luminance(t, render.ShiftLuminance(sols[i].hex, render.GroundGrainAmplitude))
		if clair <= sombre {
			t.Errorf("sous le grain, %s tombe à %.1f et %s monte à %.1f : "+
				"le joueur lit l'un pour l'autre",
				sols[i-1].nom, clair, sols[i].nom, sombre)
		}
	}
}

// solLivre est un sol de la palette livrée, avec sa luminance.
type solLivre struct {
	nom string
	hex string
	l   float64
}

// solsLivres rend les sols de la palette livrée, du plus clair au plus sombre.
func solsLivres(t *testing.T) []solLivre {
	t.Helper()

	var fichier struct {
		Palette map[string]string `toml:"palette"`
	}
	chemin := filepath.Join(racine, "plugins", "base", "palette.toml")
	if _, err := toml.DecodeFile(chemin, &fichier); err != nil {
		t.Fatal(err)
	}

	var sols []solLivre
	for _, nom := range render.Grounds {
		hex, present := fichier.Palette[nom]
		if !present {
			t.Fatalf("le sol %s manque de la palette livrée", nom)
		}
		sols = append(sols, solLivre{nom, hex, luminance(t, hex)})
	}
	sort.Slice(sols, func(i, j int) bool { return sols[i].l > sols[j].l })
	return sols
}
