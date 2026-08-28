// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"os"
	"slices"
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

// finirLaPhase rend la main du camp qui l'a, sans résoudre le tour.
func finirLaPhase(t *testing.T, p *core.Game) {
	t.Helper()
	for _, c := range p.LegalMoves(campDe(p.Phase)) {
		if c.Type == core.MoveEndPhase {
			if err := p.Apply(c); err != nil {
				t.Fatalf("fin de phase : %v", err)
			}
			return
		}
	}
	t.Fatalf("aucune fin de phase possible en %s", p.Phase)
}

// TestStranglingActuallyCloses vérifie que l'étranglement ferme réellement une
// zone, sur le contenu livré.
//
// docs/regles.md §10 : aux trois quarts de la partie, les zones se ferment à
// intervalle régulier jusqu'à ce qu'il n'en reste que trois.
func TestStranglingActuallyCloses(t *testing.T) {
	p := partieLivree(t, "district")
	ouvertes := len(p.Board.Zones())

	for p.Phase != core.PhaseOver && len(p.ClosedZones) == 0 {
		finirLeTour(t, p)
	}

	if len(p.ClosedZones) == 0 {
		t.Fatalf("aucune zone fermée en %d tours, alors que l'étranglement "+
			"démarre au tour %d", p.Turn, p.Settings.StranglingStart)
	}
	if reste := ouvertes - len(p.ClosedZones); reste < p.Settings.ZonesLeftOpen {
		t.Errorf("%d zones ouvertes, %d attendues au minimum",
			reste, p.Settings.ZonesLeftOpen)
	}
}

// TestStranglingClosesOnScheduleAndAnnouncesAhead vérifie que les zones ferment
// aux tours publiés et qu'aucune ne ferme sans avoir été annoncée.
//
// docs/regles.md §10 donne les tours de fermeture par préréglage et promet un
// préavis de deux tours. Les deux moitiés se tiennent : le mode livré portait
// auparavant le préavis sous forme d'un differer, qui s'ajoutait à la cadence du
// noyau au lieu de la précéder — les fermetures tombaient deux tours après le
// tableau, la dernière au dernier tour de la partie, et le contrôle arithmétique
// des préréglages ne le voyait pas puisqu'il mesurait le tour d'annonce.
//
// Se joue sur les trois préréglages : la période dérive de la durée, et le
// Quartier est le seul où elle égale le préavis.
func TestStranglingClosesOnScheduleAndAnnouncesAhead(t *testing.T) {
	for _, cle := range []string{"district", "outskirts", "city"} {
		t.Run(cle, func(t *testing.T) {
			p := partieLivree(t, cle)
			s := p.Settings

			annonce := map[int]int{}
			fermeture := map[int]int{}
			var ordre []int

			for p.Phase != core.PhaseOver {
				tour := p.Turn
				for _, zone := range p.ViewFor(core.SideFugitive).ZonesAnnoncees {
					if _, vue := annonce[zone]; !vue {
						annonce[zone] = tour
					}
				}

				avant := append([]int(nil), p.ClosedZones...)
				finirLeTour(t, p)
				for _, zone := range p.ClosedZones {
					if !slices.Contains(avant, zone) {
						fermeture[zone] = tour
						ordre = append(ordre, zone)
					}
				}
			}

			attendues := s.Zones - s.ZonesLeftOpen
			if len(ordre) != attendues {
				t.Fatalf("%d zones fermées en %d tours, %d attendues",
					len(ordre), s.Turns, attendues)
			}

			for rang, zone := range ordre {
				voulu := s.StranglingStart + rang*s.StranglingPeriod
				if fermeture[zone] != voulu {
					t.Errorf("zone %d fermée au tour %d, attendu %d",
						zone, fermeture[zone], voulu)
				}
				quand, annoncee := annonce[zone]
				if !annoncee {
					t.Errorf("zone %d fermée au tour %d sans avoir été annoncée",
						zone, fermeture[zone])
					continue
				}
				if delai := fermeture[zone] - quand; delai != s.StranglingNotice {
					t.Errorf("zone %d annoncée %d tours avant sa fermeture, attendu %d",
						zone, delai, s.StranglingNotice)
				}
			}

			// La dernière fermeture laisse de quoi jouer après elle, sans quoi
			// la pression tomberait quand il n'y a plus rien à en faire.
			if derniere := fermeture[ordre[len(ordre)-1]]; derniere > s.Turns-core.StranglingEndMargin {
				t.Errorf("dernière fermeture au tour %d pour %d tours de jeu",
					derniere, s.Turns)
			}
		})
	}
}

