// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// majNotices réécrit les notices de tiers au lieu de les comparer.
//
// Jamais automatique, pour la même raison que les autres attendus : un fichier
// régénéré sans être relu ne prouve plus rien, et celui-ci porte des obligations
// juridiques.
var majNotices = flag.Bool("maj-notices", false, "réécrire THIRD-PARTY-NOTICES")

// cheminNotices est le fichier livré dans chaque archive, à côté de LICENSE.
var cheminNotices = filepath.Join(racine, "THIRD-PARTY-NOTICES")

// enTeteNotices explique le fichier à qui l'ouvre, et pourquoi il existe.
const enTeteNotices = `Notices des logiciels tiers
===========================

Le binaire Filature incorpore les bibliothèques ci-dessous. Leurs licences
exigent que ces notices accompagnent toute redistribution, forme binaire
comprise.

Ce fichier est généré depuis les dépendances réellement liées au binaire :
« make notices ». Ne pas le modifier à la main.

`

// TestThirdPartyNoticesAreCurrent vérifie que les notices livrées couvrent
// exactement ce que le binaire incorpore.
//
// La licence MIT de BurntSushi/toml, comme la plupart des licences permissives,
// exige que sa notice accompagne « toute copie ou partie substantielle » du
// logiciel — un binaire qui la compile en est une. Sans ce test, une dépendance
// ajoutée entrerait dans les archives sans sa notice, et personne ne s'en
// apercevrait avant que quelqu'un le remarque.
func TestThirdPartyNoticesAreCurrent(t *testing.T) {
	attendu, err := notices()
	if err != nil {
		t.Fatal(err)
	}

	if *majNotices {
		if err := os.WriteFile(cheminNotices, []byte(attendu), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Log("THIRD-PARTY-NOTICES réécrit — relire le diff")
		return
	}

	actuel, err := os.ReadFile(cheminNotices)
	if err != nil {
		t.Fatalf("%v — lancer « make notices »", err)
	}
	if string(actuel) != attendu {
		t.Error("les notices de tiers ne correspondent plus aux dépendances du binaire — " +
			"lancer « make notices » et relire le diff")
	}
}

// notices assemble le fichier depuis les modules réellement liés au binaire.
func notices() (string, error) {
	modules, err := modulesLies()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(enTeteNotices)

	for _, m := range modules {
		texte, err := licenceDe(m.dossier)
		if err != nil {
			return "", fmt.Errorf("%s: %w", m.chemin, err)
		}
		fmt.Fprintf(&b, "-------------------------------------------------------------------------------\n%s %s\n%s\n\n%s\n",
			m.chemin, m.version, m.origine, strings.TrimRight(texte, "\n"))
	}
	return b.String(), nil
}

// module décrit une dépendance liée au binaire.
type module struct {
	chemin  string
	version string
	dossier string
	origine string
}

// modulesLies interroge le compilateur plutôt que go.mod.
//
// go.mod porte aussi ce que seuls les tests et l'outillage utilisent ; ce qui
// compte ici est ce que le binaire publié incorpore, et « go list -deps » sur la
// commande le donne exactement.
func modulesLies() ([]module, error) {
	sortie, err := exec.Command("go", "list", "-deps",
		"-f", "{{if .Module}}{{.Module.Path}}\t{{.Module.Version}}\t{{.Module.Dir}}{{end}}",
		"github.com/sprimault/filature/cmd/filature").Output()
	if err != nil {
		return nil, fmt.Errorf("liste des dépendances : %w", err)
	}

	vus := map[string]module{}
	for _, ligne := range strings.Split(strings.ReplaceAll(string(sortie), "\r\n", "\n"), "\n") {
		champs := strings.Split(strings.TrimSpace(ligne), "\t")
		if len(champs) != 3 || champs[0] == "" {
			continue
		}
		// Le module du jeu lui-même porte sa propre licence, dans LICENSE.
		if champs[0] == "github.com/sprimault/filature" {
			continue
		}
		vus[champs[0]] = module{
			chemin:  champs[0],
			version: champs[1],
			dossier: champs[2],
			origine: "https://" + champs[0],
		}
	}

	if len(vus) == 0 {
		return nil, fmt.Errorf("aucune dépendance trouvée, ce qui n'arrive pas : la commande a-t-elle compilé ?")
	}

	chemins := make([]string, 0, len(vus))
	for c := range vus {
		chemins = append(chemins, c)
	}
	sort.Strings(chemins)

	modules := make([]module, 0, len(chemins))
	for _, c := range chemins {
		modules = append(modules, vus[c])
	}
	return modules, nil
}

// licenceDe lit le fichier de licence d'un module.
//
// Les noms varient d'un projet à l'autre — COPYING chez BurntSushi, LICENSE
// ailleurs, parfois avec une extension. Ne pas en trouver est une erreur et non
// un cas à ignorer : une dépendance sans licence lisible ne peut pas être
// redistribuée en confiance.
func licenceDe(dossier string) (string, error) {
	for _, nom := range []string{
		"LICENSE", "LICENSE.txt", "LICENSE.md",
		"COPYING", "COPYING.txt",
		"LICENCE", "LICENCE.txt",
	} {
		if brut, err := os.ReadFile(filepath.Join(dossier, nom)); err == nil {
			return string(brut), nil
		}
	}
	return "", fmt.Errorf("aucun fichier de licence dans %s", dossier)
}
