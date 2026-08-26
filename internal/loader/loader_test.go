// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sprimault/filature/internal/core"
	"github.com/sprimault/filature/plugins"
)

// source monte un système de fichiers en mémoire depuis des couples chemin /
// contenu.
//
// Écrire les manifestes d'essai en TOML plutôt que de bâtir des structures Go
// est délibéré : c'est le décodage qu'on teste autant que la validation, et un
// manifeste construit en mémoire sauterait justement l'étape qui casse.
func source(fichiers map[string]string) fstest.MapFS {
	fs := fstest.MapFS{}
	for chemin, contenu := range fichiers {
		fs[chemin] = &fstest.MapFile{Data: []byte(contenu)}
	}
	return fs
}

// manifesteValide rend un plugin de règles minimal mais complet.
func manifesteValide(nom string) string {
	return `name = "` + nom + `"
version = "1.0.0"
effects_version = 2
rules = true

[ability.` + nom + `]
name = "Capacité"
side = "inspectors"
trigger = "inspectors_phase"

  [[ability.` + nom + `.effect]]
  type = "change_range"
  target = "current_piece"
  value = 1
`
}

// TestLoadsShippedContent vérifie que le jeu démarre avec ses règles.
//
// C'est le test qui compte le plus de ce fichier : le contenu livré passe par
// le même chargeur qu'un plugin tiers, donc s'il se charge, ce chemin est
// exercé à chaque partie plutôt qu'une fois de temps en temps.
func TestLoadsShippedContent(t *testing.T) {
	r, err := Load(plugins.Shipped(), "")
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}

	for _, cle := range []string{"lookout", "runner", "tracker", "blocker", "chief"} {
		if _, connue := r.Abilities[cle]; !connue {
			t.Errorf("la capacité %s manque", cle)
		}
	}
	for _, cle := range []core.Expense{
		core.ExpenseDoubleStep, core.ExpenseSilence, core.ExpenseWipeTrails,
		core.ExpenseChangeZone, core.ExpenseMurder,
	} {
		if _, connue := r.Expenses[cle]; !connue {
			t.Errorf("la dépense %s manque", cle)
		}
	}
	if _, connu := r.Modes["strangling"]; !connu {
		t.Fatal("l'étranglement manque : aucune zone ne se fermerait jamais")
	}

	if len(r.Manifest) != 2 {
		t.Errorf("%d entrées de manifeste, attendu 2", len(r.Manifest))
	}
}

// TestKeyCarriedFromTable vérifie que chaque entrée connaît sa propre clé.
//
// Elle n'est pas dans le fichier : c'est le nom de la table TOML. Sans report,
// une capacité ne saurait pas comment on la désigne, et l'interface afficherait
// une chaîne vide là où le joueur attend un nom.
func TestKeyCarriedFromTable(t *testing.T) {
	r, err := Load(plugins.Shipped(), "")
	if err != nil {
		t.Fatal(err)
	}

	for cle, c := range r.Abilities {
		if c.Key != cle {
			t.Errorf("capacité %s porte la clé %q", cle, c.Key)
		}
	}
	for cle, d := range r.Expenses {
		if core.Expense(d.Key) != cle {
			t.Errorf("dépense %s porte la clé %q", cle, d.Key)
		}
	}
}

// TestDiskPluginTakesSamePath vérifie qu'un plugin posé à la main
// s'ajoute à ce qui est livré.
func TestDiskPluginTakesSamePath(t *testing.T) {
	racine := t.TempDir()
	dossier := filepath.Join(racine, "essai")
	if err := os.MkdirAll(dossier, 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dossier, ManifestName), manifesteValide("essai"))

	r, err := Load(plugins.Shipped(), racine)
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}

	if _, connue := r.Abilities["essai"]; !connue {
		t.Error("la capacité du plugin posé sur le disque manque")
	}
	if _, connue := r.Abilities["lookout"]; !connue {
		t.Error("le contenu livré a disparu")
	}
	if len(r.Manifest) != 3 {
		t.Errorf("%d entrées de manifeste, attendu 3", len(r.Manifest))
	}
}

// TestMissingPluginFolder vérifie que l'installation ordinaire démarre.
//
// Personne n'a rien ajouté, il n'y a donc pas de dossier : refuser de démarrer
// pour ça rendrait le binaire inutilisable tel qu'il est livré.
func TestMissingPluginFolder(t *testing.T) {
	r, err := Load(plugins.Shipped(), filepath.Join(t.TempDir(), "rien"))
	if err != nil {
		t.Fatalf("un dossier absent fait échouer le chargement : %v", err)
	}
	if len(r.Abilities) == 0 {
		t.Error("le contenu livré n'a pas été chargé")
	}
}

