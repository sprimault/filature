# Format d'un greffon

Comment se compose un greffon Filature, fichier par fichier et champ par champ.

Les trois contrats qu'il peut toucher ont leur propre document, et font foi sur
leur domaine : [`contrat-formes.md`](contrat-formes.md) pour la géométrie,
[`vocabulaire-effets.md`](vocabulaire-effets.md) pour les règles,
[`protocole-bot.md`](protocole-bot.md) pour les adversaires externes. Le présent
document décrit ce qui les précède : l'arborescence, le manifeste, et ce qui se
passe au chargement.

---

## 1. Un dossier, des fichiers TOML

Un greffon est un dossier posé dans `greffons/`. Rien à compiler, rien à
enregistrer.

```
filature                l'exécutable, qui se suffit à lui-même
greffons/
  mes-vehicules/
    manifeste.toml      obligatoire
    formes.toml         facultatif
    palette.toml        facultatif
    langue.toml         facultatif
```

**Le contenu livré n'est pas dans ce dossier** : règles, formes, palette,
français et anglais sont embarqués dans l'exécutable, qui fonctionne donc sans
aucun fichier à côté.

Pour le lire ou le recopier, `filature exemples <dossier>` l'écrit sur le
disque. C'est le chemin d'un traducteur qui part de l'anglais, ou de quiconque
veut voir comment une capacité est déclarée avant d'écrire la sienne. La
commande n'écrase jamais un fichier existant.

`manifeste.toml` est le seul fichier obligatoire. Les autres n'existent que si
le greffon les remplit.

**Le contenu livré passe par le même chemin que le vôtre.** Il vient d'un
système de fichiers embarqué au lieu du disque, et c'est la seule différence :
le code qui le lit est celui qui lira le vôtre. C'est ce qui garantit que ce
chemin est exercé à chaque partie, plutôt qu'une fois de temps en temps.

---

## 2. `manifeste.toml`

### Identité

| Champ | Type | Obligatoire | Rôle |
|---|---|---|---|
| `nom` | chaîne | oui | identifiant, qui est aussi le nom du dossier |
| `version` | chaîne | oui | la vôtre, libre |
| `description` | chaîne | non | une phrase, affichée dans l'écran des greffons |
| `licence` | chaîne | au catalogue | identifiant SPDX |

`nom` suit `^[a-z][a-z0-9-]{1,31}$` : minuscules, chiffres et tirets, de 2 à 32
caractères.

`version` n'a aucune valeur de preuve. Ce qui identifie réellement un greffon
est l'empreinte de son contenu, calculée au chargement : deux greffons qui se
disent tous deux « 1.2.0 » sans être identiques sont détectés comme différents
lors d'une partie en réseau.

`licence` est une liste fermée — `MIT`, `Apache-2.0`, `CC0-1.0`, `CC-BY-4.0`,
`CC-BY-SA-4.0`, `BSD-3-Clause`. Un champ libre laisserait passer « à voir » et
rendrait les entrées du catalogue inexploitables. Facultatif hors catalogue.

### Nature

| Champ | Type | Rôle |
|---|---|---|
| `regles` | booléen | vrai si le greffon change les règles |
| `version_effets` | entier | version du vocabulaire d'effets visée |
| `wasm` | chaîne | chemin d'un module WebAssembly |

`regles` détermine la poignée de main réseau : deux joueurs doivent avoir
exactement les mêmes greffons de règles, alors qu'un greffon purement visuel
n'engage que celui qui l'installe. **La déclaration ne suffit pas, elle est
vérifiée** : `regles = false` avec une capacité dans le fichier est un refus au
chargement, pas un avertissement.

`version_effets` devient obligatoire dès qu'une `capacite`, une `depense` ou un
`mode` est déclaré. Sans lui, un greffon écrit contre une primitive apparue plus
tard échouerait sur un message de champ inconnu au lieu d'être refusé
proprement.

### Contenu

| Champ | Rôle |
|---|---|
| `capacite` | ce qu'un pion peut déclencher |
| `depense` | ce que le fugitif peut acheter avec sa résistance |
| `mode` | ce que le jeu déclenche de lui-même |
| `bot` | un adversaire en processus séparé |
| `langue` | un dictionnaire de libellés |

Les trois premiers sont des tables indexées par identifiant. La clé est
l'identifiant, le champ `nom` est le libellé affiché :

```toml
[capacite.guetteur]     # « guetteur » est l'identifiant
nom = "Guetteur"        # « Guetteur » est ce que le joueur lit
```

---

## 3. Capacités et dépenses

