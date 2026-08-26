// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

// TypeCoup énumère ce qui peut être joué. Les valeurs partent en base et sur le
// réseau : elles ne se renomment pas sans migration.
type TypeCoup string

// CoupFinDePhase est explicite plutôt que déduit du quota : les inspecteurs
// peuvent rendre la main avant leurs trois déplacements, et le journal doit
// distinguer ce choix d'une phase épuisée.
const (
	CoupPlacer      TypeCoup = "placer"
	CoupDeplacer    TypeCoup = "deplacer"
	CoupCapacite    TypeCoup = "capacite"
	CoupDepense     TypeCoup = "depense"
	CoupChangerZone TypeCoup = "changer_zone"
	CoupPasser      TypeCoup = "passer"
	CoupFinDePhase  TypeCoup = "fin_de_phase"
)

// Depense énumère les usages de la résistance. Le nom d'une dépense est une
// clé d'effets : un greffon en ajoute une sans toucher au noyau.
type Depense string

// Les cinq dépenses de la règle standard. Leurs coûts ne sont pas ici mais
// dans le manifeste de greffons/base, ce qui permet de les rééquilibrer sans
// recompiler.
const (
	DepenseDoubleDeplacement Depense = "double_deplacement"
	DepenseSilence           Depense = "silence"
	DepenseEffacement        Depense = "effacement"
	DepenseChangerZone       Depense = "changer_zone"
	DepenseMeurtre           Depense = "meurtre"
)

// Coup est volontairement un enregistrement plat plutôt qu'une interface.
//
// Il doit se sérialiser sans effort pour le journal, le réseau et le rejeu, et
// se comparer par égalité pour tester qu'un coup proposé figure bien dans
// CoupsLegaux. Les champs inutilisés restent à zéro.
type Coup struct {
	Tour     int      `json:"tour"`
	Acteur   Acteur   `json:"acteur"`
	Type     TypeCoup `json:"type"`
	Pion     int      `json:"pion,omitempty"`
	Depart   Position `json:"depart,omitempty"`
	Arrivee  Position `json:"arrivee,omitempty"`
	Capacite string   `json:"capacite,omitempty"`
	Depense  Depense  `json:"depense,omitempty"`
	Zone     int      `json:"zone,omitempty"`
}
