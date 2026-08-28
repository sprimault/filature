// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"flag"
	"testing"
)

// mesuresLongues élargit l'échantillon des tests de caractérisation.
//
// Vingt graines en routine : assez pour qu'une régression franche sorte, assez
// peu pour que make race et le job « Tests » ne s'allongent pas — ni l'un ni
// l'autre ne passe -short, donc un test lourd s'y paie à chaque exécution.
// Opt-in comme -maj-notices, quand on veut savoir plutôt que se garder.
var mesuresLongues = flag.Bool("mesures", false, "élargir l'échantillon des tests de caractérisation")

// grainesDeCaracterisation rend la taille d'échantillon en vigueur.
func grainesDeCaracterisation() int64 {
	if *mesuresLongues {
		return 200
	}
	return 20
}

// pourChaquePlateau joue la mise en place sur un échantillon de graines et
// appelle la mesure demandée.
//
// Par NewGame et Generate, jamais sur un terrain bricolé : ce qu'on caractérise
// est ce que la génération produit, et un plateau écrit à la main ne dit rien
// de ce qu'un joueur reçoit.
func pourChaquePlateau(t *testing.T, p Settings, mesurer func(*Game)) {
	t.Helper()

	for g := int64(1); g <= grainesDeCaracterisation(); g++ {
		plateau, retenue, err := Generate(g, p)
		if err != nil {
			t.Fatalf("graine %d : %v", g, err)
		}
		// La graine retenue et non celle demandée : c'est elle qui a produit le
		// terrain, et le fugitif se place dessus.
		partie, err := NewGame(plateau, retenue, p, testRegistry())
		if err != nil {
			t.Fatalf("graine %d : %v", g, err)
		}
		mesurer(partie)
	}
}

// TestMobilityRatioHoldsPerSide caractérise ce que le terrain laisse à chaque
// camp, en destinations par case.
//
// docs/regles.md §14 : le fugitif a huit directions de règlement, l'inspecteur
// quatre, mais le terrain en rend 3,5 contre 2,3 — un rapport de 1,5 et non de
// 2. C'est sur cette compensation que repose l'ordre du tour, et elle est plus
// faible qu'annoncé : la borne garde ce qui reste, pas ce qu'on espérait.
//
// Bornes et non valeurs : la génération bougera, et un test qui figerait 3,5
// rougirait au premier réglage d'avenue sans que rien soit cassé.
func TestMobilityRatioHoldsPerSide(t *testing.T) {
	for _, pre := range Presets() {
		t.Run(pre.Key, func(t *testing.T) {
			var fugitif, inspecteur, cases int

			pourChaquePlateau(t, pre.Settings, func(p *Game) {
				rayon := p.Settings.Size / 2
				for _, c := range p.Board.CellsWithin(Position{Column: rayon, Row: rayon}, rayon) {
					if !p.IsWalkable(c) {
						continue
					}
					cases++
					fugitif += len(p.terrainSteps(c))
					for _, d := range Orthogonales {
						if p.IsWalkable(c.Step(d)) {
							inspecteur++
						}
					}
				}
			})

			if cases == 0 {
				t.Fatal("aucune case de rue mesurée")
			}
			parCase := float64(fugitif) / float64(cases)
			parPion := float64(inspecteur) / float64(cases)
			t.Logf("fugitif %.2f destinations par case, inspecteur %.2f, rapport %.2f",
				parCase, parPion, parCase/parPion)

			// Le petit préréglage est le plus ouvert et frôle la borne haute :
			// s'il la franchit, c'est la densité de sa trame qui a bougé.
			if parCase < 3.3 || parCase > 3.8 {
				t.Errorf("fugitif à %.2f destinations, attendu entre 3,3 et 3,8 : "+
					"le terrain a changé de densité", parCase)
			}
			if r := parCase / parPion; r < 1.4 {
				t.Errorf("rapport de %.2f entre les camps, attendu au moins 1,4 : "+
					"c'est lui que l'ordre du tour compense (§2)", r)
			}
		})
	}
}

// TestSetupCannotSeeTheWholeCentre caractérise la part du noyau de départ que
// cinq inspecteurs couvrent au mieux, avant que quiconque ait joué.
//
// docs/regles.md §2 : sans l'interdiction de se poser dans le noyau, cinq pions
// placés au mieux en voyaient plus de neuf dixièmes dès le premier tour, et
// l'invisibilité que le §1 promet au fugitif n'existait pas. C'est cette mesure
// qui a produit la règle, et c'est elle que la borne garde.
//
// **La couverture du noyau et non le repérage d'une partie.** Le fugitif est
// tiré au hasard dans le noyau : savoir s'il a été vu mesure ce tirage autant
// que le placement, et il faut beaucoup de parties pour que le bruit s'efface.
// La part couverte, elle, est une propriété du plateau et du placement seuls.
//
// **Le placement est glouton et non naïf.** Poser les pions au plus près du
// centre paraît agressif et ne l'est pas : leurs lignes de vue se recouvrent, et
// la mesure passe alors sans rien garder — vérifié en retirant l'exclusion du
// noyau, qui ne faisait pas bouger le résultat. Chaque pion prend donc la case
// qui découvre le plus de noyau que les précédents ne voient pas.
func TestSetupCannotSeeTheWholeCentre(t *testing.T) {
	for _, pre := range Presets() {
		t.Run(pre.Key, func(t *testing.T) {
			var couverture float64
			parties := 0

			pourChaquePlateau(t, pre.Settings, func(p *Game) {
				parties++
				couverture += couvrirLeNoyauAuMieux(t, p)
			})

			part := couverture * 100 / float64(parties)
			t.Logf("cinq pions placés au mieux voient %.0f %% du noyau", part)

			if part >= 50 {
				t.Errorf("cinq pions couvrent %.0f %% du noyau au placement optimal, "+
					"attendu moins de la moitié : l'exclusion du noyau ne protège plus "+
					"l'invisibilité que le §1 promet au fugitif", part)
			}
		})
	}
}

