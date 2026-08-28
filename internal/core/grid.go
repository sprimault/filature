// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import "errors"

// Bornes de la génération, toutes tirées de docs/regles.md §3.
const (
	// MinAvenueGap et MaxAvenueGap donnent l'irrégularité de la grid. Un
	// écart constant produirait un damier où toutes les fuites se valent ;
	// c'est la variation qui crée des rues longues et des recoins.
	MinAvenueGap = 3
	MaxAvenueGap = 6

	// MinStreetRatio et MaxStreetRatio bornent la part de cases praticables que
	// produit la trame. Trop peu, le fugitif n'a nulle part où aller ; trop, les
	// inspecteurs ne peuvent plus rien couvrir.
	//
	// La trame seule, et les blocs percés par-dessus n'y entrent pas. Ce sont
	// deux populations : la trame est du terrain, une case de zone ou de lieu
	// est un dispositif — publique, désignée, et que les inspecteurs occupent au
	// lieu d'avoir à la fouiller. Les agréger faisait dépendre le seuil d'une
	// composition qui varie avec la taille : dix blocs de neuf cases pèsent un
	// cinquième d'un Quartier contre un vingtième d'une Ville, si bien que le
	// petit préréglage dépassait le plafond par ses seules zones et que
	// Generate y jetait quatre-vingt-douze tirages sur cent.
	MinStreetRatio = 35
	MaxStreetRatio = 50

	// ZoneSize est le bloc d'une zone d'extraction, et StreetsPerZone le nombre
	// de cases praticables qu'il doit contenir pour qu'on puisse s'y tenir.
	ZoneSize       = 3
	StreetsPerZone = 5

	// MaxDeadEndLength borne un couloir borgne, AreaPerDeadEnd en donne
	// le nombre : une par carré de huit cases de côté.
	//
	// Une surface et non le côté du plateau. Un nombre proportionnel au côté
	// rendrait la Ville deux fois moins piégeuse qu'un Quartier sans que
	// personne ne l'ait décidé, alors que le fugitif n'y circule pas en ligne
	// droite : il tourne, revient, se terre, et ce qu'il croise dépend de
	// l'aire qu'il couvre.
	MaxDeadEndLength = 3
	AreaPerDeadEnd   = 64

	// maxAttempts borne la recherche d'un plateau valide. Un jeu de
	// paramètres qui échoue autant de fois de suite ne produira pas de plateau
	// jouable, et le dire vaut mieux que boucler.
	maxAttempts = 200
)

// Generate produit un plateau déterministe à partir d'une graine.
//
// Le générateur retente avec la graine suivante tant que validate échoue, et
// renvoie la graine effectivement utilisée : c'est elle qui part en base, pas
// celle demandée. Une partie rejouée depuis son journal retrouve donc le même
// terrain, même si la graine d'origine ne donnait rien de jouable.
func Generate(graine int64, p Settings) (*BoundedBoard, int64, error) {
	if err := p.Validate(); err != nil {
		return nil, 0, err
	}

	for essai := 0; essai < maxAttempts; essai++ {
		retenue := graine + int64(essai)
		b, trame := draw(retenue, p)
		if err := b.validate(p, trame); err == nil {
			// Après validation seulement : un plateau rejeté n'a pas à payer le
			// précalcul.
			b.vues = precomputeSight(b, b.cote)
			return b, retenue, nil
		}
	}
	return nil, 0, ErrNoPlayableBoard
}

// ErrNoPlayableBoard dit qu'aucune graine voisine n'a donné de plateau
// jouable.
//
// Distincte d'un défaut de paramètres : celle-ci dit que le tirage n'aboutit
// pas, pas que la demande est absurde. L'appelant peut vouloir réessayer
// ailleurs plutôt que de renoncer.
var ErrNoPlayableBoard = errors.New("aucun plateau jouable pour cette graine")

