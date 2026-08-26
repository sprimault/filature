// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

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

// TestStubsMarques échoue sur une fonction au corps trivial qui ne porte pas
// son marqueur d'étape.
//
// Le décompte de `grep -rn "à implémenter"` est la mesure d'avancement de la
// feuille de route. Une fonction dont la signature ne renvoie pas d'erreur ne
// peut pas porter le marqueur dans un errors.New, et échappait donc au grep :
// le compteur a annoncé 25 stubs pour 46.
//
// Les fichiers de test sont hors du contrôle, contrairement au contrôle de
// documentation : un double d'essai qui renvoie une valeur nulle est la norme
// et non un travail en attente, et aucune étape ne se mesure à eux.
func TestStubsMarques(t *testing.T) {
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
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			return nil
		}

		f, err := parser.ParseFile(fset, chemin, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil || !corpsTrivial(fn.Body) || marque(f, fn.Body) {
				continue
			}
			manques = append(manques,
				fset.Position(fn.Pos()).String()+": "+fn.Name.Name+" a un corps vide sans marqueur d'étape")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range manques {
		t.Error(m)
	}
}

// corpsTrivial dit si un corps ne fait rien : vide, ou un unique return de
// valeurs nulles.
//
// Une fonction de production qui se réduirait à « return nil » serait signalée
// à tort. Le cas ne s'est pas présenté ; s'il se présente, l'exception se
// déclare ici plutôt qu'en désarmant le test.
func corpsTrivial(b *ast.BlockStmt) bool {
	switch len(b.List) {
	case 0:
		return true
	case 1:
		r, ok := b.List[0].(*ast.ReturnStmt)
		if !ok {
			return false
		}
		for _, v := range r.Results {
			if !valeurNulle(v) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// valeurNulle reconnaît les littéraux qu'un stub renvoie faute de mieux : nil,
// false, zéro, chaîne vide, et la structure ou la tranche vide.
func valeurNulle(e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name == "nil" || v.Name == "false"
	case *ast.BasicLit:
		return v.Value == "0" || v.Value == `""`
	case *ast.CompositeLit:
		return len(v.Elts) == 0
	case *ast.UnaryExpr:
		return v.Op == token.AND && valeurNulle(v.X)
	}
	return false
}

// marque dit si un corps contient le commentaire d'étape. Le comparer par
// position est le seul moyen : l'AST rattache les commentaires au fichier, pas
// aux blocs.
func marque(f *ast.File, b *ast.BlockStmt) bool {
	for _, g := range f.Comments {
		if g.Pos() > b.Pos() && g.End() < b.End() && strings.Contains(g.Text(), "à implémenter") {
			return true
		}
	}
	return false
}
