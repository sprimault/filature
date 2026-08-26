// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Package stockage persiste les parties dans SQLite.
//
// Le journal des coups est la source de vérité ; l'instantané n'est qu'un cache
// de reprise rapide. C'est ce qui donne d'un coup la reprise, le rejeu pas à pas
// pour déboguer, l'annulation, et le corpus d'entraînement de l'IA.
package storage

import (
	"context"
	"errors"

	"github.com/sprimault/filature/internal/core"
)

// Depot ouvre la base et sert les opérations de partie.
//
// modernc.org/sqlite plutôt que mattn/go-sqlite3 : implémentation en Go pur,
// donc pas de cgo ajouté par le stockage. Ebitengine en impose déjà sur darwin
// et linux, mais en ajouter une deuxième source rendrait la cible windows et la
// cible wasm impossibles.
type Depot struct{}

// Ouvrir crée le fichier au besoin et applique le schéma.
func Ouvrir(chemin string) (*Depot, error) {
	return nil, errors.New("à implémenter : étape 8")
}

// Enregistrer écrit l'instantané et les coups nouveaux depuis le dernier appel.
func (d *Depot) Enregistrer(ctx context.Context, nom string, p *core.Partie) error {
	return errors.New("à implémenter : étape 8")
}

// Reprendre reconstruit une partie en rejouant son journal.
//
// Le rejeu, pas la lecture de l'instantané : c'est ce qui vérifie en continu
// que le journal reste suffisant. Un instantané chargé sans rejeu masquerait le
// jour où une règle cesse d'être reproductible.
func (d *Depot) Reprendre(ctx context.Context, nom string) (*core.Partie, error) {
	return nil, errors.New("à implémenter : étape 8")
}

// Lister renvoie les parties enregistrées, la plus récente en tête.
func (d *Depot) Lister(ctx context.Context) ([]Resume, error) {
	return nil, errors.New("à implémenter : étape 8")
}

// Resume est ce qu'affiche l'écran de reprise.
type Resume struct {
	Nom        string
	Tour       int
	Terminee   bool
	ModifieeLe string
}
