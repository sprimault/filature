// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package preview

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/sprimault/filature/internal/ai"
	"github.com/sprimault/filature/internal/core"
	"github.com/sprimault/filature/internal/render"
)

// Réglages de l'aperçu de plateau. La graine est figée : deux exécutions sur le
// même plugin doivent donner le même fichier, sans quoi le diff d'un aperçu
// n'apprendrait rien.
const (
	graineApercu = 34
	toursApercu  = 16
	margePlateau = 40
)

// Board écrit un plateau en situation.
//
// Une partie est réellement jouée sur une graine figée plutôt qu'une position
// composée à la main : les pions, les traces et les barrages tombent alors où
// le jeu les met, et l'aperçu montre ce qu'on verra en jouant. Une position
// arrangée flatterait les formes exactement là où elles doivent être jugées.
func Board(w io.Writer, j *render.ShapeSet, registre *core.Registry, preset string) error {
	partie, err := jouer(registre, preset)
	if err != nil {
		return err
	}

	var s strings.Builder
	rendre(&s, partie, j)

	_, err = io.WriteString(w, s.String())
	return err
}

// jouer déroule quelques tours pour qu'il y ait des marqueurs à montrer.
//
// Sans cela le plateau n'a ni trace, ni barrage, ni inspecteur placé : la mise
// en place occupe les premiers coups, et l'aperçu ne montrerait qu'un décor.
func jouer(registre *core.Registry, preset string) (*core.Game, error) {
	reglage, connu := core.PresetByKey(preset)
	if !connu {
		return nil, fmt.Errorf("préréglage %q inconnu", preset)
	}

	plateau, graine, err := core.Generate(graineApercu, reglage.Settings)
	if err != nil {
		return nil, err
	}
	partie, err := core.NewGame(plateau, graine, reglage.Settings, registre)
	if err != nil {
		return nil, err
	}

	cerveau := ai.RandomBrain{}
	des := core.NewRandom(graine, "preview")
	for partie.Turn <= toursApercu && partie.Phase != core.PhaseOver {
		camp := core.SideFugitive
		if partie.Phase == core.PhaseInspectors || partie.Phase == core.PhaseInspectorsSetup {
			camp = core.SideInspectors
		}

		coup, err := cerveau.Play(partie.ViewFor(camp), des)
		if err != nil {
			break
		}
		if err := partie.Apply(coup); err != nil {
			break
		}
	}
	return partie, nil
}

// rendre projette l'état en SVG, dans l'ordre de peintre.
func rendre(s *strings.Builder, p *core.Game, j *render.ShapeSet) {
	cote := p.Settings.Size
	largeurCases, hauteurCases := render.Span(cote)
	largeur := largeurCases + 2*margePlateau
	hauteur := hauteurCases + 2*margePlateau

	// L'origine ramène la colonne la plus à gauche contre la marge.
	ox := float64(margePlateau + (cote-1)*render.TileWidth/2)

	fmt.Fprintf(s, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`,
		largeur, hauteur, largeur, hauteur)
	fmt.Fprintf(s, `<title>Plateau au tour %d</title>`, p.Turn)
	fmt.Fprintf(s, `<rect width="%d" height="%d" fill="%s"/>`, largeur, hauteur, j.Palette["backdrop"])

	occupants := releve(p)

	for _, pos := range ordreDePeintre(cote) {
		sx, sy := render.ToScreen(pos.Column, pos.Row)
		x, y := float64(sx)+ox, float64(sy)+margePlateau

		if !p.Board.IsStreet(pos) {
			for _, t := range j.Shapes["building"].Strokes {
				trait(s, t, x, y, 1, j.Palette)
			}
			continue
		}

		losange(s, x, y, 1, j.Palette[occupants.sol[pos]], grain(p.Seed, pos))
		for _, nom := range occupants.formes(pos) {
			for _, t := range j.Shapes[nom].Strokes {
				trait(s, t, x, y, 1, j.Palette)
			}
		}
	}
	s.WriteString(`</svg>`)
}

// occupation dit ce que porte chaque case, sol compris.
type occupation struct {
	sol         map[core.Position]string
	trails      map[core.Position]bool
	roadblocks  map[core.Position]bool
	crimeScenes map[core.Position]bool
	inspectors  map[core.Position]bool
	fugitive    core.Position
}

// formes rend les noms à dessiner sur une case, du sol vers le dessus.
//
// Une trace sous un barrage n'est pas dessinée : la case n'est plus
// franchissable, et le renseignement qu'elle portait n'a plus d'objet.
func (o occupation) formes(p core.Position) []string {
	var noms []string
	if o.trails[p] && !o.roadblocks[p] {
		noms = append(noms, "trail")
	}
	if o.crimeScenes[p] {
		noms = append(noms, "crime_scene")
	}
	if o.roadblocks[p] {
		noms = append(noms, "roadblock")
	}
	if o.inspectors[p] {
		noms = append(noms, "inspector")
	}
	if o.fugitive == p {
		noms = append(noms, "fugitive")
	}
	return noms
}

// releve rassemble en une passe ce que chaque case porte.
func releve(p *core.Game) occupation {
	o := occupation{
		sol:         map[core.Position]string{},
		trails:      map[core.Position]bool{},
		roadblocks:  map[core.Position]bool{},
		crimeScenes: map[core.Position]bool{},
		inspectors:  map[core.Position]bool{},
		fugitive:    p.Fugitive.Position,
	}

	for pos := range p.Trails {
		o.trails[pos] = true
	}
	for pos := range p.Roadblocks {
		o.roadblocks[pos] = true
	}
	for _, c := range p.CrimeScenes {
		o.crimeScenes[c.Position] = true
	}
	for _, i := range p.Inspectors {
		o.inspectors[i.Position] = true
	}

	for _, z := range p.Board.Zones() {
		nom := "zone_open"
		if z.Closed {
			nom = "zone_closed"
		}
		for _, c := range z.Cells {
			o.sol[c] = nom
		}
	}

	cote := p.Settings.Size
	for c := range cote {
		for r := range cote {
			pos := core.Position{Column: c, Row: r}
			if _, dansZone := o.sol[pos]; !dansZone {
				o.sol[pos] = "street"
			}
		}
	}
	return o
}

// ordreDePeintre trie les cases du fond vers l'avant.
//
// colonne + ligne croissants : c'est ce que la projection impose, et c'est ce
// qui permet à un bâtiment de masquer ce qui est derrière lui sans qu'aucun
// calcul de profondeur ne soit nécessaire.
func ordreDePeintre(cote int) []core.Position {
	cases := make([]core.Position, 0, cote*cote)
	for c := range cote {
		for r := range cote {
			cases = append(cases, core.Position{Column: c, Row: r})
		}
	}

	sort.Slice(cases, func(i, k int) bool {
		a, b := cases[i], cases[k]
		if a.Column+a.Row != b.Column+b.Row {
			return a.Column+a.Row < b.Column+b.Row
		}
		return a.Column < b.Column
	})
	return cases
}

// grain traduit l'écart de luminosité d'une case en facteur multiplicatif.
//
// Le calcul lui-même vit dans render : l'aperçu doit montrer le grain du jeu et
// non un grain qui lui ressemble, sans quoi il flatterait ou noircirait des
// formes qu'on cherche justement à juger.
func grain(graine int64, p core.Position) float64 {
	return 1 + float64(render.GroundGrain(graine, p.Column, p.Row))/100
}
