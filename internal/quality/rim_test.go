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

// fondsSousUnPion énumère tout ce sur quoi un pion peut se dessiner, sous les
// noms que le contrat de formes leur donne.
//
// Les cinq sols ne suffisent pas : un pion se dessine par-dessus les cubes
// situés devant lui, donc sa moitié supérieure est sur du bâti, dont les trois
// faces ont trois luminances. Le hors plateau en fait partie — le pourtour est
// rendu, et une pièce peut s'y trouver.
//
// **Les deux marqueurs aussi, et c'est ce que cette liste oubliait.** L'ordre de
// peinture pose le pion par-dessus la trace ou le barrage de sa case, et le
// Barreur peut fermer une case occupée. Le pire fond du plateau se trouve être
// l'un d'eux, si bien que le plancher publié valait celui du neuvième fond et
// non du onzième.
func fondsSousUnPion(t *testing.T) map[string]string {
	t.Helper()

	palette := paletteLivree(t)
	fonds := map[string]string{
		"rue":              palette["street"],
		"lieu actif":       palette["shelter_open"],
		"zone ouverte":     palette["zone_open"],
		"lieu en recharge": palette["shelter_used"],
		"zone fermée":      palette["zone_closed"],
		"hors plateau":     palette["backdrop"],
		"trace":            palette["trail"],
		"barrage":          palette["roadblock"],
	}

	// Par render.Lit et non par les trois nombres recopiés : le bâti n'est pas
	// un fond mais trois, et un contrôle qui les réécrit ne mesure plus que
	// lui-même.
	for nom, face := range map[string]int{
		"dessus d'un bâtiment":      render.FaceTop,
		"face droite d'un bâtiment": render.FaceRight,
		"face gauche d'un bâtiment": render.FaceLeft,
	} {
		fonds[nom] = render.Lit(palette["building"], face)
	}
	return fonds
}

// tenus rend, pour chaque fond, ce que le contour seul garde et ce que le
// liseré lui ajoute.
//
// Le meilleur des deux, parce que c'est la promesse du liseré : le contour tient
// sur les fonds clairs, le liseré sur les sombres, et il suffit que l'un des deux
// se voie. Le pire des trois contours, parce qu'une seule forme illisible suffit.
func tenus(t *testing.T, fond string) (contourSeul, avecLisere float64) {
	t.Helper()

	palette := paletteLivree(t)
	contourSeul = 99
	for _, contour := range []string{"fugitive_detail", "inspector_detail", "marker_outline"} {
		contourSeul = min(contourSeul, contraste(t, palette[contour], fond))
	}
	return contourSeul, max(contourSeul, contraste(t, render.RimColor, fond))
}

// TestRimHoldsOnEveryBackground garde ce que le liseré promet : sur n'importe
// quel fond, l'un des deux traits tranche.
//
// C'est la seule justification d'une couleur fixe que nul plugin ne peut
// déplacer, et rien ne la mesurait — le contrôle des contours ne voyait ni le
// liseré, ni les faces du bâti, ni les lieux de ressourcement. Il a ensuite
// ignoré les deux marqueurs, sur lesquels un pion se dessine pourtant : le pire
// fond du plateau est l'un d'eux.
func TestRimHoldsOnEveryBackground(t *testing.T) {
	// WCAG 2.1, critère 1.4.11 : trois pour un sur un élément non textuel. Le
	// 4,5 souvent cité est celui du texte courant, et ne s'applique pas à un
	// pion posé sur un fond.
	const seuil = 3.0

	pire, pireFond := 99.0, ""
	for nom, fond := range fondsSousUnPion(t) {
		if _, avec := tenus(t, fond); avec < pire {
			pire, pireFond = avec, nom
		}
	}
	t.Logf("pire fond : %s, contraste de %.2f", pireFond, pire)

	if pire < seuil {
		t.Errorf("le pire fond ne tient qu'à %.2f (%s), sous les %.1f de WCAG 1.4.11 : "+
			"le liseré existe pour que ce cas n'arrive pas", pire, pireFond, seuil)
	}
}

