// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import "fmt"

// EffectType est le vocabulaire de modding.
//
// C'est le contrat central du projet : une capacité, une dépense de résistance
// ou un mode de jeu se décrit par composition de ces primitives, sans une ligne
// de code. Les cinq capacités de base sont écrites dans ce format dès le
// premier jour — c'est le test qui prouve que le vocabulaire suffit. Si l'une
// d'elles ne s'exprime pas ici, c'est qu'il manque une primitive, pas qu'il
// faut un cas particulier.
//
// Ajouter une primitive est une décision lourde : elle entre dans le contrat
// public et ne peut plus être retirée sans casser les plugins existants.
type EffectType string

// Le vocabulaire livré, décrit en entier dans docs/vocabulaire-effets.md.
// EffectTeleport ne sert à aucune capacité de base — la téléportation a été
// retirée des règles — et reste offert aux plugins qui voudraient la rétablir.
const (
	EffectMove           EffectType = "step"
	EffectChangeRange    EffectType = "change_range"
	EffectChangeMobility EffectType = "change_mobility"
	EffectBlockCell      EffectType = "block_cell"
	EffectOpenCell       EffectType = "open_cell"
	EffectRevealTrails   EffectType = "reveal_trails"
	EffectRevealPosition EffectType = "reveal_position"
	EffectCancelReveal   EffectType = "cancel_reveal"
	EffectCostStamina    EffectType = "cost_stamina"
	EffectRestoreStamina EffectType = "restore_stamina"
	EffectEraseTrails    EffectType = "erase_trails"
	EffectDecoyTrail     EffectType = "decoy_trail"
	EffectCloseZone      EffectType = "close_zone"
	EffectOpenZone       EffectType = "open_zone"
	EffectSealZone       EffectType = "seal_zone"
	EffectTeleport       EffectType = "teleport"
	EffectDefer          EffectType = "defer"
	EffectEndGame        EffectType = "end_game"
)

// EffectTypes énumère le vocabulaire, dans l'ordre de sa déclaration.
//
// Le chargeur de plugins en a besoin pour refuser une primitive qu'il ne sait
// pas appliquer. La list vit ici et pas là-bas : une seconde énumération dans
// un autre paquet se désynchroniserait au premier ajout, et un manifeste
// parfaitement valide serait refusé sans qu'on comprenne pourquoi. Un test de
// internal/qualite la compare aux constantes déclarées.
func EffectTypes() []EffectType {
	return []EffectType{
		EffectMove, EffectChangeRange, EffectChangeMobility,
		EffectBlockCell, EffectOpenCell, EffectRevealTrails,
		EffectRevealPosition, EffectCancelReveal,
		EffectCostStamina, EffectRestoreStamina,
		EffectEraseTrails, EffectDecoyTrail, EffectCloseZone, EffectOpenZone,
		EffectSealZone, EffectTeleport, EffectDefer, EffectEndGame,
	}
}

// EffectTypeKnown dit si une primitive fait partie du vocabulaire.
func EffectTypeKnown(t EffectType) bool {
	for _, connu := range EffectTypes() {
		if connu == t {
			return true
		}
	}
	return false
}

// Target désigne ce sur quoi un effet s'applique.
type Target string

// TargetCurrentPiece est le cas ordinaire ; les autres valeurs supposent que le
// contexte porte le pion, la case ou la zone visée.
const (
	TargetCurrentPiece Target = "current_piece"
	TargetAllPieces    Target = "all_pieces"
	TargetOtherPiece   Target = "other_piece"
	TargetFugitive     Target = "fugitive"
	TargetCell         Target = "cell"
	TargetZone         Target = "zone"
)

// Targets énumère les cibles du vocabulaire, pour la même raison que
// EffectTypes.
func Targets() []Target {
	return []Target{
		TargetCurrentPiece, TargetAllPieces, TargetOtherPiece,
		TargetFugitive, TargetCell, TargetZone,
	}
}

