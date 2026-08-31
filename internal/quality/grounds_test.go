// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/sprimault/evasion/internal/render"
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

// TestGroundRangeIsWhatTheContractSays vérifie l'écart entre le sol le plus
// clair et le plus sombre, que deux textes publient.
//
// Le schéma de formes a dit « du simple au triple » quand le paquet de rendu
// disait « du simple au double et demi », pour la même palette : c'est le
// second qui avait raison. L'écart justifie une règle — un contour est
// nécessaire parce qu'aucun remplissage ne se détache de tous les sols —, donc
// il ne peut pas dériver sans que la règle change avec lui.
//
// Les bornes sont larges à dessein : ce qu'on garde est l'ordre de grandeur
// que la phrase énonce, pas une valeur au centième que la moindre retouche de
// palette ferait rougir.
func TestGroundRangeIsWhatTheContractSays(t *testing.T) {
	sols := solsLivres(t)
	rapport := sols[0].l / sols[len(sols)-1].l

	if rapport < 2.4 || rapport >= 3 {
		t.Errorf("du plus clair au plus sombre, rapport de %.2f : les textes annoncent "+
			"« du simple au double et demi », qui ne tient plus", rapport)
	}
}

// TestGrainAmplitudeIsWhatTheContractSays vérifie que l'amplitude publiée par le
// contrat de formes est celle du moteur.
//
// Le grain a été décrit en pourcents pendant des mois, alors qu'il est absolu :
// la godoc de la constante garde trace que la formulation fautive est restée
// « assez longtemps pour que l'aperçu les applique ». C'est le chiffre du §8 qui
// se garde ici, tout le reste du document en découlant.
func TestGrainAmplitudeIsWhatTheContractSays(t *testing.T) {
	contenu, err := os.ReadFile(filepath.Join(racine, "docs", "contrat-formes.md"))
	if err != nil {
		t.Fatal(err)
	}

	trouvaille := regexp.MustCompile(`±(\d+) niveaux de luminance`).FindStringSubmatch(string(contenu))
	if trouvaille == nil {
		t.Fatal("le contrat n'annonce plus l'amplitude du grain en niveaux de luminance")
	}
	annonce, err := strconv.Atoi(trouvaille[1])
	if err != nil {
		t.Fatal(err)
	}
	if annonce != render.GroundGrainAmplitude {
		t.Errorf("le contrat annonce ±%d niveaux, le moteur en applique %d",
			annonce, render.GroundGrainAmplitude)
	}
}

// TestGroundsAreListedFromLightestToDarkest vérifie que render.Grounds est dans
// l'ordre que sa godoc annonce.
//
// Rien ne le gardait : les autres contrôles retrient par luminance avant de
// mesurer, et la liste a passé deux versions avec ses rangs deux et trois
// inversés — le lot qui a reposé les couleurs a corrigé l'énoncé de la palette
// et laissé la liste. La planche d'aperçu s'y adosse et sortait donc avec ses
// bandes du milieu permutées.
func TestGroundsAreListedFromLightestToDarkest(t *testing.T) {
	sols := solsDeLaPalette(t)

	for i := 1; i < len(render.Grounds); i++ {
		precedent, courant := render.Grounds[i-1], render.Grounds[i]
		if sols[precedent] <= sols[courant] {
			t.Errorf("render.Grounds place %s (%.1f) avant %s (%.1f) : la liste "+
				"se dit du plus clair au plus sombre",
				precedent, sols[precedent], courant, sols[courant])
		}
	}
}

// solsDeLaPalette rend la luminance de chaque sol livré, sans les trier.
//
// solsLivres trie, ce qui convient à ce qui mesure des écarts et interdit de
// vérifier un rang.
func solsDeLaPalette(t *testing.T) map[string]float64 {
	t.Helper()

	luminances := make(map[string]float64, len(render.Grounds))
	for _, sol := range solsLivres(t) {
		luminances[sol.nom] = sol.l
	}
	return luminances
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
