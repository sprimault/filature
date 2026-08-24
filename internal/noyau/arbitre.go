// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

// Resultat clôt la partie. Motif sert à l'affichage et aux statistiques
// d'équilibrage : savoir si les inspecteurs gagnent par épuisement ou par
// blocage change ce qu'il faut corriger.
type Resultat struct {
	Vainqueur Acteur `json:"vainqueur"`
	Motif     string `json:"motif"`
	Tour      int    `json:"tour"`
}

// Les quatre motifs de fin. Ils partent dans le journal et dans le message
// `fin` du protocole de bot : les renommer périmerait les parties enregistrées.
const (
	MotifExtraction  = "extraction"
	MotifResistance  = "resistance_epuisee"
	MotifBlocage     = "fugitif_bloque"
	MotifTempsEcoule = "temps_ecoule"
)

// Resultat teste les conditions de fin. Le second retour distingue « partie en
// cours » de « match nul », qui n'existe pas ici.
func (p *Partie) Resultat() (Resultat, bool) {
	// à implémenter : étape 1
	return Resultat{}, false
}

// extractionEnCours dit si le fugitif tient sa zone. Une zone occupée par un
// inspecteur est neutralisée : le compte ne démarre pas et s'interrompt s'il
// était engagé.
func (p *Partie) extractionEnCours() bool {
	// à implémenter : étape 1
	return false
}

// zonesAFermer renvoie les zones que l'étranglement vise à ce tour. L'ordre
// vient de la graine.
//
// Elle ne ferme rien : elle donne la cadence et la cible au mode etranglement
// de greffons/base, qui porte le préavis et la fermeture. Un greffon qui
// remplace ce mode change ce qui se passe, jamais quand.
func (p *Partie) zonesAFermer() (maintenant, annoncees []int) {
	// à implémenter : étape 1
	return nil, nil
}
