// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"fmt"
	"sort"
)

// EffectsVersion est la version du vocabulaire que ce binaire sait appliquer.
//
// Un plugin écrit contre une version inconnue est refusé plutôt qu'appliqué de
// travers. Sans ce numéro, un manifeste employant une primitive apparue plus
// tard échouerait sur un message de champ inconnu.
const EffectsVersion = 3

// Trigger dit quand une capacité ou un mode entre en jeu.
type Trigger string

// Les six moments de déclenchement. OnContact et OnReveal n'ont aucun
// usage dans le contenu livré : ils existent pour les plugins de règles.
const (
	OnInspectorsPhase Trigger = "inspectors_phase"
	OnFugitivePhase   Trigger = "fugitive_phase"
	OnTurnEnd         Trigger = "turn_end"
	OnContact         Trigger = "contact"
	OnReveal          Trigger = "reveal"
	OnStrangling      Trigger = "strangling"
)

// Triggers énumère les moments de déclenchement, dans l'ordre de leur
// déclaration.
//
// Comme EffectTypes et Targets, et pour la même raison : le chargeur en a
// besoin pour refuser ce que le noyau ne saura pas déclencher. Sans elle, un
// manifeste pouvait poser n'importe quelle chaîne dans trigger — le contrôle
// n'existait pas, et une capacité mal déclarée n'entrait jamais en jeu sans
// qu'un message le dise.
func Triggers() []Trigger {
	return []Trigger{
		OnInspectorsPhase, OnFugitivePhase, OnTurnEnd,
		OnContact, OnReveal, OnStrangling,
	}
}

// Ability est une entrée déclarative, chargée depuis un manifeste. Les cinq
// capacités livrées ne sont pas codées en dur : elles vivent dans
// plugins/base, au même format que celles d'un tiers.
type Ability struct {
	Key     string   `toml:"-" json:"key"`
	Name    string   `toml:"name" json:"name"`
	Camp    Side     `toml:"side" json:"side"`
	Uses    int      `toml:"uses" json:"uses"`
	Cost    int      `toml:"cost" json:"cost"`
	Trigger Trigger  `toml:"trigger" json:"trigger"`
	Passive bool     `toml:"passive" json:"passive"`
	Effects []Effect `toml:"effect" json:"effects"`
}

// Mode est une règle de partie déclarée en effets, que le noyau déclenche sans
// qu'un joueur la choisisse.
//
// L'étranglement en est un. La cadence — à partir de quel tour, tous les
// combien — reste dans Settings, où l'interface l'expose : le mode dit ce qui
// se passe, le paramètre dit quand. Les inscrire tous deux ici donnerait deux
// sources de vérité pour un même réglage.
type Mode struct {
	Key     string   `toml:"-" json:"key"`
	Name    string   `toml:"name" json:"name"`
	Trigger Trigger  `toml:"trigger" json:"trigger"`
	Effects []Effect `toml:"effect" json:"effects"`
}

// Registry rassemble tout ce que les plugins ont apporté, plus le contenu de
// base. Le noyau ne connaît que le registre, jamais un plugin en particulier.
type Registry struct {
	Abilities map[string]Ability
	Expenses  map[Expense]Ability
	Modes     map[string]Mode

	// Generators et Brains sont les deux points d'extension qui ne se
	// décrivent pas en données : un générateur de plateau et une IA. Ils
	// passent par WebAssembly, jamais par du Go chargé dynamiquement.
	Generators map[string]BoardFactory
	Brains     map[string]BrainFactory

	// Manifest identifie les plugins actifs. Il part en base avec la
	// partie et se compare à l'établissement d'une connexion réseau : une
	// sauvegarde ne se recharge pas sans ses plugins, et le jeu le dit
	// au lieu de rejouer faux.
	Manifest []ManifestEntry
}

// ManifestEntry identifie un plugin de façon vérifiable. L'empreinte porte
// sur le contenu, pas sur le numéro de version : deux plugins qui se disent
// « 1.2.0 » sans être identiques doivent être détectés.
type ManifestEntry struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Fingerprint string `json:"fingerprint"`
	Rules       bool   `json:"rules"`
}

// Les deux signatures qu'un plugin exécutable peut honorer. Elles sont
// volontairement étroites : tout ce qui passe par elles est déterministe et
// sans effet de bord.
type (
	// BoardFactory produit un plateau depuis la graine de la partie.
	BoardFactory func(graine int64, p Settings) (Board, error)

	// BrainFactory choisit un coup. Elle reçoit une View et non *Game : un
	// plugin ne peut donc lire ni la position cachée du fugitif ni sa zone
	// scellée.
	BrainFactory func(v View, a *Random) (Move, error)
)

// Le registre se construit dans internal/plugins, jamais ici : le remplir
// demande de lire des manifestes, et ce paquet n'a aucune dépendance disque.
// C'est ce qui en fait une feuille du graphe de dépendances, et ce qui garantit
// qu'aucune règle n'attend un fichier pour s'appliquer.

// Merge ajoute un plugin au registre. Une clé déjà prise est un conflit,
// pas un écrasement silencieux : deux plugins qui redéfinissent la même
// capacité doivent faire échouer le chargement avec un message lisible.
//
// Les clés sont parcourues triées. L'ordre d'une map n'est pas stable en Go, et
// il déciderait ici duquel de deux conflits l'utilisateur entend parler — donc
// du message qu'il voit, qui changerait d'un lancement à l'autre.
func (r *Registry) Merge(nom string, autre *Registry) error {
	if autre == nil {
		return nil
	}

	if err := mergeInto(&r.Abilities, autre.Abilities, nom, "capacite"); err != nil {
		return err
	}
	if err := mergeInto(&r.Expenses, autre.Expenses, nom, "depense"); err != nil {
		return err
	}
	if err := mergeInto(&r.Modes, autre.Modes, nom, "mode"); err != nil {
		return err
	}
	if err := mergeInto(&r.Generators, autre.Generators, nom, "generateur"); err != nil {
		return err
	}
	if err := mergeInto(&r.Brains, autre.Brains, nom, "cerveau"); err != nil {
		return err
	}

	r.Manifest = append(r.Manifest, autre.Manifest...)
	return nil
}

// mergeInto verse une table dans une autre en refusant les clés déjà prises.
//
// La destination est un pointeur : un registre neuf a ses tables à nil, et les
// initialiser au premier ajout évite de les construire pour les plugins qui
// n'apportent rien de ce genre — la plupart, un dictionnaire n'ayant ni
// capacité ni mode.
func mergeInto[T any, C comparable](vers *map[C]T, depuis map[C]T, plugin, genre string) error {
	if len(depuis) == 0 {
		return nil
	}
	if *vers == nil {
		*vers = make(map[C]T, len(depuis))
	}

	cles := make([]C, 0, len(depuis))
	for cle := range depuis {
		cles = append(cles, cle)
	}
	sort.Slice(cles, func(i, j int) bool {
		return fmt.Sprint(cles[i]) < fmt.Sprint(cles[j])
	})

	for _, cle := range cles {
		if _, pris := (*vers)[cle]; pris {
			return fmt.Errorf("plugin %s: la %s %v est deja definie", plugin, genre, cle)
		}
		(*vers)[cle] = depuis[cle]
	}
	return nil
}
