# Vocabulaire d'effets

Version du vocabulaire : **3**

Un plugin qui déclare une capacité, une dépense ou un mode porte
`effects_version` dans son manifeste. C'est le quatrième numéro de contrat du
projet, indépendant des trois autres : le vocabulaire peut changer sans que le
contrat de formes ni celui des bots bougent, et l'inverse est vrai aussi.

Une capacité, une dépense de résistance, un mode de jeu se décrit par
composition de primitives, jamais par du code. C'est le contrat central du
projet : il rend le modding accessible sans bac à sable, sans faille, sans
problème de compatibilité de version.

Les capacités et dépenses livrées avec le jeu sont écrites dans ce format, au
même titre qu'un plugin tiers. C'est le test qui prouve que le vocabulaire
suffit : **si une mécanique du jeu de base ne s'y exprime pas, il manque une
primitive** — et c'est une décision de contrat public, pas un cas particulier à
coder dans le noyau.

---

## 1. Trois règles qui gouvernent tout

**Un plugin ne touche jamais `*Game`.** Il produit des `Effect`, le noyau les
applique. C'est ce qui l'empêche de lire la position cachée du fugitif ou sa
zone scellée, et ce qui garde `Undo` praticable.

**Tout effet est réversible.** `Appliquer1Effet` renvoie de quoi se défaire. Un
effet qui ne sait pas se défaire n'entre pas dans le vocabulaire : sans ça,
l'IA ne peut plus explorer des milliers de positions sans copier l'état, et le
rejeu du journal diverge.

**Tout effet est déterministe.** L'aléatoire disponible est `core.Random`,
alimenté par la graine de la partie. Ni horloge, ni entropie système, ni
parcours de `map` — dont l'ordre n'est pas stable en Go.

---

## 2. Les primitives

Ajouter une primitive est une décision lourde : elle entre dans le contrat
public et ne peut plus en sortir sans casser les plugins existants. Avant d'en
ajouter une, vérifier que la composition des existantes ne suffit pas.

### Déplacement et position

| Type | Paramètres | Effet |
|---|---|---|
| `step` | `target`, `value` (cases) | Déplace un pion, sans consommer son quota de tour |
| `teleport` | `target` | Place un pion sur la case du contexte |
| `change_mobility` | `target`, `value`, `duration`, `mode` | Ajoute des cases de déplacement pour la durée |

`value` négative est légale : `change_mobility` à -1 immobilise.

**`mode` dit comment `value` s'applique.** Absent, elle s'ajoute ; à `multiply`,
elle multiplie. Un manifeste écrit avant ce champ garde donc son sens.

Il existe parce que la portée, la durée et le rayon du noyau dérivent de la
taille du plateau : une valeur absolue ne peut plus dire une intention relative.
Le Guetteur déclarait « portée 8 », juste tant qu'un seul préréglage existait, et
qui triplait la portée d'un Quartier. Il déclare maintenant « portée × 2 ».

### Information

| Type | Paramètres | Effet |
|---|---|---|
| `change_range` | `target`, `value`, `duration`, `mode` | Modifie la portée de vue |
| `reveal_position` | `target` | Rend une position publique ce tour |
| `cancel_reveal` | `target` | Neutralise les révélations provoquées par les inspecteurs, pour le tour |
| `reveal_trails` | `target`, `radius` | Découvre les traces dans le rayon, en distance de Manhattan |
| `erase_trails` | `target`, `duration` | Supprime les traces plus récentes que `duration` tours |
| `decoy_trail` | `target` | Substitue aux traces du tour une seule, fausse |

`decoy_trail` **remplace**, il n'ajoute pas. C'est une propriété du §8 des
règles — les traces d'un tour sont toutes vraies ou toutes fausses —, et elle
tient parce qu'une seule primitive fait les deux gestes. Poser une fausse trace
et effacer les vraies par deux primitives distinctes laisserait composer les
deux dans l'autre sens, c'est-à-dire mélanger : **une règle qu'on peut enfreindre
par composition n'est pas une règle.**

La case où la trace se pose et celle vers laquelle elle pointe viennent du coup,
pas du manifeste. Un leurre a la forme d'un pas qui n'a pas eu lieu, si bien que
les règles de déplacement s'y appliquent telles quelles : il ne peut produire
qu'une trace qu'un vrai déplacement aurait pu laisser.