// TargetKnown dit si une cible fait partie du vocabulaire.
func TargetKnown(c Target) bool {
	for _, connue := range Targets() {
		if connue == c {
			return true
		}
	}
	return false
}

// Effect est une primitive paramétrée. Les champs inutiles restent à zéro : un
// enregistrement plat se sérialise et se journalise sans effort, contrairement
// à une hiérarchie de types.
type Effect struct {
	Type     EffectType `toml:"type" json:"type"`
	Target   Target     `toml:"target" json:"target,omitempty"`
	Value    int        `toml:"value" json:"value,omitempty"`
	Duration int        `toml:"duration" json:"duration,omitempty"`
	Radius   int        `toml:"radius" json:"radius,omitempty"`

	// Announced et Then n'ont de sens que pour EffectDefer.
	//
	// Un effet différé annoncé figure dans la View des deux camps, et c'est
	// tout son intérêt : un mur qui apparaît sans prévenir transformerait un
	// plan raisonné en coup de dé et rendrait la carte de croyance inutile.
	//
	// Un differer imbriqué dans un differer est refusé au chargement : deux
	// durées s'additionnent, donc ça n'ajoute rien, et ça permettrait des
	// chaînes qu'aucune annulation ne saurait dérouler.
	Announced bool     `toml:"announced" json:"announced,omitempty"`
	Then      []Effect `toml:"then" json:"then,omitempty"`

	// Mode dit comment Value s'applique. Absent vaut ModeAdd, donc tout
	// manifeste écrit avant ce champ garde son sens.
	//
	// Un champ plutôt qu'une primitive de plus : il vaut pour les dix-neuf d'un
	// coup, là où un change_range_multiply aurait laissé change_mobility
	// entier. Un champ nommé plutôt qu'un booléen : une troisième forme se
	// déclare en ajoutant une valeur, sans toucher au contrat.
	Mode ValueMode `toml:"mode" json:"mode,omitempty"`
}

// ValueMode dit comment la valeur d'un effet s'applique à ce qu'elle modifie.
type ValueMode string

// Les deux formes. ModeAdd est la valeur nulle, donc le comportement par
// défaut.
//
// La multiplication existe parce que la portée, la durée et le rayon du noyau
// dérivent désormais de la taille du plateau : une valeur absolue ne peut plus
// dire une intention relative. Le Guetteur déclarait « portée 8 », juste tant
// qu'un seul préréglage existait, et qui triplait la portée d'un Quartier.
const (
	ModeAdd      ValueMode = ""
	ModeMultiply ValueMode = "multiply"
)

// ValueModes énumère les formes, pour la même raison que EffectTypes.
func ValueModes() []ValueMode { return []ValueMode{ModeAdd, ModeMultiply} }

// ValueModeKnown dit si une forme fait partie du vocabulaire.
func ValueModeKnown(m ValueMode) bool {
	for _, connue := range ValueModes() {
		if connue == m {
			return true
		}
	}
	return false
}

// apply combine une valeur d'effet avec la base qu'elle modifie.
func (e Effect) apply(base int) int {
	if e.Mode == ModeMultiply {
		return base * e.Value
	}
	return base + e.Value
}

// PendingEffect est une entrée de la file des effets différés.
//
// Résolue en fin de tour, avant le test de fin de partie. L'annulation défait
// la mise en file, pas l'effet : annuler le tour où le differer a été posé le
// retire de la file.
type PendingEffect struct {
	Effects       []Effect      `json:"effects"`
	Turn          int           `json:"turn"`
	Announced     bool          `json:"announced"`
	EffectContext EffectContext `json:"context"`
}