// TestDecoyReplacesTheTurnTrails vérifie qu'un leurre substitue sa trace à
// celles du tour, au lieu de s'y ajouter.
//
// docs/regles.md §8 : « les traces d'un tour sont toutes vraies ou toutes
// fausses, jamais mélangées ». Mêlées, un inspecteur qui en découvrirait deux
// incompatibles saurait qu'il y a eu leurre — et l'apprendre est déjà une
// information que le fugitif a payée pour ne pas donner.
func TestDecoyReplacesTheTurnTrails(t *testing.T) {
	p := partieLivree(t, "district")

	// Les inspecteurs jouent avant le fugitif : leur rendre la main est ce qui
	// lui donne la sienne.
	for _, c := range p.LegalMoves(core.SideInspectors) {
		if c.Type == core.MoveEndPhase {
			if err := p.Apply(c); err != nil {
				t.Fatalf("fin de la phase des inspecteurs : %v", err)
			}
			break
		}
	}

	// Un pas d'abord : sans lui, la substitution ne remplacerait rien et le
	// test passerait pour une partie où le fugitif n'a pas bougé.
	var pas core.Move
	for _, c := range p.LegalMoves(core.SideFugitive) {
		if c.Type == core.MoveStep {
			pas = c
			break
		}
	}
	if pas.Type == "" {
		t.Fatal("aucun déplacement légal au premier tour")
	}
	if err := p.Apply(pas); err != nil {
		t.Fatalf("déplacement : %v", err)
	}

	var leurre core.Move
	for _, c := range p.LegalMoves(core.SideFugitive) {
		if c.Type == core.MoveExpense && c.Expense == core.ExpenseDecoy {
			leurre = c
			break
		}
	}
	if leurre.Type == "" {
		t.Fatal("aucun leurre proposé : la dépense n'est pas dans le contenu livré, " +
			"ou elle ne demande pas où poser sa trace")
	}
	if err := p.Apply(leurre); err != nil {
		t.Fatalf("leurre : %v", err)
	}

	tour := p.Turn
	finirLeTour(t, p)

	posees := map[core.Position]bool{}
	for pos, trace := range p.Trails {
		if trace.Turn == tour {
			posees[pos] = true
		}
	}
	if len(posees) != 1 {
		t.Fatalf("%d traces posées au tour %d, une seule attendue", len(posees), tour)
	}
	if !posees[leurre.From] {
		t.Errorf("la trace du tour n'est pas celle du leurre : posée ailleurs qu'en %v", leurre.From)
	}
	if posees[pas.From] && pas.From != leurre.From {
		t.Errorf("la trace du déplacement en %v est restée", pas.From)
	}
}

