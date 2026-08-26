# Vocabulaire d'effets

Version du vocabulaire : **1**

Un plugin qui déclare une capacité, une dépense ou un mode porte
`version_effets` dans son manifeste. C'est le quatrième numéro de contrat du
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

**Un plugin ne touche jamais `*Partie`.** Il produit des `Effet`, le noyau les
applique. C'est ce qui l'empêche de lire la position cachée du fugitif ou sa
zone scellée, et ce qui garde `Annuler` praticable.

**Tout effet est réversible.** `Appliquer1Effet` renvoie de quoi se défaire. Un
effet qui ne sait pas se défaire n'entre pas dans le vocabulaire : sans ça,
l'IA ne peut plus explorer des milliers de positions sans copier l'état, et le
rejeu du journal diverge.

**Tout effet est déterministe.** L'aléatoire disponible est `core.Alea`,
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
| `deplacer` | `cible`, `valeur` (cases) | Déplace un pion, sans consommer son quota de tour |
| `teleporter` | `cible` | Place un pion sur la case du contexte |
| `modifier_mobilite` | `cible`, `valeur`, `duree` | Ajoute des cases de déplacement pour la durée |

`valeur` négative est légale : `modifier_mobilite` à -1 immobilise.

### Information

| Type | Paramètres | Effet |
|---|---|---|
| `modifier_portee` | `cible`, `valeur`, `duree` | Modifie la portée de vue |
| `reveler_position` | `cible` | Rend une position publique ce tour |
| `marquer_scene` | `cible` | Inscrit un lieu, durablement et pour les deux camps |
| `annuler_revelation` | `cible` | Neutralise la prochaine révélation périodique |
| `partager_vue` | `cible`, `duree` | Un pion voit ce que voit un autre |
| `reveler_traces` | `cible`, `rayon` | Découvre les traces dans le rayon, en distance de Manhattan |
| `effacer_traces` | `cible`, `duree` | Supprime les traces plus récentes que `duree` tours |

`reveler_position` et `marquer_scene` sont volontairement distincts, et le
meurtre les compose : le premier ne vaut qu'un tour, le second reste sur le
plateau. Les fondre en un seul interdirait de révéler sans laisser de marque, ce
dont toute mécanique de repérage a besoin.

### Terrain

| Type | Paramètres | Effet |
|---|---|---|
| `bloquer_case` | `cible`, `duree` | Ferme une case ; bloque le déplacement **et la vue** |
| `ouvrir_case` | `cible`, `duree` | Rouvre une case bâtie |
| `fermer_zone` | `cible`, `zone` | Neutralise un point d'extraction |
| `ouvrir_zone` | `cible`, `zone` | Le rouvre |
| `sceller_zone` | `cible`, `zone` | Désigne la zone que le fugitif vise |

`sceller_zone` écrit la zone scellée sans permettre de la lire. C'est la seule
façon d'exprimer le changement de zone, que la règle facture 2 points, et un
plugin qui l'emploie ne gagne aucun accès à l'information la plus sensible du
jeu.

Un barrage bloque la vue comme un bâtiment. Sans ça, la capacité ne serait
qu'un mur de déplacement, sans effet sur l'information — ce qui, dans ce jeu,
revient à n'avoir aucun effet.

**Un barrage l'emporte sur un percement.** Une case visée par `ouvrir_case`
puis par `bloquer_case` est fermée, et elle l'est aussi dans l'ordre inverse :
sans priorité déclarée, le résultat dépendrait de l'ordre d'application et deux
rejeux du même journal pourraient diverger.

### Ressource

| Type | Paramètres | Effet |
|---|---|---|
| `couter_resistance` | `cible`, `valeur` | Retire des points |
| `rendre_resistance` | `cible`, `valeur` | En rend |

### Contrôle

| Type | Paramètres | Effet |
|---|---|---|
| `differer` | `duree`, `annonce`, effets imbriqués | Applique les effets dans `duree` tours |
| `fin_partie` | `cible` | Termine la partie au profit du camp visé |

---

## 3. `differer`

La primitive la plus lourde du vocabulaire, et la seule dont l'ajout demande
d'être défendu. Deux justifications indépendantes.

**Elle rend déclaratif ce qui est codé en dur.** L'étranglement — les zones
d'extraction qui se ferment à partir du tour 30, annoncées deux tours à
l'avance — est aujourd'hui dans le noyau. Il s'exprime avec `differer` :

```toml
[[mode.etranglement.effet]]
type = "differer"
duree = 2
annonce = true

  [[mode.etranglement.effet.puis]]
  type = "fermer_zone"
  cible = "zone"
```

**Elle ouvre les plateaux qui se transforment** — la mécanique de blocs qui
tombent. Un mur qui apparaît sans prévenir transformerait un plan raisonné en
coup de dé et rendrait la carte de croyance inutilisable ; un mur annoncé deux
tours à l'avance monte la tension sans que le hasard tranche.

### Champs

