// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package preview

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/sprimault/filature/internal/render"
)

// Dimensions de la planche, en pixels. La colonne est large parce qu'un pion
// s'élève de quarante unités au-dessus de son ancrage.
const (
	pasPlanche     = 150
	hauteurBande   = 200
	margeEtiquette = 8
)

// sols sont les trois fonds sur lesquels une forme peut se poser.
//
// Les trois et non le seul sol de rue : leurs luminances vont de 210 à 82, et
// une forme lisible sur l'un peut disparaître sur l'autre. C'est précisément ce
// qu'un auteur doit voir avant de publier.
var sols = []string{"street", "zone_open", "zone_closed"}

// Shapes écrit la planche des formes, chacune sur les trois sols possibles.
//
// origine désigne le plugin dont on fait l'aperçu : ses formes portent une
// marque, les autres retombent sur le contenu livré. C'est ce qui permet de voir
// qu'une clé mal orthographiée n'a rien surchargé — elle passe la validation
// sans rien changer, et rien d'autre ne le dirait.
func Shapes(w io.Writer, j *render.ShapeSet, origine string) error {
	noms := make([]string, 0, len(j.Shapes))
	for nom := range j.Shapes {
		noms = append(noms, nom)
	}
	slices.Sort(noms)

	largeur := pasPlanche*max(len(noms), 1) + pasPlanche/2
	hauteur := hauteurBande * len(sols)

	var s strings.Builder
	fmt.Fprintf(&s, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`,
		largeur, hauteur, largeur, hauteur)
	fmt.Fprintf(&s, `<title>Formes de %s</title>`, echapper(origine))

	for bande, sol := range sols {
		haut := float64(bande * hauteurBande)
		fmt.Fprintf(&s, `<rect y="%.0f" width="%d" height="%d" fill="%s"/>`,
			haut, largeur, hauteurBande, j.Palette[sol])
		etiquette(&s, margeEtiquette, haut+18, sol, contraste(j.Palette[sol]), 12, "start")

		oy := haut + float64(hauteurBande) - 50
		for i, nom := range noms {
			ox := float64(pasPlanche*i) + pasPlanche/2 + margeEtiquette

			cadre(&s, ox, oy, j.Palette[sol])
			for _, t := range j.Shapes[nom].Strokes {
				trait(&s, t, ox, oy, 1, j.Palette)
			}

			libelle := nom
			if j.Origins[nom] == origine && origine != "" {
				libelle += " *"
			}
			etiquette(&s, ox, haut+float64(hauteurBande)-8, libelle, contraste(j.Palette[sol]), 11, "middle")
		}
	}

	if origine != "" {
		etiquette(&s, float64(largeur)-margeEtiquette, 18,
			"* surchargée par "+origine, contraste(j.Palette[sols[0]]), 11, "end")
	}
	s.WriteString(`</svg>`)

	_, err := io.WriteString(w, s.String())
	return err
}

// cadre trace la case sous une forme, en pointillés.
//
// Sans elle on ne peut pas juger d'une taille : une silhouette isolée paraît
// toujours de la bonne dimension, et c'est son rapport à la case qui compte.
func cadre(s *strings.Builder, x, y float64, sol string) {
	dx, dy := float64(render.TileWidth/2), float64(render.TileHeight/2)
	fmt.Fprintf(s, `<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="none" stroke="%s" stroke-opacity="0.35" stroke-dasharray="3"/>`,
		x-dx, y, x, y-dy, x+dx, y, x, y+dy, contraste(sol))
}

// contraste choisit du noir ou du blanc selon le fond.
//
// Une étiquette d'une seule couleur disparaîtrait sur l'un des trois sols, et
// l'aperçu deviendrait illisible là où il sert le plus.
func contraste(fond string) string {
	var r, v, b int
	if _, err := fmt.Sscanf(fond, "#%02x%02x%02x", &r, &v, &b); err != nil {
		return "#ffffff"
	}
	// Luminance perceptuelle approchée : les coefficients exacts n'importent
	// pas pour trancher entre deux extrêmes.
	if (299*r+587*v+114*b)/1000 > 128 {
		return "#111111"
	}
	return "#f0f0f0"
}
