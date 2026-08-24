# Architecture

## Le découpage

```
cmd/filature/     drapeaux, injection des dépendances
internal/noyau/   règles pures — aucune dépendance UI, réseau ni disque
internal/ia/      croyance bayésienne, IA embarquée, pilotage des bots
internal/rendu/   projection isométrique, contrat de formes
internal/stockage/ SQLite : journal, instantanés, poids d'IA
internal/serveur/ WebSocket, hébergement et jonction
internal/greffons/ chargement des manifestes, bac à sable WebAssembly
greffons/         le contenu livré, embarqué dans le binaire par go:embed
schemas/          les contrats publics, versionnés à part
```

## Quatre invariants

Ce sont les seules choses qui coûteraient cher à rétablir plus tard. Tout le
reste est révisable.

### La vue filtrée

Le noyau expose `VuePour(acteur)` et l'interface ne consomme rien d'autre, **y
compris en partie locale**. Sans ça, un joueur lit la position du fugitif dans
le trafic réseau ou dans un fichier de sauvegarde, et le jeu n'a plus d'objet.

Règle de relecture : tout champ ajouté à `Partie` doit être explicitement copié
dans `VuePour`, jamais par recopie de structure. Une omission fait fuiter, un
oubli ne fait qu'afficher moins.

Un bot externe reçoit la même `Vue` que l'interface. Ce n'est pas une politesse :
c'est la même projection, appliquée au même endroit.

### Le déterminisme

Graine explicite portée par l'état, jamais de générateur global, IA comprise.
`noyau.Alea` est le seul générateur autorisé, y compris pour les greffons.

C'est ce qui rend possible le rejeu depuis le journal, la reproduction d'un
défaut, et la comparaison de deux versions d'IA sur les mêmes plateaux.

Piège propre à Go : le parcours d'une `map` n'a pas d'ordre stable. Tout ce qui
influence une décision passe par une tranche triée.

### La réversibilité

`Appliquer` a son `Annuler`, effets de greffons compris. Ce n'est pas un confort
d'interface : c'est ce qui permet à l'IA d'explorer des milliers de positions
sans copier l'état à chaque nœud.

### Aucune coordonnée d'écran dans l'état

L'état ne connaît que colonne et ligne. La conversion isométrique vit dans
`internal/rendu`, et nulle part ailleurs.

## Le journal est la source de vérité

L'instantané n'est qu'un cache de reprise. `Reprendre` rejoue le journal plutôt
que de charger l'instantané : c'est ce qui vérifie en continu que le journal
reste suffisant. Un instantané chargé sans rejeu masquerait le jour où une règle
cesse d'être reproductible.

Conséquence utile : un bot non déterministe reste parfaitement jouable, puisque
le journal enregistre ses **coups** et non son état interne.

## Le vocabulaire d'effets

Une capacité, une dépense ou un mode de jeu se décrit par composition de
primitives — `modifier_portee`, `bloquer_case`, `reveler_traces` — et non par du
code.

Les cinq capacités livrées sont écrites dans ce format dès le premier jour :
c'est le test qui prouve que le vocabulaire suffit. Si l'une d'elles ne s'y
exprime pas, il manque une primitive, ce qui est une décision de contrat public
— pas un cas particulier à coder dans le noyau.

La référence complète du vocabulaire est
[`vocabulaire-effets.md`](vocabulaire-effets.md), et le format des fichiers qui
le portent est dans [`greffons.md`](greffons.md).

Un greffon ne touche jamais `*Partie`. Il produit des `Effet` ou des `Coup`, le
noyau les applique. C'est ce qui fait qu'il ne peut pas lire la zone scellée du
fugitif, et que `Annuler` reste praticable.

## Le rendu

Ebitengine. Un seul langage du noyau au pixel, pas de webview, pas de bundle
JavaScript, et les cibles web, Android et iOS depuis le même code.

Le prix, à connaître : la compilation croisée ne fonctionne que vers Windows et
WebAssembly. Les cibles Linux et macOS se construisent en natif, sur des runners
d'intégration continue. Voir [`construction.md`](construction.md).

Deux vues coexistent :

- **une vue de débogage 2D**, une touche pour l'activer, gardée en permanence.
  Elle sert à superposer les lignes de vue, la carte de croyance de l'IA et les
  cases atteignables — illisible en isométrique ;
- **la vue isométrique**, losanges en rapport 2:1, tri en profondeur par
  `colonne + ligne`.

Le piège de l'isométrique est l'occlusion : les bâtiments en volume sont ce qui
rend la vue jolie et ce qui masque les cases derrière. Or le jeu porte
entièrement sur ce qu'on voit. D'où l'extrusion plafonnée à une demi-case, la
transparence des bâtiments devant un pion, et le marquage au sol des cases
visibles.
