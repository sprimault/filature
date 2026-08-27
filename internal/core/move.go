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
	MovePlace      MoveType = "place"
	MoveStep       MoveType = "step"
	MoveAbility    MoveType = "ability"
	MoveExpense    MoveType = "expense"
	MoveChangeZone MoveType = "change_zone"
	MovePass       MoveType = "pass"
	MoveEndPhase   MoveType = "end_phase"
)

// Expense énumère les usages de la résistance. Le nom d'une dépense est une
// clé d'effets : un plugin en ajoute une sans toucher au noyau.
type Expense string

// Les dépenses de la règle standard. Leurs coûts ne sont pas ici mais dans le
// manifeste de plugins/base, ce qui permet de les rééquilibrer sans recompiler.
const (
	ExpenseDoubleStep Expense = "double_step"
	ExpenseSilence    Expense = "silence"
	ExpenseWipeTrails Expense = "wipe_trails"
	ExpenseChangeZone Expense = "change_zone"
	ExpenseDecoy      Expense = "decoy"
	ExpenseMurder     Expense = "murder"
)

// Move est volontairement un enregistrement plat plutôt qu'une interface.
//
// Il doit se sérialiser sans effort pour le journal, le réseau et le rejeu, et
// se compare par égalité pour tester qu'un coup proposé figure bien dans
// LegalMoves. Les champs inutilisés restent à zéro.
// From et To ne servent pas qu'aux déplacements : une dépense qui agit sur une
// case les porte aussi, et le leurre s'en sert pour dire où poser sa trace et
// vers quoi elle pointe. Un leurre est un déplacement qui n'a pas lieu, donc
// les règles de déplacement s'y appliquent telles quelles — praticabilité,
// angles fermés — sans qu'une seconde validation ait à les redire.
type Move struct {
	Turn    int      `json:"turn"`
	Side    Side     `json:"side"`
	Type    MoveType `json:"type"`
	Piece   int      `json:"piece,omitempty"`
	From    Position `json:"from,omitempty"`
	To      Position `json:"to,omitempty"`
	Ability string   `json:"ability,omitempty"`
	Expense Expense  `json:"expense,omitempty"`
	Zone    int      `json:"zone,omitempty"`
}
