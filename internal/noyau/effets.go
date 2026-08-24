// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import "fmt"

// TypeEffet est le vocabulaire de modding.
//
// C'est le contrat central du projet : une capacité, une dépense de résistance
// ou un mode de jeu se décrit par composition de ces primitives, sans une ligne
// de code. Les cinq capacités de base sont écrites dans ce format dès le
// premier jour — c'est le test qui prouve que le vocabulaire suffit. Si l'une
// d'elles ne s'exprime pas ici, c'est qu'il manque une primitive, pas qu'il
// faut un cas particulier.
//
// Ajouter une primitive est une décision lourde : elle entre dans le contrat
// public et ne peut plus être retirée sans casser les greffons existants.
type TypeEffet string

// Le vocabulaire livré, décrit en entier dans docs/vocabulaire-effets.md.
// EffetTeleporter ne sert à aucune capacité de base — la téléportation a été
// retirée des règles — et reste offert aux greffons qui voudraient la rétablir.
const (
	EffetDeplacer          TypeEffet = "deplacer"
	EffetModifierPortee    TypeEffet = "modifier_portee"
	EffetModifierMobilite  TypeEffet = "modifier_mobilite"
	EffetBloquerCase       TypeEffet = "bloquer_case"
	EffetOuvrirCase        TypeEffet = "ouvrir_case"
	EffetRevelerTraces     TypeEffet = "reveler_traces"
	EffetRevelerPosition   TypeEffet = "reveler_position"
	EffetMarquerScene      TypeEffet = "marquer_scene"
	EffetAnnulerRevelation TypeEffet = "annuler_revelation"
	EffetPartagerVue       TypeEffet = "partager_vue"
	EffetCouterResistance  TypeEffet = "couter_resistance"
	EffetRendreResistance  TypeEffet = "rendre_resistance"
	EffetEffacerTraces     TypeEffet = "effacer_traces"
	EffetFermerZone        TypeEffet = "fermer_zone"
	EffetOuvrirZone        TypeEffet = "ouvrir_zone"
	EffetTeleporter        TypeEffet = "teleporter"
	EffetDifferer          TypeEffet = "differer"
	EffetFinPartie         TypeEffet = "fin_partie"
)

// Cible désigne ce sur quoi un effet s'applique.
type Cible string

// CiblePionCourant est le cas ordinaire ; les autres valeurs supposent que le
// contexte porte le pion, la case ou la zone visée.
const (
	CiblePionCourant Cible = "pion_courant"
	CibleTousPions   Cible = "tous_pions"
	CibleAutrePion   Cible = "autre_pion"
	CibleFugitif     Cible = "fugitif"
	CibleCase        Cible = "case"
	CibleZone        Cible = "zone"
)

// Effet est une primitive paramétrée. Les champs inutiles restent à zéro : un
// enregistrement plat se sérialise et se journalise sans effort, contrairement
// à une hiérarchie de types.
type Effet struct {
	Type   TypeEffet `toml:"type" json:"type"`
	Cible  Cible     `toml:"cible" json:"cible,omitempty"`
	Valeur int       `toml:"valeur" json:"valeur,omitempty"`
	Duree  int       `toml:"duree" json:"duree,omitempty"`
	Rayon  int       `toml:"rayon" json:"rayon,omitempty"`

	// Annonce et Puis n'ont de sens que pour EffetDifferer.
	//
	// Un effet différé annoncé figure dans la Vue des deux camps, et c'est
	// tout son intérêt : un mur qui apparaît sans prévenir transformerait un
	// plan raisonné en coup de dé et rendrait la carte de croyance inutile.
	//
	// Un differer imbriqué dans un differer est refusé au chargement : deux
	// durées s'additionnent, donc ça n'ajoute rien, et ça permettrait des
	// chaînes qu'aucune annulation ne saurait dérouler.
	Annonce bool    `toml:"annonce" json:"annonce,omitempty"`
	Puis    []Effet `toml:"puis" json:"puis,omitempty"`
}

// EffetEnAttente est une entrée de la file des effets différés.
//
// Résolue en fin de tour, avant le test de fin de partie. L'annulation défait
// la mise en file, pas l'effet : annuler le tour où le differer a été posé le
// retire de la file.
type EffetEnAttente struct {
	Effets   []Effet  `json:"effets"`
	Tour     int      `json:"tour"`
	Annonce  bool     `json:"annonce"`
	Contexte Contexte `json:"contexte"`
}

