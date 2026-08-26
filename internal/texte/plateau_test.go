// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package texte

import (
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/noyau"
)

// vueDEssai monte une vue de cinq cases de côté, dégagée sauf un bâtiment.
//
// Écrite à la main plutôt que tirée d'une partie : ce paquet doit se tester
// sans monter de plateau ni de registre, faute de quoi un défaut de génération
// ferait échouer un test d'affichage.
func vueDEssai() noyau.Vue {
	v := noyau.Vue{
		Acteur:     noyau.CampFugitif,
		Tour:       3,
		Phase:      noyau.PhaseFugitif,
		Parametres: noyau.Parametres{Cote: 5, Tours: 20},
	}

	for ligne := 0; ligne < 5; ligne++ {
		for colonne := 0; colonne < 5; colonne++ {
			if colonne == 2 && ligne == 2 {
				continue // le seul bâtiment
			}
			v.Rues = append(v.Rues, noyau.Position{Colonne: colonne, Ligne: ligne})
		}
	}
	return v
}

// ligneDe rend une ligne du plateau dessiné, sans la marge des coordonnées.
func ligneDe(t *testing.T, rendu string, ligne int) string {
	t.Helper()

	lignes := strings.Split(rendu, "\n")
	if len(lignes) < ligne+3 {
		t.Fatalf("rendu trop court :\n%s", rendu)
	}
	// Deux lignes d'en-tête, puis quatre caractères de marge.
	return strings.TrimPrefix(lignes[ligne+2], "  ")[2:]
}

// TestPlateauDessineLeTerrain vérifie que rue et bâtiment se distinguent.
func TestPlateauDessineLeTerrain(t *testing.T) {
	rendu := Plateau(vueDEssai())

	if got := ligneDe(t, rendu, 0); got != "....." {
		t.Errorf("ligne 0 = %q, attendu %q", got, ".....")
	}
	if got := ligneDe(t, rendu, 2); got != "..#.." {
		t.Errorf("ligne 2 = %q, attendu %q", got, "..#..")
	}
}

// TestPionsCouvrentLeReste vérifie l'ordre d'empilement des symboles.
//
// Savoir qu'un inspecteur se tient sur une trace importe plus que la trace :
// l'inverse cacherait un pion derrière ce qu'il piétine.
func TestPionsCouvrentLeReste(t *testing.T) {
	v := vueDEssai()
	sur := noyau.Position{Colonne: 1, Ligne: 1}

	v.TracesConnues = map[string]noyau.Trace{sur.Cle(): {Tour: 2, Direction: noyau.Est}}
	v.Scenes = []noyau.Scene{{Position: sur, Tour: 2}}
	v.Inspecteurs = []noyau.Inspecteur{{Position: sur}}

	if got := ligneDe(t, Plateau(v), 1); got[1] != CarInspecteur {
		t.Errorf("ligne 1 = %q, l'inspecteur devrait couvrir la scène et la trace", got)
	}
}

// TestFugitifAbsentDeLaVueDesInspecteurs vérifie que l'affichage ne montre pas
// ce que la vue ne porte pas.
//
// C'est la vue qui filtre, jamais le rendu : s'il fallait que l'affichage
// s'abstienne de montrer un champ rempli, la fuite aurait déjà eu lieu côté
// réseau.
func TestFugitifAbsentDeLaVueDesInspecteurs(t *testing.T) {
	v := vueDEssai()
	v.Acteur = noyau.CampInspecteurs
	v.PositionFugitif = nil

	if strings.ContainsRune(Plateau(v), CarFugitif) {
		t.Error("le fugitif apparaît alors que la vue ne le porte pas")
	}
}

// TestEtatTaitCeQuIlIgnore vérifie que la résistance n'est affichée que
// lorsqu'elle est connue.
//
// Un zéro affiché par défaut dirait que le fugitif est à bout, c'est-à-dire
// exactement le contraire de « on n'en sait rien ».
func TestEtatTaitCeQuIlIgnore(t *testing.T) {
	v := vueDEssai()
	v.Acteur = noyau.CampInspecteurs
	v.Resistance = nil

	if strings.Contains(Etat(v), "Résistance") {
		t.Error("la résistance est affichée alors que la vue ne la porte pas")
	}

	huit := 8
	v.Resistance = &huit
	if !strings.Contains(Etat(v), "Résistance : 8") {
		t.Error("la résistance connue n'est pas affichée")
	}
}

