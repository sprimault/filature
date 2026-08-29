// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/render"
)

// TestShapeNamesMatchTheContract vérifie que la table des rôles imposés et le
// §6 du contrat de formes nomment les mêmes formes.
//
// La table décide du gabarit qu'une forme recevra, le document dit à l'auteur de
// plugin ce que le jeu ira chercher : une forme présente d'un seul côté est soit
// un nom que rien ne contraint, soit une contrainte que personne n'a annoncée.
func TestShapeNamesMatchTheContract(t *testing.T) {
	dansLeDocument := nomsDeFormesDuContrat(t)

	for nom := range render.ShapeRoles {
		if !dansLeDocument[nom] {
			t.Errorf("render.ShapeRoles impose un rôle à %q, que le §6 du contrat ne nomme pas", nom)
		}
	}
	for nom := range dansLeDocument {
		if _, impose := render.RoleOf(nom); !impose {
			t.Errorf("le §6 du contrat nomme %q, dont render.RoleOf n'impose pas le rôle", nom)
		}
	}
}

// nomsDeFormesDuContrat rend les noms cités par le §6, entre accents graves.
//
// Les points de suspension de « inspector_1 … inspector_5 » ne se lisent pas :
// seules les deux bornes sont citées, et RoleOf les couvre par leur préfixe.
func nomsDeFormesDuContrat(t *testing.T) map[string]bool {
	t.Helper()

	contenu, err := os.ReadFile(filepath.Join(racine, "docs", "contrat-formes.md"))
	if err != nil {
		t.Fatal(err)
	}

	const titre = "## 6. Noms de formes"
	_, apres, trouve := strings.Cut(string(contenu), titre)
	if !trouve {
		t.Fatalf("section %q absente du contrat de formes", titre)
	}
	section, _, _ := strings.Cut(apres, "\n---")

	// Les lignes du tableau seulement : la prose qui le suit abrège la plage en
	// « inspector_1 à 5 », et le « 5 » isolé n'est le nom de rien.
	nom := regexp.MustCompile("`([a-z_0-9]+)`")
	noms := map[string]bool{}
	for _, ligne := range strings.Split(section, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(ligne), "|") {
			continue
		}
		for _, trouvaille := range nom.FindAllStringSubmatch(ligne, -1) {
			noms[trouvaille[1]] = true
		}
	}
	if len(noms) == 0 {
		t.Fatal("aucun nom de forme lu dans le §6 : le contrôle ne dirait rien")
	}
	return noms
}
