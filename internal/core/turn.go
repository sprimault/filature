// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import "slices"

// PlafondContacts borne la perte de résistance d'un seul tour.
//
// Être encerclé doit faire très mal sans être instantanément fatal : sans
// plafond, cinq inspecteurs coûteraient la moitié de la jauge en un tour et la
// partie se terminerait avant que le fugitif puisse réagir.
const PlafondContacts = 3

// resolveTurnEnd enchaîne visibilité, contacts, traces, révélation, effets
// différés, étranglement et décompte d'extraction.
//
// L'ordre est un contrat : le décompte des contacts a lieu après le déplacement
// du fugitif et non avant, les effets différés arrivent à échéance avant le test
// de fin, et l'extraction se compte en dernier pour qu'une zone fermée à
// l'instant interrompe le compte du tour même. Le changer change le jeu,
// silencieusement.
//
// La capture se constate ici et non dans Outcome, pour la même raison que la
// visibilité : elle dépend d'un instant précis. Comparée à la demande, elle
// vaudrait dès qu'un inspecteur déjà au contact reste où il est, sans laisser
// au fugitif la phase que la règle lui accorde pour s'écarter.
func (p *Game) resolveTurnEnd() []func() {
	var defaire []func()
	for _, etape := range []func() []func(){
		p.recomputeSpotting,
		p.takeContacts,
		p.checkCapture,
		p.dropTrails,
		p.wipeOldTrails,
		p.revealIfDue,
		p.resolveDeferred,
		p.strangle,
		p.countExtraction,
	} {
		defaire = append(defaire, etape()...)
	}
	return defaire
}

// recomputeSpotting met à jour le pion du fugitif.
//
// La visibilité se recalcule entièrement : un fugitif qui sort d'une ligne de
// vue redevient invisible, ce qu'un simple « devient visible » ne rendrait pas.
//
// Un seul inspecteur suffit, d'où l'arrêt au premier : la vue est partagée, et
// savoir lequel a repéré le fugitif ne change rien à la suite du tour.
func (p *Game) recomputeSpotting() []func() {
	vu := false
	occupees := p.occupiedCells()
	for i := range p.Inspectors {
		if IsVisible(p.Board, p.Inspectors[i].Position, p.Fugitive.Position, p.RangeOf(i), occupees) {
			vu = true
			break
		}
	}

	if p.Fugitive.Visible == vu {
		return nil
	}
	precedent := p.Fugitive.Visible
	p.Fugitive.Visible = vu
	return []func(){func() { p.Fugitive.Visible = precedent }}
}

// occupiedCells rassemble les cases qui coupent une ligne de vue.
//
// Les inspecteurs et les barrages bloquent la vue de la même façon : sans cela,
// la capacité du Barreur ne serait qu'un mur de déplacement, sans effet sur
// l'information — ce qui, dans ce jeu, revient à n'avoir aucun effet.
func (p *Game) occupiedCells() map[Position]bool {
	occupees := make(map[Position]bool, len(p.Inspectors)+len(p.Roadblocks))
	for _, i := range p.Inspectors {
		occupees[i.Position] = true
	}
	for pos := range p.Roadblocks {
		occupees[pos] = true
	}
	return occupees
}

// takeContacts retire au fugitif un point par inspecteur au contact.
func (p *Game) takeContacts() []func() {
	perte := min(len(p.contacting()), PlafondContacts)
	if perte == 0 {
		return nil
	}
	return []func(){p.adjustStamina(-perte)}
}

// checkCapture constate la capture, puis retient les contacts du tour.
//
// Un même inspecteur à deux résolutions consécutives capture. Un seul suffit :
// en terrain ouvert le fugitif rompt le contact d'une diagonale, qui porte la
// distance orthogonale de un à trois quand le poursuivant n'en regagne qu'un.
// Ce qui se maintient, c'est le talonnage dans un couloir d'une case — et c'est
// là que le double déplacement redevient une dépense de survie.
func (p *Game) checkCapture() []func() {
	actuels := p.contacting()

	pris := false
	for _, i := range actuels {
		if slices.Contains(p.LastContacts, i) {
			pris = true
			break
		}
	}

	precedents, precedentPris := p.LastContacts, p.Fugitive.Captured
	p.LastContacts = actuels
	p.Fugitive.Captured = p.Fugitive.Captured || pris

	return []func(){func() {
		p.LastContacts = precedents
		p.Fugitive.Captured = precedentPris
	}}
}

// contacting rend les pions au contact du fugitif, dans l'ordre du camp.
//
// Un inspecteur est au contact s'il occupe la case du fugitif ou une case
// orthogonalement adjacente. Les diagonales ne comptent pas : elles font du
// fugitif un pion plus rapide, pas un pion plus vulnérable.
//
// La case commune en fait partie, et il le faut : rien n'interdit à un
// inspecteur de marcher sur un fugitif caché, et l'en empêcher lui apprendrait
// où il est. Un pion à distance nulle qui ne compterait pour rien serait le
// seul du plateau à ne rien faire de sa position.
//
// Qui et non combien : la capture demande le même inspecteur à deux
// résolutions consécutives, donc leur identité, pas leur nombre.
func (p *Game) contacting() []int {
	var pions []int
	for i := range p.Inspectors {
		if ManhattanDistance(p.Inspectors[i].Position, p.Fugitive.Position) <= 1 {
			pions = append(pions, i)
		}
	}
	return pions
}

