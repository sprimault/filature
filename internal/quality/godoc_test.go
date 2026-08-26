// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Package qualite porte les contrôles qui valent pour le dépôt entier plutôt
// que pour un paquet.
package quality

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// racine est le dépôt vu depuis ce paquet.
const racine = "../.."

// ignores sont les répertoires que le parcours ne descend pas. .tmp porte le
// cache de compilation, qui contient du Go tiers.
var ignores = map[string]bool{".git": true, ".tmp": true, ".idea": true}

// TestGodocComplete échoue sur toute déclaration de haut niveau sans
// documentation.
//
// revive n'a que la règle exported : les fonctions non exportées, main et les
// variables de paquet lui échappent, alors que la règle du projet ne fait pas
// cette distinction. Les fichiers de test sont soumis au même contrôle.
func TestGodocComplete(t *testing.T) {
	fset := token.NewFileSet()
	var manques []string

	err := filepath.WalkDir(racine, func(chemin string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			if ignores[e.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(e.Name(), ".go") {
			return nil
		}

		f, err := parser.ParseFile(fset, chemin, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		manques = append(manques, sansDoc(fset, f)...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range manques {
		t.Error(m)
	}
}

// sansDoc relève les déclarations non documentées d'un fichier.
//
// Un bloc parenthésé est accepté de deux façons : un commentaire sur le bloc,
// ou un commentaire sur chacune de ses entrées. Les deux se pratiquent, et
// imposer l'un des deux forcerait à réécrire du code correct.
func sansDoc(fset *token.FileSet, f *ast.File) []string {
	var manques []string

	releve := func(pos token.Pos, quoi, nom string) {
		manques = append(manques, fset.Position(pos).String()+": "+quoi+" "+nom+" sans documentation")
	}

	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Doc == nil {
				releve(d.Pos(), "fonction", d.Name.Name)
			}

		case *ast.GenDecl:
			if d.Tok == token.IMPORT || d.Doc != nil {
				continue
			}
			for _, s := range d.Specs {
				switch s := s.(type) {
				case *ast.TypeSpec:
					if s.Doc == nil {
						releve(s.Pos(), "type", s.Name.Name)
					}
				case *ast.ValueSpec:
					if s.Doc == nil {
						releve(s.Pos(), d.Tok.String(), s.Names[0].Name)
					}
				}
			}
		}
	}
	return manques
}
