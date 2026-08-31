// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sprimault/evasion/internal/core"
)

// ailleurs est un dossier de plugins actifs qui n'entre en conflit avec aucune
// destination d'essai.
const ailleurs = "/un/dossier/qui/n/est/pas/la/destination"

// TestExtractWritesShippedPlugins vérifie que la commande rend le contenu
// embarqué recopiable.
func TestExtractWritesShippedPlugins(t *testing.T) {
	dossier := t.TempDir()
	if err := extract(dossier, ailleurs); err != nil {
		t.Fatalf("extraction refusée : %v", err)
	}

	for _, attendu := range []string{
		filepath.Join("base", "manifest.toml"),
		filepath.Join("english", "language.toml"),
	} {
		if _, err := os.Stat(filepath.Join(dossier, attendu)); err != nil {
			t.Errorf("%s n'a pas été écrit", attendu)
		}
	}
}

// TestExtractDoesNotOverwrite vérifie qu'un fichier existant arrête la commande.
//
// Un traducteur qui relance l'extraction sur son dossier de travail perdrait
// ses libellés. Le refus est ce qui rend la commande sûre à relancer.
func TestExtractDoesNotOverwrite(t *testing.T) {
	dossier := t.TempDir()
	if err := extract(dossier, ailleurs); err != nil {
		t.Fatalf("première extraction refusée : %v", err)
	}

	temoin := filepath.Join(dossier, "english", "language.toml")
	if err := os.WriteFile(temoin, []byte("# mon travail\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := extract(dossier, ailleurs); err == nil {
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

// TestExtractWithoutFolder vérifie que la commande dit son usage plutôt que
// d'écrire dans le répertoire courant.
func TestExtractWithoutFolder(t *testing.T) {
	if err := extract("", ailleurs); err == nil {
		t.Fatal("un dossier vide a été accepté")
	}
}

// TestExtractRejectsActiveFolder ferme le piège que la commande tendait.
//
// Extraire le contenu livré là où le jeu charge le déclarerait deux fois : une
// fois depuis le binaire, une fois depuis le disque. Deux plugins qui
// définissent la même clé sont un conflit, donc suivre l'invitation cassait le
// chargement.
func TestExtractRejectsActiveFolder(t *testing.T) {
	actif := t.TempDir()

	if err := extract(actif, actif); err == nil {
		t.Fatal("extraction acceptée dans le dossier des plugins actifs")
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

// TestExtractRejectsDifferentCase vérifie que le refus tient sous
// Windows, où « Plugins » et « plugins » désignent le même dossier.
func TestExtractRejectsDifferentCase(t *testing.T) {
	actif := filepath.Join(t.TempDir(), "plugins")
	if err := os.MkdirAll(actif, 0o750); err != nil {
		t.Fatal(err)
	}

	autreCasse := filepath.Join(filepath.Dir(actif), "GREFFONS")
	if err := extract(autreCasse, actif); err == nil {
		t.Skip("système sensible à la casse : les deux chemins sont bien distincts")
	}
}

// TestDefaultPluginsFollowExecutable vérifie que le dossier ne dépend pas du
// répertoire courant.
//
// Sans cela, un raccourci sur le bureau ferait chercher les plugins ailleurs,
// et ceux qui sont installés seraient ignorés sans un mot.
func TestDefaultPluginsFollowExecutable(t *testing.T) {
	defaut := pluginsParDefaut()

	if !filepath.IsAbs(defaut) {
		t.Fatalf("%s n'est pas absolu : il dépendrait du répertoire courant", defaut)
	}
	if filepath.Base(defaut) != "plugins" {
		t.Errorf("dossier %s, attendu un chemin finissant par plugins", defaut)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Skip("le système ne dit pas où est le binaire")
	}
	if resolu, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolu
	}
	if attendu := filepath.Join(filepath.Dir(exe), "plugins"); defaut != attendu {
		t.Errorf("dossier %s, attendu %s", defaut, attendu)
	}
}

// TestCommandRejectsUnknownWord vérifie qu'un mot qui n'est pas une commande
// ne lance pas de partie.
//
// Le cas n'est pas théorique : « exemples » et « valide » ont existé sous ces
// noms jusqu'à ce lot. Sans ce refus, quelqu'un qui garde l'ancien réflexe
// lancerait une partie en croyant validate un plugin.
func TestCommandRejectsUnknownWord(t *testing.T) {
	for _, nom := range []string{"exemples", "valide", "n'importe quoi"} {
		t.Run(nom, func(t *testing.T) {
			traitee, err := command(&strings.Builder{}, nom, "", "", t.TempDir())

			if !traitee {
				t.Error("le mot est passé à la boucle de jeu au lieu d'être refusé")
			}
			if err == nil {
				t.Fatal("accepté sans rien dire")
			}
			if !strings.Contains(err.Error(), "commande inconnue") {
				t.Errorf("message %q, attendu qu'il dise la commande inconnue", err)
			}
		})
	}
}

// TestEmptyCommandLetsPlay vérifie que le double-clic ordinaire passe.
func TestEmptyCommandLetsPlay(t *testing.T) {
	traitee, err := command(&strings.Builder{}, "", "", "", t.TempDir())
	if err != nil {
		t.Fatalf("sans argument : %v", err)
	}
	if traitee {
		t.Error("aucune sous-commande ne devrait avoir répondu")
	}
}

// TestCommandVersion vérifie que le numéro sort sur la sortie qu'on lui donne.
func TestCommandVersion(t *testing.T) {
	var sortie strings.Builder
	if _, err := command(&sortie, "version", "", "", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sortie.String(), version) {
		t.Errorf("sortie %q, attendu qu'elle porte %q", sortie.String(), version)
	}
}

// TestSideAcceptsOneSpellingEach vérifie les trois valeurs de --side.
func TestSideAcceptsOneSpellingEach(t *testing.T) {
	cas := map[string]core.Side{
		"fugitive":   core.SideFugitive,
		"inspectors": core.SideInspectors,
		"watch":      "",
		"":           "",
	}

	for choix, attendu := range cas {
		got, err := camp(choix)
		if err != nil {
			t.Fatalf("--side %q : %v", choix, err)
		}
		if got != attendu {
			t.Errorf("--side %q donne %q, attendu %q", choix, got, attendu)
		}
	}
}

// TestSideRejectsFrenchSpellings vérifie que les anciennes valeurs sont
// refusées et non silencieusement acceptées.
//
// Elles ont existé, et un refus franc vaut mieux qu'une tolérance : une valeur
// acceptée mais non documentée finit par diverger de celle qui l'est.
func TestSideRejectsFrenchSpellings(t *testing.T) {
	for _, choix := range []string{"fugitif", "inspecteurs", "spectateur"} {
		if _, err := camp(choix); err == nil {
			t.Errorf("--side %q accepté alors qu'il n'existe plus", choix)
		}
	}
}
