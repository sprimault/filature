// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package text

import (
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/core"
)

// testView monte une vue de cinq cases de côté, dégagée sauf un bâtiment.
//
// Écrite à la main plutôt que tirée d'une partie : ce paquet doit se tester
// sans monter de plateau ni de registre, faute de quoi un défaut de génération
// ferait échouer un test d'affichage.
func testView() core.View {
	v := core.View{
		Side:     core.SideFugitive,
		Turn:     3,
		Phase:    core.PhaseFugitive,
		Settings: core.Settings{Size: 5, Turns: 20},
	}

	for ligne := 0; ligne < 5; ligne++ {
		for colonne := 0; colonne < 5; colonne++ {
			if colonne == 2 && ligne == 2 {
				continue // le seul bâtiment
			}
			v.Streets = append(v.Streets, core.Position{Column: colonne, Row: ligne})
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

// TestBoardDrawsTerrain vérifie que rue et bâtiment se distinguent.
func TestBoardDrawsTerrain(t *testing.T) {
	rendu := Board(testView())

	if got := ligneDe(t, rendu, 0); got != "....." {
		t.Errorf("ligne 0 = %q, attendu %q", got, ".....")
	}
	if got := ligneDe(t, rendu, 2); got != "..#.." {
		t.Errorf("ligne 2 = %q, attendu %q", got, "..#..")
	}
}

// TestPiecesCoverTheRest vérifie l'ordre d'empilement des symboles.
//
// Savoir qu'un inspecteur se tient sur une trace importe plus que la trace :
// l'inverse cacherait un pion derrière ce qu'il piétine.
func TestPiecesCoverTheRest(t *testing.T) {
	v := testView()
	sur := core.Position{Column: 1, Row: 1}

	v.TracesConnues = map[string]core.Trail{sur.Key(): {Turn: 2, Direction: core.Est}}
	v.CrimeScenes = []core.CrimeScene{{Position: sur, Turn: 2}}
	v.Inspectors = []core.Inspector{{Position: sur}}

	if got := ligneDe(t, Board(v), 1); got[1] != CarInspecteur {
		t.Errorf("ligne 1 = %q, l'inspecteur devrait couvrir la scène et la trace", got)
	}
}

// TestFugitiveAbsentFromInspectorsView vérifie que l'affichage ne montre pas
// ce que la vue ne porte pas.
//
// C'est la vue qui filtre, jamais le rendu : s'il fallait que l'affichage
// s'abstienne de montrer un champ rempli, la fuite aurait déjà eu lieu côté
// réseau.
func TestFugitiveAbsentFromInspectorsView(t *testing.T) {
	v := testView()
	v.Side = core.SideInspectors
	v.PositionFugitif = nil

	if strings.ContainsRune(Board(v), CarFugitif) {
		t.Error("le fugitif apparaît alors que la vue ne le porte pas")
	}
}

// TestStatusHidesWhatItIgnores vérifie que la résistance n'est affichée que
// lorsqu'elle est connue.
//
// Un zéro affiché par défaut dirait que le fugitif est à bout, c'est-à-dire
// exactement le contraire de « on n'en sait rien ».
func TestStatusHidesWhatItIgnores(t *testing.T) {
	v := testView()
	v.Side = core.SideInspectors
	v.Stamina = nil

	if strings.Contains(Status(v), "Résistance") {
		t.Error("la résistance est affichée alors que la vue ne la porte pas")
	}

	huit := 8
	v.Stamina = &huit
	if !strings.Contains(Status(v), "Résistance : 8") {
		t.Error("la résistance connue n'est pas affichée")
	}
}

// TestZonesNumbered vérifie que chaque zone porte son numéro sur le plateau.
func TestZonesNumbered(t *testing.T) {
	v := testView()
	v.Zones = []core.Zone{
		{Number: 0, Cells: []core.Position{{Column: 0, Row: 0}}},
		{Number: 3, Cells: []core.Position{{Column: 4, Row: 4}}, Closed: true},
	}

	rendu := Board(v)
	if ligneDe(t, rendu, 0)[0] != '0' {
		t.Error("la zone 0 n'est pas dessinée")
	}
	if ligneDe(t, rendu, 4)[4] != '3' {
		t.Error("la zone 3 n'est pas dessinée")
	}
	if !strings.Contains(Status(v), "Zones fermées : 3") {
		t.Errorf("la fermeture n'est pas signalée :\n%s", Status(v))
	}
}

// TestMergeGivesWatcherWhatNobodyKnows vérifie que la superposition des
// deux vues suffit à tout montrer.
//
// C'est ce qui permet de se passer d'un accès à l'état complet : si un champ
// manquait aux deux vues, il manquerait aussi au contrat, et le trou se verrait
// ici plutôt que de rester silencieux.
func TestMergeGivesWatcherWhatNobodyKnows(t *testing.T) {
	cache := core.Position{Column: 3, Row: 1}
	zone := 2
	resistance := 7

	fugitif := testView()
	fugitif.PositionFugitif = &cache
	fugitif.SealedZone = &zone
	fugitif.Stamina = &resistance
	fugitif.TracesConnues = map[string]core.Trail{
		cache.Key(): {Turn: 2, Direction: core.Nord},
	}

	inspecteurs := testView()
	inspecteurs.Side = core.SideInspectors
	inspecteurs.Inspectors = []core.Inspector{{Position: core.Position{Column: 0, Row: 4}}}
	ailleurs := core.Position{Column: 4, Row: 0}
	inspecteurs.TracesConnues = map[string]core.Trail{
		ailleurs.Key(): {Turn: 1, Direction: core.Sud},
	}

	v := Merge(fugitif, inspecteurs)

	if v.PositionFugitif == nil || *v.PositionFugitif != cache {
		t.Error("le spectateur ne voit pas le fugitif")
	}
	if v.SealedZone == nil || *v.SealedZone != zone {
		t.Error("le spectateur ne voit pas la zone scellée")
	}
	if v.Stamina == nil || *v.Stamina != resistance {
		t.Error("le spectateur ne voit pas la résistance")
	}
	if len(v.Inspectors) != 1 {
		t.Error("le spectateur ne voit pas les inspecteurs")
	}
	if len(v.TracesConnues) != 2 {
		t.Errorf("%d traces, attendu les deux camps réunis", len(v.TracesConnues))
	}
	if len(v.LegalMoves) != 0 {
		t.Error("un spectateur ne joue pas, il n'a pas à recevoir de coups")
	}

	rendu := Board(v)
	if !strings.ContainsRune(rendu, CarFugitif) || !strings.ContainsRune(rendu, CarInspecteur) {
		t.Errorf("la vue fusionnée ne montre pas les deux camps :\n%s", rendu)
	}
}

// TestPieceLetterBounded vérifie qu'un rang hors alphabet se dit en clair.
//
// Un tel rang ne peut venir que d'un coup mal formé, d'un bot par exemple. Le
// convertir sans borne produirait un caractère dont personne ne saurait dire
// d'où il sort.
func TestPieceLetterBounded(t *testing.T) {
	for rang, attendu := range map[int]string{0: "A", 4: "E", 25: "Z"} {
		if got := PieceLetter(rang); got != attendu {
			t.Errorf("rang %d rendu %q, attendu %q", rang, got, attendu)
		}
	}
	for _, rang := range []int{-1, 26, 9000} {
		if got := PieceLetter(rang); !strings.Contains(got, "pion") {
			t.Errorf("rang %d rendu %q, attendu qu'il se dise en clair", rang, got)
		}
	}
}

// TestEndingNamesTheReason vérifie que chaque fin de partie se lit en français.
func TestEndingNamesTheReason(t *testing.T) {
	for motif, attendu := range map[string]string{
		core.OutcomeExtraction:   "extrait",
		core.OutcomeStaminaSpent: "à bout",
		core.OutcomeCornered:     "pris",
		core.OutcomeTimeUp:       "écoulé",
	} {
		rendu := Ending(core.Outcome{Winner: core.SideFugitive, Reason: motif, Turn: 12})
		if !strings.Contains(rendu, attendu) {
			t.Errorf("motif %s rendu %q, attendu qu'il contienne %q", motif, rendu, attendu)
		}
	}
}
