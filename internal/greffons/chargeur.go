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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strings"

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
//
// Un dossier de greffons absent n'est pas une erreur — c'est l'installation
// ordinaire, personne n'ayant rien ajouté.
func Charger(livres fs.FS, racine string) (*noyau.Registre, error) {
	r := &noyau.Registre{}

	if err := verser(r, livres, "livré"); err != nil {
		return nil, err
	}

	if racine != "" {
		if _, err := os.Stat(racine); err == nil {
			if err := verser(r, os.DirFS(racine), racine); err != nil {
				return nil, err
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("dossier des greffons %s: %w", racine, err)
		}
	}

	return r, nil
}

// verser lit tous les greffons d'une source et les ajoute au registre.
//
// Un dossier sans manifeste est ignoré sans bruit : c'est ce que produit un
// dossier de travail laissé à côté, et refuser le démarrage pour ça
// n'apporterait rien.
func verser(r *noyau.Registre, source fs.FS, origine string) error {
	entrees, err := fs.ReadDir(source, ".")
	if err != nil {
		return fmt.Errorf("lecture des greffons %s: %w", origine, err)
	}

	for _, e := range entrees {
		if !e.IsDir() {
			continue
		}
		dossier := e.Name()
		if _, err := fs.Stat(source, path(dossier, NomManifeste)); err != nil {
			continue
		}

		m, err := lireManifeste(source, dossier)
		if err != nil {
			return err
		}

		somme, err := empreinte(source, dossier)
		if err != nil {
			return err
		}

		if err := r.Fusionner(m.Nom, m.versRegistre(somme)); err != nil {
			return err
		}
	}
	return nil
}

// versRegistre projette un manifeste dans ce que le noyau consomme.
//
// Les libellés n'y figurent pas : le noyau n'affiche rien, et lui confier un
// dictionnaire lui donnerait une responsabilité d'interface. Ils se lisent
// depuis langue.toml par qui en a besoin.
func (m *manifeste) versRegistre(somme string) *noyau.Registre {
	r := &noyau.Registre{
		Manifeste: []noyau.EntreeManifeste{{
			Nom:       m.Nom,
			Version:   m.Version,
			Empreinte: somme,
			Regles:    m.Regles,
		}},
	}

	if len(m.Capacites) > 0 {
		r.Capacites = make(map[string]noyau.Capacite, len(m.Capacites))
		for cle, c := range m.Capacites {
			c.Cle = cle
			r.Capacites[cle] = c
		}
	}

	if len(m.Depenses) > 0 {
		r.Depenses = make(map[noyau.Depense]noyau.Capacite, len(m.Depenses))
		for cle, d := range m.Depenses {
			d.Cle = cle
			r.Depenses[noyau.Depense(cle)] = d
		}
	}

	if len(m.Modes) > 0 {
		r.Modes = make(map[string]noyau.Mode, len(m.Modes))
		for cle, mode := range m.Modes {
			mode.Cle = cle
			r.Modes[cle] = mode
		}
	}

	return r
}

// Valider lit un greffon depuis le disque et rend tous ses manquements.
//
// C'est le même contrôle que celui du chargement, et c'est ce qui en fait la
// promesse : un greffon accepté ici se chargera chez les autres. Le vérifier
// avant de publier vaut mieux que d'apprendre le problème par un jeu qui ne
// démarre plus.
func Valider(dossier string) error {
	parent, nom := filepathSplit(dossier)

	if _, err := lireManifeste(os.DirFS(parent), nom); err != nil {
		return err
	}
	return nil
}

// Empreinte calcule la somme du contenu d'un greffon, manifeste et module
// WebAssembly compris.
//
// Elle porte sur le contenu et pas sur le numéro de version, parce que c'est
// ce qui permet de détecter deux greffons qui se disent identiques sans
// l'être — cas normal pendant le développement d'un mod, et cas litigieux en
// réseau.
func Empreinte(dossier string) (string, error) {
	parent, nom := filepathSplit(dossier)
	return empreinte(os.DirFS(parent), nom)
}

// empreinte parcourt un greffon dans l'ordre alphabétique de ses chemins.
//
// Le nom de chaque fichier entre dans la somme avant son contenu : sans lui,
// renommer un fichier ou déplacer une ligne d'un fichier à l'autre laisserait
// l'empreinte inchangée.
//
// Elle garantit que le fichier est celui qui a été relu, pas que son auteur est
// honnête. Ce n'est pas une signature, et une vraie demanderait une gestion de
// clés qui ne vaut pas son coût ici.
func empreinte(source fs.FS, dossier string) (string, error) {
	var chemins []string
	err := fs.WalkDir(source, dossier, func(chemin string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !e.IsDir() {
			chemins = append(chemins, chemin)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("empreinte de %s: %w", dossier, err)
	}
	sort.Strings(chemins)

	somme := sha256.New()
	for _, chemin := range chemins {
		// hash.Hash promet que Write ne renvoie jamais d'erreur, contrairement
		// à la lecture du fichier juste en dessous.
		relatif := strings.TrimPrefix(chemin, dossier+"/")
		_, _ = fmt.Fprintf(somme, "%s\n", relatif)

		f, err := source.Open(chemin)
		if err != nil {
			return "", fmt.Errorf("empreinte de %s: %w", chemin, err)
		}
		_, err = io.Copy(somme, f)
		_ = f.Close()
		if err != nil {
			return "", fmt.Errorf("empreinte de %s: %w", chemin, err)
		}
	}
	return hex.EncodeToString(somme.Sum(nil)), nil
}

// filepathSplit sépare un chemin en son parent et son dernier élément, en
// acceptant les deux séparateurs.
//
// filepath.Split ne connaît que celui du système : sous Windows il laisserait
// intact un chemin écrit avec des barres obliques, alors qu'un dossier de
// greffons peut venir d'un drapeau tapé à la main.
func filepathSplit(chemin string) (parent, nom string) {
	nettoye := strings.TrimRight(strings.ReplaceAll(chemin, "\\", "/"), "/")
	if i := strings.LastIndex(nettoye, "/"); i >= 0 {
		return nettoye[:i], nettoye[i+1:]
	}
	return ".", nettoye
}
