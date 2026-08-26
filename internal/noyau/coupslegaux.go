// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import "sort"

// CoupsLegaux énumère ce que l'acteur peut jouer dans la phase courante.
//
// C'est la seule source de vérité sur la légalité : l'interface s'en sert pour
// surligner les cases, l'IA pour explorer, le serveur pour valider ce qui
// arrive du réseau. Aucun de ces trois ne réimplémente la règle.
func (p *Partie) CoupsLegaux(a Acteur) []Coup {
	switch p.Phase {
	case PhasePlacementFugitif:
		if a == CampFugitif {
			return p.coupsSceller()
		}
	case PhasePlacementInspecteurs:
		if a == CampInspecteurs {
			return p.coupsPlacer()
		}
	case PhaseInspecteurs:
		if a == CampInspecteurs {
			return p.coupsInspecteurs()
		}
	case PhaseFugitif:
		if a == CampFugitif {
			return p.coupsFugitif()
		}
	}
	return nil
}

// PionsDeplaces compte les inspecteurs qui ont bougé ce tour.
//
// Calculé et non stocké : le total se déduit des pions, et deux sources pour
// un même chiffre finiraient par se contredire.
func (p *Partie) PionsDeplaces() int {
	n := 0
	for _, i := range p.Inspecteurs {
		if i.DeplacementsFaits > 0 {
			n++
		}
	}
	return n
}

// coupsSceller énumère les zones que le fugitif peut choisir à la mise en
// place. Sa case, elle, est tirée au sort : il ne la choisit jamais.
func (p *Partie) coupsSceller() []Coup {
	zones := p.Plateau.Zones()
	coups := make([]Coup, 0, len(zones))
	for _, z := range zones {
		coups = append(coups, Coup{Tour: p.Tour, Acteur: CampFugitif, Type: CoupPlacer, Zone: z.Numero})
	}
	return coups
}

// coupsPlacer énumère où poser le prochain inspecteur.
//
// La case du fugitif n'est pas retirée de la liste, bien qu'il soit déjà sur le
// plateau : l'en exclure dirait aux inspecteurs où il n'est pas, ce qui est une
// fuite exactement comme dire où il est. Un pion qui tombe dessus l'a trouvé
// par hasard, et c'est un résultat de partie, pas un coup illégal.
func (p *Partie) coupsPlacer() []Coup {
	if len(p.Inspecteurs) >= p.Parametres.Inspecteurs {
		return nil
	}
	rayon := p.Parametres.Cote / 2
	centre := Position{Colonne: rayon, Ligne: rayon}

	cases := p.Plateau.CasesDans(centre, rayon)
	coups := make([]Coup, 0, len(cases))
	for _, c := range cases {
		if !p.EstPraticable(c) {
			continue
		}
		coups = append(coups, Coup{
			Tour:    p.Tour,
			Acteur:  CampInspecteurs,
			Type:    CoupPlacer,
			Pion:    len(p.Inspecteurs),
			Arrivee: c,
		})
	}
	return coups
}

// coupsInspecteurs énumère déplacements et capacités de la phase.
//
// Le quota porte sur le nombre de pions distincts, pas sur le nombre de
// déplacements : un pion déjà entamé continue sur sa mobilité propre sans
// prendre une place de plus.
func (p *Partie) coupsInspecteurs() []Coup {
	var coups []Coup
	placeLibre := p.PionsDeplaces() < p.Parametres.PionsParTour

	for i := range p.Inspecteurs {
		entame := p.Inspecteurs[i].DeplacementsFaits > 0
		if !entame && !placeLibre {
			continue
		}
		if p.Inspecteurs[i].DeplacementsFaits >= p.MobiliteDe(CampInspecteurs, i) {
			continue
		}
		depart := p.Inspecteurs[i].Position
		for _, d := range Orthogonales {
			arrivee := depart.Avance(d)
			if !p.EstPraticable(arrivee) {
				continue
			}
			coups = append(coups, Coup{
				Tour: p.Tour, Acteur: CampInspecteurs, Type: CoupDeplacer,
				Pion: i, Depart: depart, Arrivee: arrivee,
			})
		}
	}

	coups = append(coups, p.coupsCapacite()...)
	return append(coups, Coup{Tour: p.Tour, Acteur: CampInspecteurs, Type: CoupFinDePhase})
}

