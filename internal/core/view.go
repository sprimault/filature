// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import "sort"

// View est ce qu'un camp a le droit de savoir. Le noyau n'expose rien d'autre à
// l'interface, y compris en partie locale.
//
// C'est l'invariant le plus coûteux à rétrofiter et le plus facile à poser
// maintenant : si l'interface consomme l'état complet, un joueur lit la
// position du fugitif dans le trafic réseau ou dans les outils de
// développement du navigateur, et le jeu n'a plus d'objet.
type View struct {
	Side     Side     `json:"side"`
	Turn     int      `json:"turn"`
	Phase    Phase    `json:"phase"`
	Settings Settings `json:"settings"`

	// Streets est la portion de plateau connue du client. Sur plateau borné
	// c'est tout ; sur plateau infini, ce sera ce qui a été exploré.
	Streets    []Position `json:"streets"`
	Zones      []Zone     `json:"zones"`
	Roadblocks []Position `json:"roadblocks"`

	// Shelters et ShelterReady sont publics pour les deux camps, et il le faut :
	// les inspecteurs ne peuvent pas couvrir ce qu'ils ne voient pas, et un lieu
	// en recharge est l'indice que le fugitif paie en s'y ressourçant. Il dit
	// qu'il est passé là, et quand le lieu reviendra.
	Shelters     []Shelter `json:"shelters"`
	ShelterReady []int     `json:"shelter_ready"`

	Inspectors []Inspector `json:"inspectors"`

	// PositionFugitif n'est renseigné que pour le camp fugitif, ou pour les
	// inspecteurs quand il est visible ou révélé.
	PositionFugitif *Position `json:"fugitive_position,omitempty"`
	SealedZone      *int      `json:"sealed_zone,omitempty"`

	// Stamina n'est renseignée que pour le fugitif.
	//
	// Un pointeur et non un entier : à zéro, le champ dirait qu'il est mort au
	// lieu de dire qu'on n'en sait rien.
	//
	// Ce que les inspecteurs apprennent de sa jauge tient à deux choses : les
	// contacts qu'ils infligent **quand ils le voient**, et l'annonce d'un
	// silence payé. Rien ne le leur dit autrement — un contact infligé à un
	// fugitif caché ne leur est pas signalé, sans quoi ils le localiseraient à
	// une case près sans l'avoir vu. Montrer le solde rendrait l'annonce
	// inutile et leur livrerait par la bande chaque dépense : une baisse de
	// deux sans contact ne peut être qu'un double déplacement ou un changement
	// de zone.
	Stamina *int `json:"stamina,omitempty"`

	// KnownTrails ne contient que ce que les inspecteurs ont effectivement
	// découvert. Le fugitif, lui, voit les siennes.
	KnownTrails map[string]Trail `json:"known_trails"`

	// CrimeScenes est identique pour les deux camps : un meurtre est public, c'est
	// ce que le fugitif paie. Ne jamais la filtrer par acteur.
	CrimeScenes []CrimeScene `json:"crime_scenes"`

	CasesVisibles   []Position `json:"visible_cells"`
	LegalMoves      []Move     `json:"legal_moves"`
	ProchaineReveal int        `json:"next_reveal"`
	SilencePaye     bool       `json:"silence_paid"`
	ZonesAnnoncees  []int      `json:"announced_zones"`

	// AnnouncedEffects ne porte que les differer déclarés avec annonce, et les
	// porte à l'identique pour les deux camps. Un differer sans annonce reste
	// invisible jusqu'à sa résolution, sinon le champ le trahirait.
	AnnouncedEffects []PendingEffect `json:"announced_effects"`

	Outcome *Outcome `json:"outcome,omitempty"`
}

