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
	return `nom = "` + nom + `"
version = "1.0.0"
version_effets = 1
regles = true

[capacite.` + nom + `]
nom = "Capacité"
camp = "inspecteurs"
declenchement = "phase_inspecteurs"

  [[capacite.` + nom + `.effet]]
  type = "modifier_portee"
  cible = "pion_courant"
  valeur = 1
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

	for _, cle := range []string{"guetteur", "coureur", "traqueur", "barreur", "chef"} {
		if _, connue := r.Capacites[cle]; !connue {
			t.Errorf("la capacité %s manque", cle)
		}
	}
	for _, cle := range []core.Expense{
		core.ExpenseDoubleStep, core.ExpenseSilence, core.ExpenseWipeTrails,
		core.ExpenseChangeZone, core.ExpenseMurder,
	} {
		if _, connue := r.Depenses[cle]; !connue {
			t.Errorf("la dépense %s manque", cle)
		}
	}
	if _, connu := r.Modes["etranglement"]; !connu {
		t.Fatal("l'étranglement manque : aucune zone ne se fermerait jamais")
	}

	if len(r.Manifeste) != 2 {
		t.Errorf("%d entrées de manifeste, attendu 2", len(r.Manifeste))
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

	for cle, c := range r.Capacites {
		if c.Key != cle {
			t.Errorf("capacité %s porte la clé %q", cle, c.Key)
		}
	}
	for cle, d := range r.Depenses {
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

	if _, connue := r.Capacites["essai"]; !connue {
		t.Error("la capacité du plugin posé sur le disque manque")
	}
	if _, connue := r.Capacites["guetteur"]; !connue {
		t.Error("le contenu livré a disparu")
	}
	if len(r.Manifeste) != 3 {
		t.Errorf("%d entrées de manifeste, attendu 3", len(r.Manifeste))
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
	if len(r.Capacites) == 0 {
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
		"un/manifeste.toml":   manifesteValide("un"),
		"deux/manifeste.toml": strings.ReplaceAll(manifesteValide("deux"), "capacite.deux", "capacite.un"),
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
			manifeste: "nom = \"essai\"\n",
			attendu:   "version manquante",
		},
		{
			nom:       "nom qui ne suit pas le dossier",
			manifeste: "nom = \"autre\"\nversion = \"1.0.0\"\n",
			attendu:   "dossier",
		},
		{
			nom:       "champ inconnu",
			manifeste: "nom = \"essai\"\nversion = \"1.0.0\"\ncouleur = \"bleu\"\n",
			attendu:   "champs inconnus",
		},
		{
			nom: "regles fausses avec une capacité",
			manifeste: `nom = "essai"
version = "1.0.0"
version_effets = 1
regles = false

[capacite.x]
nom = "X"
camp = "inspecteurs"
`,
			attendu: "regles = false",
		},
		{
			nom:       "regles fausses avec un module",
			manifeste: "nom = \"essai\"\nversion = \"1.0.0\"\nregles = false\nwasm = \"m.wasm\"\n",
			attendu:   "module wasm",
		},
		{
			nom: "version d'effets inconnue",
			manifeste: `nom = "essai"
version = "1.0.0"
version_effets = 99
regles = true

[capacite.x]
nom = "X"
camp = "inspecteurs"
`,
			attendu: "version_effets 99",
		},
		{
			nom: "camp inconnu",
			manifeste: `nom = "essai"
version = "1.0.0"
version_effets = 1
regles = true

[capacite.x]
nom = "X"
camp = "arbitres"
`,
			attendu: "camp",
		},
		{
			nom: "primitive inconnue",
			manifeste: `nom = "essai"
version = "1.0.0"
version_effets = 1
regles = true

[capacite.x]
nom = "X"
camp = "inspecteurs"

  [[capacite.x.effet]]
  type = "faire_pleuvoir"
`,
			attendu: "inconnu",
		},
		{
			nom: "differer imbriqué",
			manifeste: `nom = "essai"
version = "1.0.0"
version_effets = 1
regles = true

[mode.x]
nom = "X"
declenchement = "fin_de_tour"

  [[mode.x.effet]]
  type = "differer"
  duree = 1

    [[mode.x.effet.puis]]
    type = "differer"
    duree = 1

      [[mode.x.effet.puis.puis]]
      type = "fin_partie"
`,
			attendu: "imbrique",
		},
		{
			nom: "differer sans suite",
			manifeste: `nom = "essai"
version = "1.0.0"
version_effets = 1
regles = true

[mode.x]
nom = "X"
declenchement = "fin_de_tour"

  [[mode.x.effet]]
  type = "differer"
  duree = 1
`,
			attendu: "sans puis",
		},
		{
			nom: "un bot et des effets",
			manifeste: `nom = "essai"
version = "1.0.0"
version_effets = 1
regles = true

[bot]
camp = "fugitif"
commande = "mon-bot"

[capacite.x]
nom = "X"
camp = "inspecteurs"
`,
			attendu: "bot et des effets",
		},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			_, err := Load(source(map[string]string{"essai/manifeste.toml": c.manifeste}), "")
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
		"un/manifeste.toml":   manifesteValide("un"),
		"brouillon/notes.txt": "rien à voir",
	}), "")
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	if len(r.Manifeste) != 1 {
		t.Errorf("%d entrées, attendu 1", len(r.Manifeste))
	}
}

// TestFingerprintOverContent vérifie ce qui distingue l'empreinte d'un numéro de
// version.
//
// Deux plugins qui se disent « 1.0.0 » sans être identiques doivent se
// distinguer : cas normal pendant le développement d'un mod, et cas litigieux
// en réseau.
func TestFingerprintOverContent(t *testing.T) {
	un := source(map[string]string{"g/manifeste.toml": manifesteValide("g")})
	pareil := source(map[string]string{"g/manifeste.toml": manifesteValide("g")})
	autre := source(map[string]string{
		"g/manifeste.toml": strings.Replace(manifesteValide("g"), "valeur = 1", "valeur = 8", 1),
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
		"g/manifeste.toml": manifesteValide("g"),
		"g/formes.toml":    "x = 1\n",
	}), "g")
	if err != nil {
		t.Fatal(err)
	}

	b, err := fingerprint(source(map[string]string{
		"g/manifeste.toml": manifesteValide("g"),
		"g/palette.toml":   "x = 1\n",
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
// next y passe la soirée, et c'est exactement ce que la command doit lui
// éviter.
func TestValidateReturnsEveryFailure(t *testing.T) {
	racine := t.TempDir()
	dossier := filepath.Join(racine, "casse")
	if err := os.MkdirAll(dossier, 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dossier, ManifestName), `nom = "autre"
regles = false

[capacite.x]
camp = "arbitres"

  [[capacite.x.effet]]
  type = "faire_pleuvoir"
`)

	err := Validate(dossier)
	if err == nil {
		t.Fatal("un manifeste truffé de fautes est accepté")
	}

	for _, attendu := range []string{
		"version manquante",
		"dossier",
		"regles = false",
		"camp",
		"capacite.x.effet[0]",
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
	for _, dossier := range []string{"base", "anglais"} {
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
