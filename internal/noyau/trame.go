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
	CoteZone           = 3
	RuesParZone        = 5
	LongueurImpasseMax = 3

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

// creuserImpasses ouvre des couloirs sans issue depuis une avenue.
//
// Sans bords à exploiter, ce sont elles qui permettent le piégeage : un fugitif
// engagé dans une impasse n'en ressort que par où il est entré. Le Barreur y
// gagne sa raison d'être.
func (b *PlateauBorne) creuserImpasses(a *Alea) {
	impasses := b.cote / 3
	for i := 0; i < impasses; i++ {
		depart := Position{Colonne: a.Entier(b.cote), Ligne: a.Entier(b.cote)}
		if !b.EstRue(depart) {
			continue
		}

		direction := Orthogonales[a.Entier(len(Orthogonales))]
		longueur := 1 + a.Entier(LongueurImpasseMax)
		for pas := 0; pas < longueur; pas++ {
			depart = depart.Avance(direction)
			if depart.Colonne < 0 || depart.Ligne < 0 ||
				depart.Colonne >= b.cote || depart.Ligne >= b.cote {
				break
			}
			b.ouvrir(depart)
		}
	}
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
