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
	"strings"

	livres "github.com/sprimault/filature/greffons"
	"github.com/sprimault/filature/internal/greffons"
)

// version est injectée à la compilation par -ldflags.
var version = "dev"

// main lit les drapeaux et fixe le code de sortie. Tout ce qui peut échouer est
// dans executer.
func main() {
	heberger := flag.Bool("heberger", false, "héberger une partie en réseau")
	rejoindre := flag.String("rejoindre", "", "adresse d'une partie à rejoindre")
	dossierGreffons := flag.String("greffons", greffonsParDefaut(), "dossier des greffons")
	partie := flag.String("partie", "", "nom d'une partie enregistrée à reprendre")
	flag.Parse()

	switch flag.Arg(0) {
	case "version":
		fmt.Println("filature", version)
		return
	case "exemples":
		if err := extraire(flag.Arg(1), *dossierGreffons); err != nil {
			fmt.Fprintln(os.Stderr, "filature:", err)
			os.Exit(1)
		}
		return
	case "valide":
		if err := valider(flag.Arg(1)); err != nil {
			fmt.Fprintln(os.Stderr, "filature:", err)
			os.Exit(1)
		}
		return
	}

	if err := executer(*heberger, *rejoindre, *dossierGreffons, *partie); err != nil {
		fmt.Fprintln(os.Stderr, "filature:", err)
		os.Exit(1)
	}
}

// memeDossier compare deux chemins après résolution.
//
// La comparaison est insensible à la casse : Windows l'est, et « Greffons »
// y désigne le même dossier que « greffons ». Se tromper dans ce sens fait
// refuser une extraction légitime, ce qui se voit et se corrige ; l'inverse
// laisserait passer celle qui casse le chargement.
func memeDossier(a, b string) bool {
	absA, err := filepath.Abs(a)
	if err != nil {
		return false
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false
	}
	return strings.EqualFold(filepath.Clean(absA), filepath.Clean(absB))
}

// greffonsParDefaut situe le dossier des greffons à côté de l'exécutable.
//
// Et non dans le répertoire courant : un raccourci sur le bureau, un lancement
// depuis ailleurs, et le jeu chercherait dans un dossier qui n'est pas le sien
// sans que rien ne le signale — les greffons installés seraient simplement
// ignorés.
//
// Le repli sur un chemin relatif ne couvre qu'un cas où le système refuse de
// dire où est le binaire, ce qui n'arrive pas sur les cibles publiées.
func greffonsParDefaut() string {
	exe, err := os.Executable()
	if err != nil {
		return "greffons"
	}
	if resolu, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolu
	}
	return filepath.Join(filepath.Dir(exe), "greffons")
}

// valider contrôle un greffon avant qu'il soit installé, et affiche son
// empreinte quand il tient.
//
// L'empreinte est ce qu'un auteur publie à côté de son greffon : elle porte sur
// le contenu et pas sur le numéro de version, donc elle distingue deux
// « 1.2.0 » qui ne sont pas le même fichier.
func valider(dossier string) error {
	if dossier == "" {
		return errors.New("usage: filature valide <dossier>")
	}
	if err := greffons.Valider(dossier); err != nil {
		return err
	}

	somme, err := greffons.Empreinte(dossier)
	if err != nil {
		return err
	}
	fmt.Printf("%s: valide\nempreinte: %s\n", dossier, somme)
	return nil
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
//
// Le dossier des greffons actifs est refusé, et c'est le piège que la commande
// tendait : le contenu livré est déjà dans le binaire, l'extraire là où le jeu
// charge reviendrait à le déclarer deux fois, et deux greffons qui définissent
// la même clé sont un conflit. Suivre l'invitation cassait le chargement.
func extraire(dossier, actifs string) error {
	if dossier == "" {
		return errors.New("usage: filature exemples <dossier>")
	}
	if memeDossier(dossier, actifs) {
		return fmt.Errorf("%s est le dossier des greffons actifs : le contenu livré y serait déclaré deux fois", dossier)
	}

	return fs.WalkDir(livres.Livres(), ".", func(chemin string, e fs.DirEntry, err error) error {
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

		contenu, err := fs.ReadFile(livres.Livres(), chemin)
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
