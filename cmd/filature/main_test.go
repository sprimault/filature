// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ailleurs est un dossier de greffons actifs qui n'entre en conflit avec aucune
// destination d'essai.
const ailleurs = "/un/dossier/qui/n/est/pas/la/destination"

// TestExtraireEcritLesGreffonsLivres vérifie que la commande rend le contenu
// embarqué recopiable.
func TestExtraireEcritLesGreffonsLivres(t *testing.T) {
	dossier := t.TempDir()
	if err := extraire(dossier, ailleurs); err != nil {
		t.Fatalf("extraction refusée : %v", err)
	}

	for _, attendu := range []string{
		filepath.Join("base", "manifeste.toml"),
		filepath.Join("anglais", "langue.toml"),
	} {
		if _, err := os.Stat(filepath.Join(dossier, attendu)); err != nil {
			t.Errorf("%s n'a pas été écrit", attendu)
		}
	}
}

// TestExtraireNEcrasePas vérifie qu'un fichier existant arrête la commande.
//
// Un traducteur qui relance l'extraction sur son dossier de travail perdrait
// ses libellés. Le refus est ce qui rend la commande sûre à relancer.
func TestExtraireNEcrasePas(t *testing.T) {
	dossier := t.TempDir()
	if err := extraire(dossier, ailleurs); err != nil {
		t.Fatalf("première extraction refusée : %v", err)
	}

	temoin := filepath.Join(dossier, "anglais", "langue.toml")
	if err := os.WriteFile(temoin, []byte("# mon travail\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := extraire(dossier, ailleurs); err == nil {
		t.Fatal("la seconde extraction aurait dû échouer")
	}

	contenu, err := os.ReadFile(temoin)
	if err != nil {
		t.Fatal(err)
	}
	if string(contenu) != "# mon travail\n" {
		t.Error("le fichier existant a été écrasé")
	}
}

// TestExtraireSansDossier vérifie que la commande dit son usage plutôt que
// d'écrire dans le répertoire courant.
func TestExtraireSansDossier(t *testing.T) {
	if err := extraire("", ailleurs); err == nil {
		t.Fatal("un dossier vide a été accepté")
	}
}

// TestExtraireRefuseLeDossierActif ferme le piège que la commande tendait.
//
// Extraire le contenu livré là où le jeu charge le déclarerait deux fois : une
// fois depuis le binaire, une fois depuis le disque. Deux greffons qui
// définissent la même clé sont un conflit, donc suivre l'invitation cassait le
// chargement.
func TestExtraireRefuseLeDossierActif(t *testing.T) {
	actif := t.TempDir()

	if err := extraire(actif, actif); err == nil {
		t.Fatal("extraction acceptée dans le dossier des greffons actifs")
	}

	// Le refus doit venir avant toute écriture : un dossier à moitié rempli
	// serait pire qu'un refus, puisqu'il faudrait deviner quoi effacer.
	entrees, err := os.ReadDir(actif)
	if err != nil {
		t.Fatal(err)
	}
	if len(entrees) != 0 {
		t.Errorf("%d entrées écrites malgré le refus", len(entrees))
	}
}

// TestExtraireRefuseMemeAvecUneCasseDifferente vérifie que le refus tient sous
// Windows, où « Greffons » et « greffons » désignent le même dossier.
func TestExtraireRefuseMemeAvecUneCasseDifferente(t *testing.T) {
	actif := filepath.Join(t.TempDir(), "greffons")
	if err := os.MkdirAll(actif, 0o750); err != nil {
		t.Fatal(err)
	}

	autreCasse := filepath.Join(filepath.Dir(actif), "GREFFONS")
	if err := extraire(autreCasse, actif); err == nil {
		t.Skip("système sensible à la casse : les deux chemins sont bien distincts")
	}
}

// TestGreffonsParDefautSuitLExecutable vérifie que le dossier ne dépend pas du
// répertoire courant.
//
// Sans cela, un raccourci sur le bureau ferait chercher les greffons ailleurs,
// et ceux qui sont installés seraient ignorés sans un mot.
func TestGreffonsParDefautSuitLExecutable(t *testing.T) {
	defaut := greffonsParDefaut()

	if !filepath.IsAbs(defaut) {
		t.Fatalf("%s n'est pas absolu : il dépendrait du répertoire courant", defaut)
	}
	if filepath.Base(defaut) != "greffons" {
		t.Errorf("dossier %s, attendu un chemin finissant par greffons", defaut)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Skip("le système ne dit pas où est le binaire")
	}
	if resolu, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolu
	}
	if attendu := filepath.Join(filepath.Dir(exe), "greffons"); defaut != attendu {
		t.Errorf("dossier %s, attendu %s", defaut, attendu)
	}
}