`reveal_position` ne vaut qu'un tour : elle rend une position publique sans
rien laisser sur le plateau. C'est la seule primitive de révélation, et elle est
volontairement sans marque durable — une mécanique de repérage a besoin de dire
« il est là maintenant » sans inscrire « il était là » pour le reste de la
partie.

### Terrain

| Type | Paramètres | Effet |
|---|---|---|
| `block_cell` | `target`, `duration` | Ferme une case ; bloque le déplacement **et la vue** |
| `open_cell` | `target`, `duration` | Rouvre une case bâtie |
| `close_zone` | `target`, `zone` | Neutralise un point d'extraction |
| `open_zone` | `target`, `zone` | Le rouvre |
| `seal_zone` | `target`, `zone` | Désigne la zone que le fugitif vise |

`seal_zone` écrit la zone scellée sans permettre de la lire. C'est la seule
façon d'exprimer le changement de zone, que la règle facture 2 points, et un
plugin qui l'emploie ne gagne aucun accès à l'information la plus sensible du
jeu.

Un barrage bloque la vue comme un bâtiment. Sans ça, la capacité ne serait
qu'un mur de déplacement, sans effet sur l'information — ce qui, dans ce jeu,
revient à n'avoir aucun effet.

**Un barrage l'emporte sur un percement.** Une case visée par `open_cell`
puis par `block_cell` est fermée, et elle l'est aussi dans l'ordre inverse :
sans priorité déclarée, le résultat dépendrait de l'ordre d'application et deux
rejeux du même journal pourraient diverger.

### Ressource

| Type | Paramètres | Effet |
|---|---|---|
| `cost_stamina` | `target`, `value` | Retire des points |
| `restore_stamina` | `target`, `value` | En rend |

### Contrôle

| Type | Paramètres | Effet |
|---|---|---|
| `defer` | `duration`, `announced`, effets imbriqués | Applique les effets dans `duration` tours |
| `end_game` | `target` | Termine la partie au profit du camp visé |

---

## 3. `defer`

La primitive la plus lourde du vocabulaire, et la seule dont l'ajout demande
d'être défendu. Deux justifications indépendantes.

**Elle rend déclaratif ce qui demanderait sinon du code.** Un mode qui ferme une
zone deux tours après son déclenchement s'écrit ainsi, sans qu'une ligne du
noyau connaisse ce délai :

```toml
[[mode.exemple.effect]]
type = "defer"
duration = 2
announced = true

  [[mode.exemple.effect.then]]
  type = "close_zone"
  target = "zone"
```

**Ce n'est pas ainsi que l'étranglement livré est écrit**, et c'est instructif :
son préavis a été retiré du mode parce que « personne ne subit une fermeture par
surprise » est une règle du jeu, et qu'une règle ne se négocie pas par
manifeste. Un plugin qui déclarait un préavis nul ne réglait pas le jeu, il en
cassait une garantie. `defer` reste le bon outil pour ce qu'un plugin décide
lui-même de retarder.

**Elle ouvre les plateaux qui se transforment** — la mécanique de blocs qui
tombent. Un mur qui apparaît sans prévenir transformerait un plan raisonné en
coup de dé et rendrait la carte de croyance inutilisable ; un mur annoncé deux
tours à l'avance monte la tension sans que le hasard tranche.

### Champs

| Champ | Rôle |
|---|---|
| `duration` | Nombre de tours avant application. Minimum 1 |
| `announced` | Si vrai, l'effet en attente figure dans la `View` des deux camps, celui du fugitif sans son contexte |
| `then` | Les effets à appliquer, mêmes primitives, **jamais un `defer`** |

L'imbrication d'un `defer` dans un `defer` est refusée au chargement : ça
n'ajoute aucune expressivité — deux durées s'additionnent — et ça permettrait
des chaînes indéfinies qu'aucune annulation ne saurait dérouler.

### Ce qu'elle impose au noyau

Une file d'effets en attente dans `Game`, sérialisée avec le reste, résolue en
fin de tour **avant** le test de fin de partie.

