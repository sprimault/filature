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
greffons/
  base/                 le contenu livré, au même format qu'un greffon tiers
  mes-vehicules/
    manifeste.toml      obligatoire
    formes.toml         facultatif
    palette.toml        facultatif
```

`manifeste.toml` est le seul fichier obligatoire. Les autres n'existent que si
le greffon les remplit.

**Le contenu livré passe par le même chemin que le vôtre.** `greffons/base` n'a
aucun statut particulier : il est lu par le code qui lira le vôtre. C'est ce qui
garantit que ce chemin est exercé à chaque partie, plutôt qu'une fois de temps
en temps.

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

## 6. Bots

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

## 7. Ce qui se passe au chargement

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

---

## 8. Ce qui est refusé

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