// EffetActif est un effet qui dure, avec le tour où il cesse.
//
// Portée, mobilité et rayon de détection ne sont pas stockés dans les pions
// mais recalculés à partir de cette liste. Un bonus est dérivable de ce qui
// l'a produit ; le figer dans Inspecteur en ferait un cache que rien ne
// réconcilie, et fermerait la porte aux capacités qu'un greffon invente.
type EffetActif struct {
	Effet    Effet    `json:"effet"`
	Contexte Contexte `json:"contexte"`

	// Echeance est le dernier tour où l'effet vaut. Zéro pour un effet
	// permanent — la capacité passive du Traqueur en est une.
	Echeance int `json:"echeance"`
}

// Vaut dit si l'effet court encore au tour donné.
func (a EffetActif) Vaut(tour int) bool {
	return a.Echeance == 0 || tour <= a.Echeance
}

// Vise dit si l'effet porte sur un pion d'inspecteur donné.
func (a EffetActif) Vise(pion int) bool {
	switch a.Effet.Cible {
	case CibleTousPions:
		return true
	case CiblePionCourant:
		return a.Contexte.Pion == pion
	case CibleAutrePion:
		return a.Contexte.AutrePion == pion
	}
	return false
}

// Contexte est ce dont un effet dispose pour s'appliquer. Il ne donne pas accès
// à la Partie entière : un greffon ne doit pas pouvoir lire la zone scellée du
// fugitif ni écrire dans le journal.
type Contexte struct {
	Acteur Acteur   `json:"acteur"`
	Pion   int      `json:"pion"`
	Case   Position `json:"case"`
	Zone   int      `json:"zone"`

	// AutrePion est le second pion d'un effet qui en relie deux. Le Chef, qui
	// voit à travers un coéquipier, est le seul cas livré.
	AutrePion int `json:"autre_pion"`
}

// Appliquer1Effet exécute un effet et renvoie de quoi le défaire.
//
// Le retour n'est pas optionnel : Partie.Annuler doit rester praticable, sinon
// l'IA ne peut plus explorer et le rejeu du journal diverge dès qu'un greffon
// est actif.
//
// Rien n'est validé ici. La légalité d'un coup relève de CoupsLegaux, et la
// cohérence d'un effet de son manifeste, contrôlée au chargement. Seul un type
// inconnu échoue, et il signale un greffon entré sans validation.
//
// Les annulations se rappellent dans l'ordre inverse des applications. Celles
// qui tronquent une tranche en dépendent : les défaire dans le désordre
// retirerait l'entrée d'un autre effet.
func (p *Partie) Appliquer1Effet(e Effet, ctx Contexte) (annulation func(), err error) {
	if ctx.Acteur == CampInspecteurs && (ctx.Pion < 0 || ctx.Pion >= len(p.Inspecteurs)) {
		return nil, fmt.Errorf("pion hors bornes: %d", ctx.Pion)
	}

	switch e.Type {
	case EffetDeplacer, EffetTeleporter:
		return p.placer(ctx), nil

	case EffetModifierPortee, EffetModifierMobilite, EffetRevelerTraces, EffetPartagerVue:
		return p.activer(e, ctx), nil

	case EffetBloquerCase:
		return p.alterer(&p.Barrages, ctx.Case, e.Duree), nil

	case EffetOuvrirCase:
		return p.alterer(&p.Ouvertures, ctx.Case, e.Duree), nil

	case EffetRevelerPosition:
		precedent := p.Fugitif.Visible
		p.Fugitif.Visible = true
		return func() { p.Fugitif.Visible = precedent }, nil

	case EffetMarquerScene:
		return p.marquerScene(e, ctx), nil

	case EffetAnnulerRevelation:
		precedent := p.Fugitif.SilenceAchete
		p.Fugitif.SilenceAchete = true
		return func() { p.Fugitif.SilenceAchete = precedent }, nil

	case EffetCouterResistance:
		return p.ajusterResistance(-e.Valeur), nil

	case EffetRendreResistance:
		return p.ajusterResistance(e.Valeur), nil

	case EffetEffacerTraces:
		return p.effacerTraces(e.Duree), nil

	case EffetFermerZone:
		return p.basculerZone(ctx.Zone, true), nil

	case EffetOuvrirZone:
		return p.basculerZone(ctx.Zone, false), nil

	case EffetDifferer:
		return p.mettreEnAttente(e, ctx), nil

	case EffetFinPartie:
		return p.forcerFin(e), nil
	}

	return nil, fmt.Errorf("effet inconnu: %s", e.Type)
}

