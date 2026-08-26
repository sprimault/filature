// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import "sort"

// Vue est ce qu'un camp a le droit de savoir. Le noyau n'expose rien d'autre à
// l'interface, y compris en partie locale.
//
// C'est l'invariant le plus coûteux à rétrofiter et le plus facile à poser
// maintenant : si l'interface consomme l'état complet, un joueur lit la
// position du fugitif dans le trafic réseau ou dans les outils de
// développement du navigateur, et le jeu n'a plus d'objet.
type Vue struct {
	Acteur     Acteur     `json:"acteur"`
	Tour       int        `json:"tour"`
	Phase      Phase      `json:"phase"`
	Parametres Parametres `json:"parametres"`

	// Rues est la portion de plateau connue du client. Sur plateau borné
	// c'est tout ; sur plateau infini, ce sera ce qui a été exploré.
	Rues     []Position `json:"rues"`
	Zones    []Zone     `json:"zones"`
	Barrages []Position `json:"barrages"`

	Inspecteurs []Inspecteur `json:"inspecteurs"`

	// PositionFugitif n'est renseigné que pour le camp fugitif, ou pour les
	// inspecteurs quand il est visible ou révélé.
	PositionFugitif *Position `json:"position_fugitif,omitempty"`
	ZoneScellee     *int      `json:"zone_scellee,omitempty"`

	// Resistance n'est renseignée que pour le fugitif.
	//
	// Un pointeur et non un entier : à zéro, le champ dirait qu'il est mort au
	// lieu de dire qu'on n'en sait rien. Les inspecteurs comptent les contacts
	// qu'ils infligent et apprennent qu'un silence a été payé — c'est tout, et
	// c'est ce que la règle leur accorde. Montrer le solde rendrait cette
	// annonce inutile, et leur livrerait par la bande chaque dépense du
	// fugitif : une baisse de deux sans contact ne peut être qu'un double
	// déplacement ou un changement de zone.
	Resistance *int `json:"resistance,omitempty"`

	// TracesConnues ne contient que ce que les inspecteurs ont effectivement
	// découvert. Le fugitif, lui, voit les siennes.
	TracesConnues map[string]Trace `json:"traces_connues"`

	// Scenes est identique pour les deux camps : un meurtre est public, c'est
	// ce que le fugitif paie. Ne jamais la filtrer par acteur.
	Scenes []Scene `json:"scenes"`

	CasesVisibles   []Position `json:"cases_visibles"`
	CoupsLegaux     []Coup     `json:"coups_legaux"`
	ProchaineReveal int        `json:"prochaine_revelation"`
	SilencePaye     bool       `json:"silence_paye"`
	ZonesAnnoncees  []int      `json:"zones_annoncees"`

	// EffetsAnnonces ne porte que les differer déclarés avec annonce, et les
	// porte à l'identique pour les deux camps. Un differer sans annonce reste
	// invisible jusqu'à sa résolution, sinon le champ le trahirait.
	EffetsAnnonces []EffetEnAttente `json:"effets_annonces"`

	Resultat *Resultat `json:"resultat,omitempty"`
}