// TestZonesNumerotees vérifie que chaque zone porte son numéro sur le plateau.
func TestZonesNumerotees(t *testing.T) {
	v := vueDEssai()
	v.Zones = []noyau.Zone{
		{Numero: 0, Cases: []noyau.Position{{Colonne: 0, Ligne: 0}}},
		{Numero: 3, Cases: []noyau.Position{{Colonne: 4, Ligne: 4}}, Fermee: true},
	}

	rendu := Plateau(v)
	if ligneDe(t, rendu, 0)[0] != '0' {
		t.Error("la zone 0 n'est pas dessinée")
	}
	if ligneDe(t, rendu, 4)[4] != '3' {
		t.Error("la zone 3 n'est pas dessinée")
	}
	if !strings.Contains(Etat(v), "Zones fermées : 3") {
		t.Errorf("la fermeture n'est pas signalée :\n%s", Etat(v))
	}
}

// TestFusionnerDonneAuSpectateurCeQueNulNeSait vérifie que la superposition des
// deux vues suffit à tout montrer.
//
// C'est ce qui permet de se passer d'un accès à l'état complet : si un champ
// manquait aux deux vues, il manquerait aussi au contrat, et le trou se verrait
// ici plutôt que de rester silencieux.
func TestFusionnerDonneAuSpectateurCeQueNulNeSait(t *testing.T) {
	cache := noyau.Position{Colonne: 3, Ligne: 1}
	zone := 2
	resistance := 7

	fugitif := vueDEssai()
	fugitif.PositionFugitif = &cache
	fugitif.ZoneScellee = &zone
	fugitif.Resistance = &resistance
	fugitif.TracesConnues = map[string]noyau.Trace{
		cache.Cle(): {Tour: 2, Direction: noyau.Nord},
	}

	inspecteurs := vueDEssai()
	inspecteurs.Acteur = noyau.CampInspecteurs
	inspecteurs.Inspecteurs = []noyau.Inspecteur{{Position: noyau.Position{Colonne: 0, Ligne: 4}}}
	ailleurs := noyau.Position{Colonne: 4, Ligne: 0}
	inspecteurs.TracesConnues = map[string]noyau.Trace{
		ailleurs.Cle(): {Tour: 1, Direction: noyau.Sud},
	}

	v := Fusionner(fugitif, inspecteurs)

	if v.PositionFugitif == nil || *v.PositionFugitif != cache {
		t.Error("le spectateur ne voit pas le fugitif")
	}
	if v.ZoneScellee == nil || *v.ZoneScellee != zone {
		t.Error("le spectateur ne voit pas la zone scellée")
	}
	if v.Resistance == nil || *v.Resistance != resistance {
		t.Error("le spectateur ne voit pas la résistance")
	}
	if len(v.Inspecteurs) != 1 {
		t.Error("le spectateur ne voit pas les inspecteurs")
	}
	if len(v.TracesConnues) != 2 {
		t.Errorf("%d traces, attendu les deux camps réunis", len(v.TracesConnues))
	}
	if len(v.CoupsLegaux) != 0 {
		t.Error("un spectateur ne joue pas, il n'a pas à recevoir de coups")
	}

	rendu := Plateau(v)
	if !strings.ContainsRune(rendu, CarFugitif) || !strings.ContainsRune(rendu, CarInspecteur) {
		t.Errorf("la vue fusionnée ne montre pas les deux camps :\n%s", rendu)
	}
}

// TestLettrePionBornee vérifie qu'un rang hors alphabet se dit en clair.
//
// Un tel rang ne peut venir que d'un coup mal formé, d'un bot par exemple. Le
// convertir sans borne produirait un caractère dont personne ne saurait dire
// d'où il sort.
func TestLettrePionBornee(t *testing.T) {
	for rang, attendu := range map[int]string{0: "A", 4: "E", 25: "Z"} {
		if got := LettrePion(rang); got != attendu {
			t.Errorf("rang %d rendu %q, attendu %q", rang, got, attendu)
		}
	}
	for _, rang := range []int{-1, 26, 9000} {
		if got := LettrePion(rang); !strings.Contains(got, "pion") {
			t.Errorf("rang %d rendu %q, attendu qu'il se dise en clair", rang, got)
		}
	}
}

// TestFinNommeLeMotif vérifie que chaque fin de partie se lit en français.
func TestFinNommeLeMotif(t *testing.T) {
	for motif, attendu := range map[string]string{
		noyau.MotifExtraction:  "extrait",
		noyau.MotifResistance:  "à bout",
		noyau.MotifBlocage:     "pris",
		noyau.MotifTempsEcoule: "écoulé",
	} {
		rendu := Fin(noyau.Resultat{Vainqueur: noyau.CampFugitif, Motif: motif, Tour: 12})
		if !strings.Contains(rendu, attendu) {
			t.Errorf("motif %s rendu %q, attendu qu'il contienne %q", motif, rendu, attendu)
		}
	}
}
