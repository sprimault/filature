// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package quality

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// quantificateurs sont les tournures qui affirment un nombre sans le mesurer.
//
// Ce sont celles qui ont produit une note de version fausse : « un cinquième des
// fonctions du dépôt l'enfreignait, toutes des helpers de test » — un chiffre
// venu d'un comptage partiel, et un « toutes » venu de rien. Les fractions et
// les totalisations en font partie ; un nombre écrit en chiffres n'y est pas, la
// mesure qui le produit étant en général citée avec lui.
// tournures énumère les formes refusées, une par ligne plutôt qu'en alternatives
// d'une expression : c'est ce qui permet à un test de les éprouver une par une.
//
// Elles vivaient dans le motif, où rien ne pouvait les atteindre — et deux
// d'entre elles n'ont jamais rien reconnu.
var tournures = []string{
	"un tiers", "un quart", "un cinquième", "un dixième",
	"la moitié", "la totalité",
	"tous les", "toutes les",
	"chacun des", "chacune des",
	"aucun des", "aucune des",
	"le seul", "la seule",
	"systématiquement",
}

// quantificateurs reconnaît l'une des tournures, isolée dans son texte.
//
// La borne de fin est « pas une lettre ASCII » et non `\b` : RE2 n'offre que la
// limite de mot ASCII, si bien que « moitié » et « totalité », qui finissent sur
// un é, n'en produisaient jamais aucune. Les deux étaient mortes à la naissance,
// et le contrôle a laissé passer « la moitié » deux fois dans une section
// publiée avant qu'une relecture le remarque.
var quantificateurs = regexp.MustCompile(
	`\b(` + strings.Join(tournures, "|") + `)([^a-zA-Z]|$)`)

// TestChangelogAvoidsUnheldQuantifiers refuse une affirmation de nombre dans la
// section non publiée du journal.
//
// Le projet applique une règle : une affirmation de nombre s'adosse à un test,
// ou perd son quantificateur. Elle est outillée sur les godoc, sur les règles,
// sur les README, sur la palette — et le CHANGELOG n'était gardé par rien, alors
// que c'est le seul texte du dépôt qui devient automatiquement une page
// publique : la publication en extrait les notes sans que personne les relise à
// ce moment-là.
//
// Seule la section non publiée est contrôlée. Les sections déjà parues sont un
// historique : les corriger se fait, mais rétroactivement et à la main, comme
// pour la note de la 0.6.1 que ce contrôle aurait arrêtée.
func TestChangelogAvoidsUnheldQuantifiers(t *testing.T) {
	section := sectionNonPubliee(t)
	if section == "" {
		t.Skip("aucune section non publiée : rien à contrôler")
	}

	for _, ligne := range strings.Split(section, "\n") {
		for _, trouvaille := range quantificateurs.FindAllString(ligne, -1) {
			t.Errorf("« %s » dans les notes non publiées : %s\n"+
				"Une affirmation de nombre s'adosse à un test, ou perd son quantificateur — "+
				"ces notes partent telles quelles sur la page des versions.",
				trouvaille, strings.TrimSpace(ligne))
		}
	}
}

// sectionNonPubliee rend le corps de la section « Non publié », vide s'il n'y en
// a pas — ce qui est le cas juste après une publication.
func sectionNonPubliee(t *testing.T) string {
	t.Helper()

	contenu, err := os.ReadFile(filepath.Join(racine, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}

	_, apres, trouve := strings.Cut(string(contenu), "## [Non publié]")
	if !trouve {
		return ""
	}
	section, _, _ := strings.Cut(apres, "\n## [")
	return section
}

// TestEveryQuantifierIsRecognised vérifie que chaque tournure de la table est
// effectivement reconnue par le motif.
//
// C'est le contrôle du contrôle, et il n'est pas superflu : deux tournures sur
// quinze ne reconnaissaient rien, et le défaut n'a pu se voir qu'en relisant le
// motif — jamais en le lançant, puisqu'une tournure morte laisse simplement
// passer. Un garde-fou qu'on ne peut pas éprouver donne une assurance que rien
// ne porte.
func TestEveryQuantifierIsRecognised(t *testing.T) {
	for _, tournure := range tournures {
		for _, phrase := range []string{
			"le texte dit " + tournure + " des fonctions",
			"il en couvre " + tournure + ".",
			tournure + " y passent",
		} {
			if !quantificateurs.MatchString(phrase) {
				t.Errorf("%q n'est pas reconnue dans %q", tournure, phrase)
			}
		}
	}
}

// TestQuantifierNeedsAWholeWord vérifie que le motif ne mord pas au milieu d'un
// mot, ce que la borne de fin doit empêcher.
func TestQuantifierNeedsAWholeWord(t *testing.T) {
	for _, phrase := range []string{
		"la seuleté n'est pas un mot",
		"un quartier de la ville",
		"tous lesquels sont là",
	} {
		if quantificateurs.MatchString(phrase) {
			t.Errorf("%q est refusée alors qu'elle ne porte aucun quantificateur", phrase)
		}
	}
}