Les effets en attente et annoncés entrent dans `View` — c'est tout leur intérêt.
Un `defer` non annoncé n'y figure pas.

**Celui qu'un plugin pose du côté fugitif part aux inspecteurs sans son
contexte : ils apprennent qu'un effet vient et quand, jamais où.** Le contexte
d'une dépense porte la case exacte du fugitif, et parfois la zone qu'il vient de
sceller ; le servir tel quel donnerait par la bande ce que la vue tait partout
ailleurs. Un auteur qui a besoin d'annoncer une case au camp adverse la fait
donc poser par une capacité d'inspecteur, dont les positions sont publiques.

L'annulation défait la mise en file, pas l'effet : annuler le tour où le
`defer` a été posé le retire de la file.

---

## 4. Cibles

| Cible | Désigne |
|---|---|
| `current_piece` | Le pion qui déclenche |
| `other_piece` | Un autre pion du même camp, choisi au déclenchement |
| `all_pieces` | Tous les pions du camp |
| `fugitive` | Le fugitif |
| `cell` | La case portée par le contexte du coup |
| `zone` | La zone portée par le contexte |

**Une cible qui ne désigne personne est refusée au chargement** : une dépense du
fugitif ne vise ni `other_piece` ni `all_pieces`, puisqu'il est seul.

**La case vient du coup, jamais du manifeste**, et c'est le noyau qui énumère
celles qui sont possibles : une capacité dont un effet vise `cell` est proposée
une fois par case atteignable, et le joueur choisit parmi ces coups. Un plugin
n'a donc rien à déclarer pour qu'on choisisse où son effet tombe — et il ne
choisit pas non plus la portée, qui est une règle de jeu : le voisinage du pion
pour une capacité d'inspecteur, celui du fugitif pour un leurre.

Viser le camp adverse, en revanche, est le cas ordinaire — le Chef révèle la
position du fugitif, le Barreur lui ferme une case. Le chargeur ne cherche donc
pas à distinguer ce qui l'avantage de ce qui le gêne : il faudrait juger chaque
couple effet-cible, c'est-à-dire porter le raisonnement que ce vocabulaire
refuse d'avoir. Un plugin qui déclarerait côté inspecteurs un effet rendant de
la résistance au fugitif se charge ; il est mal écrit, pas invalide.

`other_piece` est la seule cible qui en désigne deux : celui qui déclenche et
celui qui subit. **Aucune capacité livrée ne l'emploie** depuis que le Chef force
une révélation au lieu de partager une vue — elle reste au contrat pour les
plugins, et le noyau sait la lire.

---

## 5. Déclenchements

| Déclenchement | Quand | Où |
|---|---|---|
| `inspectors_phase` | Pendant leur phase, au choix du joueur | capacité |
| `fugitive_phase` | Pendant la sienne, au choix du joueur | dépense |
| `strangling` | Au tour où une zone se ferme | **mode seulement** |

Une capacité `passive = true` n'a pas de déclenchement : elle s'applique en
permanence, tant que le pion est en jeu.

**Trois et pas davantage, un par endroit où le noyau consulte le registre.** Les
deux phases portent ce qu'un joueur choisit ; l'étranglement est le seul moment
que le jeu déclenche de lui-même, d'où sa réserve aux modes — un pion qui s'y
accrocherait agirait sans que son camp l'ait joué.

**La cadence est en dur dans le noyau.** Le vocabulaire ne sait exprimer que
celle de l'étranglement, dont le début et la période sont des paramètres de
partie ; rien ne dit « tous les cinq tours » ni « à chaque fin de tour ». Ce
document a listé pendant des mois trois déclenchements de plus — `turn_end`,
`contact`, `reveal` — que le noyau ne produisait nulle part : un plugin qui s'y
accrochait restait inerte sans un message. Ils ont été retirés plutôt que
branchés, parce que les brancher demande de trancher ce qu'ils portent —
lequel des trois pions au contact, une révélation avant ou après la dépense du
silence —, et qu'un déclenchement entre au contrat public sans en ressortir. La
première mécanique périodique retenue paiera ce coût, en connaissance de cause.

---

## 6. Modes

