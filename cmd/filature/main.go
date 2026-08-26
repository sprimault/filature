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
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sprimault/filature/internal/core"
	"github.com/sprimault/filature/internal/loader"
	"github.com/sprimault/filature/internal/session"
	"github.com/sprimault/filature/plugins"
)

// version est injectée à la compilation par -ldflags.
var version = "dev"

// command traite les sous-commandes qui ne lancent pas de partie, et dit si
// l'une d'elles a répondu.
//
// Séparée de main pour être vérifiable : main fixe des codes de sortie, ce qui
// ne se teste pas. Or c'est précisément ici que se joue le refus d'un mot
// inconnu, et ce refus compte — « exemples » et « valide » ont existé sous ces
// noms, et sans lui l'ancien mot lancerait une partie au lieu de dire qu'il
// n'existe plus.
func command(sortie io.Writer, nom, argument, dossierPlugins string) (traitee bool, err error) {
	switch nom {
	case "":
		// Sans argument, on joue : c'est le cas ordinaire du double-clic.
		return false, nil

	case "version":
		_, err = fmt.Fprintln(sortie, "filature", version)
		return true, err

	case "examples":
		return true, extract(argument, dossierPlugins)

	case "validate":
		return true, validate(sortie, argument)

	default:
		return true, fmt.Errorf("commande inconnue %q, attendu version, examples ou validate", nom)
	}
}

// main lit les drapeaux et fixe le code de sortie. Tout ce qui peut échouer est
// dans command et dans run.
func main() {
	heberger := flag.Bool("host", false, "héberger une partie en réseau")
	rejoindre := flag.String("join", "", "adresse d'une partie à rejoindre")
	dossierPlugins := flag.String("plugins", pluginsParDefaut(), "dossier des plugins")
	partie := flag.String("game", "", "nom d'une partie enregistrée à reprendre")
	cote := flag.String("side", "inspectors", "camp joué : fugitive, inspectors, ou watch pour regarder")
	preset := flag.String("preset", "ville", "préréglage : quartier, faubourg ou ville")
	graine := flag.Int64("seed", 1, "graine de la partie")

	// Alias de la sous-commande. Les trois sous-commandes ne lancent pas le jeu
	// et deux d'entre elles prennent un opérande, ce qui les rend maladroites en
	// drapeau ; mais --version est ce qu'on essaie en premier, et le refuser
	// donnerait une erreur là où la réponse est évidente.
	versionSeule := flag.Bool("version", false, "afficher la version puis sortir")
	flag.Parse()

	nom := flag.Arg(0)
	if *versionSeule {
		nom = "version"
	}

	traitee, err := command(os.Stdout, nom, flag.Arg(1), *dossierPlugins)
	if err != nil {
		fmt.Fprintln(os.Stderr, "filature:", err)
		os.Exit(1)
	}
	if traitee {
		return
	}

	o, err := options(*cote, *preset, *graine, *dossierPlugins)
	if err != nil {
		fmt.Fprintln(os.Stderr, "filature:", err)
		os.Exit(1)
	}

	if err := run(o, *heberger, *rejoindre, *partie); err != nil {
		fmt.Fprintln(os.Stderr, "filature:", err)
		os.Exit(1)
	}
}

// sameFolder compare deux chemins après résolution.
//
// La comparaison est insensible à la casse : Windows l'est, et « Plugins »
// y désigne le même dossier que « plugins ». Se tromper dans ce sens fait
// refuser une extraction légitime, ce qui se voit et se corrige ; l'inverse
// laisserait passer celle qui casse le chargement.
func sameFolder(a, b string) bool {
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

// pluginsParDefaut situe le dossier des plugins à côté de l'exécutable.
//
// Et non dans le répertoire courant : un raccourci sur le bureau, un lancement
// depuis ailleurs, et le jeu chercherait dans un dossier qui n'est pas le sien
// sans que rien ne le signale — les plugins installés seraient simplement
// ignorés.
//
// Le repli sur un chemin relatif ne couvre qu'un cas où le système refuse de
// dire où est le binaire, ce qui n'arrive pas sur les cibles publiées.
func pluginsParDefaut() string {
	exe, err := os.Executable()
	if err != nil {
		return "plugins"
	}
	if resolu, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolu
	}
	return filepath.Join(filepath.Dir(exe), "plugins")
}