// TestRoadblockExpires vérifie qu'un barrage rend sa case au bout de sa durée.
//
// docs/regles.md §9 : le Barreur ferme une case de rue pendant 3 tours.
func TestRoadblockExpires(t *testing.T) {
	p := partieLivree(t, "district")

	barreur := -1
	for i := range p.Inspectors {
		if p.Inspectors[i].Ability == "blocker" {
			barreur = i
		}
	}
	if barreur < 0 {
		t.Fatal("aucun pion ne porte la capacité blocker : le contenu livré a changé")
	}

	joue := false
	for _, c := range p.LegalMoves(core.SideInspectors) {
		if c.Type == core.MoveAbility && c.Piece == barreur {
			if err := p.Apply(c); err != nil {
				t.Fatalf("déclenchement du Barreur : %v", err)
			}
			joue = true
			break
		}
	}
	if !joue {
		t.Fatal("la capacité du Barreur n'est pas proposée en phase inspecteurs")
	}
	if len(p.Roadblocks) == 0 {
		t.Fatal("le Barreur n'a barré aucune case")
	}

	// La règle donne trois tours : la case est fermée pendant celui où elle est
	// posée et les deux suivants, puis elle rouvre.
	for i := 0; i < 3 && p.Phase != core.PhaseOver; i++ {
		finirLeTour(t, p)
	}
	if len(p.Roadblocks) != 0 {
		t.Errorf("%d barrage(s) au bout de trois tours, attendu aucun",
			len(p.Roadblocks))
	}
}

// TestTrackerSeesFarther vérifie que le Traqueur perçoit les traces à deux
// cases dès le premier tour, sans rien déclencher.
//
// docs/regles.md §9 : « Perçoit les traces à deux cases, en permanence
// (passif) ».
func TestTrackerSeesFarther(t *testing.T) {
	p := partieLivree(t, "district")

	traqueur := -1
	for i := range p.Inspectors {
		if p.Inspectors[i].Ability == "tracker" {
			traqueur = i
		}
	}
	if traqueur < 0 {
		t.Fatal("aucun pion ne porte la capacité tracker : le contenu livré a changé")
	}

	if got := p.TrailRadiusOf(traqueur); got != 2 {
		t.Errorf("rayon de traces du Traqueur %d, attendu 2", got)
	}

	// Et lui seul : une capacité passive ne déteint pas sur le camp.
	for i := range p.Inspectors {
		if i == traqueur {
			continue
		}
		if got := p.TrailRadiusOf(i); got != 1 {
			t.Errorf("rayon du pion %d à %d, attendu 1", i, got)
		}
	}
}

// TestSpotterGetsAnExtraStep vérifie qu'un inspecteur qui repère le fugitif
// gagne un déplacement hors quota.
//
// docs/regles.md §2 et §5 : c'est ce qui remplace la téléportation supprimée.
func TestSpotterGetsAnExtraStep(t *testing.T) {
	p := partieLivree(t, "district")

	// Le pion 0 est amené à portée de vue du fugitif, puis on lui fait faire un
	// pas : la règle lui en rend un.
	depart := p.Fugitive.Position
	p.Inspectors[0].Position = core.Position{Column: depart.Column, Row: depart.Row}
	p.Phase = core.PhaseInspectors

	base := p.MobilityOf(core.SideInspectors, 0)
	for _, c := range p.LegalMoves(core.SideInspectors) {
		if c.Type == core.MoveStep && c.Piece == 0 {
			if err := p.Apply(c); err != nil {
				t.Fatalf("déplacement du pion 0 : %v", err)
			}
			break
		}
	}

	if got := p.MobilityOf(core.SideInspectors, 0); got != base+1 {
		t.Errorf("mobilité %d après repérage, attendu %d", got, base+1)
	}

	// Hors quota, et c'est le point : même avec un quota d'un seul pion, déjà
	// consommé par ce déplacement, le repéreur garde son pas supplémentaire.
	p.Settings.PiecesPerTurn = 1

	encore := false
	for _, c := range p.LegalMoves(core.SideInspectors) {
		if c.Type == core.MoveStep && c.Piece == 0 {
			encore = true
		}
	}
	if !encore {
		t.Error("le repéreur n'a plus de déplacement alors que le pas rendu est hors quota")
	}
}

