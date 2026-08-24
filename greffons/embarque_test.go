// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package greffons

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestLivresPortentLeContenuAttendu vérifie que le binaire embarque bien tout
// le contenu du dépôt.
//
// Un `//go:embed` qui oublie un dossier ne fait pas échouer la compilation : le
// jeu se construit, et se retrouve sans règles ou sans traduction à
// l'exécution. Ce test est le seul endroit où l'oubli se voit.
func TestLivresPortentLeContenuAttendu(t *testing.T) {
	livres := Livres()

	for _, attendu := range []string{
		"base/manifeste.toml",
		"base/formes.toml",
		"base/palette.toml",
		"base/langue.toml",
		"anglais/manifeste.toml",
		"anglais/langue.toml",
	} {
		if _, err := fs.Stat(livres, attendu); err != nil {
			t.Errorf("%s n'est pas embarqué", attendu)
		}
	}
}

// TestEmbarqueSuitLeDepot vérifie que tout greffon du dépôt est embarqué.
//
// Sans ce contrôle, ajouter une langue reviendrait à la livrer dans le dépôt
// sans la livrer dans le binaire — et l'écart ne se verrait qu'après
// publication.
func TestEmbarqueSuitLeDepot(t *testing.T) {
	surDisque, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	for _, d := range surDisque {
		if !d.IsDir() {
			continue
		}
		if _, err := fs.Stat(Livres(), d.Name()); err != nil {
			t.Errorf("le greffon %s est dans le dépôt mais pas dans le binaire", d.Name())
		}
	}
}

// TestContenuEmbarqueIdentiqueAuDepot compare octet pour octet.
//
// Le contenu embarqué est figé à la compilation : une divergence signifierait
// que le binaire livre autre chose que ce qui est relu en revue de code.
func TestContenuEmbarqueIdentiqueAuDepot(t *testing.T) {
	err := fs.WalkDir(Livres(), ".", func(chemin string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		embarque, err := fs.ReadFile(Livres(), chemin)
		if err != nil {
			return err
		}
		disque, err := os.ReadFile(filepath.FromSlash(chemin))
		if err != nil {
			return err
		}
		if string(embarque) != string(disque) {
			t.Errorf("%s diffère entre le binaire et le dépôt", chemin)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
