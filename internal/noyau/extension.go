// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import "errors"

// Declenchement dit quand une capacité peut être jouée.
type Declenchement string

// Les cinq moments où une capacité peut se déclencher. SurFinDeTour,
// SurContact et SurRevelation n'ont aucun usage dans le contenu livré : ils
// existent pour les greffons de règles.
const (
	SurPhaseInspecteurs Declenchement = "phase_inspecteurs"
	SurPhaseFugitif     Declenchement = "phase_fugitif"
	SurFinDeTour        Declenchement = "fin_de_tour"
	SurContact          Declenchement = "contact"
	SurRevelation       Declenchement = "revelation"
)

// Capacite est une entrée déclarative, chargée depuis un manifeste. Les cinq
// capacités livrées ne sont pas codées en dur : elles vivent dans
// greffons/base, au même format que celles d'un tiers.
type Capacite struct {
	Cle           string        `toml:"-" json:"cle"`
	Nom           string        `toml:"nom" json:"nom"`
	Camp          Acteur        `toml:"camp" json:"camp"`
	Usages        int           `toml:"usages" json:"usages"`
	Cout          int           `toml:"cout" json:"cout"`
	Declenchement Declenchement `toml:"declenchement" json:"declenchement"`
	Passive       bool          `toml:"passive" json:"passive"`
	Effets        []Effet       `toml:"effet" json:"effets"`
}

// Registre rassemble tout ce que les greffons ont apporté, plus le contenu de
// base. Le noyau ne connaît que le registre, jamais un greffon en particulier.
type Registre struct {
	Capacites map[string]Capacite
	Depenses  map[Depense]Capacite

	// Generateurs et Cerveaux sont les deux points d'extension qui ne se
	// décrivent pas en données : un générateur de plateau et une IA. Ils
	// passent par WebAssembly, jamais par du Go chargé dynamiquement.
	Generateurs map[string]FabriquePlateau
	Cerveaux    map[string]FabriqueCerveau

	// Manifeste identifie les greffons actifs. Il part en base avec la
	// partie et se compare à l'établissement d'une connexion réseau : une
	// sauvegarde ne se recharge pas sans ses greffons, et le jeu le dit
	// au lieu de rejouer faux.
	Manifeste []EntreeManifeste
}

// EntreeManifeste identifie un greffon de façon vérifiable. L'empreinte porte
// sur le contenu, pas sur le numéro de version : deux greffons qui se disent
// « 1.2.0 » sans être identiques doivent être détectés.
type EntreeManifeste struct {
	Nom       string `json:"nom"`
	Version   string `json:"version"`
	Empreinte string `json:"empreinte"`
	Regles    bool   `json:"regles"`
}

// Les deux signatures qu'un greffon exécutable peut honorer. Elles sont
// volontairement étroites : tout ce qui passe par elles est déterministe et
// sans effet de bord.
type (
	// FabriquePlateau produit un plateau depuis la graine de la partie.
	FabriquePlateau func(graine int64, p Parametres) (Plateau, error)

	// FabriqueCerveau choisit un coup. Elle reçoit une Vue et non *Partie : un
	// greffon ne peut donc lire ni la position cachée du fugitif ni sa zone
	// scellée.
	FabriqueCerveau func(v Vue, a *Alea) (Coup, error)
)

// RegistreBase charge les capacités et dépenses livrées avec le jeu. Une partie
// sans aucun greffon tiers utilise déjà ce chemin de code : il n'y a pas de
// voie rapide qui court-circuiterait le registre et resterait donc non testée.
func RegistreBase() (*Registre, error) {
	return nil, errors.New("à implémenter : étape 1")
}

// Fusionner ajoute un greffon au registre. Une clé déjà prise est un conflit,
// pas un écrasement silencieux : deux greffons qui redéfinissent la même
// capacité doivent faire échouer le chargement avec un message lisible.
func (r *Registre) Fusionner(nom string, autre *Registre) error {
	return errors.New("à implémenter : étape 8")
}
