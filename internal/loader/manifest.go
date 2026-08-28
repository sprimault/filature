// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/sprimault/filature/internal/core"
)

// ManifestName est le fichier que tout plugin porte à sa racine.
const ManifestName = "manifest.toml"

// nomDePlugin est le motif que schemas/plugin-manifest.schema.json publie.
//
// Recopié ici et non dérivé du schéma : le chargeur doit rester autonome au
// démarrage, sans lire de fichier de contrat. Un test rapproche les deux.
var nomDePlugin = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)

// licencesAdmises est la liste blanche SPDX du catalogue, dans l'ordre où le
// schéma la publie.
//
// Courte et fermée par choix : un champ libre laisserait passer « à voir » et
// rendrait les entrées inexploitables, comme les fichiers binaires que le
// catalogue refuse. Rien à juger, donc rien à relire à la main.
var licencesAdmises = []string{
	"MIT", "Apache-2.0", "CC0-1.0", "CC-BY-4.0", "CC-BY-SA-4.0", "BSD-3-Clause",
}

// nomValide dit si un nom de plugin suit le motif du contrat.
func nomValide(nom string) bool { return nomDePlugin.MatchString(nom) }

// codeDeLangue est l'étiquette BCP 47 que le schéma publie : de, pt-BR,
// zh-Hans.
//
// Recopié comme le motif d'un nom, et pour la même raison : le chargeur ne lit
// pas de fichier de contrat au démarrage. Un test rapproche les deux.
var codeDeLangue = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z]{2,8})*$`)

// manifeste est la forme d'un manifeste.toml, telle que
// schemas/manifeste-plugin.schema.json la décrit.
//
// Les capacités, dépenses et modes se décodent directement dans les types du
// noyau : ce sont les mêmes structures, et les faire transiter par des doubles
// locaux donnerait deux descriptions du même contrat à tenir d'accord.
type manifeste struct {
	Name           string `toml:"name"`
	Version        string `toml:"version"`
	EffectsVersion int    `toml:"effects_version"`
	Description    string `toml:"description"`
	Rules          bool   `toml:"rules"`
	Wasm           string `toml:"wasm"`
	Licence        string `toml:"license"`

	Abilities map[string]core.Ability `toml:"ability"`
	Expenses  map[string]core.Ability `toml:"expense"`
	Modes     map[string]core.Mode    `toml:"mode"`

	Langue *langue `toml:"language"`
	Bot    *bot    `toml:"bot"`
}

// langue identifie le dictionnaire posé dans language.toml à côté du manifeste.
// Seul ce couple entre au registre, où il rend le conflit détectable ; les
// libellés eux-mêmes n'y passent pas, le noyau n'affichant rien.
type langue struct {
	Code string `toml:"code"`
	Name string `toml:"name"`
}

// bot décrit un adversaire en processus séparé.
//
// Le déterminisme est déclaré et non vérifié ici : un bot qui ment reste
// jouable, seule la reproduction d'un défaut en pâtit.
type bot struct {
	Camp         core.Side `toml:"side"`
	Commande     string    `toml:"command"`
	Arguments    []string  `toml:"arguments"`
	Deterministe bool      `toml:"deterministic"`
}

// readManifest décode le manifeste d'un plugin et le valide.
//
// Le nom du dossier l'emporte sur le champ « nom » en cas de désaccord : c'est
// lui qui sert de clé partout ailleurs, et deux sources pour un identifiant
// finissent par diverger. Le désaccord est signalé plutôt que rattrapé.
func readManifest(source fs.FS, dossier string) (*manifeste, error) {
	chemin := path(dossier, ManifestName)

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
	// donnerait un plugin qui se charge sans faire ce que son auteur a écrit.
	if restes := meta.Undecoded(); len(restes) > 0 {
		var cles []string
		for _, cle := range restes {
			cles = append(cles, cle.String())
		}
		sort.Strings(cles)
		return nil, fmt.Errorf("%s: champs inconnus: %s", chemin, strings.Join(cles, ", "))
	}

	if manquements := m.validate(chemin, dossier); len(manquements) > 0 {
		return nil, errors.Join(manquements...)
	}
	return &m, nil
}

// validate applique les refus de docs/plugins.md §9 et rend tout ce qui cloche.
//
// Tous les manquements, et non le premier : un auteur de plugin qui corrige
// une ligne pour en découvrir une autre au lancement qui y passe la soirée.
// C'est aussi ce qui donne son intérêt à « filature valide ».
//
// Rien n'est versé dans un registre tant qu'il en reste un : un plugin à
// moitié actif est pire qu'un plugin absent.
func (m *manifeste) validate(chemin, dossier string) []error {
	var manquements []error
	ajouter := func(format string, args ...any) {
		manquements = append(manquements, fmt.Errorf("%s: "+format, append([]any{chemin}, args...)...))
	}

	if m.Name == "" {
		ajouter("nom manquant")
	}
	if m.Version == "" {
		ajouter("version manquante")
	}
	if m.Name != "" && m.Name != dossier {
		ajouter("nom %q dans un dossier %q", m.Name, dossier)
	}

	// Le nom sert d'identifiant partout — dossier, empreinte, index du
	// catalogue —, et le motif est celui que le schéma publie. Il n'était
	// contrôlé nulle part : « A_x » passait, alors que le contrat l'exclut.
	if m.Name != "" && !nomValide(m.Name) {
		ajouter("nom %q : attendu de 2 a 32 caracteres, minuscules, chiffres et tirets, "+
			"commencant par une lettre", m.Name)
	}

	// La licence est une liste fermée et non un champ libre : le catalogue
	// refuse ce qu'il ne peut pas juger, et « a voir » est exactement l'entrée
	// que la liste existe pour écarter. Facultative hors catalogue, d'où le
	// contrôle sur la valeur seulement.
	if m.Licence != "" && !slices.Contains(licencesAdmises, m.Licence) {
		ajouter("licence %q inconnue, attendu l'une de %s",
			m.Licence, strings.Join(licencesAdmises, ", "))
	}

	porteDesEffets := len(m.Abilities) > 0 || len(m.Expenses) > 0 || len(m.Modes) > 0

	if porteDesEffets && m.EffectsVersion != core.EffectsVersion {
		ajouter("effects_version %d, ce binaire applique la %d",
			m.EffectsVersion, core.EffectsVersion)
	}

	// La frontière règles / cosmétique n'est pas déclarative sur parole. C'est
	// elle qui rend la poignée de main réseau fiable : deux joueurs peuvent
	// avoir des habillages différents en sachant que la partie ne divergera
	// pas.
	if !m.Rules {
		if porteDesEffets {
			ajouter("rules = false avec des capacites, depenses ou modes")
		}
		if m.Wasm != "" {
			ajouter("rules = false avec un module wasm")
		}
	}

	// Le code de langue sert de clé au sélecteur et à la poignée de main : deux
	// plugins qui déclarent le même sont un conflit, ce qui n'a de sens que si
	// la forme est fixée. Le motif était publié par le schéma et lu par
	// personne.
	if m.Langue != nil {
		if m.Langue.Code == "" {
			ajouter("langue sans code")
		} else if !codeDeLangue.MatchString(m.Langue.Code) {
			ajouter("code de langue %q : attendu une etiquette BCP 47, comme de, pt-BR ou zh-Hans",
				m.Langue.Code)
		}
		if m.Langue.Name == "" {
			ajouter("langue %q sans nom : c'est lui qui s'affiche dans le selecteur",
				m.Langue.Code)
		}
	}

	// Un bot choisit ses coups, il ne légifère pas. S'il pouvait aussi changer
	// la règle, deux joueurs compareraient des manifestes identiques en jouant
	// deux jeux différents.
	if m.Bot != nil && porteDesEffets {
		ajouter("un bot et des effets dans le meme plugin")
	}

	return append(manquements, m.checkAllEffects(chemin)...)
}

// checkAllEffects parcourt capacités, dépenses et modes.
//
// Le chemin de clé se construit à la descente et non après coup : « capacite
// guetteur effet[0] cible » situe le manquement dans le fichier, là où « cible
// invalide » oblige à le chercher.
func (m *manifeste) checkAllEffects(chemin string) []error {
	var manquements []error

	for _, cle := range sortedKeys(m.Abilities) {
		manquements = append(manquements,
			checkAbility(m.Abilities[cle], chemin, "ability."+cle)...)
	}
	for _, cle := range sortedKeys(m.Expenses) {
		manquements = append(manquements,
			checkAbility(m.Expenses[cle], chemin, "expense."+cle)...)
	}
	// Les trois champs d'un mode sont obligatoires au schéma publié, et
	// seul le déclenchement l'était ici. Un mode sans nom n'a rien à afficher
	// quand il agit, un mode sans effet ne fait rien : les deux se chargeaient
	// sans un mot.
	for _, cle := range sortedKeys(m.Modes) {
		mode := m.Modes[cle]
		if mode.Name == "" {
			manquements = append(manquements,
				fmt.Errorf("%s: mode.%s: nom manquant", chemin, cle))
		}
		switch {
		case mode.Trigger == "":
			manquements = append(manquements,
				fmt.Errorf("%s: mode.%s: declenchement manquant", chemin, cle))
		case !triggerConnu(mode.Trigger):
			manquements = append(manquements,
				fmt.Errorf("%s: mode.%s: declenchement %q inconnu, attendu l'un de %s",
					chemin, cle, mode.Trigger, listeDesTriggers()))
		}
		if len(mode.Effects) == 0 {
			manquements = append(manquements,
				fmt.Errorf("%s: mode.%s: aucun effet", chemin, cle))
		}
		manquements = append(manquements,
			checkEffects(mode.Effects, "", chemin, "mode."+cle, false)...)
	}
	return manquements
}

// triggerConnu dit si un déclenchement fait partie du vocabulaire.
func triggerConnu(t core.Trigger) bool {
	return slices.Contains(core.Triggers(), t)
}

// listeDesTriggers rend les déclenchements pour un message d'erreur.
//
// Un refus qui ne dit pas ce qui était attendu oblige l'auteur à ouvrir le
// schéma, alors qu'il a le message sous les yeux.
func listeDesTriggers() string {
	noms := make([]string, 0, len(core.Triggers()))
	for _, t := range core.Triggers() {
		noms = append(noms, string(t))
	}
	return strings.Join(noms, ", ")
}

// checkAbility contrôle une capacité ou une dépense, qui partagent leur
// forme.
func checkAbility(c core.Ability, chemin, ou string) []error {
	var manquements []error

	if c.Name == "" {
		manquements = append(manquements, fmt.Errorf("%s: %s: nom manquant", chemin, ou))
	}
	if c.Camp != core.SideFugitive && c.Camp != core.SideInspectors {
		manquements = append(manquements, fmt.Errorf("%s: %s: camp %q, attendu %q ou %q",
			chemin, ou, c.Camp, core.SideFugitive, core.SideInspectors))
	}

	// Le déclenchement n'était contrôlé nulle part : n'importe quelle chaîne
	// passait, et la capacité n'entrait alors jamais en jeu sans qu'un message
	// le dise. Une capacité passive n'en déclare pas, elle vaut toute la partie
	// — mais si elle en déclare un, il doit exister : le schéma ne fait pas
	// dépendre son énumération de « passive », et une chaîne inventée dit que
	// l'auteur croit à un déclenchement que rien ne produira.
	if c.Trigger != "" && !triggerConnu(c.Trigger) {
		manquements = append(manquements, fmt.Errorf("%s: %s: declenchement %q inconnu, attendu l'un de %s",
			chemin, ou, c.Trigger, listeDesTriggers()))
	}
	if c.Trigger == core.OnStrangling {
		manquements = append(manquements, fmt.Errorf(
			"%s: %s: declenchement %q reserve a un mode : le jeu le declenche lui-meme, "+
				"un pion qui s'y accroche agirait sans que son camp l'ait joue",
			chemin, ou, core.OnStrangling))
	}
	return append(manquements, checkEffects(c.Effects, c.Camp, chemin, ou, false)...)
}

// cibleHorsDuCamp dit pourquoi une cible ne convient pas au camp qui la déclare,
// et rend une chaîne vide quand elle convient.
//
// **Ce qui se refuse est ce qui ne désigne personne, pas ce qui avantage
// l'adversaire.** docs/vocabulaire-effets.md §4 promettait le second, avec pour
// exemple une capacité d'inspecteur qui rendrait de la résistance au fugitif —
// mais cibler le fugitif depuis le camp adverse est le cas ordinaire : le Chef
// révèle sa position, le Barreur lui ferme une case. Séparer le bénéfique du
// nuisible demanderait au chargeur de juger chaque couple effet-cible, c'est-à-dire
// exactement le raisonnement que le vocabulaire déclaratif refuse de porter.
//
// Le document a été corrigé pour dire ce qui est réellement vérifié. Une
// promesse qu'aucun code ne peut tenir vaut moins qu'une garantie plus étroite
// qui, elle, s'applique.
//
// Un mode n'a pas de camp : c'est le jeu qui le déclenche, et il agit sur qui la
// règle désigne. Camp vide, donc rien à contrôler.
func cibleHorsDuCamp(cible core.Target, camp core.Side) string {
	// Le fugitif est seul : « un autre pion » et « tous les pions » ne
	// désignent rien chez lui, et un effet qui les vise ne s'appliquerait à
	// personne sans qu'aucun message ne le dise. current_piece reste admis, il
	// le désigne lui-même.
	if camp == core.SideFugitive &&
		(cible == core.TargetOtherPiece || cible == core.TargetAllPieces) {
		return "le fugitif est seul, cette cible ne designe aucun pion"
	}
	return ""
}

// maxThen borne les effets d'un differer, comme le schéma publié.
//
// Une chaîne sans borne se déroule à l'échéance, dans une résolution qui doit
// rester finie et dont chaque pas s'annule.
const maxThen = 8

// checkEffects contrôle une liste d'effets, et refuse un differer imbriqué.
//
// Deux durées s'additionnent, donc l'imbrication n'ajoute rien ; et elle
// permettrait des chaînes qu'aucune annulation ne saurait dérouler, ce qui
// coûterait l'invariant de réversibilité pour rien.
//
// Les bornes numériques sont celles du schéma publié, et docs/plugins.md §8
// promet que les deux disent la même chose. Sans elles, un auteur obtenait un
// refus de son validateur JSON là où le jeu acceptait — et c'est alors la
// partie qui tranche, au pire moment.
func checkEffects(effets []core.Effect, camp core.Side, chemin, ou string, dansUnDiffere bool) []error {
	var manquements []error

	for i, e := range effets {
		place := fmt.Sprintf("%s.effect[%d]", ou, i)
		ajouter := func(format string, args ...any) {
			manquements = append(manquements,
				fmt.Errorf("%s: %s: "+format, append([]any{chemin, place}, args...)...))
		}

		if !core.EffectTypeKnown(e.Type) {
			ajouter("type %q inconnu", e.Type)
		}
		if e.Target != "" && !core.TargetKnown(e.Target) {
			ajouter("cible %q inconnue", e.Target)
		}
		if raison := cibleHorsDuCamp(e.Target, camp); raison != "" {
			ajouter("cible %q dans une declaration du camp %q : %s", e.Target, camp, raison)
		}
		if !core.ValueModeKnown(e.Mode) {
			ajouter("mode %q inconnu, attendu multiply ou rien", e.Mode)
		}
		if e.Radius < 0 {
			ajouter("rayon de %d, attendu positif ou nul", e.Radius)
		}

		if e.Type != core.EffectDefer {
			if e.Duration < 0 {
				ajouter("duree de %d, attendue positive ou nulle", e.Duration)
			}
			if len(e.Then) > 0 {
				ajouter("puis n'a de sens que sur un differer")
			}
			if e.Announced {
				ajouter("annonce n'a de sens que sur un differer")
			}
			continue
		}
		if dansUnDiffere {
			ajouter("differer imbrique dans un differer")
		}
		if len(e.Then) == 0 {
			ajouter("differer sans puis")
		}
		if len(e.Then) > maxThen {
			ajouter("%d effets differes, %d au plus", len(e.Then), maxThen)
		}

		// Une échéance nulle appliquerait à la résolution du tour courant, ce
		// que le differer existe précisément pour ne pas faire — et le joueur
		// qui a lu l'annonce compterait sur un tour qui n'arrive jamais.
		if e.Duration < 1 {
			ajouter("duree d'un differer de %d, attendue au moins 1", e.Duration)
		}
		manquements = append(manquements,
			checkEffects(e.Then, camp, chemin, place+".then", true)...)
	}
	return manquements
}

// sortedKeys rend les clés d'une table dans un ordre stable, pour que deux
// chargements du même plugin signalent le même manquement en premier.
func sortedKeys[T any](table map[string]T) []string {
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
