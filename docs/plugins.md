# Format d'un plugin

Comment se compose un plugin Filature, fichier par fichier et champ par champ.

Les trois contrats qu'il peut toucher ont leur propre document, et font foi sur
leur domaine : [`contrat-formes.md`](contrat-formes.md) pour la géométrie,
[`vocabulaire-effets.md`](vocabulaire-effets.md) pour les règles,
[`protocole-bot.md`](protocole-bot.md) pour les adversaires externes. Le présent
document décrit ce qui les précède : l'arborescence, le manifeste, et ce qui se
passe au chargement.

---

## 1. Un dossier, des fichiers TOML

Un plugin est un dossier posé dans `plugins/`. Rien à compiler, rien à
enregistrer.

```
filature                l'exécutable, qui se suffit à lui-même
plugins/
  mes-vehicules/
    manifest.toml      obligatoire
    shapes.toml         facultatif
    palette.toml        facultatif
    language.toml         facultatif
```

**Le contenu livré n'est pas dans ce dossier** : règles, formes, palette,
français et anglais sont embarqués dans l'exécutable, qui fonctionne donc sans
aucun fichier à côté.

Pour le lire ou le recopier, `filature examples <dossier>` l'écrit sur le
disque. C'est le chemin d'un traducteur qui part de l'anglais, ou de quiconque
veut voir comment une capacité est déclarée avant d'écrire la sienne. La
commande n'écrase jamais un fichier existant.

`manifest.toml` est le seul fichier obligatoire. Les autres n'existent que si
le plugin les remplit.

**Le contenu livré passe par le même chemin que le vôtre.** Il vient d'un
système de fichiers embarqué au lieu du disque, et c'est la seule différence :
le code qui le lit est celui qui lira le vôtre. C'est ce qui garantit que ce
chemin est exercé à chaque partie, plutôt qu'une fois de temps en temps.

---

## 2. `manifest.toml`

### Identité

| Champ | Type | Obligatoire | Rôle |
|---|---|---|---|
| `name` | chaîne | oui | identifiant, qui est aussi le nom du dossier |
| `version` | chaîne | oui | la vôtre, libre |
| `description` | chaîne | non | une phrase, affichée dans l'écran des plugins |
| `license` | chaîne | au catalogue | identifiant SPDX |

`name` suit `^[a-z][a-z0-9-]{1,31}$` : minuscules, chiffres et tirets, de 2 à 32
caractères. Le contrôle est appliqué au chargement.

`version` n'a aucune valeur de preuve. Ce qui identifie réellement un plugin
est l'empreinte de son contenu, calculée au chargement : deux plugins qui se
disent tous deux « 1.2.0 » sans être identiques sont détectés comme différents
lors d'une partie en réseau.

`license` est une liste fermée — `MIT`, `Apache-2.0`, `CC0-1.0`, `CC-BY-4.0`,
`CC-BY-SA-4.0`, `BSD-3-Clause`. Un champ libre laisserait passer « à voir » et
rendrait les entrées du catalogue inexploitables. Facultatif hors catalogue.

### Nature

| Champ | Type | Rôle |
|---|---|---|
| `rules` | booléen | vrai si le plugin change les règles |
| `effects_version` | entier | version du vocabulaire d'effets visée |
| `wasm` | chaîne | chemin d'un module WebAssembly |

`rules` détermine la poignée de main réseau : deux joueurs doivent avoir
exactement les mêmes plugins de règles, alors qu'un plugin purement visuel
n'engage que celui qui l'installe. **La déclaration ne suffit pas, elle est
vérifiée** : `rules = false` avec une capacité dans le fichier est un refus au
chargement, pas un avertissement.

`effects_version` devient obligatoire dès qu'une `ability`, une `expense` ou un
`mode` est déclaré. Sans lui, un plugin écrit contre une primitive apparue plus
tard échouerait sur un message de champ inconnu au lieu d'être refusé
proprement.

### Contenu

| Champ | Rôle |
|---|---|
| `ability` | ce qu'un pion peut déclencher |
| `expense` | ce que le fugitif peut acheter avec sa résistance |
| `mode` | ce que le jeu déclenche de lui-même |
| `bot` | un adversaire en processus séparé |
| `language` | un dictionnaire de libellés |

