// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import "errors"

// Bornes de la génération, toutes tirées de docs/regles.md §3.
const (
	// EcartAvenueMin et EcartAvenueMax donnent l'irrégularité de la trame. Un
	// écart constant produirait un damier où toutes les fuites se valent ;
	// c'est la variation qui crée des rues longues et des recoins.
	EcartAvenueMin = 3
	EcartAvenueMax = 6

	// TauxRuesMin et TauxRuesMax bornent la part de cases praticables. Trop
	// peu, le fugitif n'a nulle part où aller ; trop, les inspecteurs ne
	// peuvent plus rien couvrir.
	TauxRuesMin = 35
	TauxRuesMax = 50

	// CoteZone est le bloc d'une zone d'extraction, et RuesParZone le nombre
	// de cases praticables qu'il doit contenir pour qu'on puisse s'y tenir.
	CoteZone    = 3
	RuesParZone = 5

	// LongueurImpasseMax borne un couloir borgne, SurfaceParImpasse en donne
	// le nombre : une par carré de huit cases de côté.
	//
	// Une surface et non le côté du plateau. Un nombre proportionnel au côté
	// rendrait la Ville deux fois moins piégeuse qu'un Quartier sans que
	// personne ne l'ait décidé, alors que le fugitif n'y circule pas en ligne
	// droite : il tourne, revient, se terre, et ce qu'il croise dépend de
	// l'aire qu'il couvre.
	LongueurImpasseMax = 3
	SurfaceParImpasse  = 64

	// tentativesMax borne la recherche d'un plateau valide. Un jeu de
	// paramètres qui échoue autant de fois de suite ne produira pas de plateau
	// jouable, et le dire vaut mieux que boucler.
	tentativesMax = 200
)

// Generer produit un plateau déterministe à partir d'une graine.
//
// Le générateur retente avec la graine suivante tant que valider échoue, et
// renvoie la graine effectivement utilisée : c'est elle qui part en base, pas
// celle demandée. Une partie rejouée depuis son journal retrouve donc le même
// terrain, même si la graine d'origine ne donnait rien de jouable.
func Generer(graine int64, p Parametres) (*PlateauBorne, int64, error) {
	if err := p.Valider(); err != nil {
		return nil, 0, err
	}

	for essai := 0; essai < tentativesMax; essai++ {
		retenue := graine + int64(essai)
		b := dessiner(retenue, p)
		if err := b.valider(p); err == nil {
			return b, retenue, nil
		}
	}
	return nil, 0, ErrPlateauIntrouvable
}

// ErrPlateauIntrouvable dit qu'aucune graine voisine n'a donné de plateau
// jouable.
//
// Distincte d'un défaut de paramètres : celle-ci dit que le tirage n'aboutit
// pas, pas que la demande est absurde. L'appelant peut vouloir réessayer
// ailleurs plutôt que de renoncer.
var ErrPlateauIntrouvable = errors.New("aucun plateau jouable pour cette graine")

// dessiner trace un plateau complet, sans juger de sa qualité.
//
// Les cinq étapes de docs/regles.md §3, dans l'ordre : trame, îlots, perçages,
// impasses, zones. Chacune tire sur son propre flux, ce qui permet d'en
// modifier une sans déplacer les tirages des autres.
func dessiner(graine int64, p Parametres) *PlateauBorne {
	b := &PlateauBorne{
		graine: graine,
		cote:   p.Cote,
		rues:   make([]bool, p.Cote*p.Cote),
	}

	b.tracerAvenues(NouvelAlea(graine, "trame"))
	b.percerCours(NouvelAlea(graine, "percages"))
	b.creuserImpasses(NouvelAlea(graine, "impasses"))
	b.poserZones(p.Zones)
	return b
}

// tracerAvenues ouvre des rues orthogonales à intervalle irrégulier.
//
// Tout est bâti au départ ; les avenues sont ce qu'on ouvre. Une grille
// d'avenues est connexe par construction, et c'est ce qui donne à la validation
// une chance d'aboutir avant les deux cents tentatives.
func (b *PlateauBorne) tracerAvenues(a *Alea) {
	for _, ligne := range b.axes(a) {
		for colonne := 0; colonne < b.cote; colonne++ {
			b.ouvrir(Position{Colonne: colonne, Ligne: ligne})
		}
	}
	for _, colonne := range b.axes(a) {
		for ligne := 0; ligne < b.cote; ligne++ {
			b.ouvrir(Position{Colonne: colonne, Ligne: ligne})
		}
	}
}