// VuePour projette l'état pour un camp.
//
// La règle de relecture : tout champ ajouté à Partie doit être explicitement
// copié ici, jamais par recopie de structure. Une omission fait fuiter, un
// oubli ne fait qu'afficher moins.
//
// Trois choses seulement sont cachées, et ce sont les trois qui font le jeu :
// où est le fugitif, quelle zone il vise, et quelles traces il a laissées hors
// de portée. Tout le reste est public, y compris sa résistance — les contacts
// et les silences payés sont annoncés, la déduire serait un exercice sans
// intérêt.
func (p *Partie) VuePour(a Acteur) Vue {
	v := Vue{
		Acteur:          a,
		Tour:            p.Tour,
		Phase:           p.Phase,
		Parametres:      p.Parametres,
		Rues:            liste(p.ruesConnues()),
		Zones:           liste(p.zonesVues()),
		Barrages:        liste(p.barrages()),
		Inspecteurs:     liste(append([]Inspecteur(nil), p.Inspecteurs...)),
		TracesConnues:   p.tracesPour(a),
		Scenes:          liste(append([]Scene(nil), p.Scenes...)),
		CasesVisibles:   liste(p.casesVisiblesPour(a)),
		CoupsLegaux:     liste(p.CoupsLegaux(a)),
		ProchaineReveal: p.prochaineRevelation(),
		SilencePaye:     p.Fugitif.SilenceAchete,
		EffetsAnnonces:  liste(p.effetsAnnonces()),
	}
	v.ZonesAnnoncees = liste(zonesAnnoncees(v.EffetsAnnonces))

	// Le fugitif voit tout de lui-même. Les inspecteurs ne voient sa position
	// que s'il est repéré ou révélé, et sa zone jamais.
	if a == CampFugitif {
		position := p.Fugitif.Position
		resistance := p.Fugitif.Resistance
		v.PositionFugitif = &position
		v.Resistance = &resistance

		// Tant qu'il n'a pas scellé, le champ reste absent plutôt que de
		// porter la sentinelle du noyau : « -1 » ne veut rien dire pour qui
		// lit le JSON, et l'omission dit exactement ce qui est vrai — le
		// choix n'a pas été fait.
		if zone := p.Fugitif.ZoneScellee; zone >= 0 {
			v.ZoneScellee = &zone
		}
	} else if p.Fugitif.Visible {
		position := p.Fugitif.Position
		v.PositionFugitif = &position
	}

	if r, fini := p.Resultat(); fini {
		v.Resultat = &r
	}
	return v
}