Les trois premiers sont des tables indexées par identifiant. La clé est
l'identifiant, le champ `name` est le libellé affiché :

```toml
[ability.lookout]     # « lookout » est l'identifiant
name = "Guetteur"        # « Guetteur » est ce que le joueur lit
```

---

## 3. Capacités et dépenses

Même structure : c'est le sens du coût qui change, pas la forme.

| Champ | Type | Rôle |
|---|---|---|
| `name` | chaîne | libellé affiché |
| `side` | `fugitive` ou `inspectors` | qui peut la déclencher |
| `uses` | entier | déclenchements par partie ; absent vaut illimité |
| `cost` | entier | points de résistance prélevés |
| `passive` | booléen | s'applique en permanence, sans déclenchement |
| `trigger` | énumération | moment où elle entre en jeu |
| `effect` | tableau | les primitives appliquées, dans l'ordre |

```toml
[ability.blocker]
name = "Barreur"
side = "inspectors"
uses = 1
trigger = "inspectors_phase"

  [[ability.blocker.effect]]
  type = "block_cell"
  target = "cell"
  duration = 3
```

Les primitives disponibles, leurs paramètres et leurs cibles sont dans
[`vocabulaire-effets.md`](vocabulaire-effets.md).

Une cible qui ne désigne personne est refusée au chargement : une dépense du
fugitif ne vise ni `other_piece` ni `all_pieces`, puisqu'il est seul. Viser le
camp adverse, en revanche, est le cas ordinaire — voir §9.

---

## 4. Modes

Un mode est une règle que le jeu déclenche sans qu'un joueur la choisisse.
L'étranglement en est un.

| Champ | Obligatoire | Rôle |
|---|---|---|
| `name` | oui | libellé affiché aux deux joueurs |
| `trigger` | oui | moment où le mode agit |
| `effect` | oui | ce qu'il applique |

Le déclenchement `strangling` n'est ouvert qu'à un mode : sa cadence vient des
paramètres de la partie, qu'une capacité ne lit pas.

**La cadence ne se déclare pas dans le mode.** À partir de quel tour et tous les
combien restent des paramètres de partie, réglés dans l'interface. Le mode dit
ce qui se passe, le paramètre dit quand.

---

## 5. Apparence

`shapes.toml` et `palette.toml`, tous deux facultatifs et indépendants l'un de
l'autre. Un plugin de palette seule est le mod le moins cher qui existe : un
fichier de quinze lignes, aucune géométrie.

Les deux portent `shapes_version`, qui est un numéro distinct de
`effects_version` : le contrat d'apparence et celui des règles évoluent
séparément.

**Un plugin ne déclare que ce qu'il remplace.** Tout le reste retombe sur le
contenu livré. Changer la seule allure du fugitif tient en un dossier et deux
fichiers.

La géométrie, les gabarits, les quatre primitives de dessin et les noms de
formes attendus sont dans [`contrat-formes.md`](contrat-formes.md).

`filature preview <dossier>` en sort un aperçu SVG sans lancer le jeu : les
formes sur chacun des sols et un plateau en situation. Voir §8.

---

## 6. Langues

Les libellés de l'interface ne sont jamais dans le code : ils viennent d'un
dictionnaire, et un plugin de langue en fournit un.

```
filature-de/
  manifest.toml
  language.toml
```

```toml
# manifest.toml
name = "filature-de"
version = "1.0.0"
license = "CC0-1.0"
description = "Filature en allemand"

[language]
code = "de"
name = "Deutsch"
```

`code` est une étiquette BCP 47 — `de`, `pt-BR`, `zh-Hans` — et sa forme est
vérifiée au chargement. `name` est le nom de la langue **dans cette langue**,
parce que c'est lui qui s'affiche dans le sélecteur : quelqu'un qui cherche sa
langue y cherche « Deutsch », pas « allemand ».

```toml
# language.toml
[label]
menu_new_game = "Neues Spiel"
side_fugitive = "Flüchtiger"
```

**Une langue par plugin.** Un traducteur publie et met à jour la sienne sans
toucher à celles des autres, et deux plugins qui fournissent le même code sont
un conflit signalé, comme partout ailleurs.

### Où le poser pour qu'il soit reconnu

Dans le dossier des plugins, qui est `plugins/` à côté de l'exécutable, ou
celui que désigne `--plugins` :

