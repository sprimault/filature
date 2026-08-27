// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestLoaderMatchesThePublishedContract rapproche du schéma les deux valeurs
// que le chargeur recopie.
//
// Le motif d'un nom et la liste des licences sont publiés dans
// schemas/plugin-manifest.schema.json, et le chargeur ne peut pas les y lire :
// il doit rester autonome au démarrage, sans dépendre d'un fichier de contrat.
// La recopie est donc assumée, et c'est ce test qui l'empêche de dériver — sans
// lui, corriger le contrat publié laisserait le jeu appliquer l'ancien.
func TestLoaderMatchesThePublishedContract(t *testing.T) {
	chemin := filepath.Join("..", "..", "schemas", "plugin-manifest.schema.json")
	brut, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}

	var schema struct {
		Properties struct {
			Name struct {
				Pattern string `json:"pattern"`
			} `json:"name"`
			License struct {
				Enum []string `json:"enum"`
			} `json:"license"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(brut, &schema); err != nil {
		t.Fatal(err)
	}

	if motif := schema.Properties.Name.Pattern; motif != nomDePlugin.String() {
		t.Errorf("le schéma publie le motif de nom %q, le chargeur applique %q",
			motif, nomDePlugin.String())
	}
	if publiees := schema.Properties.License.Enum; !slices.Equal(publiees, licencesAdmises) {
		t.Errorf("le schéma publie les licences %v, le chargeur admet %v",
			publiees, licencesAdmises)
	}
}
