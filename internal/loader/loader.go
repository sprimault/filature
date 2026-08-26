// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Package plugins charge les extensions depuis le disque et les expose au
// noyau sous forme de registre.
//
// Le principe directeur : la donnée d'abord, le code seulement si nécessaire.
// La grande majorité de ce qu'un moddeur veut faire — une capacité, une
// dépense, un préréglage, un mode de jeu — se décrit dans un manifeste TOML, ce
// qui évite le bac à sable, les failles et les problèmes de compatibilité de
// version, et rend le modding accessible à quelqu'un qui ne programme pas.
package loader

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

	"github.com/sprimault/filature/internal/core"
	"github.com/sprimault/filature/internal/render"
	"github.com/sprimault/filature/plugins"
)

// Load construit le registre depuis le contenu livré puis le dossier de
// plugins du joueur.
//
// Les deux sources sont lues par le même code, et c'est le seul point qui
// compte : le contenu livré vient d'un système de fichiers embarqué dans le
// binaire, les plugins tiers du disque, mais rien dans le chargement ne les
// distingue. Un raccourci pour le contenu livré laisserait ce chemin non testé
// jusqu'au jour où quelqu'un installe son premier plugin.
//
// C'est aussi le seul endroit où un registre se remplit. Le noyau n'en fabrique
// pas : il n'a aucune dépendance disque, et lire un manifeste lui en donnerait
// une pour un travail qui n'est pas le sien.
//
// L'ordre de chargement est alphabétique au sein de chaque source, donc
// déterministe. Un manifeste invalide fait échouer le chargement entier plutôt
// que d'être ignoré : un plugin à moitié actif est pire qu'un plugin absent.
//
// Un dossier de plugins absent n'est pas une erreur — c'est l'installation
// ordinaire, personne n'ayant rien ajouté.
//
// L'apparence remonte à part du registre : celui-ci vit dans internal/core, qui
// est une feuille du graphe de dépendances et n'a pas à connaître le rendu.
func Load(livres fs.FS, racine string) (*core.Registry, *render.ShapeSet, error) {
	r := &core.Registry{}
	j := &render.ShapeSet{
		Shapes:  map[string]render.Shape{},
		Palette: render.Palette{},
		Origins: map[string]string{},
	}

	if err := pourInto(r, j, livres, "livré", true); err != nil {
		return nil, nil, err
	}

	if racine != "" {
		if _, err := os.Stat(racine); err == nil {
			if err := pourInto(r, j, os.DirFS(racine), racine, false); err != nil {
				return nil, nil, err
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("dossier des plugins %s: %w", racine, err)
		}
	}

	// Après fusion et non pendant : une forme peut référencer une couleur
	// qu'une palette chargée plus tard définit, et juger chaque plugin isolément
	// refuserait un ensemble pourtant cohérent.
	if manquements := j.Validate(); len(manquements) > 0 {
		return nil, nil, fmt.Errorf("apparence : %w", errors.Join(manquements...))
	}

	return r, j, nil
}

// LoadOne charge le contenu livré et un seul plugin, désigné par son chemin.
//
// Pour l'aperçu et rien d'autre : on veut voir ce qu'un plugin donne, pas ce
// que donne le dossier qui l'héberge. Passer par Load chargerait ses voisins,
// et deux plugins en conflit empêcheraient de regarder le premier.
func LoadOne(livres fs.FS, chemin string) (*core.Registry, *render.ShapeSet, error) {
	parent, nom := filepathSplit(chemin)

	r := &core.Registry{}
	j := &render.ShapeSet{
		Shapes:  map[string]render.Shape{},
		Palette: render.Palette{},
		Origins: map[string]string{},
	}

	if err := pourInto(r, j, livres, "livré", true); err != nil {
		return nil, nil, err
	}
	if err := unPlugin(r, j, os.DirFS(parent), nom, false); err != nil {
		return nil, nil, err
	}

	if manquements := j.Validate(); len(manquements) > 0 {
		return nil, nil, fmt.Errorf("apparence : %w", errors.Join(manquements...))
	}
	return r, j, nil
}

// pourInto lit tous les plugins d'une source et les ajoute au registre.
//
// Un dossier sans manifeste est ignoré sans bruit : c'est ce que produit un
// dossier de travail laissé à côté, et refuser le démarrage pour ça
// n'apporterait rien.
//
// socle distingue le contenu livré des plugins du joueur, pour l'apparence
// seulement. Le chemin de chargement reste le même — c'est ce qui garantit
// qu'il est exercé à chaque démarrage — mais le socle ne revendique pas ses
// formes : surcharger le fugitif livré est précisément ce qu'un plugin
// d'apparence vient faire, et le compter comme un conflit les interdirait tous.
func pourInto(r *core.Registry, j *render.ShapeSet, source fs.FS, origine string, socle bool) error {
	entrees, err := fs.ReadDir(source, ".")
	if err != nil {
		return fmt.Errorf("lecture des plugins %s: %w", origine, err)
	}

	for _, e := range entrees {
		if !e.IsDir() {
			continue
		}
		dossier := e.Name()
		if _, err := fs.Stat(source, path(dossier, ManifestName)); err != nil {
			continue
		}
		if err := unPlugin(r, j, source, dossier, socle); err != nil {
			return err
		}
	}
	return nil
}

// unPlugin ajoute un plugin au registre et au jeu de formes.
//
// Extrait de la boucle pour que LoadOne l'emprunte : charger un seul plugin
// doit suivre exactement le même chemin qu'en charger dix, faute de quoi un
// aperçu montrerait ce qu'une partie ne montrera pas.
func unPlugin(r *core.Registry, j *render.ShapeSet, source fs.FS, dossier string, socle bool) error {
	m, err := readManifest(source, dossier)
	if err != nil {
		return err
	}

	somme, err := fingerprint(source, dossier)
	if err != nil {
		return err
	}

	if err := r.Merge(m.Name, m.versRegistre(somme)); err != nil {
		return err
	}

	formes, err := render.Read(source, dossier)
	if err != nil {
		return err
	}

	revendiquant := m.Name
	if socle {
		revendiquant = ""
	}
	return j.Merge(revendiquant, formes)
}

// versRegistre projette un manifeste dans ce que le noyau consomme.
//
// Les libellés n'y figurent pas : le noyau n'affiche rien, et lui confier un
// dictionnaire lui donnerait une responsabilité d'interface. Ils se lisent
// depuis langue.toml par qui en a besoin.
func (m *manifeste) versRegistre(somme string) *core.Registry {
	r := &core.Registry{
		Manifest: []core.ManifestEntry{{
			Name:        m.Name,
			Version:     m.Version,
			Fingerprint: somme,
			Rules:       m.Rules,
		}},
	}

	if len(m.Abilities) > 0 {
		r.Abilities = make(map[string]core.Ability, len(m.Abilities))
		for cle, c := range m.Abilities {
			c.Key = cle
			r.Abilities[cle] = c
		}
	}

	if len(m.Expenses) > 0 {
		r.Expenses = make(map[core.Expense]core.Ability, len(m.Expenses))
		for cle, d := range m.Expenses {
			d.Key = cle
			r.Expenses[core.Expense(cle)] = d
		}
	}

	if len(m.Modes) > 0 {
		r.Modes = make(map[string]core.Mode, len(m.Modes))
		for cle, mode := range m.Modes {
			mode.Key = cle
			r.Modes[cle] = mode
		}
	}

	return r
}

// Validate lit un plugin depuis le disque et rend tous ses manquements.
//
// C'est le même contrôle que celui du chargement, et c'est ce qui en fait la
// promesse : un plugin accepté ici se chargera chez les autres. Le vérifier
// avant de publier vaut mieux que d'apprendre le problème par un jeu qui ne
// démarre plus.
func Validate(dossier string) error {
	parent, nom := filepathSplit(dossier)
	source := os.DirFS(parent)

	if _, err := readManifest(source, nom); err != nil {
		return err
	}

	formes, err := render.Read(source, nom)
	if err != nil {
		return err
	}

	// Un plugin se valide seul, donc sans la palette livrée sous lui. Ses
	// couleurs ne sont contrôlées que s'il en fournit une : sinon il s'appuie
	// légitimement sur celle du jeu, et exiger qu'il la redéclare reviendrait à
	// refuser un plugin de formes seules.
	if len(formes.Palette) == 0 {
		for nom, couleur := range base().Palette {
			formes.Palette[nom] = couleur
		}
	}

	if manquements := formes.Validate(); len(manquements) > 0 {
		return errors.Join(manquements...)
	}
	return nil
}

// base lit l'apparence livrée, sur laquelle un plugin s'appuie quand il ne
// fournit pas la sienne.
//
// Sans elle, valider un plugin de formes seules reviendrait à lui reprocher
// chaque couleur du socle — celles-là mêmes qu'il a le droit d'attendre.
func base() *render.ShapeSet {
	j, err := render.Read(plugins.Shipped(), "base")
	if err != nil {
		// Le contenu livré est embarqué et testé : s'il ne se lit plus, ce
		// n'est pas un plugin qui est en cause mais le binaire lui-même.
		return &render.ShapeSet{Palette: render.Palette{}}
	}
	return j
}

// Fingerprint calcule la somme du contenu d'un plugin, manifeste et module
// WebAssembly compris.
//
// Elle porte sur le contenu et pas sur le numéro de version, parce que c'est
// ce qui permet de détecter deux plugins qui se disent identiques sans
// l'être — cas normal pendant le développement d'un mod, et cas litigieux en
// réseau.
func Fingerprint(dossier string) (string, error) {
	parent, nom := filepathSplit(dossier)
	return fingerprint(os.DirFS(parent), nom)
}

// fingerprint parcourt un plugin dans l'ordre alphabétique de ses chemins.
//
// Le nom de chaque fichier entre dans la somme avant son contenu : sans lui,
// renommer un fichier ou déplace une ligne d'un fichier à l'autre laisserait
// l'empreinte inchangée.
//
// Elle garantit que le fichier est celui qui a été relu, pas que son auteur est
// honnête. Ce n'est pas une signature, et une vraie demanderait une gestion de
// clés qui ne vaut pas son coût ici.
func fingerprint(source fs.FS, dossier string) (string, error) {
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
// plugins peut venir d'un drapeau tapé à la main.
func filepathSplit(chemin string) (parent, nom string) {
	nettoye := strings.TrimRight(strings.ReplaceAll(chemin, "\\", "/"), "/")
	if i := strings.LastIndex(nettoye, "/"); i >= 0 {
		return nettoye[:i], nettoye[i+1:]
	}
	return ".", nettoye
}