| Champ | Rôle |
|---|---|
| `duree` | Nombre de tours avant application. Minimum 1 |
| `annonce` | Si vrai, l'effet en attente figure dans la `Vue` des deux camps |
| `puis` | Les effets à appliquer, mêmes primitives, **jamais un `differer`** |

L'imbrication d'un `differer` dans un `differer` est refusée au chargement : ça
n'ajoute aucune expressivité — deux durées s'additionnent — et ça permettrait
des chaînes indéfinies qu'aucune annulation ne saurait dérouler.

### Ce qu'elle impose au noyau

Une file d'effets en attente dans `Partie`, sérialisée avec le reste, résolue en
fin de tour **avant** le test de fin de partie.

Les effets en attente et annoncés entrent dans `Vue` — c'est tout leur intérêt.
Un `differer` non annoncé n'y figure pas.

L'annulation défait la mise en file, pas l'effet : annuler le tour où le
`differer` a été posé le retire de la file.

---

## 4. Cibles

| Cible | Désigne |
|---|---|
| `pion_courant` | Le pion qui déclenche |
| `autre_pion` | Un autre pion du même camp, choisi au déclenchement |
| `tous_pions` | Tous les pions du camp |
| `fugitif` | Le fugitif |
| `case` | La case portée par le contexte du coup |
| `zone` | La zone portée par le contexte |

Une cible incompatible avec le camp déclarant est refusée au chargement : une
capacité d'inspecteur ne cible pas `fugitif` pour lui rendre de la résistance.

`autre_pion` est la seule cible qui en désigne deux : celui qui déclenche et
celui qui subit. Le Chef, qui voit à travers un coéquipier, en est le seul usage
livré.

---

## 5. Déclenchements

| Déclenchement | Quand |
|---|---|
| `phase_inspecteurs` | Pendant leur phase, au choix du joueur |
| `phase_fugitif` | Pendant la sienne |
| `fin_de_tour` | Automatique, à la résolution |
| `contact` | Quand le fugitif est adjacent à un inspecteur |
| `revelation` | Au tour d'une révélation périodique |
| `etranglement` | Au tour où une zone se ferme — **réservé aux modes** |

Une capacité `passive = true` n'a pas de déclenchement : elle s'applique en
permanence, tant que le pion est en jeu.

---

## 6. Modes

Un mode est une règle de partie que le jeu déclenche, sans qu'un joueur la
choisisse. C'est la troisième forme déclarative, à côté des capacités et des
dépenses, et elle se lit de la même façon : un nom, un déclenchement, des
effets.

```toml
[mode.etranglement]
nom = "Étranglement"
declenchement = "etranglement"

  [[mode.etranglement.effet]]
  type = "differer"
  duree = 2
  annonce = true

    [[mode.etranglement.effet.puis]]
    type = "fermer_zone"
    cible = "zone"
```

**La cadence n'est pas dans le mode.** À partir de quel tour l'étranglement
commence et tous les combien il se répète sont des `Parametres`, réglés dans
l'interface et enregistrés avec la partie. Le mode dit *ce qui se passe*, le
paramètre dit *quand* : les écrire tous deux ici donnerait deux sources de
vérité pour un même réglage, et un préréglage de difficulté cesserait d'agir.

La `duree` du `differer` ci-dessus est le préavis, pas la période. Les deux
valent 2 dans la règle standard, et ce n'est qu'une coïncidence de chiffres.

`etranglement` n'est ouvert qu'à un mode. Une capacité ne lit pas les
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
résistance initiale, nombre de zones. Ce sont les `Parametres`, exposés dans
l'interface, enregistrés avec la partie.

**Axe 3 — les règles.** Un plugin qui ajoute une capacité, change un coût,
introduit un plateau mouvant. C'est là que `regles = true` s'applique, et donc
la poignée de main réseau.

Un préréglage de difficulté nomme une combinaison des trois. Le manifeste des
plugins actifs partant en base avec la partie, les statistiques se segmentent
par empreinte : deux parties jouées sous des règles différentes ne se comparent
jamais par accident.

---

## 8. Ce qui n'est pas dans le vocabulaire, et pourquoi

**Rien qui force un adversaire à jouer un coup.** Un effet modifie ce qui est
possible, jamais ce qui est choisi. La convergence des inspecteurs sur une scène
de meurtre n'est pas un effet : ils sont libres de l'ignorer, et c'est ce qui
fait du meurtre un pari plutôt qu'un automatisme.

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

- type de primitive connu, `version_effets` prise en charge ;
- `duree` d'un `differer` supérieure à zéro, et `annonce` comme `puis` refusés
  sur toute autre primitive ;
- `cible` compatible avec le `camp` déclaré ;
- champs obligatoires présents pour le type ;
- `duree` et `rayon` positifs, `valeur` dans les bornes du type ;
- pas de `differer` imbriqué dans un `differer` ;
- une clé de capacité ou de dépense déjà prise est un **conflit**, jamais un
  écrasement silencieux ;
- `regles = false` interdit tout effet : la déclaration est vérifiée, pas crue.
