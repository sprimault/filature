// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

// Package session monte une partie et la joue jusqu'à son terme.
//
// Il assemble ce que les autres paquets fournissent — les règles, le chargeur
// de plugins, le rendu texte, un cerveau — sans rien décider lui-même. La seule
// chose qui lui appartient est l'ordre : qui joue, quand, et ce qu'on montre
// entre deux coups.
package session

import (
	"fmt"
	"io"

	"github.com/sprimault/filature/internal/ai"
	"github.com/sprimault/filature/internal/core"
	"github.com/sprimault/filature/internal/loader"
	"github.com/sprimault/filature/internal/text"
	"github.com/sprimault/filature/plugins"
)

// coupsMax borne une partie contre un cerveau qui n'avancerait pas.
//
// Le compte est large : une partie de quarante tours en consomme quelques
// centaines. Il ne protège pas d'un joueur lent mais d'un bot tiers défaillant,
// qui rendrait la main sans jamais la rendre.
const coupsMax = 100_000

// Options décrit la partie à jouer.
//
// Human vaut le camp du joueur, ou reste vide pour une partie que deux cerveaux
// disputent sans personne. C'est un paramètre de partie et non une option de
// ligne de commande : l'écran de nouvelle partie le fournira de la même façon.
type Options struct {
	Seed     int64
	Settings core.Settings
	Human    core.Side

	// Plugins est le dossier du joueur. Vide, seul le contenu livré est
	// chargé, ce qui est l'installation ordinaire.
	Plugins string

	In  io.Reader
	Out io.Writer
}

// Run joue une partie entière et rend son issue.
//
// Rien n'est affiché sans passer par une vue filtrée, y compris pour qui
// regarde sans jouer : montrer davantage demanderait un accès à l'état complet,
// et ce qu'on ne peut pas montrer depuis deux vues serait un manque du contrat
// de vue plutôt qu'une raison de le contourner.
func Run(o Options) (core.Outcome, error) {
	// Une sortie absente vaut « ne rien montrer », ce dont une partie
	// simulée n'a que faire. Le remplacement se fait ici et non à chaque
	// écriture : un io.Writer nul ne se teste pas de façon fiable — une
	// interface qui porte un pointeur nul n'est pas nulle elle-même.
	if o.Out == nil {
		o.Out = io.Discard
	}

	registre, err := loader.Load(plugins.Shipped(), o.Plugins)
	if err != nil {
		return core.Outcome{}, err
	}

	plateau, graine, err := core.Generate(o.Seed, o.Settings)
	if err != nil {
		return core.Outcome{}, err
	}

	partie, err := core.NewGame(plateau, graine, o.Settings, registre)
	if err != nil {
		return core.Outcome{}, err
	}

	return jouer(partie, o)
}

// jouer enchaîne les coups jusqu'à ce que l'arbitre tranche.
//
// Chaque cerveau tire dans son propre flux : sans cela, le nombre de coups joués
// par l'un décalerait les tirages de l'autre, et rejouer une partie depuis sa
// graine ne redonnerait pas la même.
func jouer(p *core.Game, o Options) (core.Outcome, error) {
	des := map[core.Side]*core.Random{
		core.SideFugitive:   core.NewRandom(p.Seed, "brain_fugitive"),
		core.SideInspectors: core.NewRandom(p.Seed, "brain_inspectors"),
	}

	for joues := 0; joues < coupsMax; joues++ {
		if issue, fini := p.Outcome(); fini {
			afficherLaFin(o, p, issue)
			return issue, nil
		}

		camp, connu := campActif(p.Phase)
		if !connu {
			return core.Outcome{}, fmt.Errorf("phase %s sans camp actif", p.Phase)
		}

		coup, err := choisir(p, o, camp, des[camp])
		if err != nil {
			return core.Outcome{}, err
		}
		if err := p.Apply(coup); err != nil {
			return core.Outcome{}, fmt.Errorf("coup %s refusé : %w", coup.Type, err)
		}
	}

	return core.Outcome{}, fmt.Errorf("partie interrompue après %d coups", coupsMax)
}

// choisir demande son coup à qui doit jouer.
func choisir(p *core.Game, o Options, camp core.Side, des *core.Random) (core.Move, error) {
	vue := p.ViewFor(camp)

	if camp != o.Human {
		return ai.RandomBrain{}.Play(vue, des)
	}

	// L'affichage est un confort : si la sortie est morte, c'est la saisie
	// juste en dessous qui le dira, et elle, on la vérifie.
	_, _ = fmt.Fprint(o.Out, text.Status(vue), text.Board(vue), text.Moves(vue.LegalMoves))
	return text.ReadMove(o.In, o.Out, vue.LegalMoves)
}

// campActif dit qui a la main dans une phase donnée.
//
// La phase terminée n'en a pas : c'est l'arbitre qui l'annonce, et la boucle
// s'arrête avant d'y arriver.
func campActif(phase core.Phase) (core.Side, bool) {
	switch phase {
	case core.PhaseFugitiveSetup, core.PhaseFugitive:
		return core.SideFugitive, true
	case core.PhaseInspectorsSetup, core.PhaseInspectors:
		return core.SideInspectors, true
	default:
		return "", false
	}
}

// afficherLaFin montre le plateau tel qu'il est au dernier coup, puis l'issue.
//
// Pour qui joue, sa propre vue : découvrir la position du fugitif au moment où
// la partie s'achève lui apprendrait ce qu'il a manqué, mais lui apprendrait
// aussi ce qu'il n'avait pas le droit de savoir en jouant — et il rejouera.
func afficherLaFin(o Options, p *core.Game, issue core.Outcome) {
	vue := p.ViewFor(o.Human)
	if o.Human == "" {
		vue = text.Merge(p.ViewFor(core.SideFugitive), p.ViewFor(core.SideInspectors))
	}

	// La partie est jouée : une écriture qui échoue ici ne change plus rien à
	// son issue, que l'appelant reçoit de toute façon.
	_, _ = fmt.Fprint(o.Out, text.Status(vue), text.Board(vue))
	_, _ = fmt.Fprintln(o.Out, text.Ending(issue))
}

// ErrAbandon remonte l'abandon d'un joueur, qui n'est pas une panne.
//
// Exposé ici pour que l'appelant n'ait pas à connaître le paquet de saisie : il
// veut distinguer « il a quitté » de « quelque chose a cassé », pas savoir d'où
// vient la lecture.
var ErrAbandon = text.ErrQuit
