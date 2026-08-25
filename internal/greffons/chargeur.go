// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Package greffons charge les extensions depuis le disque et les expose au
// noyau sous forme de registre.
//
// Le principe directeur : la donnée d'abord, le code seulement si nécessaire.
// La grande majorité de ce qu'un moddeur veut faire — une capacité, une
// dépense, un préréglage, un mode de jeu — se décrit dans un manifeste TOML, ce
// qui évite le bac à sable, les failles et les problèmes de compatibilité de
// version, et rend le modding accessible à quelqu'un qui ne programme pas.
package greffons

import (
	"errors"
	"io/fs"

	"github.com/sprimault/filature/internal/noyau"
)

// Charger construit le registre depuis le contenu livré puis le dossier de
// greffons du joueur.
//
// Les deux sources sont lues par le même code, et c'est le seul point qui
// compte : le contenu livré vient d'un système de fichiers embarqué dans le
// binaire, les greffons tiers du disque, mais rien dans le chargement ne les
// distingue. Un raccourci pour le contenu livré laisserait ce chemin non testé
// jusqu'au jour où quelqu'un installe son premier greffon.
//
// C'est aussi le seul endroit où un registre se remplit. Le noyau n'en fabrique
// pas : il n'a aucune dépendance disque, et lire un manifeste lui en donnerait
// une pour un travail qui n'est pas le sien.
//
// L'ordre de chargement est alphabétique au sein de chaque source, donc
// déterministe. Un manifeste invalide fait échouer le chargement entier plutôt
// que d'être ignoré : un greffon à moitié actif est pire qu'un greffon absent.
func Charger(livres fs.FS, racine string) (*noyau.Registre, error) {
	return nil, errors.New("à implémenter : étape 8")
}

// Empreinte calcule la somme du contenu d'un greffon, manifeste et module
// WebAssembly compris.
//
// Elle porte sur le contenu et pas sur le numéro de version, parce que c'est
// ce qui permet de détecter deux greffons qui se disent identiques sans
// l'être — cas normal pendant le développement d'un mod, et cas litigieux en
// réseau.
func Empreinte(dossier string) (string, error) {
	return "", errors.New("à implémenter : étape 8")
}
