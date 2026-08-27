// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

// Preset nomme un jeu de paramètres prêt à play.
//
// La clé est un identifiant, pas un libellé : l'interface cherche
// « preset_<cle> » dans le dictionnaire actif, et la même partie s'annonce
// « Quartier » ou « District » selon la langue. Mettre le texte ici le figerait
// en français jusque dans les sauvegardes.
//
// Les clés sont donc en anglais, comme le reste des identifiants publics — et
// elles ne l'étaient pas : les dictionnaires déclaraient preset_district quand
// Presets rendait « quartier », si bien qu'aucun libellé n'aurait été trouvé.
// Le repli sur le français que docs/plugins.md promet n'aurait pas joué non
// plus, la clé n'existant dans aucun des deux.
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
		{Key: "district", Settings: SettingsForSize(21)},
		{Key: "outskirts", Settings: SettingsForSize(31)},
		{Key: "city", Settings: DefaultSettings()},
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

// SettingsForSize dérive un jeu de paramètres d'une taille de plateau.
//
// Seuls la portée, la durée, le rayon du noyau et le début de l'étranglement en
// dépendent. Le reste — résistance, inspecteurs, zones, quota — ne tient pas à
// la taille du terrain mais à l'équilibre entre les deux camps, et ne bougera
// qu'à l'étape d'équilibrage.
//
// Exportée parce que partir de DefaultSettings et n'y changer que le côté donne
// un réglage incohérent : quatre valeurs suivent la taille, et les oublier
// produit un rayon de noyau plus grand que le plateau ou une portée qui voit
// d'un bord à l'autre.
func SettingsForSize(cote int) Settings {
	p := DefaultSettings()
	p.Size = cote
	p.Range = max(MinRange, cote/5)
	p.Turns = cote

	// Le noyau suit le côté, sans quoi la portée de vue le rattrape : à rayon
	// fixe, cinq inspecteurs voient d'autant mieux le départ que la ville est
	// grande, ce qui est l'inverse de ce qu'on attend d'elle.
	p.CentreRadius = cote / 4

	// L'étranglement commence aux trois quarts de la partie, comme au tour 30
	// sur quarante. Plus tôt, il décide de l'issue ; plus tard, il n'a pas le
	// temps de fermer assez de zones pour peser.
	p.StranglingStart = p.Turns * 3 / 4

	// La période s'en déduit : les fermetures doivent tenir dans ce qui reste,
	// la dernière laissant StranglingEndMargin tours de jeu après elle. Figée,
	// elle épuiserait la pression à mi-chemin sur une longue partie et
	// déborderait la fin sur une courte.
	//
	// Cette marge vaut deux tours comme le préavis, et les deux nombres n'ont
	// rien à voir : l'un laisse au fugitif de quoi profiter de la dernière
	// fermeture, l'autre de quoi la voir venir. Les avoir confondus est ce qui a
	// laissé le mode ajouter son préavis à une cadence déjà calculée.
	fermetures := p.Zones - p.ZonesLeftOpen
	if fermetures > 1 {
		p.StranglingPeriod = max(1, (p.Turns-p.StranglingStart-StranglingEndMargin)/(fermetures-1))
	}

	// Le préavis se rabat sur la période quand elle est plus courte que lui :
	// au côté minimal, les fermetures s'enchaînent d'un tour sur l'autre et
	// deux annonces se recouvriraient. Une dérivation qui produirait des
	// paramètres que Validate refuse serait un défaut de la dérivation.
	p.StranglingNotice = min(p.StranglingNotice, p.StranglingPeriod)
	return p
}
