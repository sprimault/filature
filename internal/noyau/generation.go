// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

import "errors"

// Parametres rassemble tout ce qui est réglable depuis l'interface. Ces valeurs
// sont enregistrées avec la partie : une sauvegarde reste rejouable même si les
// valeurs par défaut changent.
type Parametres struct {
	Cote                int `json:"cote"`
	Portee              int `json:"portee"`
	Tours               int `json:"tours"`
	Resistance          int `json:"resistance"`
	Inspecteurs         int `json:"inspecteurs"`
	PionsParTour        int `json:"pions_par_tour"`
	PeriodeRevelation   int `json:"periode_revelation"`
	Zones               int `json:"zones"`
	DureeTrace          int `json:"duree_trace"`
	DebutEtranglement   int `json:"debut_etranglement"`
	PeriodeEtranglement int `json:"periode_etranglement"`
}

// ParametresDefaut correspond au préréglage « Ville ». Les valeurs sont
// justifiées dans docs/regles.md ; celle qui bougera en premier est
// PionsParTour.
func ParametresDefaut() Parametres {
	return Parametres{
		Cote:                41,
		Portee:              8,
		Tours:               40,
		Resistance:          10,
		Inspecteurs:         5,
		PionsParTour:        3,
		PeriodeRevelation:   4,
		Zones:               6,
		DureeTrace:          6,
		DebutEtranglement:   30,
		PeriodeEtranglement: 2,
	}
}

// Valider refuse les combinaisons qui rendent la partie injouable plutôt que de
// les laisser produire un plateau invalide ou une partie sans issue.
func (p Parametres) Valider() error {
	switch {
	case p.Cote < 15 || p.Cote > 81:
		return errors.New("côté hors bornes")
	case p.Zones <= p.Inspecteurs:
		return errors.New("il faut plus de zones que d'inspecteurs, sinon tout est couvrable")
	case p.Portee < 3 || p.Portee > p.Cote/2:
		return errors.New("portée hors bornes")
	case p.DebutEtranglement >= p.Tours:
		return errors.New("l'étranglement doit commencer avant la fin")
	}
	return nil
}

// PlateauBorne est l'implémentation v1 : une grille dense, cases hors limites
// traitées comme bâtiments.
type PlateauBorne struct {
	graine int64
	cote   int
	rues   []bool
	zones  []Zone
	vues   map[Position][]Position
}

// Generer produit un plateau déterministe à partir d'une graine.
//
// Le générateur retente avec la graine suivante tant que Valider échoue, et
// renvoie la graine effectivement utilisée : c'est elle qui part en base, pas
// celle demandée.
func Generer(graine int64, p Parametres) (*PlateauBorne, int64, error) {
	return nil, 0, errors.New("à implémenter : étape 3")
}

// valider vérifie qu'une seule composante connexe couvre toutes les rues, que
// chaque zone est atteignable et que le taux de rues tombe entre 35 % et 50 %.
// Un plateau qui échoue est jeté, jamais rapiécé.
func (b *PlateauBorne) valider(p Parametres) error {
	return errors.New("à implémenter : étape 3")
}

// EstRue dit si une case est praticable. Hors plateau vaut bâtiment, ce qui
// évite un test de bornes à chaque appelant.
func (b *PlateauBorne) EstRue(p Position) bool {
	if p.Colonne < 0 || p.Ligne < 0 || p.Colonne >= b.cote || p.Ligne >= b.cote {
		return false
	}
	return b.rues[p.Ligne*b.cote+p.Colonne]
}

// CasesDans énumère les rues autour d'un centre, ligne par ligne puis colonne
// par colonne. L'ordre est un contrat : il détermine celui des coups légaux, et
// donc ce qu'un rejeu doit retrouver.
func (b *PlateauBorne) CasesDans(centre Position, rayon int) []Position {
	var cases []Position
	for ligne := centre.Ligne - rayon; ligne <= centre.Ligne+rayon; ligne++ {
		for colonne := centre.Colonne - rayon; colonne <= centre.Colonne+rayon; colonne++ {
			if p := (Position{Colonne: colonne, Ligne: ligne}); b.EstRue(p) {
				cases = append(cases, p)
			}
		}
	}
	return cases
}

// Zones renvoie les zones d'extraction, connues des deux camps dès le début.
func (b *PlateauBorne) Zones() []Zone { return b.zones }

// Graine renvoie la graine dont ce plateau est issu.
func (b *PlateauBorne) Graine() int64 { return b.graine }

// Vision renvoie les cases vues depuis p, terrain seul. L'occlusion par les
// pions et les barrages s'applique à la lecture : la précalculer supposerait de
// tout recalculer à chaque déplacement.
func (b *PlateauBorne) Vision(p Position, portee int) []Position {
	return b.vues[p]
}