// ViewFor projette l'état pour un camp.
//
// La règle de relecture : tout champ ajouté à Game doit être explicitement
// copié ici, jamais par recopie de structure. Une omission fait fuiter, un
// oubli ne fait qu'afficher moins.
//
// Trois choses seulement sont cachées, et ce sont les trois qui font le jeu :
// où est le fugitif, quelle zone il vise, et quelles traces il a laissées hors
// de portée. Tout le reste est public, y compris sa résistance — les contacts
// et les silences payés sont annoncés, la déduire serait un exercice sans
// intérêt.
func (p *Game) ViewFor(a Side) View {
	v := View{
		Side:       a,
		Turn:       p.Turn,
		Phase:      p.Phase,
		Settings:   p.Settings,
		Streets:    list(p.knownStreets()),
		Zones:      list(p.seenZones()),
		Roadblocks: list(p.barrages()),
		Shelters:   list(append([]Shelter(nil), p.Board.Shelters()...)),

		ShelterReady: list(append([]int(nil), p.ShelterReady...)),
		// Recopie de structure, et c'est le seul endroit où elle est admise :
		// tout est public chez un inspecteur. Le corollaire est un piège —
		// **tout champ ajouté à Inspector devient visible des deux camps sans
		// que personne ne l'ait décidé.** Ce qui dépend de la position du
		// fugitif ne vit donc pas dans Inspector mais dans Game, hors de ce qui
		// se recopie : voir LastContacts.
		Inspectors:       list(append([]Inspector(nil), p.Inspectors...)),
		KnownTrails:      p.trailsFor(a),
		CrimeScenes:      list(append([]CrimeScene(nil), p.CrimeScenes...)),
		CasesVisibles:    list(p.visibleCellsFor(a)),
		LegalMoves:       list(p.LegalMoves(a)),
		ProchaineReveal:  p.nextReveal(),
		SilencePaye:      p.Fugitive.SilenceBought,
		AnnouncedEffects: list(p.announcedEffects()),
	}
	v.ZonesAnnoncees = list(p.announcedZones(v.AnnouncedEffects))

	// Le fugitif voit tout de lui-même. Les inspecteurs ne voient sa position
	// que s'il est repéré ou révélé, et sa zone jamais.
	if a == SideFugitive {
		position := p.Fugitive.Position
		resistance := p.Fugitive.Stamina
		v.PositionFugitif = &position
		v.Stamina = &resistance

		// Tant qu'il n'a pas scellé, le champ reste absent plutôt que de
		// porter la sentinelle du noyau : « -1 » ne veut rien dire pour qui
		// lit le JSON, et l'omission dit exactement ce qui est vrai — le
		// choix n'a pas été fait.
		if zone := p.Fugitive.SealedZone; zone >= 0 {
			v.SealedZone = &zone
		}
	} else if p.Fugitive.Visible {
		position := p.Fugitive.Position
		v.PositionFugitif = &position
	}

	if r, fini := p.Outcome(); fini {
		v.Outcome = &r
	}
	return v
}