// dropTrails marked les cases que le fugitif a quittées ce tour.
//
// Une par case quittée, pas une par tour : un double déplacement en laisse deux,
// et c'est ce qui rend la dépense lisible pour qui les découvre.
func (p *Game) dropTrails() []func() {
	var defaire []func()
	for _, c := range p.Journal {
		if c.Turn != p.Turn || c.Side != SideFugitive || c.Type != MoveStep {
			continue
		}
		direction, adjacente := DirectionTo(c.From, c.To)
		if !adjacente {
			continue
		}
		defaire = append(defaire, p.dropTrail(c.From, Trail{Turn: p.Turn, Direction: direction}))
	}
	return defaire
}

// dropTrail inscrit une trace et renvoie de quoi la retirer.
func (p *Game) dropTrail(pos Position, t Trail) func() {
	etaitNulle := p.Trails == nil
	if etaitNulle {
		p.Trails = map[Position]Trail{}
	}
	precedente, existait := p.Trails[pos]
	p.Trails[pos] = t
	return func() {
		if existait {
			p.Trails[pos] = precedente
			return
		}
		delete(p.Trails, pos)
		if etaitNulle {
			p.Trails = nil
		}
	}
}

// wipeOldTrails retire celles qui ont passé leur durée.
func (p *Game) wipeOldTrails() []func() {
	effacees := map[Position]Trail{}
	for pos, t := range p.Trails {
		if p.Turn-t.Turn >= p.Settings.TrailLifetime {
			effacees[pos] = t
			delete(p.Trails, pos)
		}
	}
	if len(effacees) == 0 {
		return nil
	}
	return []func(){func() {
		for pos, t := range effacees {
			p.Trails[pos] = t
		}
	}}
}

// revealIfDue applique la révélation périodique.
//
// Sans ce battement, l'incertitude des inspecteurs ne converge jamais et la
// partie devient une loterie. C'est aussi ce qui borne la carte de croyance de
// l'IA quelle que soit la taille du plateau.
//
// Le silence acheté consomme la révélation sans la subir. Les inspecteurs
// apprennent qu'il a payé, pas où il est : ils savent qu'il s'est appauvri.
func (p *Game) revealIfDue() []func() {
	if p.Settings.RevealPeriod <= 0 || p.Turn%p.Settings.RevealPeriod != 0 {
		return nil
	}

	if p.Fugitive.SilenceBought {
		p.Fugitive.SilenceBought = false
		return []func(){func() { p.Fugitive.SilenceBought = true }}
	}

	if p.Fugitive.Visible {
		return nil
	}
	p.Fugitive.Visible = true
	return []func(){func() { p.Fugitive.Visible = false }}
}

// resolveDeferred applique les effets arrivés à échéance et vide la file de
// ce qu'elle a rendu.
//
// L'annulation remet la file telle quelle : c'est elle qui portait l'annonce,
// et un barrage annoncé puis escamoté serait pire que pas d'annonce du tout.
func (p *Game) resolveDeferred() []func() {
	var arrives, restants []PendingEffect
	for _, e := range p.PendingEffects {
		if e.Turn <= p.Turn {
			arrives = append(arrives, e)
			continue
		}
		restants = append(restants, e)
	}
	if len(arrives) == 0 {
		return nil
	}

	file := p.PendingEffects
	p.PendingEffects = restants
	defaire := []func(){func() { p.PendingEffects = file }}

	for _, e := range arrives {
		effets, err := p.applyEffects(e.Effects, e.EffectContext)
		if err != nil {
			// Un effet différé qui échoue à l'échéance vient d'un plugin
			// chargé sans validation. Le tour continue plutôt que de s'arrêter
			// sur une partie qu'on ne peut plus reprendre.
			continue
		}
		defaire = append(defaire, effets...)
	}
	return defaire
}

// strangle déclenche le mode qui ferme les zones, quand le tour l'impose.
//
// Le noyau donne la cadence et la cible, le mode dit ce qui se passe. Un
// plugin qui remplace ce mode change le préavis ou l'effet, jamais le moment.
func (p *Game) strangle() []func() {
	zone, cetour := p.zoneToStrangle()
	if !cetour || p.Extensions == nil {
		return nil
	}
	mode, connu := p.Extensions.Modes["etranglement"]
	if !connu || mode.Trigger != OnStrangling {
		return nil
	}

	effets, err := p.applyEffects(mode.Effects, EffectContext{Side: SideInspectors, Zone: zone})
	if err != nil {
		return nil
	}
	return effets
}
