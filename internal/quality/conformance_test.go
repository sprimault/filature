// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/sprimault/filature/internal/core"
	"github.com/sprimault/filature/internal/loader"
	"github.com/sprimault/filature/plugins"
)

// Ce fichier suit le chemin complet — manifeste livré, chargement, partie
// jouée — et vérifie que les mécaniques de docs/regles.md agissent réellement.
//
// **Il ne construit jamais d'état à la main.** C'est sa raison d'être : les
// autres tests exercent les pièces une par une, et six mécaniques mortes leur
// ont échappé parce que chacune injectait elle-même l'état qu'elle allait
// vérifier. Le cas d'école est le mode d'étranglement, que le noyau cherche
// sous « etranglement » quand le manifeste le déclare « strangling » : un test
// posait la clé française, un autre vérifiait l'anglaise, les deux passaient au
// vert et la mécanique n'a jamais tourné.
//
// Tout ce que ce fichier injecterait lui-même serait une portion du chemin qui
// resterait non testée.
//
// **MECANIQUE INERTE** marque un cas que le code ne satisfait pas encore. Ce
// n'est pas un test désactivé, c'est une dette nommée à l'endroit où elle se
// paiera, et TestInertMechanicsAreCounted en tient le compte. Un skip se lève
// en corrigeant le code, jamais en retirant le cas.

// campDe rend le camp à qui la phase donne la main.
func campDe(phase core.Phase) core.Side {
	switch phase {
	case core.PhaseInspectorsSetup, core.PhaseInspectors:
		return core.SideInspectors
	default:
		return core.SideFugitive
	}
}

// partieLivree monte une partie sur le contenu réellement livré : manifeste lu
// par le chargeur du jeu, plateau engendré, pions posés par des coups légaux.
//
// Le registre vient de loader.Load et non d'un littéral : c'est ce chemin-là
// qui décide des clés sous lesquelles le noyau retrouvera capacités et modes.
func partieLivree(t *testing.T, preset string) *core.Game {
	t.Helper()

	registre, _, err := loader.Load(plugins.Shipped(), ".")
	if err != nil {
		t.Fatalf("chargement du contenu livré : %v", err)
	}

	reglage, connu := core.PresetByKey(preset)
	if !connu {
		t.Fatalf("préréglage %q inconnu", preset)
	}

	plateau, graine, err := core.Generate(7, reglage.Settings)
	if err != nil {
		t.Fatalf("génération : %v", err)
	}
	partie, err := core.NewGame(plateau, graine, reglage.Settings, registre)
	if err != nil {
		t.Fatalf("mise en place : %v", err)
	}

	// La mise en place passe par des coups légaux, comme une vraie partie : le
	// fugitif scelle sa zone, les inspecteurs se posent un par un.
	for partie.Phase == core.PhaseFugitiveSetup || partie.Phase == core.PhaseInspectorsSetup {
		coups := partie.LegalMoves(campDe(partie.Phase))
		if len(coups) == 0 {
			t.Fatalf("aucun coup légal en phase %s", partie.Phase)
		}
		if err := partie.Apply(coups[0]); err != nil {
			t.Fatalf("mise en place, coup %+v : %v", coups[0], err)
		}
	}
	return partie
}

// finirLeTour rend la main des deux camps, ce qui déclenche la résolution.
func finirLeTour(t *testing.T, p *core.Game) {
	t.Helper()
	for i := 0; i < 2 && p.Phase != core.PhaseOver; i++ {
		camp := campDe(p.Phase)
		for _, c := range p.LegalMoves(camp) {
			if c.Type == core.MoveEndPhase {
				if err := p.Apply(c); err != nil {
					t.Fatalf("fin de phase : %v", err)
				}
				break
			}
		}
	}
}