// TestRimTableMatchesTheMeasure vérifie que le tableau du contrat de formes
// porte les valeurs qu'on mesure sur la palette livrée.
//
// Il ne les portait plus : ses chiffres dataient de la palette d'avant le lot
// qui a reposé les dix couleurs, et il a manqué deux fois des fonds — les lieux
// de ressourcement d'abord, les deux marqueurs ensuite. Le tableau et la liste
// mesurée se contrôlent l'un l'autre : un fond ajouté ici sans y être fait
// rougir ce test.
func TestRimTableMatchesTheMeasure(t *testing.T) {
	publie := tableauDuLisere(t)
	mesures := fondsSousUnPion(t)

	// Le sens qui manquait : sans lui, retirer un fond de la liste mesurée
	// laissait sa ligne au tableau sans que rien ne le dise — soit exactement la
	// façon dont les deux marqueurs en sont restés absents.
	for nom := range publie {
		if _, present := mesures[nom]; !present {
			t.Errorf("le tableau du liseré publie %q, qui n'est pas un fond mesuré", nom)
		}
	}

	for nom, fond := range mesures {
		ligne, present := publie[nom]
		if !present {
			t.Errorf("le fond %q ne figure pas dans le tableau du liseré", nom)
			continue
		}
		seul, avec := tenus(t, fond)

		// Deux centièmes de tolérance : le tableau arrondit, et resserrer plus
		// ferait rougir sur un chiffre que personne ne lit à cette précision.
		for _, cas := range []struct {
			colonne         string
			annonce, mesure float64
		}{
			{"contour seul", ligne.seul, seul},
			{"avec liseré", ligne.avec, avec},
		} {
			if diff := cas.annonce - cas.mesure; diff > 0.02 || diff < -0.02 {
				t.Errorf("%s, %s : le contrat annonce %.2f, la mesure donne %.2f",
					nom, cas.colonne, cas.annonce, cas.mesure)
			}
		}
	}
}

// ligneDuLisere est ce que le contrat annonce pour un fond.
type ligneDuLisere struct {
	seul, avec float64
}

// tableauDuLisere lit le tableau du §2, dont les valeurs portent une virgule
// décimale et parfois des astérisques de mise en gras.
func tableauDuLisere(t *testing.T) map[string]ligneDuLisere {
	t.Helper()

	contenu, err := os.ReadFile(filepath.Join(racine, "docs", "contrat-formes.md"))
	if err != nil {
		t.Fatal(err)
	}

	const entete = "| Fond | Contour seul | Avec liseré |"
	_, apres, trouve := strings.Cut(string(contenu), entete)
	if !trouve {
		t.Fatal("le tableau du liseré est absent du contrat de formes")
	}
	section, _, _ := strings.Cut(apres, "\n\n")

	ligne := regexp.MustCompile(`^\|([^|]+)\|([^|]+)\|([^|]+)\|$`)
	lignes := map[string]ligneDuLisere{}
	for _, brute := range strings.Split(section, "\n") {
		trouvaille := ligne.FindStringSubmatch(strings.TrimSpace(brute))
		if trouvaille == nil {
			continue
		}
		nombre := func(champ string) (float64, bool) {
			net := strings.ReplaceAll(strings.TrimSpace(champ), "*", "")
			v, err := strconv.ParseFloat(strings.ReplaceAll(net, ",", "."), 64)
			return v, err == nil
		}
		seul, seulLu := nombre(trouvaille[2])
		avec, avecLu := nombre(trouvaille[3])
		if !seulLu || !avecLu {
			continue
		}
		lignes[strings.TrimSpace(trouvaille[1])] = ligneDuLisere{seul, avec}
	}
	if len(lignes) == 0 {
		t.Fatal("aucune ligne lue dans le tableau du liseré : le contrôle ne dirait rien")
	}
	return lignes
}

// TestBackgroundRangeMatchesTheMeasure vérifie la plage de luminance que le
// contrat annonce pour les fonds.
//
// C'est l'argument qui justifie de poser deux traits plutôt qu'un : aucune
// couleur unique ne couvre cette plage. Elle n'était juste que par accident —
// sa borne haute est la trace, que le tableau voisin n'énumérait pas.
func TestBackgroundRangeMatchesTheMeasure(t *testing.T) {
	contenu, err := os.ReadFile(filepath.Join(racine, "docs", "contrat-formes.md"))
	if err != nil {
		t.Fatal(err)
	}
	trouvaille := regexp.MustCompile(`fonds possibles vont de (\d+) à (\d+) en luminance`).
		FindStringSubmatch(string(contenu))
	if trouvaille == nil {
		t.Fatal("le contrat n'annonce plus la plage des fonds")
	}

	bas, haut := 255.0, 0.0
	for _, fond := range fondsSousUnPion(t) {
		l := luminance(t, fond)
		bas, haut = min(bas, l), max(haut, l)
	}

	for _, cas := range []struct {
		bout    string
		annonce string
		mesure  float64
	}{
		{"basse", trouvaille[1], bas},
		{"haute", trouvaille[2], haut},
	} {
		annonce, err := strconv.Atoi(cas.annonce)
		if err != nil {
			t.Fatal(err)
		}
		// Une unité de tolérance : le document arrondit, et la plage sert un
		// ordre de grandeur, pas un calcul.
		if ecart := float64(annonce) - cas.mesure; ecart > 1 || ecart < -1 {
			t.Errorf("borne %s : le contrat annonce %d, la mesure donne %.1f",
				cas.bout, annonce, cas.mesure)
		}
	}
}