// validate contrôle un plugin avant qu'il soit installé, et affiche son
// fingerprint quand il tient.
//
// L'empreinte est ce qu'un auteur publie à côté de son plugin : elle porte sur
// le contenu et pas sur le numéro de version, donc elle distingue deux
// « 1.2.0 » qui ne sont pas le même fichier.
func validate(sortie io.Writer, dossier string) error {
	if dossier == "" {
		return errors.New("usage: filature validate <dossier>")
	}
	if err := loader.Validate(dossier); err != nil {
		return err
	}

	somme, err := loader.Fingerprint(dossier)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(sortie, "%s: valide\nempreinte: %s\n", dossier, somme)
	return err
}

// run lance la boucle de jeu, ou dit ce qui n'est pas encore écrit.
//
// Les options arrivent construites : le camp est un paramètre de partie, et
// l'écran de nouvelle partie devra le fournir sans contourner cette fonction.
func run(o session.Options, heberger bool, rejoindre, partie string) error {
	switch {
	case heberger || rejoindre != "":
		return errors.New("à implémenter : étape 12")
	case partie != "":
		return errors.New("à implémenter : étape 8")
	}

	// L'abandon n'est pas une panne : le joueur a tapé « q », le binaire sort
	// sans rien signaler.
	if _, err := session.Run(o); err != nil && !errors.Is(err, session.ErrAbandon) {
		return err
	}
	return nil
}

// options assemble ce que la session attend à partir des drapeaux.
//
// C'est le seul endroit du binaire qui les lit : `internal/` reçoit ses
// dépendances construites, jamais un `flag`.
func options(cote, preset string, graine int64, dossierPlugins string) (session.Options, error) {
	choisi, err := camp(cote)
	if err != nil {
		return session.Options{}, err
	}

	reglage, connu := core.PresetByKey(preset)
	if !connu {
		return session.Options{}, fmt.Errorf("préréglage %q inconnu", preset)
	}

	return session.Options{
		Seed:     graine,
		Settings: reglage.Settings,
		Human:    choisi,
		Plugins:  dossierPlugins,
		In:       os.Stdin,
		Out:      os.Stdout,
	}, nil
}

// camp traduit le drapeau --side en côté du noyau.
//
// Vide vaut spectateur : deux cerveaux jouent et personne ne saisit rien. C'est
// aussi ce qui permet de dérouler une partie entière sans intervention.
//
// Une seule orthographe acceptée par valeur. Les formes françaises ont existé
// et sont refusées : une interface qui en accepte deux n'en documente qu'une,
// et celle qu'on ne documente pas finit par diverger sans que personne s'en
// aperçoive.
func camp(choix string) (core.Side, error) {
	switch choix {
	case "fugitive":
		return core.SideFugitive, nil
	case "inspectors":
		return core.SideInspectors, nil
	case "", "watch":
		return "", nil
	default:
		return "", fmt.Errorf("camp %q inconnu, attendu fugitive, inspectors ou watch", choix)
	}
}

// extract écrit les plugins livrés dans un dossier, pour servir de modèle.
//
// Le contenu vit dans le binaire, ce qui le met hors de portée de celui qui
// voudrait s'en inspirer. Cette commande est la contrepartie : un traducteur
// recopie « english », change le code et les libellés, et pose le résultat dans
// son dossier de plugins.
//
// Un fichier existant n'est jamais écrasé. Quelqu'un qui relance la commande
// sur un dossier où il a déjà travaillé perdrait son travail, et il n'y a pas
// de bonne raison de lui offrir ça.
//
// Le dossier des plugins actifs est refusé, et c'est le piège que la commande
// tendait : le contenu livré est déjà dans le binaire, l'extraire là où le jeu
// charge reviendrait à le déclarer deux fois, et deux plugins qui définissent
// la même clé sont un conflit. Suivre l'invitation cassait le chargement.
func extract(dossier, actifs string) error {
	if dossier == "" {
		return errors.New("usage: filature examples <dossier>")
	}
	if sameFolder(dossier, actifs) {
		return fmt.Errorf("%s est le dossier des plugins actifs : le contenu livré y serait déclaré deux fois", dossier)
	}

	return fs.WalkDir(plugins.Shipped(), ".", func(chemin string, e fs.DirEntry, err error) error {
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

		contenu, err := fs.ReadFile(plugins.Shipped(), chemin)
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