// placer pose le pion visé sur la case du contexte.
//
// deplacer et teleporter aboutissent au même geste, et c'est voulu : ce qui les
// sépare est la légalité, vérifiée en amont — l'un exige une case atteignable,
// l'autre non. Les distinguer ici reviendrait à réimplémenter la règle.
func (p *Partie) placer(ctx Contexte) func() {
	if ctx.Acteur == CampFugitif {
		precedente := p.Fugitif.Position
		p.Fugitif.Position = ctx.Case
		return func() { p.Fugitif.Position = precedente }
	}
	precedente := p.Inspecteurs[ctx.Pion].Position
	p.Inspecteurs[ctx.Pion].Position = ctx.Case
	return func() { p.Inspecteurs[ctx.Pion].Position = precedente }
}

// activer met un effet dans la liste des effets en cours.
//
// Une durée de 1 vaut le tour courant seulement, d'où l'échéance à Tour+Duree-1.
// Une durée absente donne un effet permanent, ce dont la capacité passive du
// Traqueur a besoin.
func (p *Partie) activer(e Effet, ctx Contexte) func() {
	echeance := 0
	if e.Duree > 0 {
		echeance = p.Tour + e.Duree - 1
	}
	p.EffetsActifs = append(p.EffetsActifs, EffetActif{Effet: e, Contexte: ctx, Echeance: echeance})
	return func() { p.EffetsActifs = tronquer(p.EffetsActifs) }
}

// tronquer retire la dernière entrée d'une tranche, et rend nil quand elle se
// vide.
//
// La remise à nil n'est pas cosmétique : une tranche vide se sérialise en []
// là où nil donne null. Sans elle, appliquer puis annuler un effet laisserait
// un état qui se relit différemment, et le rejeu du journal cesserait d'être
// identique octet pour octet.
func tronquer[T any](s []T) []T {
	if s = s[:len(s)-1]; len(s) == 0 {
		return nil
	}
	return s
}

// alterer inscrit une case dans une couche d'altération du terrain, jusqu'au
// tour d'expiration. Une case déjà inscrite voit sa date remplacée, et
// l'annulation rend l'ancienne plutôt que d'effacer.
func (p *Partie) alterer(couche *map[Position]int, pos Position, duree int) func() {
	etaitNulle := *couche == nil
	if etaitNulle {
		*couche = make(map[Position]int)
	}
	precedent, existait := (*couche)[pos]
	(*couche)[pos] = p.Tour + duree
	return func() {
		if existait {
			(*couche)[pos] = precedent
			return
		}
		delete(*couche, pos)
		if etaitNulle {
			*couche = nil
		}
	}
}

// marquerScene inscrit un lieu de meurtre, à la case du contexte ou à celle du
// fugitif selon la cible.
//
// Contrairement à reveler_position, qui ne vaut qu'un tour, la scène reste :
// c'est ce que le fugitif achète en payant, et ce sur quoi les inspecteurs
// devront parier longtemps après.
func (p *Partie) marquerScene(e Effet, ctx Contexte) func() {
	pos := ctx.Case
	if e.Cible == CibleFugitif {
		pos = p.Fugitif.Position
	}
	p.Scenes = append(p.Scenes, Scene{Position: pos, Tour: p.Tour})
	return func() { p.Scenes = tronquer(p.Scenes) }
}

// ajusterResistance ajoute un delta à la résistance du fugitif, plancher à
// zéro. L'annulation restitue la valeur exacte, pas le delta inverse : le
// plancher rendrait les deux différents.
func (p *Partie) ajusterResistance(delta int) func() {
	precedente := p.Fugitif.Resistance
	p.Fugitif.Resistance += delta
	if p.Fugitif.Resistance < 0 {
		p.Fugitif.Resistance = 0
	}
	return func() { p.Fugitif.Resistance = precedente }
}

// effacerTraces supprime les traces de moins de duree tours.
func (p *Partie) effacerTraces(duree int) func() {
	effacees := make(map[Position]Trace)
	for pos, t := range p.Traces {
		if p.Tour-t.Tour < duree {
			effacees[pos] = t
			delete(p.Traces, pos)
		}
	}
	return func() {
		for pos, t := range effacees {
			p.Traces[pos] = t
		}
	}
}

