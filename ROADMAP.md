# Feuille de route

Les numéros font foi : ce sont eux que portent les
`errors.New("à implémenter : étape N")` du code.

```
grep -rn "à implémenter" internal cmd | wc -l
```

C'est la mesure d'avancement la plus honnête du projet. Elle descend toute
seule, et elle ne ment pas.

Un mineur de version marque une étape franchie. Voir `CHANGELOG.md` pour la
clause du zéro.

---

## 1 — Noyau

`internal/noyau` : état, coups légaux, application et annulation, contacts,
arbitrage, vocabulaire d'effets et file des différés, registre.

Livré quand une partie complète se joue de bout en bout depuis des appels Go,
sans interface, avec les cas limites de `docs/regles.md` §14 couverts par des
tests.

C'est ici que se posent les quatre invariants de l'architecture. Les rétablir
plus tard coûterait cher ; tout le reste est révisable.

## 2 — Vue filtrée

`VuePour` et sa sérialisation, plus `schemas/vue.schema.json`, généré depuis le
Go et non écrit à la main.

Séparée de l'étape 1 pour une raison : c'est l'invariant le plus facile à
enfreindre sans s'en apercevoir. Un test dédié vérifie qu'aucune vue
inspecteurs ne porte la position cachée du fugitif ni sa zone scellée.

## 3 — Génération de plateau

Trame de rues, îlots, perçages, impasses, zones d'extraction. Validation de
connexité, taux de rues, atteignabilité des zones.

Livré quand une graine donnée produit toujours le même plateau, vérifié octet
pour octet sur les trois systèmes.

## 4 — Vision

Table précalculée par le terrain, occlusion des pions et des barrages appliquée
à la lecture.

Séparée de l'étape 3 parce que le blocage de vue entre inspecteurs interdit de
tout précalculer, et que la distinction Tchebychev / Manhattan est le piège où
un prototype antérieur s'est fait prendre : un test compare `EstVisible` et
l'évaluation du fugitif sur les mêmes positions.

## 5 — Boucle de jeu

`cmd/filature` : drapeaux, assemblage des dépendances, boucle de tour.

Jouable, sans rendu : sortie texte, deux joueurs au clavier. C'est le premier
moment où les règles se valident au jeu plutôt qu'en test, et donc le moment où
`docs/regles.md` bougera.

## 6 — Contrat de formes

`internal/rendu` : lecture, validation, fusion des surcharges partielles.
Toujours pas de pixel à l'écran — le contrat se teste sur des fichiers.

## 7 — Vue isométrique

Projection, tri en profondeur, caméra, clic, grain du sol, occlusion. Plus la
vue de débogage 2D, écrite en premier et gardée en permanence.

Puis le clavier : table `touche -> direction logique`, isolée dans un module,
avec le réglage « les flèches suivent l'écran / suivent le plateau ».

## 8 — Persistance et greffons

`internal/stockage` : journal, instantané, reprise par nom.
`internal/greffons` : chargement des manifestes, empreintes, fusion au registre.

Ensemble parce qu'une sauvegarde porte le manifeste des greffons actifs : une
partie ne se recharge pas sans eux.

Livré quand le test central passe : partie jouée, journal, rejeu, état
identique octet pour octet.

## 9 — IA inspecteurs et bots

Carte de croyance bayésienne, évaluation pondérée, niveaux.
Pilotage des bots externes en JSON Lines.

Ensemble parce que l'IA livrée parle le protocole de bot : c'est ce qui prouve
qu'il suffit. Le bot minimal de `docs/protocole-bot.md` sert de test de
conformité.

## 10 — IA fugitif

Le miroir : maximiser la masse de croyance résiduelle, arbitrer les dépenses,
choisir quand tuer. Mode tout-IA, qui est aussi le partenaire d'entraînement.

## 11 — Équilibrage

Simulation en masse, taux de victoire par camp et par motif de fin, ajustement
évolutionnaire des poids.

Le premier paramètre à corriger sera le nombre de pions déplaçables par tour.
Le second, le coût du meurtre.

Ce n'est pas un test qui passe ou échoue : c'est une mesure, et une étape qui
peut renvoyer à `docs/regles.md`.

## 12 — Réseau

`internal/serveur` : hébergement, jonction, poignée de main des manifestes,
reconnexion par jeton.

Placée tard sans risque : le protocole ne construit rien de neuf, l'hôte fait
déjà autorité et `VuePour` est en place depuis l'étape 2.

## 13 — Greffons exécutables

Bac à sable wazero, générateurs de plateau et IA tierces. Hôtes sans horloge,
sans entropie système, sans disque, sans réseau.

Dernière parce que c'est le seul niveau de greffon dont la valeur est étroite :
les trois autres couvrent l'essentiel, et un bot en processus séparé fait déjà
ce dont un contributeur a besoin.

## 14 — Publication

Matrice de compilation multiplateforme, archives, sommes, attestation, version
en brouillon. La matrice et ses contraintes sont dans
[`docs/construction.md`](docs/construction.md) — c'est l'étape la plus
particulière du projet, puisque le moteur ne se compile en croisé que vers
Windows et WebAssembly.

---

## Hors périmètre v1

**Le plateau infini.** L'interface `Plateau` est conçue pour l'accueillir — la
génération par tuiles s'y substitue sans qu'aucune règle change — mais la v1
est bornée. La mécanique d'extraction est ce qui rendrait l'infini jouable ;
elle ne le rend pas nécessaire.

**Le plateau torique.** Même remarque, et plus simple à livrer que l'infini.

**Le son.** Rien n'est prévu, et le catalogue n'accepterait de toute façon aucun
fichier audio.

**Les magasins d'applications.** Ebitengine ouvre Android et iOS, mais la
distribution y demande un travail propre — signature, cycle de publication —
qui n'a rien à voir avec le jeu.

**Un serveur dédié sans écran.** Le mode réseau est de pair à pair. Un binaire
serveur exigerait une étiquette de compilation excluant le rendu ; le noyau
n'importe rien de graphique, donc c'est faisable, mais rien ne le réclame.

**La rotation de caméra.** Les quatre orientations compliqueraient la table du
clavier sans rien apporter au jeu.