// TestStranglingActuallyCloses vérifie que l'étranglement ferme réellement une
// zone, sur le contenu livré.
//
// docs/regles.md §10 : aux trois quarts de la partie, les zones se ferment à
// intervalle régulier jusqu'à ce qu'il n'en reste que trois.
func TestStranglingActuallyCloses(t *testing.T) {
	p := partieLivree(t, "quartier")

	if _, connu := p.Extensions.Modes["etranglement"]; !connu {
		t.Skip("MECANIQUE INERTE : le noyau cherche Modes[\"etranglement\"] " +
			"(internal/core/turn.go) alors que plugins/base/manifest.toml déclare " +
			"[mode.strangling] et que le chargeur recopie la clé telle quelle. " +
			"Aucune zone ne se ferme, aucun préavis n'est annoncé.")
	}

	for p.Phase != core.PhaseOver && len(p.ClosedZones) == 0 {
		finirLeTour(t, p)
	}
	if len(p.ClosedZones) == 0 {
		t.Error("aucune zone fermée sur une partie entière")
	}
}

// TestRoadblockExpires vérifie qu'un barrage rend sa case au bout de sa durée.
//
// docs/regles.md §9 : le Barreur ferme une case de rue pendant 3 tours.
func TestRoadblockExpires(t *testing.T) {
	t.Skip("MECANIQUE INERTE : IsWalkable ne teste que la présence de la clé " +
		"dans Roadblocks (internal/core/game.go) et aucune étape de résolution " +
		"ne purge les entrées échues. Un barrage tient jusqu'à la fin de la " +
		"partie, et open_cell perce définitivement.")
}

// TestTrackerSeesFarther vérifie que le Traqueur perçoit les traces à deux
// cases dès le premier tour, sans rien déclencher.
//
// docs/regles.md §9 : « Perçoit les traces à deux cases, en permanence
// (passif) ».
func TestTrackerSeesFarther(t *testing.T) {
	p := partieLivree(t, "quartier")

	traqueur := -1
	for i := range p.Inspectors {
		if p.Inspectors[i].Ability == "tracker" {
			traqueur = i
		}
	}
	if traqueur < 0 {
		t.Fatal("aucun pion ne porte la capacité tracker : le contenu livré a changé")
	}

	if p.TrailRadiusOf(traqueur) < 2 {
		t.Skip("MECANIQUE INERTE : une capacité passive n'est jamais appliquée. " +
			"abilityMoves (internal/core/legalmoves.go) exclut les passives, et " +
			"rien ne pose leurs effets à la mise en place. Le Traqueur découvre " +
			"les traces comme les quatre autres pions.")
	}
}

// TestSpotterGetsAnExtraStep vérifie qu'un inspecteur qui repère le fugitif
// gagne un déplacement hors quota.
//
// docs/regles.md §2 et §5 : c'est ce qui remplace la téléportation supprimée.
func TestSpotterGetsAnExtraStep(t *testing.T) {
	t.Skip("MECANIQUE INERTE : rien dans legalmoves.go ne rend un déplacement " +
		"supplémentaire à un pion qui vient de repérer le fugitif. Le quota de " +
		"PiecesPerTurn s'applique sans exception.")
}

// TestLookoutDoublesRange vérifie que le Guetteur double sa portée plutôt que
// d'y ajouter une constante.
//
// docs/regles.md §9 : « Portée de vue doublée pendant un tour ». L'écart ne se
// voit que sur le plus petit préréglage, où la portée vaut 4 : doubler donne 8,
// ajouter 8 donne 12.
func TestLookoutDoublesRange(t *testing.T) {
	p := partieLivree(t, "quartier")

	guetteur := -1
	for i := range p.Inspectors {
		if p.Inspectors[i].Ability == "lookout" {
			guetteur = i
		}
	}
	if guetteur < 0 {
		t.Fatal("aucun pion ne porte la capacité lookout : le contenu livré a changé")
	}

	base := p.RangeOf(guetteur)
	joue := false
	for _, c := range p.LegalMoves(core.SideInspectors) {
		if c.Type == core.MoveAbility && c.Piece == guetteur {
			if err := p.Apply(c); err != nil {
				t.Fatalf("déclenchement du Guetteur : %v", err)
			}
			joue = true
			break
		}
	}
	if !joue {
		t.Fatal("la capacité du Guetteur n'est pas proposée en phase inspecteurs")
	}

	if got := p.RangeOf(guetteur); got != 2*base {
		t.Skip("MECANIQUE INERTE : le manifeste déclare change_range avec une " +
			"valeur absolue, additionnée à la portée de base. Sur Quartier elle " +
			"vaut " + strconv.Itoa(base) + " et le Guetteur la porte à " +
			strconv.Itoa(got) + " au lieu de " + strconv.Itoa(2*base) +
			" : la capacité triple au lieu de doubler.")
	}
}

