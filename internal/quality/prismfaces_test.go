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

	"github.com/sprimault/evasion/internal/render"
)

// TestPrismFacesMatchTheContract vérifie que les coefficients d'éclairage
// publiés sont ceux que le moteur applique.
//
// Le contrat les écrit « parce que c'est exactement le genre de détail que deux
// implémentations règlent différemment sans que personne ne s'en aperçoive avant
// de comparer deux captures ». Il les écrivait sans que rien ne les tienne :
// porter le dessus à 2,50 laissait la suite entière verte, et modifier le
// document aussi. L'aperçu est la première implémentation, le rendu de l'étape 7
// sera la seconde.
func TestPrismFacesMatchTheContract(t *testing.T) {
	publies := coefficientsDuContrat(t)

	for face, nom := range map[int]string{
		render.FaceTop:   "dessus",
		render.FaceRight: "droite",
		render.FaceLeft:  "gauche",
	} {
		annonce, present := publies[nom]
		if !present {
			t.Errorf("la face %q n'a pas de coefficient publié", nom)
			continue
		}
		if annonce != render.PrismFaces[face] {
			t.Errorf("face %s : le contrat annonce × %.2f, le moteur applique × %.2f",
				nom, annonce, render.PrismFaces[face])
		}
	}
}

// coefficientsDuContrat lit la table des faces du §5, dont les valeurs portent
// une virgule décimale et un signe de multiplication.
func coefficientsDuContrat(t *testing.T) map[string]float64 {
	t.Helper()

	contenu, err := os.ReadFile(filepath.Join(racine, "docs", "contrat-formes.md"))
	if err != nil {
		t.Fatal(err)
	}

	const entete = "| Face | Coefficient |"
	_, apres, trouve := strings.Cut(string(contenu), entete)
	if !trouve {
		t.Fatal("la table des faces est absente du contrat de formes")
	}
	section, _, _ := strings.Cut(apres, "\n\n")

	ligne := regexp.MustCompile(`^\|\s*(\w+)\s*\|\s*×\s*([0-9]+,[0-9]+)\s*\|$`)
	coefficients := map[string]float64{}
	for _, brute := range strings.Split(section, "\n") {
		trouvaille := ligne.FindStringSubmatch(strings.TrimSpace(brute))
		if trouvaille == nil {
			continue
		}
		valeur, err := strconv.ParseFloat(strings.ReplaceAll(trouvaille[2], ",", "."), 64)
		if err != nil {
			t.Fatal(err)
		}
		coefficients[trouvaille[1]] = valeur
	}
	if len(coefficients) != len(render.PrismFaces) {
		t.Fatalf("%d coefficients lus dans le contrat pour %d faces",
			len(coefficients), len(render.PrismFaces))
	}
	return coefficients
}

// TestLitNeverExceedsWhite vérifie qu'un coefficient de face ne déborde pas d'un
// canal.
//
// Le débordement se borne plutôt que de reboucler : sans cette borne, un dessus
// de bâtiment ressortirait plus sombre que ses côtés dès qu'une palette monte.
func TestLitNeverExceedsWhite(t *testing.T) {
	if got := render.Lit("#c0c0c0", render.FaceTop); got != "#ffffff" {
		t.Errorf("Lit = %s, attendu que les canaux soient bornés", got)
	}
	if got := render.Lit("pas une couleur", render.FaceTop); got != "#000000" {
		t.Errorf("Lit = %s, attendu un repli plutôt qu'un échec", got)
	}
}
