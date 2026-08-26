# Feuille de route

Les numéros font foi : ce sont eux que portent les
`errors.New("à implémenter : étape N")` du code.

```
grep -rn "à implémenter : étape" internal cmd --include="*.go" --exclude="*_test.go" | wc -l
```

Le motif complet et l'exclusion des tests ne sont pas de la coquetterie : le
contrôle qui vérifie que tout stub porte son marqueur cite lui-même la chaîne,
et se comptait donc dans le total.

C'est la mesure d'avancement la plus honnête du projet. Elle descend toute
seule, et elle ne ment pas.

Un mineur de version marque une étape franchie. Voir `CHANGELOG.md` pour la
clause du zéro.

**Les numéros ordonnent les dépendances, pas le calendrier.** Une étape peut
s'attaquer avant celle qui la précède si rien ne l'en empêche. Ils ne se
renumérotent jamais : ce sont eux que portent les marqueurs du code, et les
décaler périmerait quarante-cinq lignes d'un coup. Une étape qui apparaît
s'ajoute donc à la fin, quelle que soit sa place logique.

---

## 1 — Noyau

`internal/core` : état, coups légaux, application et annulation, contacts,
arbitrage, vocabulaire d'effets et file des différés, registre.

Le registre est posé ici, mais rempli à l'étape 8 : lire un manifeste donnerait
au noyau une dépendance disque, et il n'en a aucune. De même, le plateau lui est
remis plutôt que fabriqué — la génération est l'étape 3, et une partie se monte
sans elle sur un plateau d'essai.

Livré quand une partie complète se joue de bout en bout depuis des appels Go,
sans interface, avec les cas limites des règles couverts par des tests : contact
plafonné, extraction interrompue, dernier tour, fugitif bloqué, révélation
contre silence acheté, diagonale refusée par un angle fermé.

Trois des quatre invariants de l'architecture se posent ici — déterminisme,
réversibilité, absence de coordonnée d'écran. **Le quatrième, la vue filtrée,
est l'étape 2**, et c'est le plus coûteux à rétablir : tant qu'elle n'est pas
écrite, rien n'empêche un appelant de lire l'état complet.

## 2 — Vue filtrée

`ViewFor` et sa sérialisation, plus `schemas/view.schema.json`, généré depuis le
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
un prototype antérieur s'est fait prendre : un test compare `IsVisible` et
l'évaluation du fugitif sur les mêmes positions.

## 5 — Boucle de jeu

`cmd/filature` : drapeaux, assemblage des dépendances, boucle de tour.

Jouable, sans rendu : sortie texte, deux joueurs au clavier. C'est le premier
moment où les règles se valident au jeu plutôt qu'en test, et donc le moment où
`docs/regles.md` bougera.

## 6 — Contrat de formes

`internal/render` : lecture, validation, fusion des surcharges partielles.
Toujours pas de pixel à l'écran — le contrat se teste sur des fichiers.

La validation s'applique **au chargement**, plugin local compris, et pas
seulement à `filature validate` : une forme qui déborde masque les cases
voisines, ce qui est un avantage de jeu déguisé en habillage.

Le contenu livré ne revendique pas ses formes, sans quoi le premier plugin à
toucher au fugitif serait refusé comme un conflit — alors que surcharger est
précisément ce qu'un plugin d'apparence vient faire. Deux plugins tiers sur la
même forme restent un conflit, nommé des deux côtés.

## 7 — Vue isométrique

Projection, tri en profondeur, caméra, clic, grain du sol, occlusion. Plus la
vue à plat, écrite en premier.

**Les deux vues sont affichées ensemble**, l'isométrique et la carte à plat dans
un panneau à côté d'elle. Pas un basculement : ce sont deux vues de jeu, et on a
besoin des deux en même temps.

L'isométrique ne montre jamais le plateau entier et n'essaie pas. Elle garde une
échelle de travail confortable et défile en suivant le jeu, y compris sur le
plus petit préréglage. La carte à plat porte la vue d'ensemble que
l'isométrique abandonne — c'est là que le fugitif planifie un itinéraire et que
les inspecteurs répartissent une couverture.

Elle passe par `ViewFor` comme tout le reste : celle des inspecteurs ne montre
pas le fugitif.

**La fenêtre s'adapte, elle n'impose pas.** Redimensionnement libre, plein
écran, et mise à l'échelle selon le facteur du moniteur — sur un écran très
dense, une interface en pixels fixes devient illisible, et c'est le genre de
défaut qu'on ne voit pas sur sa propre machine.