// list garantit une tranche non nulle.
//
// Une tranche vide se sérialise en null, pas en tableau vide : un bot devrait
// alors traiter les deux formes pour chacune des neuf listes de la vue, et
// celui qui ne le ferait que pour certaines tomberait sur les autres. Le
// contrat promet un tableau, il en rend un.
func list[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// knownStreets rend la portion de plateau que le client peut afficher.
//
// Tout le plateau borné aujourd'hui. Sur plateau infini, ce sera ce qui a été
// exploré, et c'est pour cela que le champ existe plutôt que de laisser le
// client interroger le plateau lui-même.
func (p *Game) knownStreets() []Position {
	rayon := p.Settings.Size / 2
	return p.Board.CellsWithin(Position{Column: rayon, Row: rayon}, rayon)
}

// seenZones rend les zones avec leur état de fermeture.
//
// Les six sont connues des deux camps dès la mise en place : seul le choix du
// fugitif est caché, pas les points d'extraction eux-mêmes.
func (p *Game) seenZones() []Zone {
	zones := append([]Zone(nil), p.Board.Zones()...)
	for i := range zones {
		for _, ferme := range p.ClosedZones {
			if zones[i].Number == ferme {
				zones[i].Closed = true
			}
		}
	}
	return zones
}

// barrages rend les cases fermées par le Barreur, triées.
//
// Publiques : elles bloquent le déplacement et la vue, un joueur qui les
// ignorerait jouerait à l'aveugle sur un terrain que l'autre voit.
func (p *Game) barrages() []Position {
	if len(p.Roadblocks) == 0 {
		return nil
	}
	cases := make([]Position, 0, len(p.Roadblocks))
	for pos := range p.Roadblocks {
		cases = append(cases, pos)
	}
	sortPositions(cases)
	return cases
}

// trailsFor filtre les traces selon ce que le camp a le droit de savoir.
//
// Le fugitif voit les siennes, toutes. Un inspecteur ne découvre une trace
// qu'en occupant sa case ou une case orthogonalement adjacente — donc en
// distance de Manhattan, jamais de Tchebychev. Les confondre étendrait la
// détection aux quatre diagonales et doublerait presque la couverture, ce qui
// est le défaut sur lequel un prototype antérieur s'est fait prendre.
func (p *Game) trailsFor(a Side) map[string]Trail {
	if len(p.Trails) == 0 {
		return map[string]Trail{}
	}

	connues := make(map[string]Trail, len(p.Trails))
	if a == SideFugitive {
		for pos, t := range p.Trails {
			connues[pos.Key()] = t
		}
		return connues
	}

	for pos, t := range p.Trails {
		for i := range p.Inspectors {
			if ManhattanDistance(p.Inspectors[i].Position, pos) <= p.TrailRadiusOf(i) {
				connues[pos.Key()] = t
				break
			}
		}
	}
	return connues
}

// visibleCellsFor rend ce que le camp voit du terrain.
//
// Les inspecteurs voient depuis chacun de leurs pions. Le fugitif n'a pas de
// vision propre dans les règles : il sait s'il est repéré, pas ce que l'autre
// couvre.
//
// La table du plateau ne connaît que le terrain : elle donne les candidats, et
// IsVisible tranche. Sans ce second passage, la vue annoncerait des cases
// situées derrière un collègue ou un barrage, qu'au même instant le fugitif ne
// serait pas repéré d'occuper — deux réponses contradictoires à la même
// question, dont l'une part sur le réseau.
func (p *Game) visibleCellsFor(a Side) []Position {
	if a == SideFugitive {
		return nil
	}

	occupees := p.occupiedCells()
	vues := map[Position]bool{}
	for i := range p.Inspectors {
		depuis, portee := p.Inspectors[i].Position, p.RangeOf(i)

		// Sa propre case, qu'aucune ligne de vue ne contient : elles partent de
		// lui sans l'inclure. Sans ça, la zone couverte aurait un trou sous
		// chaque pion.
		vues[depuis] = true

		for _, c := range p.Board.Sight(depuis, portee) {
			if !vues[c] && IsVisible(p.Board, depuis, c, portee, occupees) {
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
	sortPositions(cases)
	return cases
}

// nextReveal rend le nombre de tours avant la prochaine révélation.
func (p *Game) nextReveal() int {
	periode := p.Settings.RevealPeriod
	if periode <= 0 {
		return 0
	}
	if reste := p.Turn % periode; reste != 0 {
		return periode - reste
	}
	return periode
}

// announcedEffects rend les différés que les deux camps ont le droit de voir.
//
// Un differer sans annonce n'y figure pas : le champ le trahirait, alors que
// c'est justement le choix de son auteur de ne pas prévenir.
func (p *Game) announcedEffects() []PendingEffect {
	var annonces []PendingEffect
	for _, e := range p.PendingEffects {
		if e.Announced {
			annonces = append(annonces, e)
		}
	}
	return annonces
}

// announcedZones extrait les zones dont la fermeture est annoncée.
//
// Deux sources, et c'est voulu. L'étranglement vient du noyau, qui sait dès
// maintenant ce qu'il fermera dans StranglingNotice tours : le préavis est une
// règle (docs/regles.md §10) et ne peut donc pas dépendre d'un manifeste. Le
// reste vient des differer qu'un plugin a marqués annoncés, où prévenir ou non
// est le choix de son auteur.
//
// Dérivé plutôt que stocké : c'est la même information sous une forme que
// l'interface consomme sans walk des effets imbriqués.
func (p *Game) announcedZones(annonces []PendingEffect) []int {
	var zones []int
	if zone, prevue := p.zoneToStrangleAt(p.Turn + p.Settings.StranglingNotice); prevue {
		zones = append(zones, zone)
	}
	for _, e := range annonces {
		for _, effet := range e.Effects {
			if effet.Type == EffectCloseZone {
				zones = append(zones, e.EffectContext.Zone)
			}
		}
	}
	return zones
}

// sortPositions ordonne par ligne puis colonne.
//
// L'ordre d'une map n'est pas stable en Go : sans tri, deux vues du même état
// différeraient, et le rejeu d'un journal comme la comparaison de deux parties
// cesseraient d'être fiables.
func sortPositions(cases []Position) {
	sort.Slice(cases, func(i, j int) bool {
		if cases[i].Row != cases[j].Row {
			return cases[i].Row < cases[j].Row
		}
		return cases[i].Column < cases[j].Column
	})
}
