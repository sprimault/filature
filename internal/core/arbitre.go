// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

// Outcome clôt la partie. Reason sert à l'affichage et aux statistiques
// d'équilibrage : savoir si les inspecteurs gagnent par épuisement ou par
// blocage change ce qu'il faut corriger.
type Outcome struct {
	Winner Side   `json:"vainqueur"`
	Reason string `json:"motif"`
	Turn   int    `json:"tour"`
}

// Les motifs de fin. Ils partent dans le journal et dans le message `fin` du
// protocole de bot : les renommer périmerait les parties enregistrées.
//
// OutcomePlugin est le seul que le jeu de base ne produit jamais : il vient d'un
// effet fin_partie, dont le noyau ignore la condition.
const (
	OutcomeExtraction   = "extraction"
	OutcomeStaminaSpent = "resistance_epuisee"
	OutcomeCornered     = "fugitif_bloque"
	OutcomeTimeUp       = "temps_ecoule"
	OutcomePlugin       = "plugin"
)

// TurnsToExtract est le nombre de fins de tour consécutives que le fugitif
// doit passer dans sa zone.
//
// Deux, et non une : il faut qu'il y soit à la fin de son tour et qu'il y soit
// encore à la fin du suivant. C'est ce délai qui donne aux inspecteurs une
// chance de venir neutraliser la zone, et qui fait de l'extraction un pari
// plutôt qu'une arrivée.
const TurnsToExtract = 2

// Outcome teste les conditions de fin. Le second retour distingue « partie en
// cours » de « match nul », qui n'existe pas ici.
//
// L'ordre des tests est celui de la règle, et il compte : une extraction
// achevée le tour où le temps s'épuise est une victoire du fugitif, pas un
// temps écoulé.
func (p *Game) Outcome() (Outcome, bool) {
	// Une fin forcée par un plugin l'emporte sur tout : le noyau ne connaît
	// pas sa condition, il ne peut donc pas l'arbitrer contre les siennes.
	if p.ForcedOutcome != nil {
		return *p.ForcedOutcome, true
	}

	if p.Fugitive.TurnsInZone >= TurnsToExtract {
		return Outcome{Winner: SideFugitive, Reason: OutcomeExtraction, Turn: p.Turn}, true
	}
	if p.Fugitive.Stamina <= 0 {
		return Outcome{Winner: SideInspectors, Reason: OutcomeStaminaSpent, Turn: p.Turn}, true
	}
	if p.Settings.Turns > 0 && p.Turn > p.Settings.Turns {
		return Outcome{Winner: SideInspectors, Reason: OutcomeTimeUp, Turn: p.Turn}, true
	}

	// Le blocage se constate au début de la phase du fugitif, et là seulement :
	// plus tard dans son tour, l'absence de déplacement veut dire qu'il a déjà
	// bougé, ce qui n'est pas la même chose.
	if p.Phase == PhaseFugitive && p.Fugitive.StepsTaken == 0 && p.fugitiveStuck() {
		return Outcome{Winner: SideInspectors, Reason: OutcomeCornered, Turn: p.Turn}, true
	}

	return Outcome{}, false
}

// fugitiveStuck dit qu'aucun déplacement ne lui est offert.
//
// Passé par LegalMoves et non par un calcul propre : la légalité n'a qu'une
// source, sinon deux règles divergent et c'est la partie qui tranche.
func (p *Game) fugitiveStuck() bool {
	for _, c := range p.LegalMoves(SideFugitive) {
		if c.Type == MoveStep {
			return false
		}
	}
	return true
}

// extractionUnderway dit si le fugitif tient sa zone.
//
// Une zone occupée par un inspecteur est neutralisée : le compte ne démarre pas
// et s'interrompt s'il était engagé. Camper est une stratégie valide — mais un
// inspecteur assis sur une zone est un inspecteur qui ne cherche pas.
//
// Une zone fermée par l'étranglement ne vaut pas mieux : le fugitif qui s'y
// trouve doit repartir, et payer pour resceller ailleurs.
func (p *Game) extractionUnderway() bool {
	zone, existe := p.sealedZone()
	if !existe || !zone.Contains(p.Fugitive.Position) {
		return false
	}
	for _, ferme := range p.ClosedZones {
		if ferme == zone.Number {
			return false
		}
	}
	for _, i := range p.Inspectors {
		if zone.Contains(i.Position) {
			return false
		}
	}
	return true
}

// sealedZone retrouve la zone que le fugitif vise.
func (p *Game) sealedZone() (Zone, bool) {
	for _, z := range p.Board.Zones() {
		if z.Number == p.Fugitive.SealedZone {
			return z, true
		}
	}
	return Zone{}, false
}

// countExtraction avance ou remet à zéro le décompte d'extraction.
//
// Appelé en toute fin de résolution, après l'étranglement : une zone qui vient
// de se fermer interrompt le compte du tour même, sans quoi le fugitif
// s'extrairait d'un point d'extraction qui n'existe plus.
func (p *Game) countExtraction() []func() {
	precedent := p.Fugitive.TurnsInZone

	if p.extractionUnderway() {
		p.Fugitive.TurnsInZone++
	} else {
		p.Fugitive.TurnsInZone = 0
	}

	if p.Fugitive.TurnsInZone == precedent {
		return nil
	}
	return []func(){func() { p.Fugitive.TurnsInZone = precedent }}
}

// zoneToStrangle renvoie la zone que l'étranglement vise à ce tour, s'il en
// vise une.
//
// Elle ne ferme rien : elle donne la cadence et la cible au mode etranglement
// de plugins/base, qui porte le préavis et la fermeture. Un plugin qui
// remplace ce mode change ce qui se passe, jamais quand.
//
// Une seule zone à la fois, et non deux listes : le préavis n'est plus l'affaire
// du noyau depuis que le mode le déclare, et rendre « ce qui est annoncé » ici
// dupliquerait ce que la file des différés porte déjà.
func (p *Game) zoneToStrangle() (int, bool) {
	debut, periode := p.Settings.StranglingStart, p.Settings.PeriodeEtranglement
	if periode <= 0 || p.Turn < debut || (p.Turn-debut)%periode != 0 {
		return 0, false
	}

	ordre := p.stranglingOrder()
	rang := (p.Turn - debut) / periode
	if rang >= len(ordre) {
		return 0, false
	}
	return ordre[rang], true
}

// stranglingOrder tire l'ordre de fermeture des zones depuis la graine.
//
// Recalculé et non stocké : la graine le détermine entièrement, et le garder
// dans l'état en ferait un cache que rien ne réconcilie. Le flux nommé évite
// qu'un tirage ajouté ailleurs ne le décale.
func (p *Game) stranglingOrder() []int {
	zones := p.Board.Zones()
	numeros := make([]int, 0, len(zones))
	for _, z := range zones {
		numeros = append(numeros, z.Number)
	}
	Shuffle(NewRandom(p.Seed, "etranglement"), numeros)
	return numeros
}