// ActiveEffect est un effet qui dure, avec le tour où il cesse.
//
// Portée, mobilité et rayon de détection ne sont pas stockés dans les pions
// mais recalculés à partir de cette liste. Un bonus est dérivable de ce qui
// l'a produit ; le figer dans Inspector en ferait un cache que rien ne
// réconcilie, et fermerait la porte aux capacités qu'un plugin invente.
type ActiveEffect struct {
	Effect        Effect        `json:"effect"`
	EffectContext EffectContext `json:"context"`

	// Echeance est le dernier tour où l'effet vaut. Zéro pour un effet
	// permanent — la capacité passive du Traqueur en est une.
	Echeance int `json:"due_turn"`
}

// AppliesAt dit si l'effet court encore au tour donné.
func (a ActiveEffect) AppliesAt(tour int) bool {
	return a.Echeance == 0 || tour <= a.Echeance
}

// Aims dit si l'effet porte sur un pion d'inspecteur donné.
func (a ActiveEffect) Aims(pion int) bool {
	switch a.Effect.Target {
	case TargetAllPieces:
		return true
	case TargetCurrentPiece:
		return a.EffectContext.Piece == pion
	case TargetOtherPiece:
		return a.EffectContext.AutrePion == pion
	}
	return false
}

// EffectContext est ce dont un effet dispose pour s'appliquer. Il ne donne pas accès
// à la Game entière : un plugin ne doit pas pouvoir lire la zone scellée du
// fugitif ni écrire dans le journal.
type EffectContext struct {
	Side  Side     `json:"side"`
	Piece int      `json:"piece"`
	Case  Position `json:"cell"`
	Zone  int      `json:"zone"`

	// Toward est la seconde case d'un effet qui en relie deux. Le leurre est le
	// seul cas livré : il pose sa trace sur Case et la fait pointer vers
	// Toward, exactement comme un pas réel produit la sienne.
	Toward Position `json:"toward"`

	// AutrePion est le second pion d'un effet qui en relie deux, celui que
	// vise other_piece. Aucune capacité livrée ne l'emploie depuis que le Chef
	// force une révélation au lieu de partager une vue : la cible reste au
	// contrat pour les plugins, et Aims la sait lire.
	AutrePion int `json:"autre_pion"`
}

