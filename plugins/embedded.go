// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Package plugins embarque le contenu livré dans le binaire.
//
// C'est ce qui tient la promesse du binaire unique : le jeu a ses règles, ses
// formes et ses libellés sans qu'aucun fichier l'accompagne. Un exécutable
// déplacé continue de fonctionner, et personne ne casse ses règles en éditant
// un fichier qu'il n'a pas écrit.
//
// Le contenu embarqué n'a pour autant aucun statut particulier : il est exposé
// comme un système de fichiers ordinaire, et le chargeur le lit par le même
// chemin qu'un plugin posé sur le disque. C'est ce qui garantit que ce chemin
// est exercé à chaque démarrage plutôt qu'une fois de temps en temps.
package plugins

import (
	"embed"
	"io/fs"
)

//go:embed base anglais
var embarques embed.FS

// Shipped renvoie le contenu livré avec le jeu.
//
// Chaque entrée de la racine est un plugin, comme dans le dossier que désigne
// --plugins : « base » porte les règles, les formes et le français, « anglais »
// la traduction.
func Shipped() fs.FS {
	return embarques
}
