// Copyright 2026 Stéphane Primault <sprimault@users.noreply.github.com>
// SPDX-License-Identifier: Apache-2.0

package noyau

// Vue est ce qu'un camp a le droit de savoir. Le noyau n'expose rien d'autre à
// l'interface, y compris en partie locale.
//
// C'est l'invariant le plus coûteux à rétrofiter et le plus facile à poser
// maintenant : si l'interface consomme l'état complet, un joueur lit la
// position du fugitif dans le trafic réseau ou dans les outils de
// développement du navigateur, et le jeu n'a plus d'objet.
type Vue struct {
	Acteur     Acteur     `json:"acteur"`
	Tour       int        `json:"tour"`
	Phase      Phase      `json:"phase"`
	Parametres Parametres `json:"parametres"`

	// Rues est la portion de plateau connue du client. Sur plateau borné
	// c'est tout ; sur plateau infini, ce sera ce qui a été exploré.
	Rues     []Position `json:"rues"`
	Zones    []Zone     `json:"zones"`
	Barrages []Position `json:"barrages"`

	Inspecteurs []Inspecteur `json:"inspecteurs"`

	// PositionFugitif n'est renseigné que pour le camp fugitif, ou pour les
	// inspecteurs quand il est visible ou révélé.
	PositionFugitif *Position `json:"position_fugitif,omitempty"`
	Resistance      int       `json:"resistance"`
	ZoneScellee     *int      `json:"zone_scellee,omitempty"`

	// TracesConnues ne contient que ce que les inspecteurs ont effectivement
	// découvert. Le fugitif, lui, voit les siennes.
	TracesConnues map[string]Trace `json:"traces_connues"`

	// Scenes est identique pour les deux camps : un meurtre est public, c'est
	// ce que le fugitif paie. Ne jamais la filtrer par acteur.
	Scenes []Scene `json:"scenes"`

	CasesVisibles   []Position `json:"cases_visibles"`
	CoupsLegaux     []Coup     `json:"coups_legaux"`
	ProchaineReveal int        `json:"prochaine_revelation"`
	SilencePaye     bool       `json:"silence_paye"`
	ZonesAnnoncees  []int      `json:"zones_annoncees"`

	// EffetsAnnonces ne porte que les differer déclarés avec annonce, et les
	// porte à l'identique pour les deux camps. Un differer sans annonce reste
	// invisible jusqu'à sa résolution, sinon le champ le trahirait.
	EffetsAnnonces []EffetEnAttente `json:"effets_annonces"`

	Resultat *Resultat `json:"resultat,omitempty"`
}

// VuePour projette l'état pour un camp.
//
// La règle de relecture : tout champ ajouté à Partie doit être explicitement
// copié ici, jamais par recopie de structure. Une omission fait fuiter, un
// oubli ne fait qu'afficher moins.
func (p *Partie) VuePour(a Acteur) Vue {
	return Vue{}
}