// draw trace un plateau complet, sans juger de sa qualité, et rend avec lui le
// nombre de cases que la trame a ouvertes.
//
// Ce compte est rendu plutôt que relu sur le plateau fini : les blocs percés
// ensuite recouvrent des cases que la trame avait déjà ouvertes, donc les
// soustraire après coup enlèverait une centaine de cases de trame sur un
// Quartier. Un champ le rendrait implicite, et faux par défaut pour tout
// appelant qui n'est pas draw ; en paramètre, celui qui valide dit ce qu'il
// mesure.
//
// Les six étapes de docs/regles.md §3, dans l'ordre : grid, îlots, perçages,
// impasses, zones, lieux. Trois d'entre elles tirent au sort, chacune sur son
// propre flux — la trame, les cours, les impasses —, ce qui permet d'en modifier
// une sans déplacer les tirages des autres. Les trois autres n'en ont pas
// besoin : le remplissage des îlots suit la trame, et la position des zones
// comme des lieux se déduit de la géométrie.
func draw(graine int64, p Settings) (*BoundedBoard, int) {
	b := &BoundedBoard{
		graine: graine,
		cote:   p.Size,
		rues:   make([]bool, p.Size*p.Size),
	}

	b.traceAvenues(NewRandom(graine, "grid"))
	b.punchCourtyards(NewRandom(graine, "courtyards"))
	b.carveDeadEnds(NewRandom(graine, "deadends"))
	trame := b.countStreets()

	b.placeZones(p.Zones)
	b.placeShelters(p)
	return b, trame
}

// placeShelters pose les lieux de ressourcement, un par quadrant.
//
// Sur les diagonales et non sur les axes : les zones se répartissent
// angulairement sur le périmètre, et deux des six tombent près des axes. Le
// rayon vise le milieu de la couronne entre le noyau et l'anneau des zones,
// avec un plancher juste au-dessus du noyau — sur le plus petit préréglage, la
// couronne n'a qu'un rayon de libre, et c'est lui.
//
// Aucun tirage : comme pour les zones, la position se déduit de la géométrie et
// le percement suit l'ordre de parcours. Un lieu placé au sort décalerait tous
// les flux qui le suivent.
func (b *BoundedBoard) placeShelters(p Settings) {
	milieu := b.cote / 2
	centre := Position{Column: milieu, Row: milieu}
	rayon := max(p.CentreRadius+1, (p.CentreRadius+milieu-2)/2)

	quadrants := []Position{
		{Column: centre.Column + rayon, Row: centre.Row + rayon},
		{Column: centre.Column + rayon, Row: centre.Row - rayon},
		{Column: centre.Column - rayon, Row: centre.Row + rayon},
		{Column: centre.Column - rayon, Row: centre.Row - rayon},
	}

	b.abris = make([]Shelter, 0, p.Shelters)
	for i := 0; i < p.Shelters && i < len(quadrants); i++ {
		coin := b.pullInside(quadrants[i])
		b.abris = append(b.abris, Shelter{Number: i, Cells: b.carveBlock(coin)})
	}
}

// traceAvenues ouvre des rues orthogonales à intervalle irrégulier.
//
// Tout est bâti au départ ; les avenues sont ce qu'on ouvre. Une grille
// d'avenues est connexe par construction, et c'est ce qui donne à la validation
// une chance d'aboutir avant les deux cents tentatives.
func (b *BoundedBoard) traceAvenues(a *Random) {
	for _, ligne := range b.axes(a) {
		for colonne := 0; colonne < b.cote; colonne++ {
			b.open(Position{Column: colonne, Row: ligne})
		}
	}
	for _, colonne := range b.axes(a) {
		for ligne := 0; ligne < b.cote; ligne++ {
			b.open(Position{Column: colonne, Row: ligne})
		}
	}
}

// axes tire les indices des avenues, du premier bord au dernier.
//
// Le bord opposé est toujours une avenue : sans lui, la bande de cases qui le
// longe n'est atteignable que par un côté, et les zones qu'on y pose deviennent
// des culs-de-sac que la validation rejette.
//
// **Le tirage s'arrête assez tôt pour que ce bord respecte l'écart minimal.**
// L'ajouter sans regarder la distance au dernier axe le collait à lui : l'écart
// final tombe alors à 2,9 cases au lieu des 3 à 6 annoncés, sur les trois
// préréglages. Une avenue de trop resserre toute la trame, et d'autant plus que
// le plateau est petit — le Quartier n'en garde que soixante-neuf tirages sur
// deux cents, contre cent soixante-six (mesuré le 27 août 2026).
func (b *BoundedBoard) axes(a *Random) []int {
	bord := b.cote - 1

	var indices []int
	for x := 0; x < bord-MinAvenueGap; x += MinAvenueGap + a.Int(MaxAvenueGap-MinAvenueGap+1) {
		indices = append(indices, x)
	}

	// L'écart qui reste jusqu'au bord n'est plus tiré : il est ce que la boucle
	// laisse, et il dépasse parfois la borne haute. C'est assumé — le rattraper
	// par une avenue de plus coûterait bien davantage, l'essai ramenant le
	// Quartier de cent soixante-six plateaux acceptés sur deux cents à
	// soixante-neuf. Un îlot un peu large en périphérie se traverse ; une trame
	// resserrée ne se corrige pas.
	return append(indices, bord)
}

