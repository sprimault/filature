# Architecture

## Le découpage

```
cmd/filature/     drapeaux, injection des dépendances
internal/core/    règles pures — aucune dépendance UI, réseau ni disque
internal/ai/      croyance bayésienne, IA embarquée, pilotage des bots
internal/render/  projection isométrique, contrat de formes
internal/storage/ SQLite : journal, instantanés, poids d'IA
internal/server/  WebSocket, hébergement et jonction
internal/loader/  chargement des manifestes, bac à sable WebAssembly
internal/text/    rendu d'une vue en caractères, saisie d'un coup
plugins/          le contenu livré, embarqué dans le binaire par go:embed
schemas/          les contrats publics, versionnés à part
```

## Quatre invariants

Ce sont les seules choses qui coûteraient cher à rétablir plus tard. Tout le
reste est révisable.

### La vue filtrée

Le noyau expose `ViewFor(side)` et l'interface ne consomme rien d'autre, **y
compris en partie locale**. Sans ça, un joueur lit la position du fugitif dans
le trafic réseau ou dans un fichier de sauvegarde, et le jeu n'a plus d'objet.

Règle de relecture : tout champ ajouté à `Game` doit être explicitement copié
dans `ViewFor`, jamais par recopie de structure. Une omission fait fuiter, un
oubli ne fait qu'afficher moins.

Un bot externe reçoit la même `View` que l'interface. Ce n'est pas une politesse :
c'est la même projection, appliquée au même endroit.

### Le déterminisme

Graine explicite portée par l'état, jamais de générateur global, IA comprise.
`core.Alea` est le seul générateur autorisé, y compris pour les plugins.

C'est ce qui rend possible le rejeu depuis le journal, la reproduction d'un
défaut, et la comparaison de deux versions d'IA sur les mêmes plateaux.

Piège propre à Go : le parcours d'une `map` n'a pas d'ordre stable. Tout ce qui
influence une décision passe par une tranche triée.

### La réversibilité

`Apply` a son `Undo`, effets de plugins compris. Ce n'est pas un confort
d'interface : c'est ce qui permet à l'IA d'explorer des milliers de positions
sans copier l'état à chaque nœud.

### Aucune coordonnée d'écran dans l'état

L'état ne connaît que colonne et ligne. La conversion isométrique vit dans
`internal/render`, et nulle part ailleurs.

## Le journal est la source de vérité

L'instantané n'est qu'un cache de reprise. `Resume` rejoue le journal plutôt
que de charger l'instantané : c'est ce qui vérifie en continu que le journal
reste suffisant. Un instantané chargé sans rejeu masquerait le jour où une règle
cesse d'être reproductible.

Conséquence utile : un bot non déterministe reste parfaitement jouable, puisque
le journal enregistre ses **coups** et non son état interne.

## Le vocabulaire d'effets

Une capacité, une dépense ou un mode de jeu se décrit par composition de
primitives — `change_range`, `block_cell`, `reveal_trails` — et non par du
code.

Les cinq capacités livrées sont écrites dans ce format dès le premier jour :
c'est le test qui prouve que le vocabulaire suffit. Si l'une d'elles ne s'y
exprime pas, il manque une primitive, ce qui est une décision de contrat public
— pas un cas particulier à coder dans le noyau.

La référence complète du vocabulaire est
[`vocabulaire-effets.md`](vocabulaire-effets.md), et le format des fichiers qui
le portent est dans [`plugins.md`](plugins.md).

Un plugin ne touche jamais `*Game`. Il produit des `Effect` ou des `Move`, le
noyau les applique. C'est ce qui fait qu'il ne peut pas lire la zone scellée du
fugitif, et que `Undo` reste praticable.

## Le rendu

Ebitengine. Un seul langage du noyau au pixel, pas de webview, pas de bundle
JavaScript, et les cibles web, Android et iOS depuis le même code.

Le prix, à connaître : la compilation croisée ne fonctionne que vers Windows et
WebAssembly. Les cibles Linux et macOS se construisent en natif, sur des runners
d'intégration continue. Voir [`construction.md`](construction.md).

Deux vues **affichées ensemble**, l'isométrique et la carte à plat à côté
d'elle. Ce n'est pas un basculement : chacune fait ce que l'autre ne sait pas
faire, et on a besoin des deux en même temps.

- **la vue isométrique** occupe l'essentiel de la fenêtre. Losanges en rapport
  2:1, tri en profondeur par `colonne + ligne`. **Elle ne montre jamais le
  plateau entier et n'essaie pas** : elle défile en suivant le jeu ;
- **la carte à plat** tient dans un panneau latéral, en permanence. Elle porte
  la vue d'ensemble, que l'isométrique abandonne : le fugitif y planifie un
  itinéraire vers une zone, les inspecteurs y répartissent une couverture. Elle
  superpose aussi les lignes de vue, la carte de croyance de l'IA et les cases
  atteignables, illisibles en isométrique.

**L'échelle isométrique est un intervalle fermé, fixé par la règle et non par le
confort.** Par le bas, `MinRenderScale`. Par le haut, la vue doit contenir en
entier le champ du pion sélectionné — `Span(2*portée+1)` donne la place qu'il
demande. Le pire cas prévu, une fenêtre de 1280 sur le plus grand préréglage,
donne 55 pixels par case : le plafond commande partout et le plancher ne se
déclenche jamais en jeu normal.

La portée qui entre dans ce calcul est celle du **préréglage**, pas celle du
tour. Une capacité qui double la vue ferait autrement sauter l'échelle d'un
tiers à son déclenchement, puis la rendrait au tour suivant.

**La garantie porte sur le pion sélectionné, pas sur tous.** Un inspecteur au
bord du panneau a sa ligne de vue tronquée à l'écran, et à cinq pions dispersés
sur quarante et une cases c'est le cas courant. D'où le rôle le plus important
de la carte à plat : elle porte la couverture d'ensemble du camp inspecteurs,
qui est leur information la plus stratégique et que l'isométrique ne peut pas
montrer.

Le panneau vaut 26 % de la largeur, borné entre 320 et 520 pixels — dimensionné
par ce qu'il doit montrer, soit douze pixels par case sur le plus grand
préréglage. Cette couverture s'y rend **en teinte de fond de case, jamais en
halo autour d'un pion** : à huit pixels par case, un halo déborderait sur les
huit voisines.

**La carte à plat passe par `ViewFor` comme tout le reste.** C'est une vue de
plus, pas un accès privilégié : celle des inspecteurs ne montre pas le fugitif,
et une minicarte permanente ne donne donc aucun avantage. L'oublier ferait
fuiter la position par le chemin le plus discret du programme.

Le piège de l'isométrique est l'occlusion : les bâtiments en volume sont ce qui
rend la vue jolie et ce qui masque les cases derrière. Or le jeu porte
entièrement sur ce qu'on voit. D'où l'extrusion plafonnée à une demi-case, la
transparence des bâtiments devant un pion, et le marquage au sol des cases
visibles.
