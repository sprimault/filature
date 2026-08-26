// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

// Prereglage nomme un jeu de paramètres prêt à jouer.
//
// La clé est un identifiant, pas un libellé : l'interface cherche
// « prereglage_<cle> » dans le dictionnaire actif, et la même partie s'annonce
// « Quartier » ou « District » selon la langue. Mettre le texte ici le figerait
// en français jusque dans les sauvegardes.
type Prereglage struct {
	Cle        string
	Parametres Parametres
}

// Prereglages rend les jeux de paramètres livrés, du plus petit au plus grand.
//
// Trois tailles, comme docs/regles.md §11 les annonce. Deux valeurs suivent le
// côté plutôt que d'être posées : la portée vaut environ un cinquième du
// terrain, et la durée son côté — sur un petit plateau, quarante tours
// laisseraient au fugitif le temps de faire trois fois le tour.
//
// L'ordre est stable : c'est celui d'un sélecteur, et le voir changer d'un
// lancement à l'autre serait déroutant.
func Prereglages() []Prereglage {
	return []Prereglage{
		{Cle: "quartier", Parametres: parametresPourCote(21)},
		{Cle: "faubourg", Parametres: parametresPourCote(31)},
		{Cle: "ville", Parametres: ParametresDefaut()},
	}
}

// PrereglagePar retrouve un préréglage par sa clé.
func PrereglagePar(cle string) (Prereglage, bool) {
	for _, p := range Prereglages() {
		if p.Cle == cle {
			return p, true
		}
	}
	return Prereglage{}, false
}

// parametresPourCote dérive un jeu de paramètres d'une taille de plateau.
//
// Seuls la portée, la durée et le début de l'étranglement en dépendent. Le
// reste — résistance, inspecteurs, zones, quota — ne tient pas à la taille du
// terrain mais à l'équilibre entre les deux camps, et ne bougera qu'à l'étape
// d'équilibrage.
func parametresPourCote(cote int) Parametres {
	p := ParametresDefaut()
	p.Cote = cote
	p.Portee = max(PorteeMin, cote/5)
	p.Tours = cote

	// L'étranglement commence aux trois quarts de la partie, comme au tour 30
	// sur quarante. Plus tôt, il décide de l'issue ; plus tard, il n'a pas le
	// temps de fermer assez de zones pour peser.
	p.DebutEtranglement = p.Tours * 3 / 4
	return p
}
