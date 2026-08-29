// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import "sort"

// LegalMoves énumère ce que l'acteur peut jouer dans la phase courante.
//
// C'est la seule source de vérité sur la légalité : l'interface s'en sert pour
// surligner les cases, l'IA pour explorer, le serveur pour valider ce qui
// arrive du réseau. Aucun de ces trois ne réimplémente la règle.
func (p *Game) LegalMoves(a Side) []Move {
	switch p.Phase {
	case PhaseFugitiveSetup:
		if a == SideFugitive {
			return p.sealMoves()
		}
	case PhaseInspectorsSetup:
		if a == SideInspectors {
			return p.placeMoves()
		}
	case PhaseInspectors:
		if a == SideInspectors {
			return p.inspectorMoves()
		}
	case PhaseFugitive:
		if a == SideFugitive {
			return p.fugitiveMoves()
		}
	}
	return nil
}

// PiecesMoved compte les inspecteurs qui ont bougé ce tour.
//
// Calculé et non stocké : le total se déduit des pions, et deux sources pour
// un même chiffre finiraient par se contredire.
func (p *Game) PiecesMoved() int {
	n := 0
	for _, i := range p.Inspectors {
		if i.StepsTaken > 0 {
			n++
		}
	}
	return n
}

// sealMoves énumère les zones que le fugitif peut choisir à la mise en
// place. Sa case, elle, est tirée au sort : il ne la choisit jamais.
func (p *Game) sealMoves() []Move {
	zones := p.Board.Zones()
	coups := make([]Move, 0, len(zones))
	for _, z := range zones {
		coups = append(coups, Move{Turn: p.Turn, Side: SideFugitive, Type: MovePlace, Zone: z.Number})
	}
	return coups
}

// placeMoves énumère où poser le prochain inspecteur.
//
// La case du fugitif n'est pas retirée de la liste, bien qu'il soit déjà sur le
// plateau : l'en exclure dirait aux inspecteurs où il n'est pas, ce qui est une
// fuite exactement comme dire où il est. Un pion qui tombe dessus l'a trouvé
// par hasard, et c'est un résultat de partie, pas un coup illégal.
//
// Le noyau central en est exclu, et c'est une règle de jeu et non une commodité
// : cinq pions posés autour du départ en voient de six à neuf dixièmes dès le
// premier tour selon le préréglage, et l'invisibilité du fugitif n'existe alors
// pas. Le chiffre valait plus de neuf dixièmes partout quand la mesure a produit
// la règle ; il a baissé depuis, le noyau ayant changé de rayon.
// L'exclusion ne porte que sur le placement — rien n'empêche d'y entrer au
// premier tour.
func (p *Game) placeMoves() []Move {
	if len(p.Inspectors) >= p.Settings.Inspectors {
		return nil
	}
	rayon := p.Settings.Size / 2
	centre := Position{Column: rayon, Row: rayon}

	cases := p.Board.CellsWithin(centre, rayon)
	coups := make([]Move, 0, len(cases))
	for _, c := range cases {
		if !p.IsWalkable(c) || ChebyshevDistance(c, centre) <= p.Settings.CentreRadius {
			continue
		}
		coups = append(coups, Move{
			Turn:  p.Turn,
			Side:  SideInspectors,
			Type:  MovePlace,
			Piece: len(p.Inspectors),
			To:    c,
		})
	}
	return coups
}

// inspectorMoves énumère déplacements et capacités de la phase.
//
// Le quota porte sur le nombre de pions distincts, pas sur le nombre de
// déplacements : un pion déjà entamé continue sur sa mobilité propre sans
// prendre une place de plus.
func (p *Game) inspectorMoves() []Move {
	var coups []Move
	placeLibre := p.PiecesMoved() < p.Settings.PiecesPerTurn

	for i := range p.Inspectors {
		entame := p.Inspectors[i].StepsTaken > 0
		if !entame && !placeLibre {
			continue
		}
		if p.Inspectors[i].StepsTaken >= p.MobilityOf(SideInspectors, i) {
			continue
		}
		depart := p.Inspectors[i].Position
		for _, d := range Orthogonales {
			arrivee := depart.Step(d)
			if !p.IsWalkable(arrivee) {
				continue
			}
			coups = append(coups, Move{
				Turn: p.Turn, Side: SideInspectors, Type: MoveStep,
				Piece: i, From: depart, To: arrivee,
			})
		}
	}

	coups = append(coups, p.abilityMoves()...)
	return append(coups, Move{Turn: p.Turn, Side: SideInspectors, Type: MoveEndPhase})
}

