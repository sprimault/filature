// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestExtraireEcritLesGreffonsLivres vérifie que la commande rend le contenu
// embarqué recopiable.
func TestExtraireEcritLesGreffonsLivres(t *testing.T) {
	dossier := t.TempDir()
	if err := extraire(dossier); err != nil {
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
	if err := extraire(dossier); err != nil {
		t.Fatalf("première extraction refusée : %v", err)
	}

	temoin := filepath.Join(dossier, "anglais", "langue.toml")
	if err := os.WriteFile(temoin, []byte("# mon travail\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := extraire(dossier); err == nil {
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
	if err := extraire(""); err == nil {
		t.Fatal("un dossier vide a été accepté")
	}
}
