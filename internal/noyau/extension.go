// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import (
	"fmt"
	"sort"
)

// VersionEffets est la version du vocabulaire que ce binaire sait appliquer.
//
// Un greffon écrit contre une version inconnue est refusé plutôt qu'appliqué de
// travers. Sans ce numéro, un manifeste employant une primitive apparue plus
// tard échouerait sur un message de champ inconnu.
const VersionEffets = 1

// Declenchement dit quand une capacité ou un mode entre en jeu.
type Declenchement string

// Les six moments de déclenchement. SurContact et SurRevelation n'ont aucun
// usage dans le contenu livré : ils existent pour les greffons de règles.
const (
	SurPhaseInspecteurs Declenchement = "phase_inspecteurs"
	SurPhaseFugitif     Declenchement = "phase_fugitif"
	SurFinDeTour        Declenchement = "fin_de_tour"
	SurContact          Declenchement = "contact"
	SurRevelation       Declenchement = "revelation"
	SurEtranglement     Declenchement = "etranglement"
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

// Mode est une règle de partie déclarée en effets, que le noyau déclenche sans
// qu'un joueur la choisisse.
//
// L'étranglement en est un. La cadence — à partir de quel tour, tous les
// combien — reste dans Parametres, où l'interface l'expose : le mode dit ce qui
// se passe, le paramètre dit quand. Les inscrire tous deux ici donnerait deux
// sources de vérité pour un même réglage.
type Mode struct {
	Cle           string        `toml:"-" json:"cle"`
	Nom           string        `toml:"nom" json:"nom"`
	Declenchement Declenchement `toml:"declenchement" json:"declenchement"`
	Effets        []Effet       `toml:"effet" json:"effets"`
}

// Registre rassemble tout ce que les greffons ont apporté, plus le contenu de
// base. Le noyau ne connaît que le registre, jamais un greffon en particulier.
type Registre struct {
	Capacites map[string]Capacite
	Depenses  map[Depense]Capacite
	Modes     map[string]Mode

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

// Le registre se construit dans internal/greffons, jamais ici : le remplir
// demande de lire des manifestes, et ce paquet n'a aucune dépendance disque.
// C'est ce qui en fait une feuille du graphe de dépendances, et ce qui garantit
// qu'aucune règle n'attend un fichier pour s'appliquer.

// Fusionner ajoute un greffon au registre. Une clé déjà prise est un conflit,
// pas un écrasement silencieux : deux greffons qui redéfinissent la même
// capacité doivent faire échouer le chargement avec un message lisible.
//
// Les clés sont parcourues triées. L'ordre d'une map n'est pas stable en Go, et
// il déciderait ici duquel de deux conflits l'utilisateur entend parler — donc
// du message qu'il voit, qui changerait d'un lancement à l'autre.
func (r *Registre) Fusionner(nom string, autre *Registre) error {
	if autre == nil {
		return nil
	}

	if err := fusionner(&r.Capacites, autre.Capacites, nom, "capacite"); err != nil {
		return err
	}
	if err := fusionner(&r.Depenses, autre.Depenses, nom, "depense"); err != nil {
		return err
	}
	if err := fusionner(&r.Modes, autre.Modes, nom, "mode"); err != nil {
		return err
	}
	if err := fusionner(&r.Generateurs, autre.Generateurs, nom, "generateur"); err != nil {
		return err
	}
	if err := fusionner(&r.Cerveaux, autre.Cerveaux, nom, "cerveau"); err != nil {
		return err
	}

	r.Manifeste = append(r.Manifeste, autre.Manifeste...)
	return nil
}

// fusionner verse une table dans une autre en refusant les clés déjà prises.
//
// La destination est un pointeur : un registre neuf a ses tables à nil, et les
// initialiser au premier ajout évite de les construire pour les greffons qui
// n'apportent rien de ce genre — la plupart, un dictionnaire n'ayant ni
// capacité ni mode.
func fusionner[T any, C comparable](vers *map[C]T, depuis map[C]T, greffon, genre string) error {
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
			return fmt.Errorf("greffon %s: la %s %v est deja definie", greffon, genre, cle)
		}
		(*vers)[cle] = depuis[cle]
	}
	return nil
}