// coupsCapacite énumère les capacités déclenchables ce tour.
//
// Les clés du registre sont triées avant parcours : l'ordre d'une map n'est pas
// stable en Go, et il déciderait ici de l'ordre des coups légaux — donc de ce
// qu'un rejeu doit retrouver.
func (p *Partie) coupsCapacite() []Coup {
	if p.Extensions == nil || p.CapaciteJouee {
		return nil
	}

	cles := make([]string, 0, len(p.Extensions.Capacites))
	for cle := range p.Extensions.Capacites {
		cles = append(cles, cle)
	}
	sort.Strings(cles)

	var coups []Coup
	for _, cle := range cles {
		c := p.Extensions.Capacites[cle]
		if c.Camp != CampInspecteurs || c.Passive || c.Declenchement != SurPhaseInspecteurs {
			continue
		}
		for i := range p.Inspecteurs {
			if p.Inspecteurs[i].Capacite != cle || p.Inspecteurs[i].CapaciteUtilisee {
				continue
			}
			coups = append(coups, Coup{
				Tour: p.Tour, Acteur: CampInspecteurs, Type: CoupCapacite,
				Pion: i, Capacite: cle,
			})
		}
	}
	return coups
}

// coupsFugitif énumère déplacements, dépenses et changement de zone.
//
// Une case occupée par un inspecteur lui est fermée : il y serait à l'abri de
// tout contact, l'adjacence n'incluant pas la superposition.
func (p *Partie) coupsFugitif() []Coup {
	var coups []Coup
	depart := p.Fugitif.Position

	if p.Fugitif.DeplacementsFaits < p.MobiliteDe(CampFugitif, 0) {
		for d := Nord; d <= NordOuest; d++ {
			arrivee := depart.Avance(d)
			if !p.EstPraticable(arrivee) || p.occupee(arrivee) {
				continue
			}
			// Un angle fermé ne se franchit pas : la diagonale exige qu'au
			// moins une des deux cases orthogonales intermédiaires soit une
			// rue, sinon le bâti ne bloque plus rien.
			if d.EstDiagonale() {
				a, b := depart.Contournement(d)
				if !p.EstPraticable(a) && !p.EstPraticable(b) {
					continue
				}
			}
			coups = append(coups, Coup{
				Tour: p.Tour, Acteur: CampFugitif, Type: CoupDeplacer,
				Depart: depart, Arrivee: arrivee,
			})
		}
	}

	coups = append(coups, p.coupsDepense()...)
	coups = append(coups, Coup{Tour: p.Tour, Acteur: CampFugitif, Type: CoupPasser})
	return append(coups, Coup{Tour: p.Tour, Acteur: CampFugitif, Type: CoupFinDePhase})
}

// coupsDepense énumère ce que le fugitif peut acheter avec sa résistance.
//
// Le changement de zone est une dépense comme une autre, mais porte un type de
// coup distinct : il désigne une zone, que les autres n'ont pas à porter.
func (p *Partie) coupsDepense() []Coup {
	if p.Extensions == nil {
		return nil
	}

	cles := make([]Depense, 0, len(p.Extensions.Depenses))
	for cle := range p.Extensions.Depenses {
		cles = append(cles, cle)
	}
	sort.Slice(cles, func(i, j int) bool { return cles[i] < cles[j] })

	var coups []Coup
	for _, cle := range cles {
		d := p.Extensions.Depenses[cle]
		if d.Camp != CampFugitif || d.Cout > p.Fugitif.Resistance {
			continue
		}
		if d.Usages > 0 && p.UsagesDepense[cle] >= d.Usages {
			continue
		}
		if cle == DepenseSilence && p.Fugitif.SilenceAchete {
			continue
		}
		coups = append(coups, Coup{
			Tour: p.Tour, Acteur: CampFugitif, Type: CoupDepense, Depense: cle,
		})
	}

	return append(coups, p.coupsChangerZone()...)
}

// coupsChangerZone énumère les zones vers lesquelles le fugitif peut resceller.
func (p *Partie) coupsChangerZone() []Coup {
	if p.Extensions == nil {
		return nil
	}
	d, connue := p.Extensions.Depenses[DepenseChangerZone]
	if !connue || d.Cout > p.Fugitif.Resistance {
		return nil
	}

	var coups []Coup
	for _, z := range p.Plateau.Zones() {
		if z.Numero == p.Fugitif.ZoneScellee {
			continue
		}
		coups = append(coups, Coup{
			Tour: p.Tour, Acteur: CampFugitif, Type: CoupChangerZone, Zone: z.Numero,
		})
	}
	return coups
}

// occupee dit si un inspecteur tient la case.
func (p *Partie) occupee(pos Position) bool {
	for _, i := range p.Inspecteurs {
		if i.Position == pos {
			return true
		}
	}
	return false
}
