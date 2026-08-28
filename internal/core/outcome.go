// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import "slices"

// Outcome clôt la partie. Reason sert à l'affichage et aux statistiques
// d'équilibrage : savoir si les inspecteurs gagnent par épuisement ou par
// blocage change ce qu'il faut corriger.
type Outcome struct {
	Winner Side   `json:"winner"`
	Reason string `json:"reason"`
	Turn   int    `json:"turn"`
}

// Les motifs de fin. Ils partent dans le journal et dans le message `over` du
// protocole de bot : les renommer périmerait les parties enregistrées.
//
// OutcomePlugin est le seul que le jeu de base ne produit jamais : il vient d'un
// effet end_game, dont le noyau ignore la condition.
const (
	OutcomeExtraction   = "extraction"
	OutcomeCaptured     = "captured"
	OutcomeStaminaSpent = "stamina_spent"
	OutcomeCornered     = "fugitive_cornered"
	OutcomeTimeUp       = "time_up"
	OutcomePlugin       = "plugin"
)

// TurnsToExtract est le nombre de fins de tour consécutives que le fugitif
// doit passer dans sa zone.
//
// Deux, et non une : il faut qu'il y soit à la fin de son tour et qu'il y soit
// encore à la fin du next. C'est ce délai qui donne aux inspecteurs une
// chance de venir neutraliser la zone, et qui fait de l'extraction un pari
// plutôt qu'une arrivée.
const TurnsToExtract = 2

// Outcome teste les conditions de fin. Le second retour distingue « partie en
// cours » de « match nul », qui n'existe pas ici.
//
// **L'extraction se teste en premier, et c'est une règle et non un détail
// d'implémentation** : les inspecteurs disposent de quatre voies de conclusion
// contre une seule au fugitif, et faire tomber les égalités du côté des quatre
// refermerait l'unique sans que personne l'ait décidé. Une extraction achevée
// le tour où il est capturé, épuisé ou où le temps s'arrête est donc une
// victoire du fugitif.
//
// La règle vaut telle quelle pour les conditions qu'un plugin ajouterait, à une
// exception près, plus haut : une fin forcée l'emporte sur tout, le noyau ne
// connaissant pas sa condition.
func (p *Game) Outcome() (Outcome, bool) {
	// Une fin forcée par un plugin l'emporte sur tout : le noyau ne connaît
	// pas sa condition, il ne peut donc pas l'arbitrer contre les siennes.
	if p.ForcedOutcome != nil {
		return *p.ForcedOutcome, true
	}

	if p.Fugitive.TurnsInZone >= TurnsToExtract {
		return Outcome{Winner: SideFugitive, Reason: OutcomeExtraction, Turn: p.Turn}, true
	}
	if p.Fugitive.Captured {
		return Outcome{Winner: SideInspectors, Reason: OutcomeCaptured, Turn: p.Turn}, true
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
	//
	// Depuis la capture, c'est un cas de terrain et non de poursuite : un
	// fugitif entouré d'inspecteurs tombe par capture bien avant d'être à court
	// de cases. Ce qui reste ici, ce sont les bâtiments et les barrages du
	// Barreur, seuls capables de l'enfermer sans le toucher.
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
// **Une zone ne se neutralise plus, ses cases s'occupent.** Un inspecteur posté
// dessus n'en retire qu'une case d'entrée, puisque le fugitif ne peut pas y
// venir — il n'y a donc rien à tester ici : s'il est sur la zone, sa case est
// libre par construction.
//
// Ce qui était testé avant donnait au camp le moyen de verrouiller les
// dernières sorties en fin de partie, au moment précis où couvrir ne coûte plus
// rien puisqu'il n'y a plus rien à chercher. Fermer une zone demande maintenant
// autant de pions qu'elle a de cases de rue, cinq au minimum.
//
// Une zone fermée par l'étranglement, elle, arrête bien le compte : le fugitif
// qui s'y trouve doit repartir, et payer pour resceller ailleurs.
func (p *Game) extractionUnderway() bool {
	zone, existe := p.sealedZone()
	if !existe || !zone.Contains(p.Fugitive.Position) {
		return false
	}
	return !slices.Contains(p.ClosedZones, zone.Number)
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

// zoneToStrangle renvoie la zone que l'étranglement ferme à ce tour, s'il en
// ferme une.
//
// Elle donne la cadence et la cible au mode déclaré sur OnStrangling, qui porte
// la fermeture. Un plugin qui remplace ce mode change ce qui se passe, jamais
// quand.
func (p *Game) zoneToStrangle() (int, bool) {
	return p.zoneToStrangleAt(p.Turn)
}

// zoneToStrangleAt renvoie la zone que l'étranglement fermera au tour demandé.
//
// Interrogeable pour un tour à venir, et c'est ce qui porte l'annonce : la vue
// la lit à Turn+StranglingNotice, sans qu'aucun effet ait à être posé.
//
// **Le préavis appartient au noyau et non au mode.** docs/regles.md §10 promet
// que personne ne subit une fermeture par surprise : une promesse de règle ne
// peut pas dépendre de ce qu'un manifeste déclare, pas plus que la vue filtrée
// ne se négocie. Le mode le portait auparavant sous forme d'un differer de deux
// tours, qui s'ajoutait à la cadence au lieu de la précéder — les fermetures
// tombaient deux tours après le tableau publié, la dernière au dernier tour de
// la partie.
func (p *Game) zoneToStrangleAt(tour int) (int, bool) {
	debut, periode := p.Settings.StranglingStart, p.Settings.StranglingPeriod
	if periode <= 0 || tour < debut || (tour-debut)%periode != 0 {
		return 0, false
	}

	// L'étranglement s'arrête à ZonesLeftOpen : il doit créer un rendez-vous,
	// pas un verrou. Avec une seule issue il n'y a plus d'arbitrage et le camp
	// entier s'y assied ; à trois, couvrir demande de se diviser, et se diviser
	// coûte la masse qui capture.
	ordre := p.stranglingOrder()
	fermetures := len(ordre) - p.Settings.ZonesLeftOpen
	rang := (tour - debut) / periode
	if fermetures <= 0 || rang >= fermetures {
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
	Shuffle(NewRandom(p.Seed, "strangling"), numeros)
	return numeros
}