// TestKeyConflict vérifie qu'une clé déjà prise arrête le chargement.
//
// Deux plugins qui définissent la même capacité doivent le dire : écraser
// silencieusement donnerait une partie dont les règles dépendent de l'ordre
// alphabétique des dossiers.
func TestKeyConflict(t *testing.T) {
	_, err := Load(source(map[string]string{
		"un/manifest.toml":   manifesteValide("un"),
		"deux/manifest.toml": strings.ReplaceAll(manifesteValide("deux"), "ability.deux", "ability.un"),
	}), "")

	if err == nil {
		t.Fatal("deux plugins définissent la même capacité sans que rien ne le dise")
	}
	if !strings.Contains(err.Error(), "deja definie") {
		t.Errorf("message peu clair : %v", err)
	}
}

// TestRejections rassemble les manquements de docs/plugins.md §9.
//
// Chacun fait échouer le chargement entier : un plugin à moitié actif est pire
// qu'un plugin absent, et un manifeste écarté en silence se découvre en
// partie.
func TestRejections(t *testing.T) {
	cas := []struct {
		nom       string
		manifeste string
		attendu   string
	}{
		{
			nom:       "nom manquant",
			manifeste: "version = \"1.0.0\"\n",
			attendu:   "nom manquant",
		},
		{
			nom:       "version manquante",
			manifeste: "name = \"essai\"\n",
			attendu:   "version manquante",
		},
		{
			nom:       "nom qui ne suit pas le dossier",
			manifeste: "name = \"autre\"\nversion = \"1.0.0\"\n",
			attendu:   "dossier",
		},
		{
			nom:       "champ inconnu",
			manifeste: "name = \"essai\"\nversion = \"1.0.0\"\ncouleur = \"bleu\"\n",
			attendu:   "champs inconnus",
		},
		{
			nom: "regles fausses avec une capacité",
			manifeste: `name = "essai"
version = "1.0.0"
effects_version = 2
rules = false

[ability.x]
name = "X"
side = "inspectors"
`,
			attendu: "rules = false",
		},
		{
			nom:       "rules fausses avec un module",
			manifeste: "name = \"essai\"\nversion = \"1.0.0\"\nrules = false\nwasm = \"m.wasm\"\n",
			attendu:   "module wasm",
		},
		{
			nom: "version d'effets inconnue",
			manifeste: `name = "essai"
version = "1.0.0"
effects_version = 99
rules = true

[ability.x]
name = "X"
side = "inspectors"
`,
			attendu: "effects_version 99",
		},
		{
			nom: "camp inconnu",
			manifeste: `name = "essai"
version = "1.0.0"
effects_version = 2
rules = true

[ability.x]
name = "X"
side = "referees"
`,
			attendu: "camp",
		},
		{
			nom: "primitive inconnue",
			manifeste: `name = "essai"
version = "1.0.0"
effects_version = 2
rules = true

[ability.x]
name = "X"
side = "inspectors"

  [[ability.x.effect]]
  type = "faire_pleuvoir"
`,
			attendu: "inconnu",
		},
		{
			nom: "differer imbriqué",
			manifeste: `name = "essai"
version = "1.0.0"
effects_version = 2
rules = true

[mode.x]
name = "X"
trigger = "turn_end"

  [[mode.x.effect]]
  type = "defer"
  duration = 1

    [[mode.x.effect.then]]
    type = "defer"
    duration = 1

      [[mode.x.effect.then.then]]
      type = "end_game"
`,
			attendu: "imbrique",
		},
		{
			nom: "differer sans suite",
			manifeste: `name = "essai"
version = "1.0.0"
effects_version = 2
rules = true

[mode.x]
name = "X"
trigger = "turn_end"

  [[mode.x.effect]]
  type = "defer"
  duration = 1
`,
			attendu: "sans puis",
		},
		{
			nom: "un bot et des effets",
			manifeste: `name = "essai"
version = "1.0.0"
effects_version = 2
rules = true

[bot]
side = "fugitive"
command = "mon-bot"

[ability.x]
name = "X"
side = "inspectors"
`,
			attendu: "bot et des effets",
		},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			_, err := Load(source(map[string]string{"essai/manifest.toml": c.manifeste}), "")
			if err == nil {
				t.Fatal("accepté sans rien dire")
			}
			if !strings.Contains(err.Error(), c.attendu) {
				t.Errorf("message %q, attendu qu'il contienne %q", err, c.attendu)
			}
			if !strings.Contains(err.Error(), ManifestName) {
				t.Errorf("le message ne dit pas dans quel fichier : %v", err)
			}
		})
	}
}

