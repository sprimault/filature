// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import "errors"

// TypeEffet est le vocabulaire de modding.
//
// C'est le contrat central du projet : une capacité, une dépense de résistance
// ou un mode de jeu se décrit par composition de ces primitives, sans une ligne
// de code. Les cinq capacités de base sont écrites dans ce format dès le
// premier jour — c'est le test qui prouve que le vocabulaire suffit. Si l'une
// d'elles ne s'exprime pas ici, c'est qu'il manque une primitive, pas qu'il
// faut un cas particulier.
//
// Ajouter une primitive est une décision lourde : elle entre dans le contrat
// public et ne peut plus être retirée sans casser les greffons existants.
type TypeEffet string

// Le vocabulaire livré, décrit en entier dans docs/vocabulaire-effets.md.
// EffetTeleporter ne sert à aucune capacité de base — la téléportation a été
// retirée des règles — et reste offert aux greffons qui voudraient la rétablir.
const (
	EffetDeplacer          TypeEffet = "deplacer"
	EffetModifierPortee    TypeEffet = "modifier_portee"
	EffetModifierMobilite  TypeEffet = "modifier_mobilite"
	EffetBloquerCase       TypeEffet = "bloquer_case"
	EffetOuvrirCase        TypeEffet = "ouvrir_case"
	EffetRevelerTraces     TypeEffet = "reveler_traces"
	EffetRevelerPosition   TypeEffet = "reveler_position"
	EffetAnnulerRevelation TypeEffet = "annuler_revelation"
	EffetPartagerVue       TypeEffet = "partager_vue"
	EffetCouterResistance  TypeEffet = "couter_resistance"
	EffetRendreResistance  TypeEffet = "rendre_resistance"
	EffetEffacerTraces     TypeEffet = "effacer_traces"
	EffetFermerZone        TypeEffet = "fermer_zone"
	EffetOuvrirZone        TypeEffet = "ouvrir_zone"
	EffetTeleporter        TypeEffet = "teleporter"
	EffetDifferer          TypeEffet = "differer"
	EffetFinPartie         TypeEffet = "fin_partie"
)

// Cible désigne ce sur quoi un effet s'applique.
type Cible string

// CiblePionCourant est le cas ordinaire ; les autres valeurs supposent que le
// contexte porte le pion, la case ou la zone visée.
const (
	CiblePionCourant Cible = "pion_courant"
	CibleTousPions   Cible = "tous_pions"
	CibleAutrePion   Cible = "autre_pion"
	CibleFugitif     Cible = "fugitif"
	CibleCase        Cible = "case"
	CibleZone        Cible = "zone"
)

// Effet est une primitive paramétrée. Les champs inutiles restent à zéro : un
// enregistrement plat se sérialise et se journalise sans effort, contrairement
// à une hiérarchie de types.
type Effet struct {
	Type   TypeEffet `toml:"type" json:"type"`
	Cible  Cible     `toml:"cible" json:"cible,omitempty"`
	Valeur int       `toml:"valeur" json:"valeur,omitempty"`
	Duree  int       `toml:"duree" json:"duree,omitempty"`
	Rayon  int       `toml:"rayon" json:"rayon,omitempty"`

	// Annonce et Puis n'ont de sens que pour EffetDifferer.
	//
	// Un effet différé annoncé figure dans la Vue des deux camps, et c'est
	// tout son intérêt : un mur qui apparaît sans prévenir transformerait un
	// plan raisonné en coup de dé et rendrait la carte de croyance inutile.
	//
	// Un differer imbriqué dans un differer est refusé au chargement : deux
	// durées s'additionnent, donc ça n'ajoute rien, et ça permettrait des
	// chaînes qu'aucune annulation ne saurait dérouler.
	Annonce bool    `toml:"annonce" json:"annonce,omitempty"`
	Puis    []Effet `toml:"puis" json:"puis,omitempty"`
}

// EffetEnAttente est une entrée de la file des effets différés.
//
// Résolue en fin de tour, avant le test de fin de partie. L'annulation défait
// la mise en file, pas l'effet : annuler le tour où le differer a été posé le
// retire de la file.
type EffetEnAttente struct {
	Effets   []Effet  `json:"effets"`
	Tour     int      `json:"tour"`
	Annonce  bool     `json:"annonce"`
	Contexte Contexte `json:"contexte"`
}

// Contexte est ce dont un effet dispose pour s'appliquer. Il ne donne pas accès
// à la Partie entière : un greffon ne doit pas pouvoir lire la zone scellée du
// fugitif ni écrire dans le journal.
type Contexte struct {
	Acteur Acteur   `json:"acteur"`
	Pion   int      `json:"pion"`
	Case   Position `json:"case"`
	Zone   int      `json:"zone"`
}

// Appliquer1Effet exécute un effet et renvoie de quoi le défaire.
//
// Le retour n'est pas optionnel : Partie.Annuler doit rester praticable, sinon
// l'IA ne peut plus explorer et le rejeu du journal diverge dès qu'un greffon
// est actif.
func (p *Partie) Appliquer1Effet(e Effet, ctx Contexte) (annulation func(), err error) {
	return nil, errors.New("à implémenter : étape 1")
}
