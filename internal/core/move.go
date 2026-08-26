// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

// MoveType énumère ce qui peut être joué. Les valeurs partent en base et sur le
// réseau : elles ne se renomment pas sans migration.
type MoveType string

// MoveEndPhase est explicite plutôt que déduit du quota : les inspecteurs
// peuvent rendre la main avant leurs trois déplacements, et le journal doit
// distinguer ce choix d'une phase épuisée.
const (
	MovePlace      MoveType = "placer"
	MoveStep       MoveType = "deplacer"
	MoveAbility    MoveType = "capacite"
	MoveExpense    MoveType = "depense"
	MoveChangeZone MoveType = "changer_zone"
	MovePass       MoveType = "passer"
	MoveEndPhase   MoveType = "fin_de_phase"
)

// Expense énumère les usages de la résistance. Le nom d'une dépense est une
// clé d'effets : un plugin en ajoute une sans toucher au noyau.
type Expense string

// Les cinq dépenses de la règle standard. Leurs coûts ne sont pas ici mais
// dans le manifeste de plugins/base, ce qui permet de les rééquilibrer sans
// recompiler.
const (
	ExpenseDoubleStep Expense = "double_deplacement"
	ExpenseSilence    Expense = "silence"
	ExpenseWipeTrails Expense = "effacement"
	ExpenseChangeZone Expense = "changer_zone"
	ExpenseMurder     Expense = "meurtre"
)

// Move est volontairement un enregistrement plat plutôt qu'une interface.
//
// Il doit se sérialiser sans effort pour le journal, le réseau et le rejeu, et
// se compare par égalité pour tester qu'un coup proposé figure bien dans
// LegalMoves. Les champs inutilisés restent à zéro.
type Move struct {
	Turn    int      `json:"tour"`
	Side    Side     `json:"acteur"`
	Type    MoveType `json:"type"`
	Piece   int      `json:"pion,omitempty"`
	From    Position `json:"depart,omitempty"`
	To      Position `json:"arrivee,omitempty"`
	Ability string   `json:"capacite,omitempty"`
	Expense Expense  `json:"depense,omitempty"`
	Zone    int      `json:"zone,omitempty"`
}
