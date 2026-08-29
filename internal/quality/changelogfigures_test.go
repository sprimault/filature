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
var quantificateurs = regexp.MustCompile(
	`\b(un (?:tiers|quart|cinquième|dixième)|la (?:moitié|totalité)|toutes? les|tous les|chacune? des|aucune? des|le seul|la seule|systématiquement)\b`)

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
