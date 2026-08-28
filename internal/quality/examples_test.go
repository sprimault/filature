// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/loader"
)

// TestDocumentedManifestsLoad passe au chargeur chaque manifeste que la
// documentation donne en exemple.
//
// Ce sont des modèles à recopier, et deux d'entre eux étaient refusés tels
// quels : celui du contrat de formes ne déclarait pas de rôle, et celui des
// langues portait un nom que le dossier de son arborescence contredisait. Dans
// les deux cas l'auteur obtenait un refus sur sa première tentative, à partir
// du texte même qui lui expliquait quoi écrire.
//
// La limite à connaître : le dossier est créé au nom que le manifeste déclare,
// donc leur désaccord ne se voit pas ici. C'est ce qu'il faut relire à la main
// dans une arborescence d'exemple — le reste, motif du nom, licence, champs
// inconnus, cohérence de rules, est couvert.
func TestDocumentedManifestsLoad(t *testing.T) {
	documents := []string{"plugins.md", "contrat-formes.md", "protocole-bot.md"}

	trouves := 0
	for _, doc := range documents {
		for _, manifeste := range manifestesDe(t, filepath.Join(racine, "docs", doc)) {
			nom := nomDeclare(manifeste)
			if nom == "" {
				t.Errorf("%s : un manifeste d'exemple sans nom", doc)
				continue
			}
			trouves++

			t.Run(doc+"/"+nom, func(t *testing.T) {
				dossier := filepath.Join(t.TempDir(), nom)
				if err := os.MkdirAll(dossier, 0o750); err != nil {
					t.Fatal(err)
				}
				chemin := filepath.Join(dossier, loader.ManifestName)
				if err := os.WriteFile(chemin, []byte(manifeste), 0o600); err != nil {
					t.Fatal(err)
				}

				if err := loader.Validate(dossier); err != nil {
					t.Errorf("l'exemple de %s est refusé tel qu'il est publié :\n%v", doc, err)
				}
			})
		}
	}

	if trouves == 0 {
		t.Fatal("aucun manifeste d'exemple trouvé : le contrôle ne vérifie plus rien")
	}
}

// manifestesDe extrait les blocs TOML d'un document qui décrivent un manifeste
// de plugin.
//
// Un manifeste se reconnaît à ses deux champs obligatoires posés à la racine :
// les autres blocs du même document montrent une capacité ou un dictionnaire,
// qui portent un « name » sans être des manifestes.
func manifestesDe(t *testing.T, chemin string) []string {
	t.Helper()

	contenu, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}

	var blocs []string
	for _, bloc := range strings.Split(string(contenu), "```toml")[1:] {
		fin := strings.Index(bloc, "```")
		if fin < 0 {
			continue
		}
		brut := bloc[:fin]
		if nomDeclare(brut) != "" && strings.Contains(brut, "\nversion = ") {
			blocs = append(blocs, brut)
		}
	}
	return blocs
}

// nomDeclare rend le nom qu'un manifeste pose à sa racine.
func nomDeclare(manifeste string) string {
	for _, ligne := range strings.Split(manifeste, "\n") {
		reste, coupe := strings.CutPrefix(ligne, "name = \"")
		if !coupe {
			continue
		}
		if nom, ferme := strings.CutSuffix(reste, "\""); ferme {
			return nom
		}
	}
	return ""
}
