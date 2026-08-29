// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// majMesures relance les mesures de génération et réécrit le bloc que
// docs/regles.md leur réserve.
//
// Jamais en intégration continue, jamais automatique : c'est un travail long
// qu'on lance quand on se pose une question, comme -maj-notices pour les
// licences. Ici plutôt que dans internal/quality parce qu'un outil vit où vit
// ce qu'il observe — draw et validate ne sont pas exportées, et les exporter
// ferait payer à tout le code un besoin qui n'appartient qu'à la mesure.
var majMesures = flag.Bool("maj-mesures", false, "refaire les mesures de génération et réécrire docs/regles.md")

// grainesMesurees est l'échantillon de chaque mesure. Assez large pour que la
// queue basse d'un préréglage apparaisse : à quatre cents graines, les Quartiers
// sous cinq impasses se comptaient sur les doigts d'une main.
const grainesMesurees = 2000

// grainesEssais est l'échantillon des seuls tirages par plateau, plus court :
// cette mesure-là génère jusqu'à obtenir un plateau valide, donc elle coûte ce
// que Generate coûte à un joueur, et non ce que coûte un tirage.
//
// Le bloc annonce les deux nombres. Il n'en annonçait qu'un, et une colonne sur
// quatre reposait donc sur un quart de l'échantillon publié.
const grainesEssais = 500

// Les bornes du bloc réécrit dans docs/regles.md. Ce sont des commentaires
// HTML : Markdown ne les rend pas, et un lecteur du document ne les voit pas.
const (
	debutDesMesures = "<!-- mesures:début -->"
	finDesMesures   = "<!-- mesures:fin -->"
)

// TestRulesMeasurementsAreReproducible refait les mesures de génération que
// docs/regles.md cite, et réécrit le bloc qui les porte.
//
// **Aucun contrôle de fraîcheur, et c'est délibéré.** Un test qui rougirait
// quand le taux de rejet bouge rougirait au premier changement volontaire de
// génération : ce sont des conséquences, pas des décisions. Ce qui garde une
// décision — les bornes du taux, le nombre d'inspecteurs — est ailleurs, exercé
// par les tests qui s'y rapportent. Ici on relance et on relit.
//
// Le bloc porte sa date, écrite par la mesure elle-même. Une date saisie à la
// main peut mentir sur l'ancienneté d'un chiffre, et c'est exactement ce qui
// laisse un nombre juste vieillir sans qu'on s'en aperçoive.
func TestRulesMeasurementsAreReproducible(t *testing.T) {
	if !*majMesures {
		t.Skip("relancer avec -maj-mesures pour refaire les mesures et réécrire docs/regles.md")
	}
	if *majAttendus {
		t.Fatal("-maj-attendus et -maj-mesures ensemble : deux réécritures coûteuses dans " +
			"un seul diff, qui ne se relit alors ni pour l'une ni pour l'autre")
	}

	var bloc strings.Builder
	fmt.Fprintf(&bloc, "%s\n", debutDesMesures)
	fmt.Fprintf(&bloc, "*Mesuré le %s sur %d graines par préréglage, %d pour les tirages.*\n\n",
		time.Now().Format("2006-01-02"), grainesMesurees, grainesEssais)
	fmt.Fprintf(&bloc, "| Préréglage | Trame | Praticable | Tirages par plateau | Impasses |\n")
	fmt.Fprintf(&bloc, "|---|---|---|---|---|\n")

	for _, pre := range Presets() {
		m := mesurer(pre.Settings)
		fmt.Fprintf(&bloc, "| %s | %d %% en médiane, de %d à %d | %d %% en médiane, %d au plus | %s en moyenne, %d au pire | %d au moins, %s %% sous %d |\n",
			pre.Key, m.trameMediane, m.trameMin, m.trameMax,
			m.praticableMedian, m.praticableMax,
			virgule(m.essaisMoyens, 2), m.essaisPire,
			m.impassesMin, virgule(m.partSousPlancher, 1), m.plancher)
	}
	fmt.Fprintf(&bloc, "%s", finDesMesures)

	remplacerBloc(t, bloc.String())
	t.Log("docs/regles.md réécrit : relire le diff avant de le livrer")
}

// virgule rend un décimal comme l'écrit le document qui va le porter.
func virgule(v float64, decimales int) string {
	return strings.Replace(strconv.FormatFloat(v, 'f', decimales, 64), ".", ",", 1)
}

