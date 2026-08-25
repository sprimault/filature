// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

// PlafondContacts borne la perte de résistance d'un seul tour.
//
// Être encerclé doit faire très mal sans être instantanément fatal : sans
// plafond, cinq inspecteurs coûteraient la moitié de la jauge en un tour et la
// partie se terminerait avant que le fugitif puisse réagir.
const PlafondContacts = 3

// resoudreFinDeTour enchaîne visibilité, contacts, traces, révélation, effets
// différés et étranglement.
//
// L'ordre est un contrat : le décompte des contacts a lieu après le déplacement
// du fugitif et non avant, et les effets différés arrivent à échéance avant le
// test de fin. Le changer change le jeu, silencieusement.
func (p *Partie) resoudreFinDeTour() []func() {
	var defaire []func()
	for _, etape := range []func() []func(){
		p.recalculerVisibilite,
		p.subirLesContacts,
		p.deposerLesTraces,
		p.effacerLesTracesAnciennes,
		p.revelerSiCestLHeure,
		p.resoudreLesDifferes,
		p.etrangler,
	} {
		defaire = append(defaire, etape()...)
	}
	return defaire
}

// recalculerVisibilite met à jour le pion du fugitif.
//
// La visibilité se recalcule entièrement : un fugitif qui sort d'une ligne de
// vue redevient invisible, ce qu'un simple « devient visible » ne rendrait pas.
//
// EstVisible relève de l'étape 4 et renvoie false pour l'instant : le câblage
// est ici, la table de vision viendra sous lui.
func (p *Partie) recalculerVisibilite() []func() {
	vu := false
	occupees := p.casesOccupees()
	for i := range p.Inspecteurs {
		if EstVisible(p.Plateau, p.Inspecteurs[i].Position, p.Fugitif.Position, p.PorteeDe(i), occupees) {
			vu = true
			break
		}
	}

	if p.Fugitif.Visible == vu {
		return nil
	}
	precedent := p.Fugitif.Visible
	p.Fugitif.Visible = vu
	return []func(){func() { p.Fugitif.Visible = precedent }}
}

// casesOccupees rassemble les cases qui coupent une ligne de vue.
//
// Les inspecteurs et les barrages bloquent la vue de la même façon : sans cela,
// la capacité du Barreur ne serait qu'un mur de déplacement, sans effet sur
// l'information — ce qui, dans ce jeu, revient à n'avoir aucun effet.
func (p *Partie) casesOccupees() map[Position]bool {
	occupees := make(map[Position]bool, len(p.Inspecteurs)+len(p.Barrages))
	for _, i := range p.Inspecteurs {
		occupees[i.Position] = true
	}
	for pos := range p.Barrages {
		occupees[pos] = true
	}
	return occupees
}

// subirLesContacts retire au fugitif un point par inspecteur adjacent.
func (p *Partie) subirLesContacts() []func() {
	perte := p.contacts()
	if perte == 0 {
		return nil
	}
	return []func(){p.ajusterResistance(-perte)}
}

// contacts compte les inspecteurs orthogonalement adjacents au fugitif.
//
// Les diagonales ne comptent pas : elles font du fugitif un pion plus rapide,
// pas un pion plus vulnérable. Le total est plafonné par PlafondContacts.
func (p *Partie) contacts() int {
	n := 0
	for _, i := range p.Inspecteurs {
		if DistanceManhattan(i.Position, p.Fugitif.Position) == 1 {
			n++
		}
	}
	return min(n, PlafondContacts)
}

// deposerLesTraces marque les cases que le fugitif a quittées ce tour.
//
// Une par case quittée, pas une par tour : un double déplacement en laisse deux,
// et c'est ce qui rend la dépense lisible pour qui les découvre.
func (p *Partie) deposerLesTraces() []func() {
	var defaire []func()
	for _, c := range p.Journal {
		if c.Tour != p.Tour || c.Acteur != CampFugitif || c.Type != CoupDeplacer {
			continue
		}
		direction, adjacente := DirectionVers(c.Depart, c.Arrivee)
		if !adjacente {
			continue
		}
		defaire = append(defaire, p.poserTrace(c.Depart, Trace{Tour: p.Tour, Direction: direction}))
	}
	return defaire
}

