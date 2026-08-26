// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package core

import "testing"

// TestPrereglagesJouables vérifie que chaque préréglage livré produit un
// plateau.
//
// Un préréglage qu'on propose dans un sélecteur et qui échoue à la génération
// serait pire qu'un préréglage absent : le joueur le choisit, et la partie ne
// démarre pas.
func TestPrereglagesJouables(t *testing.T) {
	for _, p := range Prereglages() {
		t.Run(p.Cle, func(t *testing.T) {
			if err := p.Parametres.Valider(); err != nil {
				t.Fatalf("paramètres refusés : %v", err)
			}

			// Plusieurs graines : un préréglage qui ne marcherait qu'une fois
			// sur deux n'est pas jouable.
			for graine := int64(1); graine <= 30; graine++ {
				b, _, err := Generer(graine, p.Parametres)
				if err != nil {
					t.Fatalf("graine %d : %v", graine, err)
				}
				if len(b.Zones()) != p.Parametres.Zones {
					t.Fatalf("graine %d : %d zones, attendu %d",
						graine, len(b.Zones()), p.Parametres.Zones)
				}
			}
		})
	}
}

// TestPrereglagesCouvrentLesTroisTailles vérifie ce que docs/regles.md §11
// annonce : des préréglages 21, 31 et 41.
func TestPrereglagesCouvrentLesTroisTailles(t *testing.T) {
	vus := map[int]bool{}
	for _, p := range Prereglages() {
		vus[p.Parametres.Cote] = true
	}

	for _, cote := range []int{21, 31, 41} {
		if !vus[cote] {
			t.Errorf("aucun préréglage en %d×%d", cote, cote)
		}
	}
}

// TestPrereglagesOrdonnesEtDistincts vérifie qu'un sélecteur les affiche du
// plus petit au plus grand, sans doublon.
func TestPrereglagesOrdonnesEtDistincts(t *testing.T) {
	liste := Prereglages()
	cles := map[string]bool{}

	for i, p := range liste {
		if p.Cle == "" {
			t.Errorf("préréglage %d sans clé", i)
		}
		if cles[p.Cle] {
			t.Errorf("clé %s en double", p.Cle)
		}
		cles[p.Cle] = true

		if i > 0 && p.Parametres.Cote <= liste[i-1].Parametres.Cote {
			t.Errorf("%s n'est pas plus grand que %s", p.Cle, liste[i-1].Cle)
		}
	}
}

// TestPrereglagePorteeEtDureeSuiventLaTaille vérifie que les deux valeurs
// dérivées le restent.
//
// Sur un petit plateau, une portée de 8 verrait presque tout le terrain et
// quarante tours laisseraient au fugitif le temps d'en faire trois fois le
// tour : les figer viderait les petits préréglages de leur intérêt.
func TestPrereglagePorteeEtDureeSuiventLaTaille(t *testing.T) {
	for _, p := range Prereglages() {
		cote := p.Parametres.Cote

		if attendue := max(PorteeMin, cote/5); p.Parametres.Portee != attendue {
			t.Errorf("%s : portée %d, attendu %d", p.Cle, p.Parametres.Portee, attendue)
		}
		// « Environ le côté », dit la règle, et non exactement : la Ville garde
		// ses quarante tours historiques sur un plateau de quarante et un.
		if ecart := abs(p.Parametres.Tours - cote); ecart > 2 {
			t.Errorf("%s : %d tours pour un côté de %d", p.Cle, p.Parametres.Tours, cote)
		}
		if p.Parametres.DebutEtranglement >= p.Parametres.Tours {
			t.Errorf("%s : étranglement au tour %d sur %d",
				p.Cle, p.Parametres.DebutEtranglement, p.Parametres.Tours)
		}
	}
}

// TestPrereglageParCle vérifie la recherche, et qu'une clé inconnue le dit.
func TestPrereglageParCle(t *testing.T) {
	p, trouve := PrereglagePar("ville")
	if !trouve {
		t.Fatal("le préréglage par défaut est introuvable")
	}
	if p.Parametres.Cote != ParametresDefaut().Cote {
		t.Error("« ville » ne correspond pas aux paramètres par défaut")
	}

	if _, trouve := PrereglagePar("inexistant"); trouve {
		t.Error("une clé inconnue est acceptée")
	}
}

// TestPrereglageDefautEstLaVille vérifie que le défaut du jeu figure bien parmi
// les préréglages proposés, plutôt que d'être un quatrième jeu de valeurs.
func TestPrereglageDefautEstLaVille(t *testing.T) {
	defaut := ParametresDefaut()

	for _, p := range Prereglages() {
		if p.Parametres == defaut {
			return
		}
	}
	t.Error("les paramètres par défaut ne correspondent à aucun préréglage")
}