// mesures rassemble ce qu'une passe observe sur un préréglage.
type mesures struct {
	trameMediane, trameMin, trameMax int
	praticableMedian, praticableMax  int
	essaisMoyens                     float64
	essaisPire                       int
	impassesMin, plancher            int
	partSousPlancher                 float64
}

// mesurer rejoue la génération sur l'échantillon et rend ce qu'elle produit.
//
// Le taux de la trame se prend entre les impasses et les zones, là où validate
// le lit : après les blocs, il ne se retrouve plus — les percements recouvrent
// des cases que la trame avait déjà ouvertes.
func mesurer(p Settings) mesures {
	var trames, praticables, impasses []int
	var essais, pire int
	// La cible de l'étape 4, et non le quart du côté : carveDeadEnds en creuse
	// une par carré de huit cases, et grid.go écarte nommément la règle
	// proportionnelle au côté, qui rendrait la Ville deux fois moins piégeuse
	// qu'un Quartier. Le seuil publié valait 5, 7 et 10 quand la cible en demande
	// 6, 15 et 26, si bien que la colonne ne mesurait rien.
	plancher := p.Size * p.Size / AreaPerDeadEnd
	var sousPlancher int

	for g := int64(1); g <= grainesMesurees; g++ {
		b, trame := draw(g, p)
		total := p.Size * p.Size

		// Après le refus et non avant : le bloc publie ce que la génération
		// retient, pas ce qu'elle jette. Compté en amont, il annonçait une
		// trame allant jusqu'à 56 % pour une borne haute de 50, ce qui n'a de
		// sens que si l'on sait que les plateaux rejetés y sont — et la légende
		// disait l'inverse.
		if b.validate(p, trame) != nil {
			continue
		}
		trames = append(trames, trame*100/total)
		praticables = append(praticables, b.countStreets()*100/total)

		n := deadEnds(b)
		impasses = append(impasses, n)
		if n < plancher {
			sousPlancher++
		}
	}

	for g := int64(1); g <= grainesEssais; g++ {
		n := 1
		for {
			b, trame := draw(g+int64(n-1), p)
			if b.validate(p, trame) == nil {
				break
			}
			n++
		}
		essais += n
		pire = max(pire, n)
	}

	sort.Ints(trames)
	sort.Ints(praticables)
	sort.Ints(impasses)

	return mesures{
		trameMediane:     trames[len(trames)/2],
		trameMin:         trames[0],
		trameMax:         trames[len(trames)-1],
		praticableMedian: praticables[len(praticables)/2],
		praticableMax:    praticables[len(praticables)-1],
		essaisMoyens:     float64(essais) / grainesEssais,
		essaisPire:       pire,
		impassesMin:      impasses[0],
		plancher:         plancher,
		partSousPlancher: float64(sousPlancher) * 100 / float64(len(impasses)),
	}
}

// remplacerBloc réécrit ce que docs/regles.md réserve aux mesures.
//
// **Échoue franchement quand les marqueurs manquent.** Ils couplent un document
// de prose à du code Go : quelqu'un qui reformate le fichier peut en abîmer un
// sans le voir, et le harnais ajouterait alors ses mesures à la fin, ou ne
// ferait rien. Les deux laisseraient croire à une mise à jour qui n'a pas eu
// lieu — c'est le défaut qu'on a payé sur les effets différés avalés en
// silence.
func remplacerBloc(t *testing.T, bloc string) {
	t.Helper()

	chemin := filepath.Join("..", "..", "docs", "regles.md")
	brut, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}

	texte := string(brut)
	debut := strings.Index(texte, debutDesMesures)
	fin := strings.Index(texte, finDesMesures)
	switch {
	case debut < 0:
		t.Fatalf("%s ne porte plus %q : le bloc des mesures a disparu ou son marqueur a été abîmé",
			chemin, debutDesMesures)
	case fin < 0:
		t.Fatalf("%s ne porte plus %q : le bloc des mesures n'est pas refermé",
			chemin, finDesMesures)
	case fin < debut:
		t.Fatalf("%s porte les marqueurs des mesures dans le désordre", chemin)
	}

	neuf := texte[:debut] + bloc + texte[fin+len(finDesMesures):]
	if err := os.WriteFile(chemin, []byte(neuf), 0o600); err != nil {
		t.Fatal(err)
	}
}