// poserTrace inscrit une trace et renvoie de quoi la retirer.
func (p *Partie) poserTrace(pos Position, t Trace) func() {
	etaitNulle := p.Traces == nil
	if etaitNulle {
		p.Traces = map[Position]Trace{}
	}
	precedente, existait := p.Traces[pos]
	p.Traces[pos] = t
	return func() {
		if existait {
			p.Traces[pos] = precedente
			return
		}
		delete(p.Traces, pos)
		if etaitNulle {
			p.Traces = nil
		}
	}
}

// effacerLesTracesAnciennes retire celles qui ont passé leur durée.
func (p *Partie) effacerLesTracesAnciennes() []func() {
	effacees := map[Position]Trace{}
	for pos, t := range p.Traces {
		if p.Tour-t.Tour >= p.Parametres.DureeTrace {
			effacees[pos] = t
			delete(p.Traces, pos)
		}
	}
	if len(effacees) == 0 {
		return nil
	}
	return []func(){func() {
		for pos, t := range effacees {
			p.Traces[pos] = t
		}
	}}
}

// revelerSiCestLHeure applique la révélation périodique.
//
// Sans ce battement, l'incertitude des inspecteurs ne converge jamais et la
// partie devient une loterie. C'est aussi ce qui borne la carte de croyance de
// l'IA quelle que soit la taille du plateau.
//
// Le silence acheté consomme la révélation sans la subir. Les inspecteurs
// apprennent qu'il a payé, pas où il est : ils savent qu'il s'est appauvri.
func (p *Partie) revelerSiCestLHeure() []func() {
	if p.Parametres.PeriodeRevelation <= 0 || p.Tour%p.Parametres.PeriodeRevelation != 0 {
		return nil
	}

	if p.Fugitif.SilenceAchete {
		p.Fugitif.SilenceAchete = false
		return []func(){func() { p.Fugitif.SilenceAchete = true }}
	}

	if p.Fugitif.Visible {
		return nil
	}
	p.Fugitif.Visible = true
	return []func(){func() { p.Fugitif.Visible = false }}
}

// resoudreLesDifferes applique les effets arrivés à échéance et vide la file de
// ce qu'elle a rendu.
//
// L'annulation remet la file telle quelle : c'est elle qui portait l'annonce,
// et un barrage annoncé puis escamoté serait pire que pas d'annonce du tout.
func (p *Partie) resoudreLesDifferes() []func() {
	var arrives, restants []EffetEnAttente
	for _, e := range p.EffetsEnAttente {
		if e.Tour <= p.Tour {
			arrives = append(arrives, e)
			continue
		}
		restants = append(restants, e)
	}
	if len(arrives) == 0 {
		return nil
	}

	file := p.EffetsEnAttente
	p.EffetsEnAttente = restants
	defaire := []func(){func() { p.EffetsEnAttente = file }}

	for _, e := range arrives {
		effets, err := p.appliquerEffets(e.Effets, e.Contexte)
		if err != nil {
			// Un effet différé qui échoue à l'échéance vient d'un greffon
			// chargé sans validation. Le tour continue plutôt que de s'arrêter
			// sur une partie qu'on ne peut plus reprendre.
			continue
		}
		defaire = append(defaire, effets...)
	}
	return defaire
}

// etrangler déclenche le mode qui ferme les zones, quand le tour l'impose.
//
// Le noyau donne la cadence et la cible, le mode dit ce qui se passe. Un
// greffon qui remplace ce mode change le préavis ou l'effet, jamais le moment.
func (p *Partie) etrangler() []func() {
	zone, cetour := p.zoneAEtrangler()
	if !cetour || p.Extensions == nil {
		return nil
	}
	mode, connu := p.Extensions.Modes["etranglement"]
	if !connu || mode.Declenchement != SurEtranglement {
		return nil
	}

	effets, err := p.appliquerEffets(mode.Effets, Contexte{Acteur: CampInspecteurs, Zone: zone})
	if err != nil {
		return nil
	}
	return effets
}