// abilityMoves énumère les capacités déclenchables ce tour.
//
// Les clés du registre sont triées avant parcours : l'ordre d'une map n'est pas
// stable en Go, et il déciderait ici de l'ordre des coups légaux — donc de ce
// qu'un rejeu doit retrouver.
func (p *Game) abilityMoves() []Move {
	if p.Extensions == nil || p.AbilityPlayed {
		return nil
	}

	cles := make([]string, 0, len(p.Extensions.Abilities))
	for cle := range p.Extensions.Abilities {
		cles = append(cles, cle)
	}
	sort.Strings(cles)

	var coups []Move
	for _, cle := range cles {
		c := p.Extensions.Abilities[cle]
		if c.Camp != SideInspectors || c.Passive || c.Trigger != OnInspectorsPhase {
			continue
		}
		for i := range p.Inspectors {
			if p.Inspectors[i].Ability != cle || p.Inspectors[i].AbilityUsed {
				continue
			}
			base := Move{
				Turn: p.Turn, Side: SideInspectors, Type: MoveAbility,
				Piece: i, Ability: cle,
			}
			if !needsCell(c.Effects) {
				coups = append(coups, base)
				continue
			}
			// From porte la case du pion, comme pour un déplacement : sans
			// elle, rien ne distingue une capacité visant (0,0) d'une capacité
			// qui ne vise aucune case, et l'interface ne peut pas nommer le
			// sens de la cible.
			for _, cible := range p.abilityCells(p.Inspectors[i].Position) {
				coup := base
				coup.From, coup.To = p.Inspectors[i].Position, cible
				coups = append(coups, coup)
			}
		}
	}
	return coups
}

// needsCell dit si une capacité attend du coup la case sur laquelle agir.
//
// Un effet visant cell la lit dans le contexte, que triggerAbility remplit
// depuis Move.To. Tant que l'énumération n'a proposé qu'un coup par pion, ce
// champ restait nul et le Barreur fermait invariablement la case {0,0}.
//
// Les effets imbriqués comptent : un differer dont le then vise une case a
// besoin d'elle au moment de la pose, pas à l'échéance.
func needsCell(effets []Effect) bool {
	for _, e := range effets {
		if e.Target == TargetCell || needsCell(e.Then) {
			return true
		}
	}
	return false
}

// abilityCells rend les cases qu'une capacité peut viser depuis un pion.
//
// Les huit voisines, et pas n'importe quelle case du plateau : barrer à
// distance retirerait une case d'entrée d'une zone sans rien dégarnir, quand
// docs/regles.md §10 fait payer chaque case d'une zone par un pion posté
// dessus. Les huit et non les quatre parce que le fugitif s'échappe en
// diagonale — un barrage qui ne couperait que des orthogonales ne refermerait
// jamais un angle.
//
// Sa propre case est exclue : un barrage arrête la vue comme un bâtiment
// (§13), et le poser sous soi aveuglerait le pion qui vient de le poser.
//
// La règle de l'angle fermé ne s'applique pas ici, contrairement à
// terrainSteps : on pose un obstacle sur une case, on ne s'y déplace pas.
func (p *Game) abilityCells(depuis Position) []Position {
	var cases []Position
	for d := Nord; d <= NordOuest; d++ {
		if voisine := depuis.Step(d); p.IsWalkable(voisine) {
			cases = append(cases, voisine)
		}
	}
	return cases
}

// fugitiveMoves énumère déplacements, dépenses et changement de zone.
//
// Une case occupée par un inspecteur lui est fermée : il y serait à l'abri de
// tout contact, l'adjacence n'incluant pas la superposition.
func (p *Game) fugitiveMoves() []Move {
	var coups []Move
	depart := p.Fugitive.Position

	if p.Fugitive.StepsTaken < p.MobilityOf(SideFugitive, 0) {
		for _, arrivee := range p.terrainSteps(depart) {
			if p.occupied(arrivee) {
				continue
			}
			coups = append(coups, Move{
				Turn: p.Turn, Side: SideFugitive, Type: MoveStep,
				From: depart, To: arrivee,
			})
		}
	}

	coups = append(coups, p.expenseMoves()...)
	coups = append(coups, Move{Turn: p.Turn, Side: SideFugitive, Type: MovePass})
	return append(coups, Move{Turn: p.Turn, Side: SideFugitive, Type: MoveEndPhase})
}

// terrainSteps rend les cases qu'un pas depuis depart peut atteindre, terrain
// seul.
//
// Sans le quota ni l'occupation, que l'appelant ajoute s'il les veut : un
// déplacement les subit, un leurre non — une trace reste plausible sur une case
// où un inspecteur se tient maintenant, puisqu'il a pu y arriver après.
//
// Extraite parce qu'elle sert deux fois, et c'est ce qui donne au leurre sa
// contrainte sans qu'on l'écrive : il ne peut produire qu'une trace qu'un vrai
// pas aurait pu laisser. Une seconde validation aurait dérivé de celle-ci.
func (p *Game) terrainSteps(depart Position) []Position {
	var cases []Position
	for d := Nord; d <= NordOuest; d++ {
		arrivee := depart.Step(d)
		if !p.IsWalkable(arrivee) {
			continue
		}
		// Un angle fermé ne se franchit pas : la diagonale exige qu'au moins
		// une des deux cases orthogonales intermédiaires soit une rue, sinon le
		// bâti ne bloque plus rien.
		if d.IsDiagonal() {
			a, b := depart.CornerCells(d)
			if !p.IsWalkable(a) && !p.IsWalkable(b) {
				continue
			}
		}
		cases = append(cases, arrivee)
	}
	return cases
}