// couvrirLeNoyauAuMieux joue la mise en place en maximisant à chaque pion ce
// que le camp découvre du noyau, et rend la part couverte.
func couvrirLeNoyauAuMieux(t *testing.T, p *Game) float64 {
	t.Helper()

	// Le fugitif scelle d'abord : la phase de placement des inspecteurs ne
	// s'ouvre qu'ensuite.
	if err := p.Apply(p.LegalMoves(SideFugitive)[0]); err != nil {
		t.Fatal(err)
	}

	rayon := p.Settings.Size / 2
	centre := Position{Column: rayon, Row: rayon}
	noyau := map[Position]bool{}
	for _, c := range p.Board.CellsWithin(centre, p.Settings.CentreRadius) {
		if p.IsWalkable(c) && ChebyshevDistance(c, centre) <= p.Settings.CentreRadius {
			noyau[c] = true
		}
	}
	if len(noyau) == 0 {
		t.Fatal("aucune case de rue dans le noyau de départ")
	}

	couvertes := map[Position]bool{}
	for p.Phase == PhaseInspectorsSetup {
		legaux := p.LegalMoves(SideInspectors)
		if len(legaux) == 0 {
			t.Fatal("aucune case où poser un inspecteur")
		}

		meilleur, gain := legaux[0], -1
		for _, c := range legaux {
			// Un pion trop loin ne peut rien voir du noyau : l'écarter évite de
			// dérouler une ligne de vue pour chaque case du plateau.
			if ChebyshevDistance(c.To, centre) > p.Settings.CentreRadius+p.Settings.Range {
				continue
			}
			n := 0
			for _, vue := range p.Board.Sight(c.To, p.Settings.Range) {
				if noyau[vue] && !couvertes[vue] {
					n++
				}
			}
			if n > gain {
				meilleur, gain = c, n
			}
		}

		for _, vue := range p.Board.Sight(meilleur.To, p.Settings.Range) {
			if noyau[vue] {
				couvertes[vue] = true
			}
		}
		if err := p.Apply(meilleur); err != nil {
			t.Fatal(err)
		}
	}
	return float64(len(couvertes)) / float64(len(noyau))
}

// TestNearestZoneIsWithinHalfTheGame caractérise la longueur du trajet que le
// fugitif a devant lui au départ.
//
// docs/regles.md §14 tient le milieu de partie pour ce qui manque au jeu : sur
// la plus grande ville, la zone la plus proche est à quatorze pas et
// l'étranglement démarre au tour trente. Une borne haute seulement — un trajet
// court n'est pas un défaut, un trajet qui mangerait la partie en est un.
//
// **La borne porte sur la moyenne, pas sur le pire plateau.** Un tirage isolé
// qui pose la zone la plus proche à vingt et un pas sur quarante tours ne casse
// rien : le fugitif garde dix-neuf tours et n'est pas tenu d'aller à la plus
// proche. Ce qui casserait le jeu est que le trajet ordinaire mange la moitié
// de la partie, puisqu'il ne resterait plus de milieu du tout. Le pire cas
// reste affiché — il informe sans décider.
func TestNearestZoneIsWithinHalfTheGame(t *testing.T) {
	for _, pre := range Presets() {
		t.Run(pre.Key, func(t *testing.T) {
			pire, total, parties := 0, 0, 0

			pourChaquePlateau(t, pre.Settings, func(p *Game) {
				pas := pasVersLaZoneLaPlusProche(t, p)
				pire = max(pire, pas)
				total += pas
				parties++
			})

			moyenne := float64(total) / float64(parties)
			moitie := float64(pre.Settings.Turns) / 2
			t.Logf("zone la plus proche à %.1f pas en moyenne, %d au pire, pour %d tours",
				moyenne, pire, pre.Settings.Turns)

			if moyenne > moitie {
				t.Errorf("zone la plus proche à %.1f pas en moyenne pour %d tours de jeu : "+
					"le trajet mange la moitié de la partie, et le trajet n'est pas le jeu",
					moyenne, pre.Settings.Turns)
			}
		})
	}
}

// pasVersLaZoneLaPlusProche compte les pas du départ du fugitif à la zone la
// plus proche.
//
// Un parcours en largeur sur les pas que le fugitif sait faire, et non une
// distance à vol d'oiseau : c'est le bâti qui décide de la longueur d'un
// trajet, et une diagonale refusée par un angle fermé en ajoute deux.
func pasVersLaZoneLaPlusProche(t *testing.T, p *Game) int {
	t.Helper()

	dansUneZone := map[Position]bool{}
	for _, z := range p.Board.Zones() {
		for _, c := range z.Cells {
			dansUneZone[c] = true
		}
	}

	vues := map[Position]bool{p.Fugitive.Position: true}
	front := []Position{p.Fugitive.Position}
	for pas := 0; len(front) > 0; pas++ {
		var suivant []Position
		for _, c := range front {
			if dansUneZone[c] {
				return pas
			}
			for _, voisine := range p.terrainSteps(c) {
				if !vues[voisine] {
					vues[voisine] = true
					suivant = append(suivant, voisine)
				}
			}
		}
		front = suivant
	}

	t.Fatalf("aucune zone atteignable depuis %v : la validation de connexité a laissé passer un plateau",
		p.Fugitive.Position)
	return 0
}
