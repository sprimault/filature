// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/render"
)

// TestTemplateTableMatchesTheCode vérifie que la table des gabarits du contrat
// de formes dit ce que le chargeur applique.
//
// C'est la table qu'un auteur de plugin lit avant de poser le premier point, et
// elle a donné le marqueur pour vertical alors qu'il est au sol : une flèche
// écrite d'après elle passait la validation et ressortait retournée, le contrat
// donnant alors raison au plugin contre le moteur.
func TestTemplateTableMatchesTheCode(t *testing.T) {
	lignes := tableDesGabarits(t)

	plans := map[render.Plane]string{
		render.PlaneGround:   "sol",
		render.PlaneVertical: "vertical",
	}
	roles := map[render.Role]string{
		render.RolePiece:    "Pion",
		render.RoleMarker:   "Marqueur",
		render.RoleBuilding: "Bâtiment",
	}

	for role, gabarit := range render.Templates {
		ligne, present := lignes[roles[role]]
		if !present {
			t.Errorf("le rôle %s n'a pas de ligne dans la table des gabarits", role)
			continue
		}
		if plan := plans[gabarit.Plane]; ligne.plan != plan {
			t.Errorf("%s: la table le dit %q, le code %q", role, ligne.plan, plan)
		}
		if ligne.traits != gabarit.MaxStrokes {
			t.Errorf("%s: la table annonce %d traits au plus, le code en accepte %d",
				role, ligne.traits, gabarit.MaxStrokes)
		}
	}
}

// gabaritPublie est ce que la table du contrat annonce pour un rôle.
type gabaritPublie struct {
	plan   string
	traits int
}

// tableDesGabarits lit la table « Gabarits » du contrat de formes.
//
// Les colonnes x et y ne se relisent pas : le bâtiment y porte un tiret et une
// hauteur, là où les deux autres portent des bornes. Le plan et le plafond de
// traits suffisent — ce sont eux qui ont divergé.
func tableDesGabarits(t *testing.T) map[string]gabaritPublie {
	t.Helper()

	contenu, err := os.ReadFile(filepath.Join(racine, "docs", "contrat-formes.md"))
	if err != nil {
		t.Fatal(err)
	}

	const titre = "### Gabarits"
	_, apres, trouve := strings.Cut(string(contenu), titre)
	if !trouve {
		t.Fatalf("section %q absente du contrat de formes", titre)
	}
	section, _, _ := strings.Cut(apres, "\n#")

	ligne := regexp.MustCompile(`^\|\s*(\S+)\s*\|\s*(\S+)\s*\|.*\|.*\|\s*(\d+)\s*\|$`)
	lignes := map[string]gabaritPublie{}
	for _, brute := range strings.Split(section, "\n") {
		trouvaille := ligne.FindStringSubmatch(strings.TrimSpace(brute))
		if trouvaille == nil {
			continue
		}
		traits, err := strconv.Atoi(trouvaille[3])
		if err != nil {
			t.Fatal(err)
		}
		lignes[trouvaille[1]] = gabaritPublie{trouvaille[2], traits}
	}
	if len(lignes) != len(render.Templates) {
		t.Fatalf("%d lignes lues dans la table des gabarits pour %d rôles",
			len(lignes), len(render.Templates))
	}
	return lignes
}