**Les cinq inspecteurs portent leur numéro.** Chaud contre froid sépare les
deux camps même en niveaux de gris, mais cinq bleus nuancés ne se distinguent
pas — et savoir lequel est le Barreur change ce qu'on joue. Le chiffre est
tracé par le moteur, pas déclaré : cinq silhouettes différentes coûteraient
cinq formes à maintenir et brouilleraient le fait qu'un inspecteur doit d'abord
se lire comme un inspecteur.

Deux moyens de jeu, complets l'un comme l'autre :

- **la souris**, qui doit suffire à tout ce qui se joue. Un jeu de plateau se
  manipule en désignant des cases ; exiger le clavier pour une action serait
  une régression par rapport au support qu'on imite ;
- **le clavier**, table `touche -> direction logique` isolée dans un module,
  avec le réglage « les flèches suivent l'écran / suivent le plateau » — en
  isométrique le nord logique part vers le haut-droite, et les deux attentes
  existent chez de vrais joueurs.

## 8 — Persistance et plugins

`internal/storage` : journal, instantané, reprise par nom.
`internal/loader` : chargement des manifestes, empreintes, fusion au registre.

Ensemble parce qu'une sauvegarde porte le manifeste des plugins actifs : une
partie ne se recharge pas sans eux.

Avec `filature valide <chemin>`, qui charge un plugin par **le même code que
le jeu** et liste ses manquements en une fois. Deux validateurs finiraient par
diverger, et c'est alors la partie qui tranche — au pire moment.

**Chaque manquement dit où il est** : fichier, ligne quand elle est connue, et
chemin complet de la clé fautive, avec ce qui était attendu. Une erreur qu'on
doit chercher coûte plus cher que celle qu'on lit.

Elle existe pour deux raisons. Un plugin invalide fait échouer le chargement
entier, ce qui est juste en jeu et détestable en développement : le jeu ne
démarre plus, et il faut deviner lequel des fichiers pèche. Et surtout, un
plugin validé chez son auteur se charge chez n'importe qui — c'est la
condition pour installer le travail d'un inconnu sans crainte. Un code de sortie
suffit à en faire la brique du workflow que `docs/protocole-bot.md` §8 promet
aux auteurs.

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
Le second, les trois chiffres du ressourcement — nombre de lieux, gain et durée
de recharge — posés ensemble et jamais joués.

Ce n'est pas un test qui passe ou échoue : c'est une mesure, et une étape qui
peut renvoyer à `docs/regles.md`.

## 12 — Réseau

`internal/server` : hébergement, jonction, poignée de main des manifestes,
reconnexion par jeton.

Placée tard sans risque : le protocole ne construit rien de neuf, l'hôte fait
déjà autorité et `ViewFor` est en place depuis l'étape 2.

**Le retour en arrière s'arrête ici.** `Undo` existe pour l'IA et se prête à
un bouton en solo, où l'adversaire est une machine. En réseau deux joueurs
s'affrontent, et défaire un coup reviendrait à rejouer celui d'en face après
l'avoir vu.

## 13 — Plugins exécutables

Bac à sable wazero, générateurs de plateau et IA tierces. Hôtes sans horloge,
sans entropie système, sans disque, sans réseau.

Dernière parce que c'est le seul niveau de plugin dont la valeur est étroite :
les trois autres couvrent l'essentiel, et un bot en processus séparé fait déjà
ce dont un contributeur a besoin.

## 14 — Publication

Matrice de compilation multiplateforme, archives, sommes, attestation, version
en brouillon. La matrice et ses contraintes sont dans
[`docs/construction.md`](docs/construction.md) — c'est l'étape la plus
particulière du projet, puisque le moteur ne se compile en croisé que vers
Windows et WebAssembly.

## 15 — Interface hors-jeu

Menus, écran de préférences, `Échap` pour l'ouvrir, et la persistance des
réglages. Plus le chargement des langues et le sélecteur qui va avec.

Les libellés ne sont jamais dans le code : ils viennent d'un dictionnaire, et un
plugin de langue en fournit un. Le format se pose à l'étape 8 avec les autres
contrats de plugin ; c'est ici qu'il est consommé.

**L'écran des plugins montre les refusés autant que les actifs**, chacun avec
sa raison. Un plugin absent de la liste sans explication laisse son auteur
deviner, et le journal n'est pas un endroit où il pensera à regarder.

Numérotée en dernier sans être la dernière à faire : rien de tout cela ne
dépend du réseau ni des plugins exécutables, et un écran de réglages sera
utile bien avant. C'est aussi ici que se tranche `ebitenui` contre des widgets
écrits à la main — la question se règle en écrivant le premier écran, pas
avant.

---

## Hors périmètre v1

**Le plateau infini.** L'interface `Board` est conçue pour l'accueillir — la
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
