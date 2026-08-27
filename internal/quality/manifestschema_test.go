// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// schemaManifeste compile le contrat publié.
//
// Le validateur ne vit que dans les tests : il n'entre dans aucun binaire, donc
// il ne se juge pas au critère des bibliothèques liées. Ce qu'il apporte est
// que le schéma soit exécuté — un schéma que rien n'exerce peut mentir comme un
// commentaire, et il l'a fait : sa clause sur les effets différés a refusé
// pendant des mois la seule forme valide de la primitive qu'elle contraint.
func schemaManifeste(t *testing.T) *jsonschema.Schema {
	t.Helper()

	brut, err := os.ReadFile(cheminSchemaManifeste)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(brut)))
	if err != nil {
		t.Fatal(err)
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource("manifest.json", doc); err != nil {
		t.Fatal(err)
	}
	sch, err := c.Compile("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	return sch
}

// enJSON rend un manifeste TOML sous la forme que le validateur attend.
//
// Le passage par JSON n'est pas une commodité : un décodeur TOML rend des int64
// et des time.Time que JSON Schema ne sait pas typer. Valider ce que le jeu
// enverrait plutôt que ce que le décodeur a produit est aussi ce qui rapproche
// le test du cas réel.
func enJSON(t *testing.T, contenu string) any {
	t.Helper()

	var brut map[string]any
	if _, err := toml.Decode(contenu, &brut); err != nil {
		t.Fatalf("TOML invalide : %v", err)
	}
	encode, err := json.Marshal(brut)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(encode)))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// TestShippedManifestMatchesItsSchema valide le contenu livré contre le contrat
// qu'il est censé illustrer.
//
// Le manifeste de plugins/base passe par le même chemin qu'un plugin tiers ; il
// doit donc satisfaire le même contrat, sans quoi le premier exemple qu'un
// auteur recopie est celui qui ne valide pas.
func TestShippedManifestMatchesItsSchema(t *testing.T) {
	contenu, err := os.ReadFile(filepath.Join(racine, "plugins", "base", "manifest.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := schemaManifeste(t).Validate(enJSON(t, string(contenu))); err != nil {
		t.Errorf("le manifeste livré ne valide pas contre son propre schéma :\n%v", err)
	}
}

// TestSchemaRejectsOneFaultAtATime exerce chaque refus du schéma sur un
// manifeste qui ne porte que cette faute.
//
// **Une faute par manifeste, et le chemin de l'instance fautive vérifié.** Un
// manifeste qui en cumulerait deux passerait le test alors que le schéma n'en
// attrape qu'une : c'est la version « refus » du défaut qu'on a trouvé dans
// validate, dont trois critères sur six n'étaient exercés par personne.
func TestSchemaRejectsOneFaultAtATime(t *testing.T) {
	sch := schemaManifeste(t)

	const socle = `
name = "essai"
version = "0.1.0"
`
	cas := []struct {
		nom       string
		manifeste string
		fautif    string
	}{
		{
			nom: "nom hors du motif",
			manifeste: `
name = "A_x"
version = "0.1.0"
`,
			fautif: "/name",
		},
		{
			nom: "licence hors de la liste",
			manifeste: socle + `
license = "à voir"
`,
			fautif: "/license",
		},
		{
			nom: "version du vocabulaire périmée",
			manifeste: socle + `
effects_version = 1

[ability.essai]
name = "Essai"
side = "inspectors"
`,
			fautif: "/effects_version",
		},
		{
			nom: "déclenchement inconnu",
			manifeste: socle + `
effects_version = 2

[ability.essai]
name = "Essai"
side = "inspectors"
trigger = "n_importe_quoi"
`,
			fautif: "/ability/essai/trigger",
		},
		{
			nom: "étranglement sur une capacité",
			manifeste: socle + `
effects_version = 2

[ability.essai]
name = "Essai"
side = "inspectors"
trigger = "strangling"
`,
			fautif: "/ability/essai/trigger",
		},
		{
			nom: "primitive inconnue",
			manifeste: socle + `
effects_version = 2

[ability.essai]
name = "Essai"
side = "inspectors"

  [[ability.essai.effect]]
  type = "invente"
  target = "current_piece"
`,
			fautif: "/ability/essai/effect/0/type",
		},
		{
			nom: "cible inconnue",
			manifeste: socle + `
effects_version = 2

[ability.essai]
name = "Essai"
side = "inspectors"

  [[ability.essai.effect]]
  type = "change_range"
  target = "le_voisin"
`,
			fautif: "/ability/essai/effect/0/target",
		},
		{
			nom: "annonce hors d'un différé",
			manifeste: socle + `
effects_version = 2

[ability.essai]
name = "Essai"
side = "inspectors"

  [[ability.essai.effect]]
  type = "change_range"
  target = "current_piece"
  announced = true
`,
			fautif: "/ability/essai/effect/0/announced",
		},
		{
			nom: "différé sans suite",
			manifeste: socle + `
effects_version = 2

[mode.essai]
name = "Essai"
trigger = "strangling"

  [[mode.essai.effect]]
  type = "defer"
  duration = 2
`,
			fautif: "/mode/essai/effect/0",
		},
		{
			nom: "un bot et des effets",
			manifeste: socle + `
effects_version = 2

[bot]
side = "inspectors"
command = "essai"

[ability.essai]
name = "Essai"
side = "inspectors"
`,
			fautif: "",
		},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			err := sch.Validate(enJSON(t, c.manifeste))
			if err == nil {
				t.Fatal("le schéma accepte ce manifeste")
			}
			if c.fautif == "" {
				return
			}
			if !strings.Contains(err.Error(), c.fautif) {
				t.Errorf("refusé, mais pas sur %s :\n%v", c.fautif, err)
			}
		})
	}
}
