// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package greffons

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/sprimault/filature/internal/noyau"
)

// NomManifeste est le fichier que tout greffon porte à sa racine.
const NomManifeste = "manifeste.toml"

// manifeste est la forme d'un manifeste.toml, telle que
// schemas/manifeste-greffon.schema.json la décrit.
//
// Les capacités, dépenses et modes se décodent directement dans les types du
// noyau : ce sont les mêmes structures, et les faire transiter par des doubles
// locaux donnerait deux descriptions du même contrat à tenir d'accord.
type manifeste struct {
	Nom           string `toml:"nom"`
	Version       string `toml:"version"`
	VersionEffets int    `toml:"version_effets"`
	Description   string `toml:"description"`
	Regles        bool   `toml:"regles"`
	Wasm          string `toml:"wasm"`
	Licence       string `toml:"licence"`

	Capacites map[string]noyau.Capacite `toml:"capacite"`
	Depenses  map[string]noyau.Capacite `toml:"depense"`
	Modes     map[string]noyau.Mode     `toml:"mode"`

	Langue *langue `toml:"langue"`
	Bot    *bot    `toml:"bot"`
}

// langue identifie le dictionnaire posé dans langue.toml à côté du manifeste.
// Les libellés eux-mêmes ne passent pas par le registre : le noyau n'affiche
// rien.
type langue struct {
	Code string `toml:"code"`
	Nom  string `toml:"nom"`
}

// bot décrit un adversaire en processus séparé.
//
// Le déterminisme est déclaré et non vérifié ici : un bot qui ment reste
// jouable, seule la reproduction d'un défaut en pâtit.
type bot struct {
	Camp         noyau.Acteur `toml:"camp"`
	Commande     string       `toml:"commande"`
	Arguments    []string     `toml:"arguments"`
	Deterministe bool         `toml:"deterministe"`
}

// lireManifeste décode le manifeste d'un greffon et le valide.
//
// Le nom du dossier l'emporte sur le champ « nom » en cas de désaccord : c'est
// lui qui sert de clé partout ailleurs, et deux sources pour un identifiant
// finissent par diverger. Le désaccord est signalé plutôt que rattrapé.
func lireManifeste(source fs.FS, dossier string) (*manifeste, error) {
	chemin := path(dossier, NomManifeste)

	contenu, err := fs.ReadFile(source, chemin)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", chemin, err)
	}

	var m manifeste
	meta, err := toml.Decode(string(contenu), &m)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", chemin, err)
	}

	// Un champ inconnu est un refus et non un avertissement : c'est presque
	// toujours une faute de frappe sur un champ qui compte, et l'ignorer
	// donnerait un greffon qui se charge sans faire ce que son auteur a écrit.
	if restes := meta.Undecoded(); len(restes) > 0 {
		var cles []string
		for _, cle := range restes {
			cles = append(cles, cle.String())
		}
		sort.Strings(cles)
		return nil, fmt.Errorf("%s: champs inconnus: %s", chemin, strings.Join(cles, ", "))
	}

	if manquements := m.valider(chemin, dossier); len(manquements) > 0 {
		return nil, errors.Join(manquements...)
	}
	return &m, nil
}

// valider applique les refus de docs/greffons.md §9 et rend tout ce qui cloche.
//
// Tous les manquements, et non le premier : un auteur de greffon qui corrige
// une ligne pour en découvrir une autre au lancement suivant y passe la soirée.
// C'est aussi ce qui donne son intérêt à « filature valide ».
//
// Rien n'est versé dans un registre tant qu'il en reste un : un greffon à
// moitié actif est pire qu'un greffon absent.
func (m *manifeste) valider(chemin, dossier string) []error {
	var manquements []error
	ajouter := func(format string, args ...any) {
		manquements = append(manquements, fmt.Errorf("%s: "+format, append([]any{chemin}, args...)...))
	}

	if m.Nom == "" {
		ajouter("nom manquant")
	}
	if m.Version == "" {
		ajouter("version manquante")
	}
	if m.Nom != "" && m.Nom != dossier {
		ajouter("nom %q dans un dossier %q", m.Nom, dossier)
	}

	porteDesEffets := len(m.Capacites) > 0 || len(m.Depenses) > 0 || len(m.Modes) > 0

	if porteDesEffets && m.VersionEffets != noyau.VersionEffets {
		ajouter("version_effets %d, ce binaire applique la %d",
			m.VersionEffets, noyau.VersionEffets)
	}

	// La frontière règles / cosmétique n'est pas déclarative sur parole. C'est
	// elle qui rend la poignée de main réseau fiable : deux joueurs peuvent
	// avoir des habillages différents en sachant que la partie ne divergera
	// pas.
	if !m.Regles {
		if porteDesEffets {
			ajouter("regles = false avec des capacites, depenses ou modes")
		}
		if m.Wasm != "" {
			ajouter("regles = false avec un module wasm")
		}
	}

	// Un bot choisit ses coups, il ne légifère pas. S'il pouvait aussi changer
	// la règle, deux joueurs compareraient des manifestes identiques en jouant
	// deux jeux différents.
	if m.Bot != nil && porteDesEffets {
		ajouter("un bot et des effets dans le meme greffon")
	}

	return append(manquements, m.validerLesEffets(chemin)...)
}