Même structure : c'est le sens du coût qui change, pas la forme.

| Champ | Type | Rôle |
|---|---|---|
| `nom` | chaîne | libellé affiché |
| `camp` | `fugitif` \| `inspecteurs` | qui peut la déclencher |
| `usages` | entier | déclenchements par partie ; absent vaut illimité |
| `cout` | entier | points de résistance prélevés |
| `passive` | booléen | s'applique en permanence, sans déclenchement |
| `declenchement` | énumération | moment où elle entre en jeu |
| `effet` | tableau | les primitives appliquées, dans l'ordre |

```toml
[capacite.barreur]
nom = "Barreur"
camp = "inspecteurs"
usages = 1
declenchement = "phase_inspecteurs"

  [[capacite.barreur.effet]]
  type = "bloquer_case"
  cible = "case"
  duree = 3
```

Les primitives disponibles, leurs paramètres et leurs cibles sont dans
[`vocabulaire-effets.md`](vocabulaire-effets.md).

Une cible incompatible avec le `camp` déclaré est refusée au chargement : une
capacité d'inspecteur ne rend pas de résistance au fugitif.

---

## 4. Modes

Un mode est une règle que le jeu déclenche sans qu'un joueur la choisisse.
L'étranglement en est un.

| Champ | Obligatoire | Rôle |
|---|---|---|
| `nom` | oui | libellé affiché aux deux joueurs |
| `declenchement` | oui | moment où le mode agit |
| `effet` | oui | ce qu'il applique |

Le déclenchement `etranglement` n'est ouvert qu'à un mode : sa cadence vient des
paramètres de la partie, qu'une capacité ne lit pas.

**La cadence ne se déclare pas dans le mode.** À partir de quel tour et tous les
combien restent des paramètres de partie, réglés dans l'interface. Le mode dit
ce qui se passe, le paramètre dit quand.

---

## 5. Apparence

`formes.toml` et `palette.toml`, tous deux facultatifs et indépendants l'un de
l'autre. Un greffon de palette seule est le mod le moins cher qui existe : un
fichier de quinze lignes, aucune géométrie.

Les deux portent `version_formes`, qui est un numéro distinct de
`version_effets` : le contrat d'apparence et celui des règles évoluent
séparément.

**Un greffon ne déclare que ce qu'il remplace.** Tout le reste retombe sur le
contenu livré. Changer la seule allure du fugitif tient en un dossier et deux
fichiers.

La géométrie, les gabarits, les quatre primitives de dessin et les noms de
formes attendus sont dans [`contrat-formes.md`](contrat-formes.md).

---

## 6. Langues

Les libellés de l'interface ne sont jamais dans le code : ils viennent d'un
dictionnaire, et un greffon de langue en fournit un.

```
mes-traductions/
  manifeste.toml
  langue.toml
```

```toml
# manifeste.toml
nom = "filature-de"
version = "1.0.0"
licence = "CC0-1.0"
description = "Filature en allemand"

[langue]
code = "de"
nom = "Deutsch"
```

`code` est une étiquette BCP 47 — `de`, `pt-BR`, `zh-Hans`. `nom` est le nom de
la langue **dans cette langue**, parce que c'est lui qui s'affiche dans le
sélecteur : quelqu'un qui cherche sa langue y cherche « Deutsch », pas
« allemand ».

```toml
# langue.toml
[libelle]
menu_nouvelle_partie = "Neues Spiel"
camp_fugitif = "Flüchtiger"
```

**Une langue par greffon.** Un traducteur publie et met à jour la sienne sans
toucher à celles des autres, et deux greffons qui fournissent le même code sont
un conflit signalé, comme partout ailleurs.

### Où le poser pour qu'il soit reconnu

Dans le dossier des greffons, qui est `greffons/` à côté de l'exécutable, ou
celui que désigne `--greffons` :

```
filature.exe
greffons/
  mes-traductions/   le vôtre, ici
```

Rien à enregistrer, rien à recompiler : le dossier est lu au démarrage, et la
langue apparaît dans le sélecteur des préférences. Un greffon qui ne se montre
pas est un greffon dont le manifeste a été refusé — le journal dit lequel et
pourquoi.

Le français et l'anglais ne sont pas là : ils sont dans l'exécutable. Pour
partir de l'anglais, `filature exemples <dossier>` l'en ressort.

### Ce qui n'a pas de version

Un dictionnaire n'en porte pas, contrairement aux formes, aux effets et au
protocole de bot. **Aucune incompatibilité n'est possible** : une clé absente
retombe sur le français, une clé inconnue est ignorée. Une traduction en retard
n'est pas cassée, elle est partielle — et l'écran des greffons dit à quel point.