// axes tire les indices des avenues, du premier bord au dernier.
//
// Le dernier axe est ramené sur le bord opposé : sans lui, la bande de cases
// qui le longe n'est atteignable que par un côté, et les zones qu'on y pose
// deviennent des culs-de-sac que la validation rejette.
func (b *PlateauBorne) axes(a *Alea) []int {
	var indices []int
	for x := 0; x < b.cote-1; x += EcartAvenueMin + a.Entier(EcartAvenueMax-EcartAvenueMin+1) {
		indices = append(indices, x)
	}
	return append(indices, b.cote-1)
}

// percerCours ouvre des cours dans les îlots, pour casser la régularité.
//
// **Une cour donne sur la rue.** Percer une case au milieu d'un îlot créerait
// un bout de rue que personne ne peut atteindre, et la validation rejetterait
// le plateau — c'est ce qui les faisait tous échouer.
//
// Une case sur vingt-quatre est tentée : assez pour que deux quartiers ne se
// ressemblent pas, assez peu pour que le taux de rues garde de la marge sous sa
// borne haute.
func (b *PlateauBorne) percerCours(a *Alea) {
	tentatives := b.cote * b.cote / 24
	for i := 0; i < tentatives; i++ {
		c := Position{Colonne: a.Entier(b.cote), Ligne: a.Entier(b.cote)}
		if !b.EstRue(c) && b.toucheUneRue(c) {
			b.ouvrir(c)
		}
	}
}

// toucheUneRue dit si une case a une voisine orthogonale praticable.
func (b *PlateauBorne) toucheUneRue(p Position) bool {
	for _, d := range Orthogonales {
		if b.EstRue(p.Avance(d)) {
			return true
		}
	}
	return false
}

// creuserImpasses ouvre des couloirs sans issue depuis une rue.
//
// Sans bords à exploiter, ce sont elles qui permettent le piégeage : un fugitif
// engagé dans une impasse n'en ressort que par où il est entré. Le Barreur y
// gagne sa raison d'être.
//
// Les départs sont énumérés puis mélangés, et non tirés case par case. Une case
// bâtie qui ne touche qu'une seule rue est rare — une centaine sur les quatre
// cent quarante et une d'un Quartier — et la tirer au hasard échouait assez
// souvent pour que l'étape entière n'ait aucun effet mesurable.
func (b *PlateauBorne) creuserImpasses(a *Alea) {
	departs := b.amorces()
	Melanger(a, departs)

	restantes := b.cote * b.cote / SurfaceParImpasse
	for _, d := range departs {
		if restantes == 0 {
			return
		}
		// Un couloir déjà creusé a pu ouvrir une voisine de cette amorce, qui
		// déboucherait maintenant.
		if !b.creusable(d.tete, d.acces) {
			continue
		}
		b.prolonger(d, 1+a.Entier(LongueurImpasseMax))
		restantes--
	}
}

// amorce est le départ d'un couloir borgne : une case bâtie qui ne touche
// qu'une seule rue, et cette rue.
type amorce struct {
	tete  Position
	acces Position
}

// amorces énumère les départs possibles, ligne par ligne puis colonne par
// colonne.
//
// L'ordre du parcours est ce qui rend le mélange reproductible : deux rejeux de
// la même graine doivent mélanger la même liste.
func (b *PlateauBorne) amorces() []amorce {
	var liste []amorce
	for ligne := 0; ligne < b.cote; ligne++ {
		for colonne := 0; colonne < b.cote; colonne++ {
			p := Position{Colonne: colonne, Ligne: ligne}
			for _, d := range Orthogonales {
				if v := p.Avance(d); b.EstRue(v) && b.creusable(p, v) {
					liste = append(liste, amorce{tete: p, acces: v})
					break
				}
			}
		}
	}
	return liste
}