// validerLesEffets parcourt capacités, dépenses et modes.
//
// Le chemin de clé se construit à la descente et non après coup : « capacite
// guetteur effet[0] cible » situe le manquement dans le fichier, là où « cible
// invalide » oblige à le chercher.
func (m *manifeste) validerLesEffets(chemin string) []error {
	var manquements []error

	for _, cle := range triees(m.Capacites) {
		manquements = append(manquements,
			valideCapacite(m.Capacites[cle], chemin, "capacite."+cle)...)
	}
	for _, cle := range triees(m.Depenses) {
		manquements = append(manquements,
			valideCapacite(m.Depenses[cle], chemin, "depense."+cle)...)
	}
	for _, cle := range triees(m.Modes) {
		mode := m.Modes[cle]
		if mode.Declenchement == "" {
			manquements = append(manquements,
				fmt.Errorf("%s: mode.%s: declenchement manquant", chemin, cle))
		}
		manquements = append(manquements,
			valideEffets(mode.Effets, chemin, "mode."+cle, false)...)
	}
	return manquements
}

// valideCapacite contrôle une capacité ou une dépense, qui partagent leur
// forme.
func valideCapacite(c noyau.Capacite, chemin, ou string) []error {
	var manquements []error

	if c.Nom == "" {
		manquements = append(manquements, fmt.Errorf("%s: %s: nom manquant", chemin, ou))
	}
	if c.Camp != noyau.CampFugitif && c.Camp != noyau.CampInspecteurs {
		manquements = append(manquements, fmt.Errorf("%s: %s: camp %q, attendu %q ou %q",
			chemin, ou, c.Camp, noyau.CampFugitif, noyau.CampInspecteurs))
	}
	return append(manquements, valideEffets(c.Effets, chemin, ou, false)...)
}

// valideEffets contrôle une liste d'effets, et refuse un differer imbriqué.
//
// Deux durées s'additionnent, donc l'imbrication n'ajoute rien ; et elle
// permettrait des chaînes qu'aucune annulation ne saurait dérouler, ce qui
// coûterait l'invariant de réversibilité pour rien.
func valideEffets(effets []noyau.Effet, chemin, ou string, dansUnDiffere bool) []error {
	var manquements []error

	for i, e := range effets {
		place := fmt.Sprintf("%s.effet[%d]", ou, i)
		ajouter := func(format string, args ...any) {
			manquements = append(manquements,
				fmt.Errorf("%s: %s: "+format, append([]any{chemin, place}, args...)...))
		}

		if !noyau.TypeEffetConnu(e.Type) {
			ajouter("type %q inconnu", e.Type)
		}
		if e.Cible != "" && !noyau.CibleConnue(e.Cible) {
			ajouter("cible %q inconnue", e.Cible)
		}

		if e.Type != noyau.EffetDifferer {
			if len(e.Puis) > 0 {
				ajouter("puis n'a de sens que sur un differer")
			}
			continue
		}
		if dansUnDiffere {
			ajouter("differer imbrique dans un differer")
		}
		if len(e.Puis) == 0 {
			ajouter("differer sans puis")
		}
		manquements = append(manquements,
			valideEffets(e.Puis, chemin, place+".puis", true)...)
	}
	return manquements
}

// triees rend les clés d'une table dans un ordre stable, pour que deux
// chargements du même greffon signalent le même manquement en premier.
func triees[T any](table map[string]T) []string {
	cles := make([]string, 0, len(table))
	for cle := range table {
		cles = append(cles, cle)
	}
	sort.Strings(cles)
	return cles
}

// path assemble un chemin de fs.FS, qui n'emprunte que des barres obliques quel
// que soit le système.
func path(elements ...string) string {
	return strings.Join(elements, "/")
}