```
filature.exe
plugins/
  filature-de/   le vôtre, ici
```

Rien à enregistrer, rien à recompiler : le dossier est lu au démarrage, et la
langue apparaît dans le sélecteur des préférences. Un plugin qui ne se montre
pas est un plugin dont le manifeste a été refusé — le journal dit lequel et
pourquoi.

Le français et l'anglais ne sont pas là : ils sont dans l'exécutable. Pour
partir de l'anglais, `filature examples <dossier>` l'en ressort.

### Ce qui n'a pas de version

Un dictionnaire n'en porte pas, contrairement aux formes, aux effets et au
protocole de bot. **Aucune incompatibilité n'est possible** : une clé absente
retombe sur le français, une clé inconnue est ignorée. Une traduction en retard
n'est pas cassée, elle est partielle — et l'écran des plugins dit à quel point.

Le français fait exception en ceci qu'il est la langue de repli : il est livré
avec les règles, dans le plugin `base`, et couvre toutes les clés par
construction puisque c'est lui que le reste complète.

## 7. Bots

Un bot ne modifie pas les règles, il joue avec. Il se déclare dans une table
`[bot]` et vit dans un processus séparé.

```toml
[bot]
side = "inspectors"
command = "traqueur"
arguments = ["--niveau", "3"]
deterministic = true
```

`command` est cherchée dans le dossier du plugin puis dans le `PATH`. Aucun
interpréteur n'est fourni : un bot Python se livre avec son lanceur.

Un plugin qui déclare un bot **et** des effets est refusé. Ce sont deux choses,
et les mélanger casserait la poignée de main réseau, où les plugins de règles
doivent être identiques des deux côtés.

Les messages échangés sont dans [`protocole-bot.md`](protocole-bot.md).

---

## 8. Ce qui se passe au chargement

1. Lecture des manifestes, dans l'ordre alphabétique des dossiers.
2. Validation de chacun, contrôle par contrôle, par le chargeur lui-même.
3. Calcul de l'empreinte du contenu.
4. Fusion dans le registre.

**Le chargeur applique le contrat, il ne lit pas le schéma.** Le jeu se lance
sans dépendre d'un fichier de description, et
[`schemas/plugin-manifest.schema.json`](../schemas/plugin-manifest.schema.json)
reste ce que lit un auteur pour savoir contre quoi il écrit. Les deux disent la
même chose, et des tests les tiennent ensemble : l'un valide le contenu livré et
une série de manifestes fautifs contre le schéma, l'autre rapproche du contrat
publié le motif d'un nom, la liste des licences, les énumérations du vocabulaire
et les trois numéros de version.

Ce n'est pas un détail d'implémentation. Tant que rien n'exécutait le schéma, il
pouvait mentir sans qu'on le sache — sa clause sur les effets différés a refusé
pendant des mois la seule forme valide de la primitive qu'elle contraint.

**Un plugin invalide fait échouer le chargement entier.** Il n'est pas ignoré :
un plugin à moitié actif est pire qu'un plugin absent, parce que la partie se
joue alors sous des règles que personne n'a choisies.

**Deux plugins qui définissent la même clé sont un conflit signalé**, jamais un
écrasement silencieux. Cela vaut pour une capacité, une dépense, un mode comme
pour une forme.

Les manquements d'un manifeste sont listés en une fois. Quelqu'un qui met au
point un plugin veut la liste, pas un aller-retour par erreur.

### Vérifier avant d'installer

```
$ filature validate mon-plugin/
mon-plugin/manifest.toml:23: ability.lookout.effect[0].target
    « pion » n'est pas une cible connue
    attendu : current_piece, all_pieces, other_piece, fugitive, cell, zone
mon-plugin/manifest.toml: effects_version
    obligatoire dès qu'une capacité, une dépense ou un mode est déclaré
mon-plugin/shapes.toml:41: shape.fugitive.stroke[2].points[3]
    y vaut 48, hors du gabarit du rôle pion (0 à 40)

3 manquements
```

**Chaque manquement dit où il est** : le fichier, la ligne quand elle est
connue, et le chemin complet de la clé fautive. « Cible invalide » sans autre
précision oblige à relire tout un manifeste ; `ability.lookout.effect[0].target`
désigne un seul endroit.