// punchCourtyards ouvre des cours dans les îlots, pour casser la régularité.
//
// **Une cour donne sur la rue.** Percer une case au milieu d'un îlot créerait
// un bout de rue que personne ne peut atteindre, et la validation rejetterait
// le plateau — c'est ce qui les faisait tous échouer.
//
// Une case sur vingt-quatre est tentée : assez pour que deux quartiers ne se
// ressemblent pas, assez peu pour que le taux de rues garde de la marge sous sa
// borne haute.
func (b *BoundedBoard) punchCourtyards(a *Random) {
	tentatives := b.cote * b.cote / 24
	for i := 0; i < tentatives; i++ {
		c := Position{Column: a.Int(b.cote), Row: a.Int(b.cote)}
		if !b.IsStreet(c) && b.touchesStreet(c) {
			b.open(c)
		}
	}
}

// touchesStreet dit si une case a une voisine orthogonale praticable.
func (b *BoundedBoard) touchesStreet(p Position) bool {
	for _, d := range Orthogonales {
		if b.IsStreet(p.Step(d)) {
			return true
		}
	}
	return false
}

// carveDeadEnds ouvre des couloirs sans issue depuis une rue.
//
// Sans bords à exploiter, ce sont elles qui permettent le piégeage : un fugitif
// engagé dans une impasse n'en ressort que par où il est entré. Le Barreur y
// gagne sa raison d'être.
//
// Les départs sont énumérés puis mélangés, et non tirés case par case. Une case
// bâtie qui ne touche qu'une seule rue est rare — une centaine sur les quatre
// cent quarante et une d'un Quartier — et la tirer au hasard échouait assez
// souvent pour que l'étape entière n'ait aucun effet mesurable.
func (b *BoundedBoard) carveDeadEnds(a *Random) {
	departs := b.starts()
	Shuffle(a, departs)

	restantes := b.cote * b.cote / AreaPerDeadEnd
	for _, d := range departs {
		if restantes == 0 {
			return
		}
		// Un couloir déjà creusé a pu open une voisine de cette amorce, qui
		// déboucherait maintenant.
		if !b.carvable(d.tete, d.acces) {
			continue
		}
		b.extend(d, 1+a.Int(MaxDeadEndLength))
		restantes--
	}
}

// amorce est le départ d'un couloir borgne : une case bâtie qui ne touche
// qu'une seule rue, et cette rue.
type amorce struct {
	tete  Position
	acces Position
}

// starts énumère les départs possibles, ligne par ligne puis colonne par
// colonne.
//
// L'ordre du parcours est ce qui rend le mélange reproductible : deux rejeux de
// la même graine doivent mélanger la même liste.
func (b *BoundedBoard) starts() []amorce {
	var list []amorce
	for ligne := 0; ligne < b.cote; ligne++ {
		for colonne := 0; colonne < b.cote; colonne++ {
			p := Position{Column: colonne, Row: ligne}
			for _, d := range Orthogonales {
				if v := p.Step(d); b.IsStreet(v) && b.carvable(p, v) {
					list = append(list, amorce{tete: p, acces: v})
					break
				}
			}
		}
	}
	return list
}

// extend creuse tout droit depuis une amorce, et s'arrête dès que la case
// suivante déboucherait.
func (b *BoundedBoard) extend(d amorce, longueur int) {
	// Le second retour ne peut pas être faux : une amorce et son accès sont
	// voisins orthogonaux par construction.
	direction, _ := DirectionTo(d.acces, d.tete)

	b.open(d.tete)
	precedente, tete := d.tete, d.tete.Step(direction)

	for pas := 1; pas < longueur && b.carvable(tete, precedente); pas++ {
		b.open(tete)
		precedente, tete = tete, tete.Step(direction)
	}
}

