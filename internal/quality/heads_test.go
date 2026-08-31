// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/sprimault/evasion/internal/loader"
	"github.com/sprimault/evasion/internal/render"
	"github.com/sprimault/evasion/plugins"
)

// TestPieceHeadsAtTheFloorScale vérifie la taille des têtes de pions au
// plancher de rendu, que deux textes citent.
//
// docs/contrat-formes.md et la godoc de StrokeWidth s'en servent pour justifier
// le plafond d'épaisseur : à cette échelle, deux pixels de liseré avaleraient
// une tête. L'argument ne vaut que si le chiffre est le bon, et il a annoncé
// « quatre et demi » pour deux têtes qui font 5,25 et 3,75 — une valeur qui
// n'est ni l'une ni l'autre, et qu'aucune géométrie livrée ne produit.
//
// Mesuré sur les formes livrées plutôt que recopié : c'est shapes.toml qui
// décide, et il bougera à l'étape 7 quand le rendu existera pour de bon.
func TestPieceHeadsAtTheFloorScale(t *testing.T) {
	_, formes, err := loader.Load(plugins.Shipped(), "")
	if err != nil {
		t.Fatalf("chargement du contenu livré : %v", err)
	}

	attendues := map[string]float64{"fugitive": 5.25, "inspector": 4.50}
	for nom, attendu := range attendues {
		forme, present := formes.Shapes[nom]
		if !present {
			t.Errorf("la forme %s a disparu du contenu livré", nom)
			continue
		}

		tete := 0
		for _, trait := range forme.Strokes {
			if trait.Type == render.StrokeCircle {
				tete = max(tete, 2*trait.Radius)
			}
		}
		if tete == 0 {
			t.Errorf("%s n'a plus de tête : aucun cercle dans ses traits", nom)
			continue
		}

		if got := float64(tete) * render.MinRenderScale; got != attendu {
			t.Errorf("tête de %s à %.2f pixels au plancher de rendu, les textes disent %.2f",
				nom, got, attendu)
		}
	}
}

// TestPieceCoreStaysMajority vérifie qu'un pion garde un remplissage
// majoritaire, sur la plus petite tête livrée et non sur la plus grande.
//
// C'est la couleur du noyau qui dit à quel camp un pion appartient : un noyau
// minoritaire n'est pas un défaut d'esthétique mais une perte d'information.
//
// Le contrôle vivait dans render avec un diamètre de quatorze en dur, présenté
// comme « la plus petite forme d'un pion livré ». C'est celle du fugitif ; celle
// d'un inspecteur fait dix, et il y en a cinq sur le plateau. Le test passait
// donc en n'exerçant que la forme la plus favorable.
func TestPieceCoreStaysMajority(t *testing.T) {
	_, formes, err := loader.Load(plugins.Shipped(), "")
	if err != nil {
		t.Fatalf("chargement du contenu livré : %v", err)
	}

	pire, pireNom, pireTaille := 1.0, "", 0
	for nom, forme := range formes.Shapes {
		if forme.Role != render.RolePiece {
			continue
		}
		tete := 0
		for _, trait := range forme.Strokes {
			if trait.Type == render.StrokeCircle {
				tete = max(tete, 2*trait.Radius)
			}
		}
		if tete == 0 {
			continue
		}

		for _, taille := range []int{64, 48, 32, 24} {
			if part := remplissage(tete, taille); part < pire {
				pire, pireNom, pireTaille = part, nom, taille
			}
		}
	}

	if pireNom == "" {
		t.Fatal("aucun pion à tête ronde dans le contenu livré : le contrôle ne dit rien")
	}
	t.Logf("pire remplissage : %s à %d px par case, %.1f %%", pireNom, pireTaille, 100*pire)

	if pire <= 0.5 {
		t.Errorf("à %d px par case, le noyau de %s ne fait que %.1f %% du pion",
			pireTaille, pireNom, 100*pire)
	}
}

// remplissage rend la part du noyau dans la largeur d'un pion, bordure comprise.
func remplissage(diametre, taille int) float64 {
	echelle := float64(taille) / render.TileWidth
	noyau := float64(diametre) * echelle
	bordure := render.StrokeWidth(2, echelle, diametre) +
		render.StrokeWidth(render.RimWidth, echelle, diametre)
	return noyau / (noyau + 2*bordure)
}

// TestFillTableMatchesTheMeasure vérifie que le tableau de remplissage du
// contrat porte ce qu'on mesure sur les formes livrées.
//
// Le contrat en publiait trois valeurs, celles du fugitif seul, sous une phrase
// qui parlait d'« un pion ». Un tableau de chiffres qui n'est comparé à rien
// vieillit sans qu'on le voie — c'est déjà arrivé à celui du liseré.
func TestFillTableMatchesTheMeasure(t *testing.T) {
	_, formes, err := loader.Load(plugins.Shipped(), "")
	if err != nil {
		t.Fatalf("chargement du contenu livré : %v", err)
	}

	contenu, err := os.ReadFile(filepath.Join(racine, "docs", "contrat-formes.md"))
	if err != nil {
		t.Fatal(err)
	}
	const entete = "| Pixels par case | Fugitif | Inspecteur |"
	_, apres, trouve := strings.Cut(string(contenu), entete)
	if !trouve {
		t.Fatal("le tableau de remplissage est absent du contrat de formes")
	}
	section, _, _ := strings.Cut(apres, "\n\n")

	ligne := regexp.MustCompile(`^\|\s*(\d+)\s*\|\s*(\d+) %\s*\|\s*(\d+) %\s*\|$`)
	lignes := 0
	for _, brute := range strings.Split(section, "\n") {
		trouvaille := ligne.FindStringSubmatch(strings.TrimSpace(brute))
		if trouvaille == nil {
			continue
		}
		lignes++

		taille, _ := strconv.Atoi(trouvaille[1])
		for colonne, nom := range map[int]string{2: "fugitive", 3: "inspector"} {
			annonce, _ := strconv.Atoi(trouvaille[colonne])
			mesure := int(100*remplissage(teteDe(t, formes, nom), taille) + 0.5)
			if annonce != mesure {
				t.Errorf("à %d px par case, le contrat donne %s à %d %%, la mesure %d %%",
					taille, nom, annonce, mesure)
			}
		}
	}
	if lignes == 0 {
		t.Fatal("aucune ligne lue dans le tableau de remplissage")
	}
}

// teteDe rend le diamètre de la tête d'un pion livré.
func teteDe(t *testing.T, formes *render.ShapeSet, nom string) int {
	t.Helper()

	forme, present := formes.Shapes[nom]
	if !present {
		t.Fatalf("la forme %s a disparu du contenu livré", nom)
	}
	tete := 0
	for _, trait := range forme.Strokes {
		if trait.Type == render.StrokeCircle {
			tete = max(tete, 2*trait.Radius)
		}
	}
	if tete == 0 {
		t.Fatalf("%s n'a plus de tête : aucun cercle dans ses traits", nom)
	}
	return tete
}
