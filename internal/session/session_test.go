// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/core"
)

// partieRegardee monte une partie que deux cerveaux disputent sans personne.
//
// Le Quartier plutôt que la Ville : vingt et un tours suffisent à exercer tout
// le déroulé — étranglement, révélation, traces — et la suite entière tourne
// en une fraction de seconde.
func partieRegardee(seed int64, sortie io.Writer) Options {
	reglage, _ := core.PresetByKey("quartier")
	return Options{
		Seed:     seed,
		Settings: reglage.Settings,
		Out:      sortie,
	}
}

// TestWatchedGameReachesAnOutcome vérifie qu'une partie sans joueur se termine.
//
// C'est ce que la boucle doit garantir avant tout : sans elle, aucune passe
// d'équilibrage n'est possible, et le noyau reste une bibliothèque que personne
// n'exerce de bout en bout.
func TestWatchedGameReachesAnOutcome(t *testing.T) {
	var sortie strings.Builder

	issue, err := Run(partieRegardee(3, &sortie))
	if err != nil {
		t.Fatalf("partie interrompue : %v", err)
	}

	if issue.Reason == "" {
		t.Error("la partie s'achève sans motif")
	}
	if issue.Winner != core.SideFugitive && issue.Winner != core.SideInspectors {
		t.Errorf("vainqueur %q, attendu un des deux camps", issue.Winner)
	}
	if issue.Turn <= 0 {
		t.Errorf("fin au tour %d", issue.Turn)
	}
}

// TestSameSeedSameGame est l'invariant de déterminisme, appliqué à la partie
// entière.
//
// Deux parties lancées sur la même graine doivent se dérouler à l'identique,
// coup pour coup. C'est ce qui rend possible la reproduction d'un défaut, la
// comparaison de deux versions d'IA, et le rejeu depuis le journal.
func TestSameSeedSameGame(t *testing.T) {
	jouer := func(seed int64) (core.Outcome, string) {
		var sortie strings.Builder
		issue, err := Run(partieRegardee(seed, &sortie))
		if err != nil {
			t.Fatal(err)
		}
		return issue, sortie.String()
	}

	a, texteA := jouer(11)
	b, texteB := jouer(11)

	if a != b {
		t.Errorf("issues différentes : %+v puis %+v", a, b)
	}
	if texteA != texteB {
		t.Error("deux parties de même graine ne rendent pas le même écran")
	}

	if c, _ := jouer(12); c == a && a.Turn == c.Turn {
		t.Log("deux graines donnent la même issue — possible, mais à surveiller")
	}
}

// TestSeveralSeedsPlayThrough exerce le déroulé sur un lot de graines.
//
// Le bot minimal du protocole sert ici de test de conformité, comme
// docs/protocole-bot.md l'annonce : s'il joue des parties entières sans qu'un
// coup soit refusé, c'est que LegalMoves et Apply s'accordent sur toutes les
// phases et tous les types de coup.
func TestSeveralSeedsPlayThrough(t *testing.T) {
	motifs := map[string]int{}

	for seed := int64(1); seed <= 25; seed++ {
		issue, err := Run(partieRegardee(seed, nil))
		if err != nil {
			t.Fatalf("graine %d : %v", seed, err)
		}
		motifs[issue.Reason]++
	}

	if len(motifs) == 0 {
		t.Fatal("aucune partie n'est allée au bout")
	}
	t.Logf("issues sur vingt-cinq parties : %v", motifs)
}

// TestWatcherSeesBothSides vérifie que l'écran de fin montre ce qu'aucun des
// deux camps ne sait.
//
// Sans personne pour jouer, l'affichage superpose les deux vues filtrées : le
// fugitif y apparaît alors qu'il est resté caché pour les inspecteurs, et sans
// que rien n'ait lu l'état complet.
func TestWatcherSeesBothSides(t *testing.T) {
	var sortie strings.Builder
	if _, err := Run(partieRegardee(5, &sortie)); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(sortie.String(), "vue complète") {
		t.Error("l'écran de fin n'annonce pas une vue de spectateur")
	}
	if !strings.ContainsRune(sortie.String(), 'F') {
		t.Errorf("le fugitif n'apparaît pas :\n%s", sortie.String())
	}
}

// TestHumanIsAskedToPlay vérifie que le camp du joueur passe par la saisie et
// non par un cerveau.
//
// L'entrée est un texte : la boucle doit être pilotable sans clavier, sinon
// aucun test d'intégration ne peut la traverser.
func TestHumanIsAskedToPlay(t *testing.T) {
	var sortie strings.Builder
	o := partieRegardee(7, &sortie)
	o.Human = core.SideFugitive
	o.In = strings.NewReader("1\n") // puis fin d'entrée : abandon

	_, err := Run(o)
	if !errors.Is(err, ErrAbandon) {
		t.Fatalf("erreur %v, attendu l'abandon en fin d'entrée", err)
	}
	if !strings.Contains(sortie.String(), "Coup (1-") {
		t.Errorf("le joueur n'a pas été invité à jouer :\n%s", sortie.String())
	}
}

// TestUnknownPresetIsRefused vérifie qu'une partie ne démarre pas sur des
// réglages injouables.
func TestUnknownPresetIsRefused(t *testing.T) {
	o := partieRegardee(1, nil)
	o.Settings.Size = 3 // sous MinSize

	if _, err := Run(o); err == nil {
		t.Error("une partie démarre sur un plateau trop petit")
	}
}