// ApplyOneEffect exécute un effet et renvoie de quoi le défaire.
//
// Le retour n'est pas optionnel : Game.Undo doit rester praticable, sinon
// l'IA ne peut plus explorer et le rejeu du journal diverge dès qu'un plugin
// est actif.
//
// Rien n'est validé ici. La légalité d'un coup relève de LegalMoves, et la
// cohérence d'un effet de son manifeste, contrôlée au chargement. Seul un type
// inconnu échoue, et il signale un plugin entré sans validation.
//
// Les annulations se rappellent dans l'ordre inverse des applications. Celles
// qui tronquent une tranche en dépendent : les défaire dans le désordre
// retirerait l'entrée d'un autre effet.
func (p *Game) ApplyOneEffect(e Effect, ctx EffectContext) (annulation func(), err error) {
	switch e.Type {
	case EffectMove, EffectTeleport:
		// Seuls les effets qui indexent réellement les pions ont à vérifier
		// l'indice. Le contrôler pour tous refusait un fermer_zone déclenché
		// par le jeu lui-même, dont le contexte n'a pas de pion à désigner.
		if ctx.Side == SideInspectors && (ctx.Piece < 0 || ctx.Piece >= len(p.Inspectors)) {
			return nil, fmt.Errorf("pion hors bornes: %d", ctx.Piece)
		}
		return p.place(ctx), nil

	case EffectChangeRange, EffectChangeMobility, EffectRevealTrails:
		return p.activate(e, ctx), nil

	case EffectBlockCell:
		return p.alter(&p.Roadblocks, ctx.Case, e.Duration), nil

	case EffectOpenCell:
		return p.alter(&p.Openings, ctx.Case, e.Duration), nil

	// Le silence s'achète contre être trouvé, pas contre se montrer : une
	// révélation provoquée par les inspecteurs se neutralise, celle que le
	// fugitif s'inflige jamais. Le camp du contexte suffit à les distinguer,
	// et la règle couvre les cas qu'un plugin inventera.
	case EffectRevealPosition:
		if ctx.Side == SideInspectors && p.Fugitive.SilenceBought {
			return func() {}, nil
		}
		precedent := p.Fugitive.Visible
		p.Fugitive.Visible = true
		return func() { p.Fugitive.Visible = precedent }, nil

	// L'achat se date au passage : le drapeau retombe en fin de tour, la date
	// reste, et c'est elle que les inspecteurs lisent au tour suivant.
	case EffectCancelReveal:
		achete, quand := p.Fugitive.SilenceBought, p.Fugitive.SilenceTurn
		p.Fugitive.SilenceBought, p.Fugitive.SilenceTurn = true, p.Turn
		return func() {
			p.Fugitive.SilenceBought, p.Fugitive.SilenceTurn = achete, quand
		}, nil

	case EffectCostStamina:
		return p.adjustStamina(-e.Value), nil

	case EffectRestoreStamina:
		return p.adjustStamina(e.Value), nil

	case EffectEraseTrails:
		return p.wipeTrails(e.Duration), nil

	case EffectDecoyTrail:
		return p.armDecoy(ctx), nil

	case EffectCloseZone:
		return p.toggleZone(ctx.Zone, true), nil

	case EffectOpenZone:
		return p.toggleZone(ctx.Zone, false), nil

	// Écrire la zone scellée ne la fait pas fuiter : un plugin la remplace
	// sans jamais pouvoir la lire, et ViewFor continue de la retenir côté
	// inspecteurs.
	case EffectSealZone:
		precedente := p.Fugitive.SealedZone
		p.Fugitive.SealedZone = ctx.Zone
		return func() { p.Fugitive.SealedZone = precedente }, nil

	case EffectDefer:
		return p.defer_(e, ctx), nil

	case EffectEndGame:
		return p.forceEnd(e), nil
	}

	return nil, fmt.Errorf("effet inconnu: %s", e.Type)
}

// place pose le pion visé sur la case du contexte.
//
// step et teleporter aboutissent au même geste, et c'est voulu : ce qui les
// sépare est la légalité, vérifiée en amont — l'un exige une case atteignable,
// l'autre non. Les distinguer ici reviendrait à réimplémenter la règle.
func (p *Game) place(ctx EffectContext) func() {
	if ctx.Side == SideFugitive {
		precedente := p.Fugitive.Position
		p.Fugitive.Position = ctx.Case
		return func() { p.Fugitive.Position = precedente }
	}
	precedente := p.Inspectors[ctx.Piece].Position
	p.Inspectors[ctx.Piece].Position = ctx.Case
	return func() { p.Inspectors[ctx.Piece].Position = precedente }
}

// activate met un effet dans la liste des effets en cours.
//
// Une durée de 1 vaut le tour courant seulement, d'où l'échéance à Turn+Duration-1.
// Une durée absente donne un effet permanent, ce dont la capacité passive du
// Traqueur a besoin.
func (p *Game) activate(e Effect, ctx EffectContext) func() {
	echeance := 0
	if e.Duration > 0 {
		echeance = p.Turn + e.Duration - 1
	}
	p.ActiveEffects = append(p.ActiveEffects, ActiveEffect{Effect: e, EffectContext: ctx, Echeance: echeance})
	return func() { p.ActiveEffects = truncate(p.ActiveEffects) }
}

// truncate retire la dernière entrée d'une tranche, et rend nil quand elle se
// vide.
func truncate[T any](s []T) []T { return removeAt(s, len(s)-1) }