Le français fait exception en ceci qu'il est la langue de repli : il est livré
avec les règles, dans le greffon `base`, et couvre toutes les clés par
construction puisque c'est lui que le reste complète.

## 7. Bots

Un bot ne modifie pas les règles, il joue avec. Il se déclare dans une table
`[bot]` et vit dans un processus séparé.

```toml
[bot]
camp = "inspecteurs"
commande = "traqueur"
arguments = ["--niveau", "3"]
deterministe = true
```

`commande` est cherchée dans le dossier du greffon puis dans le `PATH`. Aucun
interpréteur n'est fourni : un bot Python se livre avec son lanceur.

Un greffon qui déclare un bot **et** des effets est refusé. Ce sont deux choses,
et les mélanger casserait la poignée de main réseau, où les greffons de règles
doivent être identiques des deux côtés.

Les messages échangés sont dans [`protocole-bot.md`](protocole-bot.md).

---

## 8. Ce qui se passe au chargement

1. Lecture des manifestes, dans l'ordre alphabétique des dossiers.
2. Validation de chacun contre
   [`schemas/manifeste-greffon.schema.json`](../schemas/manifeste-greffon.schema.json).
3. Calcul de l'empreinte du contenu.
4. Fusion dans le registre.

**Un greffon invalide fait échouer le chargement entier.** Il n'est pas ignoré :
un greffon à moitié actif est pire qu'un greffon absent, parce que la partie se
joue alors sous des règles que personne n'a choisies.

**Deux greffons qui définissent la même clé sont un conflit signalé**, jamais un
écrasement silencieux. Cela vaut pour une capacité, une dépense, un mode comme
pour une forme.

Les manquements d'un manifeste sont listés en une fois. Quelqu'un qui met au
point un greffon veut la liste, pas un aller-retour par erreur.

### Vérifier avant d'installer

```
$ filature valide mon-greffon/
mon-greffon/manifeste.toml:23: capacite.guetteur.effet[0].cible
    « pion » n'est pas une cible connue
    attendu : pion_courant, tous_pions, autre_pion, fugitif, case, zone
mon-greffon/manifeste.toml: version_effets
    obligatoire dès qu'une capacité, une dépense ou un mode est déclaré
mon-greffon/formes.toml:41: forme.fugitif.trait[2].points[3]
    y vaut 48, hors du gabarit du rôle pion (0 à 40)

3 manquements
```

**Chaque manquement dit où il est** : le fichier, la ligne quand elle est
connue, et le chemin complet de la clé fautive. « Cible invalide » sans autre
précision oblige à relire tout un manifeste ; `capacite.guetteur.effet[0].cible`
désigne un seul endroit.

Le chemin de clé est donné même quand la ligne manque — un décodeur ne la
connaît pas toujours pour une erreur de sens, alors qu'il sait toujours quelle
clé il lisait. Et ce qui est attendu est dit avec ce qui est refusé : la liste
des cibles connues vaut mieux que « cible invalide ».

La commande charge le greffon **par le même code que le jeu**. Elle sort avec un
code non nul s'il reste un manquement : de quoi la mettre dans une intégration
continue, ce que le catalogue attend des auteurs.

Elle évite surtout d'apprendre le problème par un jeu qui ne démarre plus. Et
elle porte une garantie qui vaut pour qui installe : un greffon validé chez son
auteur se charge chez les autres, puisque c'est la même validation.

Le jeu, lui, montre dans son écran des greffons ceux qu'il a refusés et
pourquoi — un greffon simplement absent de la liste laisserait deviner.

---

## 9. Ce qui est refusé

- `regles = false` accompagné d'une capacité, d'une dépense, d'un mode ou d'un
  module exécutable ;
- une forme qui déborde de son gabarit — au chargement, greffon local compris,
  parce que masquer les cases voisines est un avantage de jeu déguisé en
  habillage ;
- une couleur en hexadécimal dans une forme : les formes ne référencent que des
  noms de palette ;
- un `version_formes` ou un `version_effets` que ce binaire ne connaît pas ;
- un greffon qui déclare à la fois un bot et des effets.

Le catalogue ajoute une seule règle, mécanique : **aucun fichier binaire, sous
aucune extension.** C'est ce qui supprime toute question de provenance, donc
toute relecture humaine.

Hors catalogue, rien de tout cela ne s'applique au-delà du gabarit : un dossier
posé à la main se charge tel quel.
