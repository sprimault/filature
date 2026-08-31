// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sprimault/evasion/internal/core"
)

// TestEveryPrimitiveExercised échoue sur une primitive d'effets que la suite
// du noyau n'applique jamais.
//
// Le vocabulaire est un contrat public : une primitive y entre pour ne plus en
// sortir, et l'invariant de réversibilité veut que chacune sache se défaire.
// Sans ce contrôle, en ajouter une et oublier son cas de test ne se voit nulle
// part — c'est arrivé à ouvrir_zone, dont l'annulation réinsère une zone à son
// rang d'origine et n'était vérifiée par personne.
//
// Le rapprochement porte sur les identifiants et non sur les valeurs : un cas
// écrit EffectTeleport, jamais « teleporter ». Chercher les chaînes ne
// trouverait que les noms de cas, qui ressemblent aux valeurs sans les valoir.
func TestEveryPrimitiveExercised(t *testing.T) {
	fset := token.NewFileSet()

	declarees := constantesDuType(t, fset,
		filepath.Join(racine, "internal", "core", "effects.go"), "EffectType")
	if len(declarees) == 0 {
		t.Fatal("aucune primitive trouvée : le contrôle ne vérifie plus rien")
	}

	cites := identifiantsDe(t, fset,
		filepath.Join(racine, "internal", "core", "effects_test.go"))

	for _, c := range declarees {
		if !cites[c.nom] {
			t.Errorf("la primitive %s n'est exercée par aucun cas de effets_test.go", c.nom)
		}
	}
}

// TestVocabularyFullyEnumerated vérifie que EffectTypes et Targets ne
// laissent aucune constante de côté.
//
// Le chargeur de plugins s'en sert pour refuser ce qu'il ne sait pas
// appliquer. Une primitive déclarée mais absente de l'énumération serait donc
// rejetée à la lecture d'un manifeste parfaitement valide, sur un message
// disant qu'elle est inconnue alors qu'elle est écrite juste au-dessus.
func TestVocabularyFullyEnumerated(t *testing.T) {
	fset := token.NewFileSet()
	chemin := filepath.Join(racine, "internal", "core", "effects.go")

	var types []string
	for _, e := range core.EffectTypes() {
		types = append(types, string(e))
	}
	compare(t, "EffectTypes", constantesDuType(t, fset, chemin, "EffectType"), types)

	var cibles []string
	for _, c := range core.Targets() {
		cibles = append(cibles, string(c))
	}
	compare(t, "Targets", constantesDuType(t, fset, chemin, "Target"), cibles)
}

// compare signale ce qu'une énumération oublie et ce qu'elle invente.
func compare(t *testing.T, quoi string, declarees []constante, enumerees []string) {
	t.Helper()

	connues := map[string]bool{}
	for _, v := range enumerees {
		connues[v] = true
	}

	for _, c := range declarees {
		if !connues[c.valeur] {
			t.Errorf("%s oublie %s (%q)", quoi, c.nom, c.valeur)
		}
		delete(connues, c.valeur)
	}
	for reste := range connues {
		t.Errorf("%s cite %q, qui n'est déclarée nulle part", quoi, reste)
	}
}

// constante est une constante déclarée, avec la valeur qui part dans un
// manifeste.
type constante struct {
	nom    string
	valeur string
}

// constantesDuType rend les constantes déclarées avec un type donné.
//
// Les déclarations groupées ne répètent pas le type sur chaque ligne : il est
// porté par la première, et les suivantes en héritent. Sans mémoriser le
// dernier type vu, le contrôle ne verrait qu'une seule primitive sur toutes.
func constantesDuType(t *testing.T, fset *token.FileSet, chemin, typeVoulu string) []constante {
	t.Helper()

	f, err := parser.ParseFile(fset, chemin, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	var trouvees []constante
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
			for i, n := range v.Names {
				c := constante{nom: n.Name}
				if i < len(v.Values) {
					if lit, ok := v.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						c.valeur = strings.Trim(lit.Value, `"`)
					}
				}
				trouvees = append(trouvees, c)
			}
		}
	}
	return trouvees
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