// removeAt retire l'entrée de rang donné, et rend nil quand la tranche se vide.
//
// La remise à nil n'est pas cosmétique : une tranche vide se sérialise en []
// là où nil donne null. Sans elle, appliquer puis annuler un effet laisserait
// un état qui se relit différemment, et le rejeu du journal cesserait d'être
// identique octet pour octet.
//
// Point unique parce que le geste s'oublie : truncate le faisait depuis le
// début, et la réouverture de zone, qui retire au milieu, ne le faisait pas.
func removeAt[T any](s []T, i int) []T {
	if s = append(s[:i], s[i+1:]...); len(s) == 0 {
		return nil
	}
	return s
}

// setIn inscrit une valeur dans une map de l'état et rend de quoi défaire
// exactement, la map comprise.
//
// Trois choses à rendre, et la troisième est celle qu'on oublie : la valeur
// précédente si la clé existait, l'absence de clé sinon, et la nullité de la map
// si c'est cet appel qui l'a créée. Une map vide n'est pas une map nulle pour
// reflect.DeepEqual ni pour JSON — {} contre null —, si bien qu'appliquer puis
// annuler laissait un état équivalent au sens des règles et différent au sens de
// la comparaison. L'IA compare des états pour reconnaître une position déjà
// explorée : elle aurait refait un travail qu'elle croyait éviter, sans qu'aucune
// partie soit faussée.
//
// Extraite parce que le geste était écrit deux fois — les traces, les
// altérations de terrain — et manquait au troisième endroit. Deux copies ne
// déclenchent aucune alarme, et c'est exactement le seuil où la duplication
// coûte : assez fréquente pour être un motif, pas assez pour qu'on la nomme.
func setIn[K comparable, V any](m *map[K]V, cle K, valeur V) func() {
	etaitNulle := *m == nil
	if etaitNulle {
		*m = make(map[K]V)
	}
	precedente, existait := (*m)[cle]
	(*m)[cle] = valeur

	return func() {
		if existait {
			(*m)[cle] = precedente
			return
		}
		delete(*m, cle)
		if etaitNulle {
			*m = nil
		}
	}
}

// alter inscrit une case dans une couche d'altération du terrain, jusqu'au
// dernier tour où elle vaut. Une case déjà inscrite voit sa date remplacée, et
// l'annulation rend l'ancienne plutôt que d'effacer.
//
// La date se compte comme celle d'un effet actif — dernier tour inclus, d'où le
// moins un. Elle valait un tour de plus, et personne ne s'en apercevait :
// aucune lecture ne la consultait, un barrage tenait jusqu'à la fin de la
// partie. C'est expireTerrain qui la relit désormais.
func (p *Game) alter(couche *map[Position]int, pos Position, duree int) func() {
	return setIn(couche, pos, p.Turn+duree-1)
}

// armDecoy retient la trace que le fugitif substituera aux siennes.
//
// Retenue et non posée : les traces s'inscrivent à la résolution, après la
// phase du fugitif, et une trace posée maintenant coexisterait avec les vraies
// — ce que docs/regles.md §8 interdit, les traces d'un tour étant toutes vraies
// ou toutes fausses.
func (p *Game) armDecoy(ctx EffectContext) func() {
	precedent := p.Fugitive.Decoy
	p.Fugitive.Decoy = &Decoy{At: ctx.Case, Toward: ctx.Toward}
	return func() { p.Fugitive.Decoy = precedent }
}

// adjustStamina ajoute un delta à la résistance du fugitif, plancher à
// zéro. L'annulation restitue la valeur exacte, pas le delta inverse : le
// plancher rendrait les deux différents.
func (p *Game) adjustStamina(delta int) func() {
	precedente := p.Fugitive.Stamina
	p.Fugitive.Stamina += delta
	if p.Fugitive.Stamina < 0 {
		p.Fugitive.Stamina = 0
	}
	return func() { p.Fugitive.Stamina = precedente }
}

// wipeTrails supprime les traces de moins de duree tours.
func (p *Game) wipeTrails(duree int) func() {
	effacees := make(map[Position]Trail)
	for pos, t := range p.Trails {
		if p.Turn-t.Turn < duree {
			effacees[pos] = t
			delete(p.Trails, pos)
		}
	}
	return func() {
		for pos, t := range effacees {
			p.Trails[pos] = t
		}
	}
}