Un mode est une règle de partie que le jeu déclenche, sans qu'un joueur la
choisisse. C'est la troisième forme déclarative, à côté des capacités et des
dépenses, et elle se lit de la même façon : un nom, un déclenchement, des
effets.

```toml
[mode.strangling]
name = "Étranglement"
trigger = "strangling"

  [[mode.strangling.effect]]
  type = "close_zone"
  target = "zone"
```

**La cadence n'est pas dans le mode.** À partir de quel tour l'étranglement
commence, tous les combien il se répète et combien de tours à l'avance il
s'annonce sont des `Settings`, réglés dans l'interface et enregistrés avec la
partie. Le mode dit *ce qui se passe*, le paramètre dit *quand* : les écrire
tous deux ici donnerait deux sources de vérité pour un même réglage, et un
préréglage de difficulté cesserait d'agir.

**Le préavis en fait partie, et il n'en a pas toujours été ainsi.** Il vivait
dans le mode sous la forme d'un `defer` de deux tours, qui s'ajoutait à une
cadence que le noyau calculait déjà en tours de fermeture : tout tombait deux
tours après le tableau publié.

`strangling` n'est ouvert qu'à un mode. Une capacité ne lit pas les
paramètres de cadence, elle n'aurait donc aucun moyen de savoir quand se
déclencher.

---

## 7. Comment s'exprime un niveau de difficulté

Trois axes indépendants. **Les garder séparés est ce qui rend l'équilibrage
mesurable** : si « difficile » veut dire à la fois une IA plus forte et des
règles différentes, plus aucune statistique n'est comparable.

**Axe 1 — la compétence de l'adversaire.** Jeux de poids de l'IA, plus un bruit
décroissant. Ne touche ni aux règles ni aux paramètres. Persisté dans
`poids_ia`.

**Axe 2 — le cadre.** Portée de vue, nombre de pions déplaçables, durée,
résistance initiale, nombre de zones. Ce sont les `Settings`, exposés dans
l'interface, enregistrés avec la partie.

**Axe 3 — les règles.** Un plugin qui ajoute une capacité, change un coût,
introduit un plateau mouvant. C'est là que `rules = true` s'applique, et donc
la poignée de main réseau.

Un préréglage de difficulté nomme une combinaison des trois. Le manifeste des
plugins actifs partant en base avec la partie, les statistiques se segmentent
par empreinte : deux parties jouées sous des règles différentes ne se comparent
jamais par accident.

---

## 8. Ce qui n'est pas dans le vocabulaire, et pourquoi

**Rien qui force un adversaire à jouer un coup.** Un effet modifie ce qui est
possible, jamais ce qui est choisi. La convergence des inspecteurs sur une
fausse trace n'est pas un effet : ils sont libres de l'ignorer, et c'est ce qui
fait du leurre un pari plutôt qu'un automatisme.

**Rien qui lise l'état caché.** Pas de primitive « connaître la zone scellée ».
Un plugin qui en aurait besoin demanderait en fait un autre jeu.

**Rien de conditionnel.** Pas de `si`, pas de boucle. Le vocabulaire est
délibérément non calculable : une condition arbitraire ferait d'un manifeste un
programme, ce qui ramènerait tous les problèmes que le format déclaratif évite.
Un plugin qui a besoin de logique passe par WebAssembly.

**Aucune primitive de rendu.** L'apparence relève du contrat de formes, qui est
un contrat distinct avec sa propre version.

---

## 9. Validation

Contrôles appliqués au chargement, plugin local compris :

- type de primitive connu, `effects_version` prise en charge ;
- `duration` d'un `defer` supérieure à zéro, et `announced` comme `then` refusés
  sur toute autre primitive ;
- `target` compatible avec le `camp` déclaré ;
- champs obligatoires présents pour le type ;
- `duration` et `radius` positifs ou nuls. **`value` n'a pas de bornes** — le
  négatif est légal, c'est ce qui immobilise un pion ou lui retire de la portée,
  et aucune primitive n'a de plafond qui vaudrait pour toutes ;
- pas de `defer` imbriqué dans un `defer` ;
- une clé de capacité ou de dépense déjà prise est un **conflit**, jamais un
  écrasement silencieux ;
- `rules = false` interdit tout effet : la déclaration est vérifiée, pas crue.
