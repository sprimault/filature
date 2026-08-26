// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package qualite

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// TestPrimitivesToutesExercees échoue sur une primitive d'effets que la suite
// du noyau n'applique jamais.
//
// Le vocabulaire est un contrat public : une primitive y entre pour ne plus en
// sortir, et l'invariant de réversibilité veut que chacune sache se défaire.
// Sans ce contrôle, en ajouter une et oublier son cas de test ne se voit nulle
// part — c'est arrivé à ouvrir_zone, dont l'annulation réinsère une zone à son
// rang d'origine et n'était vérifiée par personne.
//
// Le rapprochement porte sur les identifiants et non sur les valeurs : un cas
// écrit EffetTeleporter, jamais « teleporter ». Chercher les chaînes ne
// trouverait que les noms de cas, qui ressemblent aux valeurs sans les valoir.
func TestPrimitivesToutesExercees(t *testing.T) {
	fset := token.NewFileSet()

	declarees := constantesDuType(t, fset,
		filepath.Join(racine, "internal", "noyau", "effets.go"), "TypeEffet")
	if len(declarees) == 0 {
		t.Fatal("aucune primitive trouvée : le contrôle ne vérifie plus rien")
	}

	cites := identifiantsDe(t, fset,
		filepath.Join(racine, "internal", "noyau", "effets_test.go"))

	for _, nom := range declarees {
		if !cites[nom] {
			t.Errorf("la primitive %s n'est exercée par aucun cas de effets_test.go", nom)
		}
	}
}

// constantesDuType rend les noms des constantes déclarées avec un type donné.
//
// Les déclarations groupées ne répètent pas le type sur chaque ligne : il est
// porté par la première, et les suivantes en héritent. Sans mémoriser le
// dernier type vu, le contrôle ne verrait qu'une primitive sur dix-neuf.
func constantesDuType(t *testing.T, fset *token.FileSet, chemin, typeVoulu string) []string {
	t.Helper()

	f, err := parser.ParseFile(fset, chemin, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	var noms []string
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}

		dernierType := ""
		for _, s := range gen.Specs {
			v, ok := s.(*ast.ValueSpec)
			if !ok {
				continue
			}
			if ident, ok := v.Type.(*ast.Ident); ok {
				dernierType = ident.Name
			}
			if dernierType != typeVoulu {
				continue
			}
			for _, n := range v.Names {
				noms = append(noms, n.Name)
			}
		}
	}
	return noms
}

// identifiantsDe rassemble tous les noms cités dans un fichier.
func identifiantsDe(t *testing.T, fset *token.FileSet, chemin string) map[string]bool {
	t.Helper()

	f, err := parser.ParseFile(fset, chemin, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	cites := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			cites[ident.Name] = true
		}
		return true
	})
	return cites
}