// basculerZone ferme ou rouvre un point d'extraction.
//
// L'annulation d'une réouverture réinsère la zone à son ancien rang : la
// tranche est parcourue en ordre par la vue et par l'IA, et une permutation
// suffirait à faire diverger un rejeu.
func (p *Partie) basculerZone(zone int, fermer bool) func() {
	rang := -1
	for i, z := range p.ZonesFermees {
		if z == zone {
			rang = i
			break
		}
	}

	if fermer {
		if rang >= 0 {
			return func() {}
		}
		p.ZonesFermees = append(p.ZonesFermees, zone)
		return func() { p.ZonesFermees = tronquer(p.ZonesFermees) }
	}

	if rang < 0 {
		return func() {}
	}
	p.ZonesFermees = append(p.ZonesFermees[:rang], p.ZonesFermees[rang+1:]...)
	return func() {
		p.ZonesFermees = append(p.ZonesFermees, 0)
		copy(p.ZonesFermees[rang+1:], p.ZonesFermees[rang:])
		p.ZonesFermees[rang] = zone
	}
}

// mettreEnAttente inscrit les effets d'un differer dans la file, pour le tour
// d'échéance. L'annulation retire l'entrée, pas les effets : ils n'ont pas
// encore eu lieu.
func (p *Partie) mettreEnAttente(e Effet, ctx Contexte) func() {
	p.EffetsEnAttente = append(p.EffetsEnAttente, EffetEnAttente{
		Effets:   e.Puis,
		Tour:     p.Tour + e.Duree,
		Annonce:  e.Annonce,
		Contexte: ctx,
	})
	return func() { p.EffetsEnAttente = tronquer(p.EffetsEnAttente) }
}

// forcerFin termine la partie au profit du camp visé. C'est le seul moyen qu'un
// greffon conclue sans que le noyau connaisse sa condition de victoire.
func (p *Partie) forcerFin(e Effet) func() {
	precedent := p.FinForcee
	vainqueur := CampInspecteurs
	if e.Cible == CibleFugitif {
		vainqueur = CampFugitif
	}
	p.FinForcee = &Resultat{Vainqueur: vainqueur, Motif: MotifGreffon, Tour: p.Tour}
	return func() { p.FinForcee = precedent }
}

// PorteeDe renvoie la portée de vue d'un inspecteur, effets en cours compris.
func (p *Partie) PorteeDe(pion int) int {
	portee := p.Parametres.Portee
	for _, a := range p.EffetsActifs {
		if a.Effet.Type == EffetModifierPortee && a.Vaut(p.Tour) && a.Vise(pion) {
			portee += a.Effet.Valeur
		}
	}
	return max(portee, 0)
}

// MobiliteDe renvoie le nombre de cases qu'un acteur peut franchir en un
// déplacement. Une valeur négative est légale dans le vocabulaire : à -1, le
// pion est immobilisé.
func (p *Partie) MobiliteDe(acteur Acteur, pion int) int {
	mobilite := 1
	for _, a := range p.EffetsActifs {
		if a.Effet.Type != EffetModifierMobilite || !a.Vaut(p.Tour) {
			continue
		}
		vise := a.Vise(pion)
		if acteur == CampFugitif {
			vise = a.Effet.Cible == CibleFugitif
		}
		if vise {
			mobilite += a.Effet.Valeur
		}
	}
	return max(mobilite, 0)
}

// RayonTracesDe renvoie à quelle distance un inspecteur découvre les traces.
//
// Un de base, ce qui couvre sa case et les quatre orthogonales ; le Traqueur
// porte ce rayon à deux, en permanence.
func (p *Partie) RayonTracesDe(pion int) int {
	rayon := 1
	for _, a := range p.EffetsActifs {
		if a.Effet.Type == EffetRevelerTraces && a.Vaut(p.Tour) && a.Vise(pion) {
			rayon = max(rayon, a.Effet.Rayon)
		}
	}
	return rayon
}

// VuePartageeDe renvoie les pions dont un inspecteur emprunte la vue.
func (p *Partie) VuePartageeDe(pion int) []int {
	var pions []int
	for _, a := range p.EffetsActifs {
		if a.Effet.Type == EffetPartagerVue && a.Vaut(p.Tour) && a.Contexte.Pion == pion {
			pions = append(pions, a.Contexte.AutrePion)
		}
	}
	return pions
}
