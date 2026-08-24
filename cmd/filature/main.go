// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Commande filature : le jeu, et son mode serveur.
//
// Toute la configuration vient des drapeaux, lus et validés ici. Aucun flag ni
// os.Getenv dans internal/ — les dépendances sont injectées.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sprimault/filature/greffons"
)

// version est injectée à la compilation par -ldflags.
var version = "dev"

// main lit les drapeaux et fixe le code de sortie. Tout ce qui peut échouer est
// dans executer.
func main() {
	heberger := flag.Bool("heberger", false, "héberger une partie en réseau")
	rejoindre := flag.String("rejoindre", "", "adresse d'une partie à rejoindre")
	greffons := flag.String("greffons", "greffons", "dossier des greffons")
	partie := flag.String("partie", "", "nom d'une partie enregistrée à reprendre")
	flag.Parse()

	switch flag.Arg(0) {
	case "version":
		fmt.Println("filature", version)
		return
	case "exemples":
		if err := extraire(flag.Arg(1)); err != nil {
			fmt.Fprintln(os.Stderr, "filature:", err)
			os.Exit(1)
		}
		return
	}

	if err := executer(*heberger, *rejoindre, *greffons, *partie); err != nil {
		fmt.Fprintln(os.Stderr, "filature:", err)
		os.Exit(1)
	}
}

// executer assemble les dépendances et lance la boucle de jeu.
func executer(heberger bool, rejoindre, greffons, partie string) error {
	return fmt.Errorf("à implémenter : étape 5")
}

// extraire écrit les greffons livrés dans un dossier, pour servir de modèle.
//
// Le contenu vit dans le binaire, ce qui le met hors de portée de celui qui
// voudrait s'en inspirer. Cette commande est la contrepartie : un traducteur
// recopie « anglais », change le code et les libellés, et pose le résultat dans
// son dossier de greffons.
//
// Un fichier existant n'est jamais écrasé. Quelqu'un qui relance la commande
// sur un dossier où il a déjà travaillé perdrait son travail, et il n'y a pas
// de bonne raison de lui offrir ça.
func extraire(dossier string) error {
	if dossier == "" {
		return errors.New("usage: filature exemples <dossier>")
	}

	return fs.WalkDir(greffons.Livres(), ".", func(chemin string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		cible := filepath.Join(dossier, filepath.FromSlash(chemin))
		if e.IsDir() {
			return os.MkdirAll(cible, 0o750)
		}
		if _, err := os.Stat(cible); err == nil {
			return fmt.Errorf("%s existe deja", cible)
		}

		contenu, err := fs.ReadFile(greffons.Livres(), chemin)
		if err != nil {
			return err
		}
		if err := os.WriteFile(cible, contenu, 0o600); err != nil {
			return err
		}
		fmt.Println(cible)
		return nil
	})
}