// toggleZone ferme ou rouvre un point d'extraction.
//
// L'annulation d'une réouverture réinsère la zone à son ancien rang : la
// tranche est parcourue en ordre par la vue et par l'IA, et une permutation
// suffirait à faire diverger un rejeu.
func (p *Game) toggleZone(zone int, fermer bool) func() {
	rang := -1
	for i, z := range p.ClosedZones {
		if z == zone {
			rang = i
			break
		}
	}

	if fermer {
		if rang >= 0 {
			return func() {}
		}
		p.ClosedZones = append(p.ClosedZones, zone)
		return func() { p.ClosedZones = truncate(p.ClosedZones) }
	}

	if rang < 0 {
		return func() {}
	}
	p.ClosedZones = removeAt(p.ClosedZones, rang)
	return func() {
		p.ClosedZones = append(p.ClosedZones, 0)
		copy(p.ClosedZones[rang+1:], p.ClosedZones[rang:])
		p.ClosedZones[rang] = zone
	}
}

// defer_ inscrit les effets d'un differer dans la file, pour le tour
// d'échéance. L'annulation retire l'entrée, pas les effets : ils n'ont pas
// encore eu lieu.
func (p *Game) defer_(e Effect, ctx EffectContext) func() {
	p.PendingEffects = append(p.PendingEffects, PendingEffect{
		Effects:       e.Then,
		Turn:          p.Turn + e.Duration,
		Announced:     e.Announced,
		EffectContext: ctx,
	})
	return func() { p.PendingEffects = truncate(p.PendingEffects) }
}

// forceEnd termine la partie au profit du camp visé. C'est le seul moyen qu'un
// plugin conclue sans que le noyau connaisse sa condition de victoire.
func (p *Game) forceEnd(e Effect) func() {
	precedent := p.ForcedOutcome
	vainqueur := SideInspectors
	if e.Target == TargetFugitive {
		vainqueur = SideFugitive
	}
	p.ForcedOutcome = &Outcome{Winner: vainqueur, Reason: OutcomePlugin, Turn: p.Turn}
	return func() { p.ForcedOutcome = precedent }
}

// RangeOf renvoie la portée de vue d'un inspecteur, effets en cours compris.
func (p *Game) RangeOf(pion int) int {
	portee := p.Settings.Range
	for _, a := range p.ActiveEffects {
		if a.Effect.Type == EffectChangeRange && a.AppliesAt(p.Turn) && a.Aims(pion) {
			portee = a.Effect.apply(portee)
		}
	}
	return max(portee, 0)
}

// MobilityOf renvoie le nombre de cases qu'un acteur peut franchir en un
// déplacement. Une valeur négative est légale dans le vocabulaire : à -1, le
// pion est immobilisé.
func (p *Game) MobilityOf(acteur Side, pion int) int {
	mobilite := 1
	for _, a := range p.ActiveEffects {
		if a.Effect.Type != EffectChangeMobility || !a.AppliesAt(p.Turn) {
			continue
		}
		vise := a.Aims(pion)
		if acteur == SideFugitive {
			vise = a.Effect.Target == TargetFugitive
		}
		if vise {
			mobilite = a.Effect.apply(mobilite)
		}
	}
	return max(mobilite, 0)
}

// TrailRadiusOf renvoie à quelle distance un inspecteur découvre les traces.
//
// Un de base, ce qui couvre sa case et les quatre orthogonales ; le Traqueur
// porte ce rayon à deux, en permanence.
func (p *Game) TrailRadiusOf(pion int) int {
	rayon := 1
	for _, a := range p.ActiveEffects {
		if a.Effect.Type == EffectRevealTrails && a.AppliesAt(p.Turn) && a.Aims(pion) {
			rayon = max(rayon, a.Effect.Radius)
		}
	}
	return rayon
}