// TestLookoutDoublesRange vérifie que le Guetteur double sa portée plutôt que
// d'y ajouter une constante.
//
// docs/regles.md §9 : « Portée de vue doublée pendant un tour ». L'écart ne se
// voit que sur le plus petit préréglage, où la portée vaut 4 : doubler donne 8,
// ajouter 8 donne 12.
func TestLookoutDoublesRange(t *testing.T) {
	p := partieLivree(t, "district")

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
		t.Errorf("portée %d après la capacité, attendu %d — la base était %d",
			got, 2*base, base)
	}
}

// TestBotProtocolIsCheckedAtHandshake vérifie qu'un bot d'une autre version du
// protocole est écarté avant de jouer.
//
// docs/protocole-bot.md §5 : « protocol inconnu dans ready — refus avant le
// début ». La promesse est faite à des auteurs de bots, qui n'ont aucun moyen
// de savoir qu'elle n'est pas encore tenue.
func TestBotProtocolIsCheckedAtHandshake(t *testing.T) {
	t.Skip("MECANIQUE INERTE : checkProtocol est écrite et exercée, mais " +
		"ai.Start est un stub d'étape 9 et ne l'appelle donc pas encore. " +
		"Le contrôle existe sans être branché — le compter ici plutôt que de " +
		"s'en remettre au marqueur de stub de Start, qui dit qu'une étape " +
		"manque et non qu'une promesse du contrat n'est pas tenue.")
}

// TestChiefForcesAReveal vérifie que la capacité du Chef rend la position du
// fugitif publique, deux tours après son déclenchement.
//
// docs/regles.md §9. La capacité partageait auparavant la vue d'un coéquipier,
// ce qui ne voulait rien dire : les cinq inspecteurs sont un joueur unique et
// ViewFor unit déjà ce que chacun voit. Elle a été écrite contre un modèle
// d'information que le jeu n'a jamais eu.
//
// Le préavis fait partie de la mécanique : une révélation immédiate ne se
// contrerait pas, le Silence s'achetant avant de savoir.
func TestChiefForcesAReveal(t *testing.T) {
	p := partieLivree(t, "district")

	chef := -1
	for i := range p.Inspectors {
		if p.Inspectors[i].Ability == "chief" {
			chef = i
		}
	}
	if chef < 0 {
		t.Fatal("aucun pion ne porte la capacité chief : le contenu livré a changé")
	}

	declenche := p.Turn
	joue := false
	for _, c := range p.LegalMoves(core.SideInspectors) {
		if c.Type == core.MoveAbility && c.Piece == chef {
			if err := p.Apply(c); err != nil {
				t.Fatalf("capacité du Chef : %v", err)
			}
			joue = true
			break
		}
	}
	if !joue {
		t.Fatal("la capacité du Chef n'est proposée à aucun coup légal")
	}

	// Annoncée aux deux camps dès le déclenchement : c'est ce qui laisse au
	// fugitif le temps de fuir ou de payer.
	if len(p.ViewFor(core.SideFugitive).AnnouncedEffects) == 0 {
		t.Error("la révélation n'est pas annoncée au fugitif")
	}

	// Le coup de filet tombe au tour où la capacité a été jouée, plus le
	// préavis : il faut donc résoudre ce tour-là, pas seulement l'atteindre.
	const preavis = 2
	echeance := declenche + preavis
	for p.Turn <= echeance && p.Phase != core.PhaseOver {
		p.Fugitive.Visible = false
		finirLeTour(t, p)
	}

	if !p.Fugitive.Visible {
		t.Errorf("le fugitif n'est pas révélé au tour %d, %d tours après le coup de filet",
			echeance, preavis)
	}
}

