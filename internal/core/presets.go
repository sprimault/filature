// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

// Preset nomme un jeu de paramètres prêt à play.
//
// La clé est un identifiant, pas un libellé : l'interface cherche
// « prereglage_<cle> » dans le dictionnaire actif, et la même partie s'annonce
// « Quartier » ou « District » selon la langue. Mettre le texte ici le figerait
// en français jusque dans les sauvegardes.
type Preset struct {
	Key      string
	Settings Settings
}

// Presets rend les jeux de paramètres livrés, du plus petit au plus grand.
//
// Trois tailles, comme docs/regles.md §11 les annonce. Deux valeurs suivent le
// côté plutôt que d'être posées : la portée vaut environ un cinquième du
// terrain, et la durée son côté — sur un petit plateau, quarante tours
// laisseraient au fugitif le temps de faire trois fois le tour.
//
// L'ordre est stable : c'est celui d'un sélecteur, et le voir changer d'un
// lancement à l'autre serait déroutant.
func Presets() []Preset {
	return []Preset{
		{Key: "quartier", Settings: settingsForSize(21)},
		{Key: "faubourg", Settings: settingsForSize(31)},
		{Key: "ville", Settings: DefaultSettings()},
	}
}

// PresetByKey retrouve un préréglage par sa clé.
func PresetByKey(cle string) (Preset, bool) {
	for _, p := range Presets() {
		if p.Key == cle {
			return p, true
		}
	}
	return Preset{}, false
}

// settingsForSize dérive un jeu de paramètres d'une taille de plateau.
//
// Seuls la portée, la durée et le début de l'étranglement en dépendent. Le
// reste — résistance, inspecteurs, zones, quota — ne tient pas à la taille du
// terrain mais à l'équilibre entre les deux camps, et ne bougera qu'à l'étape
// d'équilibrage.
func settingsForSize(cote int) Settings {
	p := DefaultSettings()
	p.Size = cote
	p.Range = max(MinRange, cote/5)
	p.Turns = cote

	// L'étranglement commence aux trois quarts de la partie, comme au tour 30
	// sur quarante. Plus tôt, il décide de l'issue ; plus tard, il n'a pas le
	// temps de fermer assez de zones pour peser.
	p.StranglingStart = p.Turns * 3 / 4
	return p
}