// decoyMoves énumère les traces qu'un leurre peut poser.
//
// La case de départ est la sienne ou une voisine praticable, la case visée en
// est atteignable par un pas : docs/regles.md §8 donne la première liberté, et
// la seconde vient du terrain. Un leurre pointant vers la case que le fugitif
// occupe reste proposé — il dirait la vérité, mais l'énumération dit ce qui est
// légal, pas ce qui est sage, et l'écarter demanderait au noyau une notion
// d'utilité qu'il n'a nulle part ailleurs.
func (p *Game) decoyMoves(cle Expense) []Move {
	depart := p.Fugitive.Position
	origines := append([]Position{depart}, p.terrainSteps(depart)...)

	var coups []Move
	for _, at := range origines {
		if !p.IsWalkable(at) {
			continue
		}
		for _, vers := range p.terrainSteps(at) {
			coups = append(coups, Move{
				Turn: p.Turn, Side: SideFugitive, Type: MoveExpense,
				Expense: cle, From: at, To: vers,
			})
		}
	}
	return coups
}

// poseUneTrace dit si une dépense demande au joueur où poser sa trace.
//
// Sur ce que la dépense fait et non sur son nom : le noyau n'a pas à savoir
// laquelle s'appelle leurre, pas plus qu'il ne sait laquelle plafonne à trois.
func poseUneTrace(d Ability) bool {
	for _, e := range d.Effects {
		if e.Type == EffectDecoyTrail {
			return true
		}
	}
	return false
}

// expenseMoves énumère ce que le fugitif peut acheter avec sa résistance.
//
// Le changement de zone est une dépense comme une autre, mais porte un type de
// coup distinct : il désigne une zone, que les autres n'ont pas à porter.
//
// Il ne peut pas y engager son dernier point. À zéro l'arbitre le déclare
// vaincu, donc l'effet acheté ne servirait jamais : ce n'est pas une liberté
// qu'on lui retire mais un piège qu'on referme. Un cerveau qui tire au hasard
// s'y ruinait en deux tours, ce qu'aucun joueur n'aurait fait.
func (p *Game) expenseMoves() []Move {
	if p.Extensions == nil {
		return nil
	}

	cles := make([]Expense, 0, len(p.Extensions.Expenses))
	for cle := range p.Extensions.Expenses {
		cles = append(cles, cle)
	}
	sort.Slice(cles, func(i, j int) bool { return cles[i] < cles[j] })

	var coups []Move
	for _, cle := range cles {
		d := p.Extensions.Expenses[cle]

		// Le déclenchement était déclaré par les cinq dépenses livrées et lu
		// par personne : une dépense marquée « phase inspecteurs » se proposait
		// quand même au fugitif, et son auteur n'avait aucun moyen de
		// l'apprendre.
		if d.Camp != SideFugitive || d.Trigger != OnFugitivePhase {
			continue
		}
		if d.Cost >= p.Fugitive.Stamina {
			continue
		}
		if d.Uses > 0 && p.ExpenseUses[cle] >= d.Uses {
			continue
		}
		if cle == ExpenseSilence && p.Fugitive.SilenceBought {
			continue
		}
		if poseUneTrace(d) {
			coups = append(coups, p.decoyMoves(cle)...)
			continue
		}
		coups = append(coups, Move{
			Turn: p.Turn, Side: SideFugitive, Type: MoveExpense, Expense: cle,
		})
	}

	return append(coups, p.changeZoneMoves()...)
}

// changeZoneMoves énumère les zones vers lesquelles le fugitif peut resceller.
func (p *Game) changeZoneMoves() []Move {
	if p.Extensions == nil {
		return nil
	}
	d, connue := p.Extensions.Expenses[ExpenseChangeZone]
	if !connue || d.Cost >= p.Fugitive.Stamina {
		return nil
	}

	var coups []Move
	for _, z := range p.Board.Zones() {
		if z.Number == p.Fugitive.SealedZone {
			continue
		}
		coups = append(coups, Move{
			Turn: p.Turn, Side: SideFugitive, Type: MoveChangeZone, Zone: z.Number,
		})
	}
	return coups
}

// occupied dit si un inspecteur tient la case.
func (p *Game) occupied(pos Position) bool {
	for _, i := range p.Inspectors {
		if i.Position == pos {
			return true
		}
	}
	return false
}