// prolonger creuse tout droit depuis une amorce, et s'arrête dès que la case
// suivante déboucherait.
func (b *PlateauBorne) prolonger(d amorce, longueur int) {
	// Le second retour ne peut pas être faux : une amorce et son accès sont
	// voisins orthogonaux par construction.
	direction, _ := DirectionVers(d.acces, d.tete)

	b.ouvrir(d.tete)
	precedente, tete := d.tete, d.tete.Avance(direction)

	for pas := 1; pas < longueur && b.creusable(tete, precedente); pas++ {
		b.ouvrir(tete)
		precedente, tete = tete, tete.Avance(direction)
	}
}

// creusable dit si une case peut prolonger un couloir borgne arrivant de
// precedente.
//
// Le refus qui compte est le dernier : une case voisine d'une rue autre que
// celle d'où l'on vient débouche, et un couloir qui débouche n'est plus une
// impasse.
func (b *PlateauBorne) creusable(p, precedente Position) bool {
	if !b.dedans(p) || b.EstRue(p) {
		return false
	}
	for _, d := range Orthogonales {
		if v := p.Avance(d); v != precedente && b.EstRue(v) {
			return false
		}
	}
	return true
}

// poserZones place les points d'extraction en périphérie, régulièrement
// espacés.
//
// Le tour du plateau est découpé en autant de segments qu'il y a de zones, et
// chacune se pose au milieu du sien. Pas de tirage : deux zones voisines
// seraient couvrables par un seul inspecteur, ce qui viderait de son sens
// l'idée d'en avoir plus que d'inspecteurs.
func (b *PlateauBorne) poserZones(combien int) {
	perimetre := 4 * (b.cote - 1)
	b.zones = make([]Zone, 0, combien)

	for i := 0; i < combien; i++ {
		centre := b.rapprocher(b.surLeTour(i * perimetre / combien))
		b.zones = append(b.zones, b.tailler(i, centre))
	}
}

// surLeTour convertit une distance parcourue sur le pourtour en position.
func (b *PlateauBorne) surLeTour(pas int) Position {
	cote := b.cote - 1
	switch {
	case pas < cote:
		return Position{Colonne: pas, Ligne: 0}
	case pas < 2*cote:
		return Position{Colonne: cote, Ligne: pas - cote}
	case pas < 3*cote:
		return Position{Colonne: 3*cote - pas, Ligne: cote}
	default:
		return Position{Colonne: 0, Ligne: 4*cote - pas}
	}
}

// rapprocher ramène un point du bord assez à l'intérieur pour qu'un bloc de
// zone y tienne entièrement.
func (b *PlateauBorne) rapprocher(p Position) Position {
	marge := CoteZone / 2
	return Position{
		Colonne: min(max(p.Colonne, marge), b.cote-1-marge),
		Ligne:   min(max(p.Ligne, marge), b.cote-1-marge),
	}
}

// tailler découpe le bloc d'une zone et l'ouvre assez pour qu'on puisse s'y
// tenir.
//
// Les cases sont ouvertes dans l'ordre de parcours du bloc, jamais au hasard :
// une zone est un lieu du plateau, et deux rejeux de la même graine doivent la
// trouver identique.
func (b *PlateauBorne) tailler(numero int, centre Position) Zone {
	marge := CoteZone / 2
	zone := Zone{Numero: numero, Cases: make([]Position, 0, CoteZone*CoteZone)}

	ouvertes := 0
	for ligne := centre.Ligne - marge; ligne <= centre.Ligne+marge; ligne++ {
		for colonne := centre.Colonne - marge; colonne <= centre.Colonne+marge; colonne++ {
			c := Position{Colonne: colonne, Ligne: ligne}
			zone.Cases = append(zone.Cases, c)

			if b.EstRue(c) {
				ouvertes++
				continue
			}
			if ouvertes < RuesParZone {
				b.ouvrir(c)
				ouvertes++
			}
		}
	}
	return zone
}

// ouvrir rend une case praticable, en ignorant ce qui est hors du plateau.
func (b *PlateauBorne) ouvrir(p Position) {
	if p.Colonne < 0 || p.Ligne < 0 || p.Colonne >= b.cote || p.Ligne >= b.cote {
		return
	}
	b.rues[p.Ligne*b.cote+p.Colonne] = true
}