Le chemin de clé est donné même quand la ligne manque — un décodeur ne la
connaît pas toujours pour une erreur de sens, alors qu'il sait toujours quelle
clé il lisait. Et ce qui est attendu est dit avec ce qui est refusé : la liste
des cibles connues vaut mieux que « cible invalide ».

La commande charge le plugin **par le même code que le jeu**. Elle sort avec un
code non nul s'il reste un manquement : de quoi la mettre dans une intégration
continue, ce que le catalogue attend des auteurs.

Elle évite surtout d'apprendre le problème par un jeu qui ne démarre plus. Et
elle porte une garantie qui vaut pour qui installe : un plugin validé chez son
auteur se charge chez les autres, puisque c'est la même validation.

Le jeu, lui, montre dans son écran des plugins ceux qu'il a refusés et
pourquoi — un plugin simplement absent de la liste laisserait deviner.

### Regarder ce que ça donne

Un plugin d'apparence peut être valide et laid, ou valide et illisible. La
validation ne dit rien de ça.

```
$ filature preview mes-vehicules/
mes-vehicules-formes.svg
mes-vehicules-plateau.svg
```

Deux fichiers, parce que les deux questions ne se jugent pas au même endroit :
la planche pose chaque forme sur chacun des cinq sols, où la même couleur
ne se lit pas pareil, et le plateau la montre en situation. Un second argument
dit où les écrire.

Le plugin est fusionné sur le contenu livré avant d'être rendu — il ne déclare
que ce qu'il remplace, et le montrer seul donnerait une pièce au milieu du
vide. La planche marque d'une étoile les formes qui viennent de lui, ce qui
rend visible la seule faute qu'aucun contrôle n'attrape : une clé mal
orthographiée est un nom de forme inconnu, elle passe la validation et ne
surcharge rien.

Le plateau est une partie réellement jouée sur une graine figée, pas une
position composée : les pions, les traces et les barrages y tombent où le jeu
les met, et la graine étant figée, deux aperçus du même plugin donnent le même
fichier — leur diff dit ce qui a bougé.

Du SVG parce qu'il s'ouvre dans un navigateur, se relit sans lancer le jeu, et
reste du texte comme le reste de ce qu'un plugin publie.

---

## 9. Ce qui est refusé

- `rules = false` accompagné d'une capacité, d'une dépense, d'un mode ou d'un
  module exécutable ;
- une forme qui déborde de son gabarit — au chargement, plugin local compris,
  parce que masquer les cases voisines est un avantage de jeu déguisé en
  habillage ;
- une couleur en hexadécimal dans une forme : les formes ne référencent que des
  noms de palette ;
- un `shapes_version` ou un `effects_version` que ce binaire ne connaît pas ;
- un plugin qui déclare à la fois un bot et des effets ;
- un nom hors du motif `^[a-z][a-z0-9-]{1,31}$`, qui sert d'identifiant partout ;
- une `license` absente de la liste fermée du §2 ;
- un `trigger` inconnu, ou `strangling` sur une capacité — le jeu le déclenche
  lui-même, et un pion qui s'y accrocherait agirait sans que son camp l'ait joué ;
- une cible qui ne désigne personne : `other_piece` ou `all_pieces` dans une
  dépense du fugitif, qui est seul ;
- un `code` de langue hors de l'étiquette BCP 47, ou une langue sans `name` —
  c'est lui qui s'affiche dans le sélecteur.

**Ce qui n'est pas refusé, et ne peut pas l'être** : un effet qui vise le camp
adverse. Le Chef révèle la position du fugitif, le Barreur lui ferme une case ;
distinguer cela d'une capacité d'inspecteur qui lui rendrait de la résistance
demanderait au chargeur de juger l'intention derrière chaque couple effet-cible,
c'est-à-dire de porter le raisonnement que le vocabulaire déclaratif refuse
d'avoir. Un tel plugin se charge : il est mal écrit, pas invalide.

Le catalogue ajoute une seule règle, mécanique : **aucun fichier binaire, sous
aucune extension.** C'est ce qui supprime toute question de provenance, donc
toute relecture humaine.

Hors catalogue, rien de tout cela ne s'applique au-delà du gabarit : un dossier
posé à la main se charge tel quel.
