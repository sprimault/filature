// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Package schema produit un JSON Schema depuis un type Go.
//
// Le contrat que lisent les bots et le mode réseau est celui de la structure
// Go, pas un document écrit à côté. Le dériver par réflexion supprime la
// question de savoir lequel des deux fait foi : un champ ajouté à la structure
// apparaît dans le schéma, un champ retiré en disparaît, et le test qui compare
// les deux échoue tant que le fichier n'a pas été régénéré et relu.
package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Document est un schéma prêt à être écrit.
//
// L'ordre des clés est celui de la sérialisation Go, pas celui du fichier :
// c'est encoding/json qui décide, et il trie les clés d'une map. Le résultat
// est donc stable d'une exécution à l'autre.
type Document struct {
	Comment     string           `json:"$comment,omitempty"`
	Schema      string           `json:"$schema"`
	ID          string           `json:"$id"`
	Title       string           `json:"title"`
	Description string           `json:"description,omitempty"`
	Ref         string           `json:"$ref"`
	Defs        map[string]*Node `json:"$defs"`
}

// Node est un schéma de type, inline ou référencé.
type Node struct {
	Ref                  string           `json:"$ref,omitempty"`
	Type                 string           `json:"type,omitempty"`
	Description          string           `json:"description,omitempty"`
	Items                *Node            `json:"items,omitempty"`
	Properties           map[string]*Node `json:"properties,omitempty"`
	Required             []string         `json:"required,omitempty"`
	AdditionalProperties *Node            `json:"additionalProperties,omitempty"`
}

// dialect est la version de JSON Schema produite, la même que les schémas
// écrits à la main du dépôt.
const dialect = "https://json-schema.org/draft/2020-12/schema"

// Generate rend le schéma d'un type, avec ses types imbriqués en $defs.
//
// Les structures partagées — une position apparaît une dizaine de fois — ne
// sont décrites qu'une fois. C'est aussi ce qui permet à un type récursif de
// tenir : un effet différé contient des effets.
func Generate(t reflect.Type, id, titre, description, entete string) ([]byte, error) {
	g := &generator{defs: map[string]*Node{}, walking: map[string]bool{}}

	racine := g.node(t)
	if racine.Ref == "" {
		return nil, fmt.Errorf("le type racine %s n'est pas une structure", t.Name())
	}

	doc := Document{
		Comment:     entete,
		Schema:      dialect,
		ID:          id,
		Title:       titre,
		Description: description,
		Ref:         racine.Ref,
		Defs:        g.defs,
	}

	var tampon bytes.Buffer
	enc := json.NewEncoder(&tampon)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return tampon.Bytes(), nil
}

// generator accumule les définitions rencontrées.
//
// walking porte les structures en cours de description : c'est le drapeau qui
// coupe la récursion, un effet différé contenant des effets.
type generator struct {
	defs    map[string]*Node
	walking map[string]bool
}

// node rend le schéma d'un type, en le plaçant en $defs si c'est une structure.
func (g *generator) node(t reflect.Type) *Node {
	switch t.Kind() {
	case reflect.Pointer:
		// Un pointeur ne change pas la forme, seulement la présence : le champ
		// porte omitempty, et son absence dit qu'on n'en sait rien.
		return g.node(t.Elem())

	case reflect.String:
		return &Node{Type: "string"}

	case reflect.Bool:
		return &Node{Type: "boolean"}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Node{Type: "integer"}

	case reflect.Float32, reflect.Float64:
		return &Node{Type: "number"}

	case reflect.Slice, reflect.Array:
		return &Node{Type: "array", Items: g.node(t.Elem())}

	case reflect.Map:
		return &Node{Type: "object", AdditionalProperties: g.node(t.Elem())}

	case reflect.Struct:
		return g.structure(t)
	}

	// Une interface ou un canal n'a pas de forme JSON. Le noyau n'en sérialise
	// pas — Board porte json:"-" — et rencontrer le cas signale un champ
	// ajouté sans y penser.
	return &Node{Description: "type sans forme JSON : " + t.String()}
}

// structure enregistre une structure en $defs et rend la référence.
func (g *generator) structure(t reflect.Type) *Node {
	nom := t.Name()
	if nom == "" {
		nom = "Anonyme"
	}
	ref := &Node{Ref: "#/$defs/" + nom}

	// Le drapeau coupe la récursion avant qu'elle ne s'emballe : un effet
	// différé contient des effets, et la définition doit se référencer
	// elle-même plutôt que se dérouler sans fin.
	if g.walking[nom] {
		return ref
	}
	g.walking[nom] = true
	defer delete(g.walking, nom)

	node := &Node{Type: "object", Properties: map[string]*Node{}}
	g.defs[nom] = node

	for i := 0; i < t.NumField(); i++ {
		champ := t.Field(i)
		if !champ.IsExported() {
			continue
		}
		cle, requis, garde := jsonName(champ)
		if !garde {
			continue
		}
		node.Properties[cle] = g.node(champ.Type)
		if requis {
			node.Required = append(node.Required, cle)
		}
	}
	sort.Strings(node.Required)

	return ref
}

// jsonName lit le tag d'un champ : sa clé, s'il est obligatoire, et s'il est
// sérialisé du tout.
//
// Un champ marqué omitempty peut manquer, donc n'est pas requis. C'est ainsi
// que la vue dit « je n'en sais rien » plutôt que de mentir avec une valeur
// nulle.
func jsonName(champ reflect.StructField) (cle string, requis, garde bool) {
	tag := champ.Tag.Get("json")
	if tag == "-" {
		return "", false, false
	}

	parties := strings.Split(tag, ",")
	cle = parties[0]
	if cle == "" {
		cle = champ.Name
	}
	for _, option := range parties[1:] {
		if option == "omitempty" {
			return cle, false, true
		}
	}
	return cle, true, true
}
