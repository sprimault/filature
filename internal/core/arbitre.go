// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

// Resultat clôt la partie. Motif sert à l'affichage et aux statistiques
// d'équilibrage : savoir si les inspecteurs gagnent par épuisement ou par
// blocage change ce qu'il faut corriger.
type Resultat struct {
	Vainqueur Acteur `json:"vainqueur"`
	Motif     string `json:"motif"`
	Tour      int    `json:"tour"`
}

// Les motifs de fin. Ils partent dans le journal et dans le message `fin` du
// protocole de bot : les renommer périmerait les parties enregistrées.
//
// MotifGreffon est le seul que le jeu de base ne produit jamais : il vient d'un
// effet fin_partie, dont le noyau ignore la condition.
const (
	MotifExtraction  = "extraction"
	MotifResistance  = "resistance_epuisee"
	MotifBlocage     = "fugitif_bloque"
	MotifTempsEcoule = "temps_ecoule"
	MotifGreffon     = "plugin"
)

// ToursPourExtraction est le nombre de fins de tour consécutives que le fugitif
// doit passer dans sa zone.
//
// Deux, et non une : il faut qu'il y soit à la fin de son tour et qu'il y soit
// encore à la fin du suivant. C'est ce délai qui donne aux inspecteurs une
// chance de venir neutraliser la zone, et qui fait de l'extraction un pari
// plutôt qu'une arrivée.
const ToursPourExtraction = 2

// Resultat teste les conditions de fin. Le second retour distingue « partie en
// cours » de « match nul », qui n'existe pas ici.
//
// L'ordre des tests est celui de la règle, et il compte : une extraction
// achevée le tour où le temps s'épuise est une victoire du fugitif, pas un
// temps écoulé.
func (p *Partie) Resultat() (Resultat, bool) {
	// Une fin forcée par un plugin l'emporte sur tout : le noyau ne connaît
	// pas sa condition, il ne peut donc pas l'arbitrer contre les siennes.
	if p.FinForcee != nil {
		return *p.FinForcee, true
	}

	if p.Fugitif.ToursDansZone >= ToursPourExtraction {
		return Resultat{Vainqueur: CampFugitif, Motif: MotifExtraction, Tour: p.Tour}, true
	}
	if p.Fugitif.Resistance <= 0 {
		return Resultat{Vainqueur: CampInspecteurs, Motif: MotifResistance, Tour: p.Tour}, true
	}
	if p.Parametres.Tours > 0 && p.Tour > p.Parametres.Tours {
		return Resultat{Vainqueur: CampInspecteurs, Motif: MotifTempsEcoule, Tour: p.Tour}, true
	}

	// Le blocage se constate au début de la phase du fugitif, et là seulement :
	// plus tard dans son tour, l'absence de déplacement veut dire qu'il a déjà
	// bougé, ce qui n'est pas la même chose.
	if p.Phase == PhaseFugitif && p.Fugitif.DeplacementsFaits == 0 && p.fugitifImmobilise() {
		return Resultat{Vainqueur: CampInspecteurs, Motif: MotifBlocage, Tour: p.Tour}, true
	}

	return Resultat{}, false
}

// fugitifImmobilise dit qu'aucun déplacement ne lui est offert.
//
// Passé par CoupsLegaux et non par un calcul propre : la légalité n'a qu'une
// source, sinon deux règles divergent et c'est la partie qui tranche.
func (p *Partie) fugitifImmobilise() bool {
	for _, c := range p.CoupsLegaux(CampFugitif) {
		if c.Type == CoupDeplacer {
			return false
		}
	}
	return true
}

// extractionEnCours dit si le fugitif tient sa zone.
//
// Une zone occupée par un inspecteur est neutralisée : le compte ne démarre pas
// et s'interrompt s'il était engagé. Camper est une stratégie valide — mais un
// inspecteur assis sur une zone est un inspecteur qui ne cherche pas.
//
// Une zone fermée par l'étranglement ne vaut pas mieux : le fugitif qui s'y
// trouve doit repartir, et payer pour resceller ailleurs.
func (p *Partie) extractionEnCours() bool {
	zone, existe := p.zoneScellee()
	if !existe || !zone.Contient(p.Fugitif.Position) {
		return false
	}
	for _, ferme := range p.ZonesFermees {
		if ferme == zone.Numero {
			return false
		}
	}
	for _, i := range p.Inspecteurs {
		if zone.Contient(i.Position) {
			return false
		}
	}
	return true
}

// zoneScellee retrouve la zone que le fugitif vise.
func (p *Partie) zoneScellee() (Zone, bool) {
	for _, z := range p.Plateau.Zones() {
		if z.Numero == p.Fugitif.ZoneScellee {
			return z, true
		}
	}
	return Zone{}, false
}

// compterExtraction avance ou remet à zéro le décompte d'extraction.
//
// Appelé en toute fin de résolution, après l'étranglement : une zone qui vient
// de se fermer interrompt le compte du tour même, sans quoi le fugitif
// s'extrairait d'un point d'extraction qui n'existe plus.
func (p *Partie) compterExtraction() []func() {
	precedent := p.Fugitif.ToursDansZone

	if p.extractionEnCours() {
		p.Fugitif.ToursDansZone++
	} else {
		p.Fugitif.ToursDansZone = 0
	}

	if p.Fugitif.ToursDansZone == precedent {
		return nil
	}
	return []func(){func() { p.Fugitif.ToursDansZone = precedent }}
}

// zoneAEtrangler renvoie la zone que l'étranglement vise à ce tour, s'il en
// vise une.
//
// Elle ne ferme rien : elle donne la cadence et la cible au mode etranglement
// de plugins/base, qui porte le préavis et la fermeture. Un plugin qui
// remplace ce mode change ce qui se passe, jamais quand.
//
// Une seule zone à la fois, et non deux listes : le préavis n'est plus l'affaire
// du noyau depuis que le mode le déclare, et rendre « ce qui est annoncé » ici
// dupliquerait ce que la file des différés porte déjà.
func (p *Partie) zoneAEtrangler() (int, bool) {
	debut, periode := p.Parametres.DebutEtranglement, p.Parametres.PeriodeEtranglement
	if periode <= 0 || p.Tour < debut || (p.Tour-debut)%periode != 0 {
		return 0, false
	}

	ordre := p.ordreEtranglement()
	rang := (p.Tour - debut) / periode
	if rang >= len(ordre) {
		return 0, false
	}
	return ordre[rang], true
}

// ordreEtranglement tire l'ordre de fermeture des zones depuis la graine.
//
// Recalculé et non stocké : la graine le détermine entièrement, et le garder
// dans l'état en ferait un cache que rien ne réconcilie. Le flux nommé évite
// qu'un tirage ajouté ailleurs ne le décale.
func (p *Partie) ordreEtranglement() []int {
	zones := p.Plateau.Zones()
	numeros := make([]int, 0, len(zones))
	for _, z := range zones {
		numeros = append(numeros, z.Numero)
	}
	Melanger(NouvelAlea(p.Graine, "etranglement"), numeros)
	return numeros
}
