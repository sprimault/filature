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

	"github.com/sprimault/filature/internal/noyau"
)

// Charger lit un dossier de greffons et construit le registre.
//
// L'ordre de chargement est alphabétique, donc déterministe. Un manifeste
// invalide fait échouer le chargement entier plutôt que d'être ignoré : un
// greffon à moitié actif est pire qu'un greffon absent.
func Charger(racine string, base *noyau.Registre) (*noyau.Registre, error) {
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