// TestFolderWithoutManifestIgnored vérifie qu'un dossier de travail laissé à côté
// n'empêche pas le jeu de démarrer.
func TestFolderWithoutManifestIgnored(t *testing.T) {
	r, err := Load(source(map[string]string{
		"un/manifest.toml":    manifesteValide("un"),
		"brouillon/notes.txt": "rien à voir",
	}), "")
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	if len(r.Manifest) != 1 {
		t.Errorf("%d entrées, attendu 1", len(r.Manifest))
	}
}

// TestFingerprintOverContent vérifie ce qui distingue l'empreinte d'un numéro de
// version.
//
// Deux plugins qui se disent « 1.0.0 » sans être identiques doivent se
// distinguer : cas normal pendant le développement d'un mod, et cas litigieux
// en réseau.
func TestFingerprintOverContent(t *testing.T) {
	un := source(map[string]string{"g/manifest.toml": manifesteValide("g")})
	pareil := source(map[string]string{"g/manifest.toml": manifesteValide("g")})
	autre := source(map[string]string{
		"g/manifest.toml": strings.Replace(manifesteValide("g"), "value = 1", "value = 8", 1),
	})

	a, err := fingerprint(un, "g")
	if err != nil {
		t.Fatal(err)
	}
	b, err := fingerprint(pareil, "g")
	if err != nil {
		t.Fatal(err)
	}
	c, err := fingerprint(autre, "g")
	if err != nil {
		t.Fatal(err)
	}

	if a != b {
		t.Error("deux plugins identiques ont des empreintes différentes")
	}
	if a == c {
		t.Error("changer la portée d'une capacité ne change pas l'empreinte")
	}
}

// TestFingerprintCountsNames vérifie qu'un fichier renommé se voit.
//
// Sans le nom dans la somme, déplace une ligne d'un fichier à l'autre — ou
// renommer une forme — laisserait l'empreinte inchangée alors que le plugin
// ne se charge plus pareil.
func TestFingerprintCountsNames(t *testing.T) {
	a, err := fingerprint(source(map[string]string{
		"g/manifest.toml": manifesteValide("g"),
		"g/formes.toml":   "x = 1\n",
	}), "g")
	if err != nil {
		t.Fatal(err)
	}

	b, err := fingerprint(source(map[string]string{
		"g/manifest.toml": manifesteValide("g"),
		"g/palette.toml":  "x = 1\n",
	}), "g")
	if err != nil {
		t.Fatal(err)
	}

	if a == b {
		t.Error("renommer un fichier ne change pas l'empreinte")
	}
}

// TestPublicFingerprintReadsDisk vérifie la commodité offerte à
// « filature valide ».
func TestPublicFingerprintReadsDisk(t *testing.T) {
	racine := t.TempDir()
	dossier := filepath.Join(racine, "essai")
	if err := os.MkdirAll(dossier, 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dossier, ManifestName), manifesteValide("essai"))

	somme, err := Fingerprint(dossier)
	if err != nil {
		t.Fatalf("empreinte : %v", err)
	}
	if len(somme) != 64 {
		t.Errorf("empreinte de %d caractères, attendu 64", len(somme))
	}
}

// TestValidateReturnsEveryFailure vérifie ce que docs/plugins.md attend :
// une liste, pas un aller-retour par erreur.
//
// Un auteur qui corrige une ligne pour en découvrir une autre au lancement
// suivant y passe la soirée, et c'est exactement ce que la commande doit lui
// éviter.
func TestValidateReturnsEveryFailure(t *testing.T) {
	racine := t.TempDir()
	dossier := filepath.Join(racine, "casse")
	if err := os.MkdirAll(dossier, 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dossier, ManifestName), `name = "autre"
rules = false

[ability.x]
side = "referees"

  [[ability.x.effect]]
  type = "faire_pleuvoir"
`)

	err := Validate(dossier)
	if err == nil {
		t.Fatal("un manifeste truffé de fautes est accepté")
	}

	for _, attendu := range []string{
		"version manquante",
		"dossier",
		"rules = false",
		"camp",
		"ability.x.effect[0]",
		"inconnu",
	} {
		if !strings.Contains(err.Error(), attendu) {
			t.Errorf("le rapport ne mentionne pas %q :\n%v", attendu, err)
		}
	}
}

// TestValidateAcceptsShippedContent vérifie que ce que le jeu embarque passe le
// contrôle qu'il impose aux autres.
func TestValidateAcceptsShippedContent(t *testing.T) {
	for _, dossier := range []string{"base", "english"} {
		t.Run(dossier, func(t *testing.T) {
			if err := Validate(filepath.Join("..", "..", "plugins", dossier)); err != nil {
				t.Errorf("le contenu livré ne passe pas son propre contrôle : %v", err)
			}
		})
	}
}

// writeFile pose un fichier d'essai, avec les droits que gosec exige.
func writeFile(t *testing.T, chemin, contenu string) {
	t.Helper()
	if err := os.WriteFile(chemin, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
}