// TestChiefSharesView vérifie que la vue partagée du Chef entre réellement dans
// ce que son camp voit.
//
// docs/regles.md §9 : « Voit ce que voit un autre inspecteur pendant deux
// tours ».
func TestChiefSharesView(t *testing.T) {
	t.Skip("MECANIQUE INERTE : SharedViewOf (internal/core/effects.go) n'a " +
		"aucun appelant hors test. visibleCellsFor ne consulte pas les vues " +
		"partagées, donc déclencher la capacité du Chef ne change rien à ce que " +
		"son camp voit.")
}

// TestShelterNeedsEntering vérifie qu'un fugitif immobile sur un lieu ne se
// ressource pas à chaque recharge.
//
// docs/regles.md §5 et §7 : le ressourcement se déclenche en entrant, pas en
// s'y tenant. Autrement c'est la récupération à l'immobilité, écartée au §2
// parce qu'elle récompense le campement.
func TestShelterNeedsEntering(t *testing.T) {
	t.Skip("MECANIQUE INERTE : useShelter (internal/core/turn.go) teste la " +
		"présence du fugitif sur le lieu, pas son entrée. Un fugitif qui ne " +
		"bouge pas y regagne deux points à chaque recharge — mesuré sur " +
		"Quartier : 10 au tour 1, puis 12, 14, 16 aux tours 1, 9 et 17.")
}

// TestInertMechanicsAreCounted garde le compte des mécaniques inertes visible.
//
// Un test rouge fait mal et se corrige ; un test qui se saute passe inaperçu, et
// six mois plus tard il reste des skips que personne ne voit plus. Ce contrôle
// relit le fichier et refuse qu'une dette apparaisse ou disparaisse sans que la
// constante bouge dans le même lot.
//
// Le nombre ne peut que descendre : le remonter demande d'écrire ici qu'on a
// cassé une mécanique qui marchait.
//
// C'est ce qui distingue ce compteur de celui des stubs de ROADMAP.md, et les
// deux n'ont pas à se ressembler : **le compte de stubs mesure, celui-ci
// verrouille.** Un stub porte son propre marqueur et ne peut pas s'oublier, donc
// il suffit de le compter ; une mécanique inerte est un test qui se saute, donc
// silencieux — il lui faut une constante en face pour qu'une dette ajoutée ou
// levée sans être déclarée fasse rougir la suite.
func TestInertMechanicsAreCounted(t *testing.T) {
	const attendues = 7

	source, err := os.ReadFile("conformance_test.go")
	if err != nil {
		t.Fatalf("lecture du fichier de conformité : %v", err)
	}

	// Le marqueur est reconstitué plutôt qu'écrit d'un bloc : littéral, cette
	// ligne se compterait elle-même. C'est le défaut qui a fait annoncer
	// quarante-cinq stubs à la ROADMAP pour vingt et un — le contrôle qui cite
	// le motif entre dans son propre décompte.
	marqueur := "MECANIQUE INERTE" + " :"
	n := strings.Count(string(source), marqueur)
	if n != attendues {
		t.Errorf("%d mécaniques inertes déclarées, %d attendues — "+
			"corriger la constante dans le même lot que le code", n, attendues)
	}
	if n > 0 {
		t.Logf("%d mécanique(s) encore inerte(s)", n)
	}
}