// carvable dit si une case peut extend un couloir borgne arrivant de
// precedente.
//
// Le refus qui compte est le dernier : une case voisine d'une rue autre que
// celle d'où l'on vient débouche, et un couloir qui débouche n'est plus une
// impasse.
func (b *BoundedBoard) carvable(p, precedente Position) bool {
	if !b.inside(p) || b.IsStreet(p) {
		return false
	}
	for _, d := range Orthogonales {
		if v := p.Step(d); v != precedente && b.IsStreet(v) {
			return false
		}
	}
	return true
}

// placeZones place les points d'extraction en périphérie, régulièrement
// espacés.
//
// Le tour du plateau est découpé en autant de segments qu'il y a de zones, et
// chacune se pose au milieu du sien. Pas de tirage : deux zones voisines
// seraient couvrables par un seul inspecteur, ce qui viderait de son sens
// l'idée d'en avoir plus que d'inspecteurs.
func (b *BoundedBoard) placeZones(combien int) {
	perimetre := 4 * (b.cote - 1)
	b.zones = make([]Zone, 0, combien)

	for i := 0; i < combien; i++ {
		centre := b.pullInside(b.onPerimeter(i * perimetre / combien))
		b.zones = append(b.zones, b.carveZone(i, centre))
	}
}

// onPerimeter convertit une distance parcourue sur le pourtour en position.
func (b *BoundedBoard) onPerimeter(pas int) Position {
	cote := b.cote - 1
	switch {
	case pas < cote:
		return Position{Column: pas, Row: 0}
	case pas < 2*cote:
		return Position{Column: cote, Row: pas - cote}
	case pas < 3*cote:
		return Position{Column: 3*cote - pas, Row: cote}
	default:
		return Position{Column: 0, Row: 4*cote - pas}
	}
}

// pullInside ramène un point du bord assez à l'intérieur pour qu'un bloc de
// zone y tienne entièrement.
func (b *BoundedBoard) pullInside(p Position) Position {
	marge := ZoneSize / 2
	return Position{
		Column: min(max(p.Column, marge), b.cote-1-marge),
		Row:    min(max(p.Row, marge), b.cote-1-marge),
	}
}

// carveZone découpe le bloc d'une zone et l'ouvre assez pour qu'on puisse s'y
// tenir.
//
// Les cases sont ouvertes dans l'ordre de parcours du bloc, jamais au hasard :
// une zone est un lieu du plateau, et deux rejeux de la même graine doivent la
// trouver identique.
func (b *BoundedBoard) carveZone(numero int, centre Position) Zone {
	return Zone{Number: numero, Cells: b.carveBlock(centre)}
}

// carveBlock découpe un bloc et l'ouvre assez pour qu'on puisse s'y tenir.
//
// Partagé par les zones et les lieux de ressourcement, et il le faut : deux
// fonctions de percement finiraient par diverger sur le seuil, et c'est
// l'identité de format qui fait qu'un joueur n'a rien de neuf à apprendre.
//
// Les cases sont ouvertes dans l'ordre de parcours du bloc, jamais au hasard :
// un bloc est un lieu du plateau, et deux rejeux de la même graine doivent le
// trouver identique.
func (b *BoundedBoard) carveBlock(centre Position) []Position {
	marge := ZoneSize / 2
	cases := make([]Position, 0, ZoneSize*ZoneSize)

	ouvertes := 0
	for ligne := centre.Row - marge; ligne <= centre.Row+marge; ligne++ {
		for colonne := centre.Column - marge; colonne <= centre.Column+marge; colonne++ {
			c := Position{Column: colonne, Row: ligne}
			cases = append(cases, c)

			if b.IsStreet(c) {
				ouvertes++
				continue
			}
			if ouvertes < StreetsPerZone {
				b.open(c)
				ouvertes++
			}
		}
	}
	return cases
}

// open rend une case praticable, en ignorant ce qui est hors du plateau.
func (b *BoundedBoard) open(p Position) {
	if p.Column < 0 || p.Row < 0 || p.Column >= b.cote || p.Row >= b.cote {
		return
	}
	b.rues[p.Row*b.cote+p.Column] = true
}
