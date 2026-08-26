// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"context"
	"errors"

	"github.com/sprimault/filature/internal/core"
)

// Trois choses ne se décrivent pas en données : un générateur de plateau, une
// IA, une condition de victoire arbitraire. Elles passent par WebAssembly.
//
// Le paquet plugin de la bibliothèque standard est écarté : Linux uniquement,
// et il exige que le plugin soit compilé avec exactement la même version du
// compilateur et les mêmes dépendances que l'hôte. Inutilisable pour
// distribuer un binaire.
//
// wazero est un runtime WebAssembly en Go pur, sans cgo, ce qui laisse la
// compilation croisée triviale. Le plugin est isolé, sans disque ni réseau, et
// s'écrit en Go, Rust, Zig ou AssemblyScript — un concours d'IA d'inspecteurs
// devient possible sans faire confiance au code des participants.

// BacASable exécute un module invité.
//
// Ce qui est délibérément absent des hôtes exposés : l'horloge, l'entropie
// système, le système de fichiers, les sockets. L'aléatoire disponible est un
// core.Random dérivé de la graine de la partie. Un plugin non déterministe
// casserait le rejeu du journal, donc les sauvegardes, le débogage et
// l'entraînement de l'IA.
type BacASable struct {
	// budget borne le temps d'exécution d'un appel : une IA tierce ne doit
	// pas pouvoir figer la partie.
	budget int
}

// Open instancie un module et vérifie qu'il expose les fonctions attendues.
func Open(ctx context.Context, wasm []byte, alea *core.Random) (*BacASable, error) {
	return nil, errors.New("à implémenter : étape 13")
}

// Brain adapte un module invité à la signature d'IA du noyau.
func (b *BacASable) Brain() core.FabriqueCerveau {
	// à implémenter : étape 13
	return nil
}

// Generator adapte un module invité à la signature de génération de plateau.
func (b *BacASable) Generator() core.FabriquePlateau {
	// à implémenter : étape 13
	return nil
}
