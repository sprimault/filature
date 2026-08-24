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

// Le vocabulaire livré. EffetTeleporter ne sert à aucune capacité de base — la
// téléportation a été retirée des règles — et reste offert aux greffons qui
// voudraient la rétablir.
const (
	EffetDeplacer          TypeEffet = "deplacer"
	EffetModifierPortee    TypeEffet = "modifier_portee"
	EffetModifierMobilite  TypeEffet = "modifier_mobilite"
	EffetBloquerCase       TypeEffet = "bloquer_case"
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
}

// Contexte est ce dont un effet dispose pour s'appliquer. Il ne donne pas accès
// à la Partie entière : un greffon ne doit pas pouvoir lire la zone scellée du
// fugitif ni écrire dans le journal.
type Contexte struct {
	Acteur Acteur
	Pion   int
	Case   Position
	Zone   int
}

// Appliquer1Effet exécute un effet et renvoie de quoi le défaire.
//
// Le retour n'est pas optionnel : Partie.Annuler doit rester praticable, sinon
// l'IA ne peut plus explorer et le rejeu du journal diverge dès qu'un greffon
// est actif.
func (p *Partie) Appliquer1Effet(e Effet, ctx Contexte) (annulation func(), err error) {
	return nil, errors.New("à implémenter : étape 1")
}