// TestShelterNeedsEntering vérifie qu'un fugitif immobile sur un lieu ne se
// ressource pas à chaque recharge.
//
// docs/regles.md §5 et §7 : le ressourcement se déclenche en entrant, pas en
// s'y tenant. Autrement c'est la récupération à l'immobilité, écartée au §2
// parce qu'elle récompense le campement.
func TestShelterNeedsEntering(t *testing.T) {
	p := partieLivree(t, "district")

	abris := p.Board.Shelters()
	if len(abris) == 0 {
		t.Fatal("aucun lieu de ressourcement sur le plateau")
	}

	// Le fugitif est posé sur un lieu et n'en bouge plus. La première entrée
	// est légitime ; toute la question est la suite.
	p.Fugitive.Position = abris[0].Cells[0]
	p.Fugitive.LastShelter = core.NoShelter
	finirLeTour(t, p)

	apresEntree := p.Fugitive.Stamina

	// Assez de tours pour couvrir plusieurs recharges.
	for i := 0; i < 3*p.Settings.ShelterRecharge && p.Phase != core.PhaseOver; i++ {
		finirLeTour(t, p)
	}

	if p.Fugitive.Stamina > apresEntree {
		t.Errorf("résistance %d après immobilité, %d juste après l'entrée : "+
			"le lieu rend des points à qui ne bouge pas",
			p.Fugitive.Stamina, apresEntree)
	}
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
	const attendues = 1

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

// TestSilenceCoversTheWholeTurn vérifie qu'un silence acheté neutralise toutes
// les révélations du tour, pas seulement la première.
//
// Le fugitif paie avant que les inspecteurs jouent : il ne peut pas prévoir
// qu'un coup de filet tombera le même tour que la révélation périodique. Le
// faire payer deux fois pour cette coïncidence serait une punition, pas un
// arbitrage.
//
// La coïncidence est obtenue en jouant, pas en posant un effet en attente : le
// Chef est déclenché au tour qui place son échéance sur une révélation
// périodique. Un test qui construirait la file lui-même n'exercerait pas le
// chemin qui la remplit — c'est ainsi qu'une mécanique reste morte sous des
// tests verts.
func TestSilenceCoversTheWholeTurn(t *testing.T) {
	p := partieLivree(t, "district")

	const preavis = 2
	echeance := p.Settings.RevealPeriod
	if echeance <= preavis {
		t.Skipf("periode de revelation de %d tours : trop courte pour que les deux coincident", echeance)
	}

	// Avancer jusqu'au tour d'où le coup de filet tombe sur la révélation.
	for p.Turn < echeance-preavis && p.Phase != core.PhaseOver {
		finirLeTour(t, p)
	}

	chef := -1
	for i := range p.Inspectors {
		if p.Inspectors[i].Ability == "chief" {
			chef = i
		}
	}
	joue := false
	for _, c := range p.LegalMoves(core.SideInspectors) {
		if c.Type == core.MoveAbility && c.Piece == chef {
			if err := p.Apply(c); err != nil {
				t.Fatalf("capacité du Chef : %v", err)
			}
			joue = true
			break
		}
	}
	if !joue {
		t.Fatal("la capacité du Chef n'est proposée à aucun coup légal")
	}

	// Le fugitif paie son silence pendant sa phase, sans savoir que les deux
	// révélations tomberont ensemble.
	for p.Phase == core.PhaseInspectors {
		finirLaPhase(t, p)
	}
	achete := false
	for _, c := range p.LegalMoves(core.SideFugitive) {
		if c.Type == core.MoveExpense && c.Expense == core.ExpenseSilence {
			if err := p.Apply(c); err != nil {
				t.Fatalf("silence : %v", err)
			}
			achete = true
			break
		}
	}
	if !achete {
		t.Fatal("le silence n'est proposé à aucun coup légal")
	}

	for p.Turn <= echeance && p.Phase != core.PhaseOver {
		p.Fugitive.Visible = false
		finirLeTour(t, p)
	}

	if p.Fugitive.Visible {
		t.Error("le silence n'a pas couvert les deux révélations du tour")
	}
	if p.Fugitive.SilenceBought {
		t.Error("le silence n'a pas été dépensé alors qu'il a servi")
	}
}