// liste garantit une tranche non nulle.
//
// Une tranche vide se sérialise en null, pas en tableau vide : un bot devrait
// alors traiter les deux formes pour chacune des neuf listes de la vue, et
// celui qui ne le ferait que pour certaines tomberait sur les autres. Le
// contrat promet un tableau, il en rend un.
func liste[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// ruesConnues rend la portion de plateau que le client peut afficher.
//
// Tout le plateau borné aujourd'hui. Sur plateau infini, ce sera ce qui a été
// exploré, et c'est pour cela que le champ existe plutôt que de laisser le
// client interroger le plateau lui-même.
func (p *Partie) ruesConnues() []Position {
	rayon := p.Parametres.Cote / 2
	return p.Plateau.CasesDans(Position{Colonne: rayon, Ligne: rayon}, rayon)
}

// zonesVues rend les zones avec leur état de fermeture.
//
// Les six sont connues des deux camps dès la mise en place : seul le choix du
// fugitif est caché, pas les points d'extraction eux-mêmes.
func (p *Partie) zonesVues() []Zone {
	zones := append([]Zone(nil), p.Plateau.Zones()...)
	for i := range zones {
		for _, ferme := range p.ZonesFermees {
			if zones[i].Numero == ferme {
				zones[i].Fermee = true
			}
		}
	}
	return zones
}

// barrages rend les cases fermées par le Barreur, triées.
//
// Publiques : elles bloquent le déplacement et la vue, un joueur qui les
// ignorerait jouerait à l'aveugle sur un terrain que l'autre voit.
func (p *Partie) barrages() []Position {
	if len(p.Barrages) == 0 {
		return nil
	}
	cases := make([]Position, 0, len(p.Barrages))
	for pos := range p.Barrages {
		cases = append(cases, pos)
	}
	trierPositions(cases)
	return cases
}

// tracesPour filtre les traces selon ce que le camp a le droit de savoir.
//
// Le fugitif voit les siennes, toutes. Un inspecteur ne découvre une trace
// qu'en occupant sa case ou une case orthogonalement adjacente — donc en
// distance de Manhattan, jamais de Tchebychev. Les confondre étendrait la
// détection aux quatre diagonales et doublerait presque la couverture, ce qui
// est le défaut sur lequel un prototype antérieur s'est fait prendre.
func (p *Partie) tracesPour(a Acteur) map[string]Trace {
	if len(p.Traces) == 0 {
		return map[string]Trace{}
	}

	connues := make(map[string]Trace, len(p.Traces))
	if a == CampFugitif {
		for pos, t := range p.Traces {
			connues[pos.Cle()] = t
		}
		return connues
	}

	for pos, t := range p.Traces {
		for i := range p.Inspecteurs {
			if DistanceManhattan(p.Inspecteurs[i].Position, pos) <= p.RayonTracesDe(i) {
				connues[pos.Cle()] = t
				break
			}
		}
	}
	return connues
}

// casesVisiblesPour rend ce que le camp voit du terrain.
//
// Les inspecteurs voient depuis chacun de leurs pions. Le fugitif n'a pas de
// vision propre dans les règles : il sait s'il est repéré, pas ce que l'autre
// couvre.
//
// La table du plateau ne connaît que le terrain : elle donne les candidats, et
// EstVisible tranche. Sans ce second passage, la vue annoncerait des cases
// situées derrière un collègue ou un barrage, qu'au même instant le fugitif ne
// serait pas repéré d'occuper — deux réponses contradictoires à la même
// question, dont l'une part sur le réseau.
func (p *Partie) casesVisiblesPour(a Acteur) []Position {
	if a == CampFugitif {
		return nil
	}

	occupees := p.casesOccupees()
	vues := map[Position]bool{}
	for i := range p.Inspecteurs {
		depuis, portee := p.Inspecteurs[i].Position, p.PorteeDe(i)

		// Sa propre case, qu'aucune ligne de vue ne contient : elles partent de
		// lui sans l'inclure. Sans ça, la zone couverte aurait un trou sous
		// chaque pion.
		vues[depuis] = true

		for _, c := range p.Plateau.Vision(depuis, portee) {
			if !vues[c] && EstVisible(p.Plateau, depuis, c, portee, occupees) {
				vues[c] = true
			}
		}
	}
	if len(vues) == 0 {
		return nil
	}

	cases := make([]Position, 0, len(vues))
	for c := range vues {
		cases = append(cases, c)
	}
	trierPositions(cases)
	return cases
}

// prochaineRevelation rend le nombre de tours avant la prochaine révélation.
func (p *Partie) prochaineRevelation() int {
	periode := p.Parametres.PeriodeRevelation
	if periode <= 0 {
		return 0
	}
	if reste := p.Tour % periode; reste != 0 {
		return periode - reste
	}
	return periode
}

// effetsAnnonces rend les différés que les deux camps ont le droit de voir.
//
// Un differer sans annonce n'y figure pas : le champ le trahirait, alors que
// c'est justement le choix de son auteur de ne pas prévenir.
func (p *Partie) effetsAnnonces() []EffetEnAttente {
	var annonces []EffetEnAttente
	for _, e := range p.EffetsEnAttente {
		if e.Annonce {
			annonces = append(annonces, e)
		}
	}
	return annonces
}

// zonesAnnoncees extrait les zones dont la fermeture est annoncée.
//
// Dérivé des effets plutôt que stocké : c'est la même information sous une
// forme que l'interface consomme sans parcourir des effets imbriqués.
func zonesAnnoncees(annonces []EffetEnAttente) []int {
	var zones []int
	for _, e := range annonces {
		for _, effet := range e.Effets {
			if effet.Type == EffetFermerZone {
				zones = append(zones, e.Contexte.Zone)
			}
		}
	}
	return zones
}

// trierPositions ordonne par ligne puis colonne.
//
// L'ordre d'une map n'est pas stable en Go : sans tri, deux vues du même état
// différeraient, et le rejeu d'un journal comme la comparaison de deux parties
// cesseraient d'être fiables.
func trierPositions(cases []Position) {
	sort.Slice(cases, func(i, j int) bool {
		if cases[i].Ligne != cases[j].Ligne {
			return cases[i].Ligne < cases[j].Ligne
		}
		return cases[i].Colonne < cases[j].Colonne
	})
}
