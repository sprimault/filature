# Conventions de code

Ce que ce dépôt attend d'un changement en Go, et pourquoi. Les règles du jeu
sont dans [`regles.md`](regles.md), le découpage des paquets dans
[`architecture.md`](architecture.md) ; ce document ne parle que du code.

---

## 1. Bibliothèques

Bibliothèque standard par défaut. Pas de framework de ligne de commande, pas de
bibliothèque d'assertion de test, pas de gestionnaire d'état. La question avant
d'ajouter quoi que ce soit : « qu'est-ce que ça m'évite d'écrire », pas « est-ce
que c'est répandu ».

Les exceptions sont connues d'avance :

| Module | Ce qu'il apporte |
|---|---|
| `github.com/hajimehoshi/ebiten/v2` | rendu, entrées, son |
| `github.com/tetratelabs/wazero` | bac à sable des plugins exécutables |
| `modernc.org/sqlite` | persistance, en Go pur |
| `github.com/BurntSushi/toml` | manifestes de plugins |
| `github.com/coder/websocket` | mode réseau |

Une dépendance ajoutée passe par `make notices` : sa licence entre dans
`THIRD-PARTY-NOTICES`, qui accompagne le binaire dans chaque archive.

## 2. cgo n'est pas uniforme, et c'est la seule entorse

| Cible | `CGO_ENABLED` |
|---|---|
| windows/amd64 | `0` — Ebitengine y est en Go pur |
| js/wasm | `0` |
| linux, darwin | `1` — Ebitengine y passe par les API système |

Le mettre à `0` sur darwin ne produit pas une erreur claire mais un échec de
liaison obscur. C'est une variable de matrice, jamais une valeur globale.

Conséquence à connaître : le binaire Linux est lié à la glibc, il n'y a pas de
construction statique. Sans importance pour un jeu de bureau, mais ça interdit
une image `scratch`.

**Aucune autre dépendance n'a le droit d'introduire cgo.** `modernc.org/sqlite`
est choisi pour cela : une seconde source rendrait les cibles Windows et
WebAssembly impossibles. C'est aussi pourquoi `js/wasm` continue d'être compilé
en intégration continue sans être publié — c'est le seul contrôle qui empêche
une dépendance d'en ajouter par surprise.

## 3. `internal/core` est une feuille du graphe

Le paquet des règles ne dépend de rien d'autre dans `internal/`, et ça doit le
rester. `ai`, `render`, `storage` et `server` en dépendent tous ; l'inverse
jamais.

Il n'importe **jamais** Ebitengine. Un test qui exigerait une fenêtre n'a pas sa
place dans la suite par défaut, les runners d'intégration étant sans écran.

## 4. La configuration entre par `cmd/`

Les drapeaux sont lus et validés dans `cmd/filature/`. Aucun `flag` ni
`os.Getenv` dans `internal/` : les dépendances sont injectées.

Le corollaire est une règle de conception plus qu'une règle de style. Un choix
lu depuis un drapeau au fond du code ne peut plus venir d'ailleurs — un menu, un
message réseau, une sauvegarde reprise. Le camp joué, par exemple, est un
paramètre de partie que la ligne de commande renseigne ; ce n'est pas la ligne
de commande qui le détient.

## 5. Structure

- Une struct de dépendances par paquet, construite dans `cmd/filature/main.go`.
- Constructeurs `New…` rendant `(*T, error)` si l'initialisation peut échouer.
- Interfaces définies côté consommateur, pas côté implémentation.
- Une interface de plus de trois méthodes signale en général deux
  responsabilités mélangées. `Board` en a quatre parce qu'elle doit survivre au
  passage au plateau infini ; c'est la seule.

Un fichier porte le nom du type qu'il déclare — `game.go` pour `Game`,
`board.go` pour `Board`. C'est ce qui permet de trouver une déclaration sans
chercher.

**Un paquet ne se coupe pas en sous-paquets pour cause de volume.** En Go un
sous-répertoire est un paquet distinct : découper `internal/core` forcerait à
exporter des champs privés et créerait un cycle entre la génération et la
partie. Le découpage se fait par fichier, un sujet par fichier.

## 6. Erreurs

```go
if err != nil {
    return fmt.Errorf("chargement du plugin %s: %w", nom, err)
}
```

- Message en français, minuscules, sans ponctuation finale ni accent — la
  convention Go pour la casse, et le `grep` reste simple pour les accents.
- Sentinelles exportées quand l'appelant doit distinguer les cas :
  `ErrIllegalMove`, `ErrNoPlayableBoard`.
- Pas de `panic` hors de `main`, pas de `log.Fatal` dans `internal/`.
- **Pas de `panic("à implémenter")`.** Un bout non écrit rend
  `errors.New("à implémenter : étape N")`, où N renvoie à
  [`ROADMAP.md`](../ROADMAP.md) : la compilation passe, le programme échoue
  proprement, et un `grep` retrouve la liste. C'est la mesure d'avancement du
  projet, et elle ne ment pas.
- **Une signature sans `error` porte quand même son marqueur**, en commentaire
  sur la première ligne du corps. Sans lui la fonction est invisible au `grep`
  et le décompte ment.

**Un manquement dit où il est.** Fichier, ligne quand le décodeur la connaît, et
chemin complet de la clé — `ability.lookout.effect[0].target`, pas « cible
invalide ». Le chemin se porte dans l'erreur au fur et à mesure de la descente,
il ne se reconstitue pas après coup. Et ce qui est attendu s'énonce avec ce qui
est refusé : la liste des valeurs connues vaut mieux qu'un adjectif.

Un plugin invalide fait échouer le chargement entier plutôt que d'être ignoré :
un plugin à moitié actif est pire qu'un plugin absent. Ses manquements sont
listés en une fois — qui met au point un plugin veut la liste, pas un
aller-retour par erreur.

## 7. Écriture de fichiers

`0o600` pour un fichier, `0o750` pour un dossier. Ce que le jeu écrit — plugins
extraits, sauvegardes, préférences — appartient à qui l'a lancé et n'a aucune
raison d'être lisible par les autres comptes de la machine.

Ce sont les seuils qu'exige `gosec`, et aucun cas n'a justifié de s'en écarter.
Le jour où il y en aura un, c'est un `#nosec` commenté, pas un assouplissement
global.

## 8. Journalisation

`log/slog` uniquement, structuré. Clés en anglais, message en français :

```go
slog.Info("plugin charge", "name", nom, "version", v, "rules", regles)
```

Jamais de `fmt.Println` de débogage laissé derrière soi. La sortie d'erreur d'un
bot externe est capturée et journalisée telle quelle — c'est là que va son
débogage, jamais sur sa sortie standard, qui porte le protocole.

## 9. Concurrence

- `context.Context` en premier paramètre de toute fonction faisant de
  l'entrée-sortie, propagé jusqu'au bout. Pas de `context.Background()` hors de
  `main` et des tests.
- La simulation d'équilibrage parallélise des milliers de parties, **avec un
  plafond** : `runtime.NumCPU()`, pas une goroutine par partie.
- Toute goroutine lancée a une condition d'arrêt explicite et est attendue. Un
  bot externe qui ne répond pas est tué, pas laissé en fond.
- **Le parallélisme ne transparaît jamais dans un résultat.** Les parties
  simulées sont agrégées dans l'ordre de leur graine, jamais dans l'ordre
  d'arrivée.
- `make race` doit passer.

---

## 10. Tests

### Aucun test n'ouvre de fenêtre

Les runners d'intégration sont sans écran, et `internal/core` comme
`internal/ai` n'importent pas Ebitengine, ce qui rend la règle facile à tenir.
Un test qui exigerait un serveur X virtuel est le signe qu'il n'a rien à faire
dans la suite par défaut : le rendu se juge à l'œil.

### Le test qui vend le projet

```
partie jouée -> journal -> rejeu -> état identique octet pour octet
```

Sans lui, la reprise, l'annulation et l'entraînement de l'IA se dégradent sans
qu'on s'en aperçoive. Il tourne sur des parties engendrées à partir de graines
figées.

Son jumeau : deux parties d'IA contre IA sur la même graine produisent la même
suite de coups. C'est le filet du déterminisme.

### Fichiers de référence

La génération de plateau étant pure, elle se teste sans rien : une graine figée
en entrée, un plateau attendu en sortie, comparaison stricte.

Le jeu de graines couvre délibérément les cas pénibles — plateau très ouvert,
plateau labyrinthique, zone d'extraction en cul-de-sac.

Mise à jour groupée des attendus derrière `go test ./internal/core -maj-attendus`,
jamais automatique : un attendu régénéré sans être relu ne teste plus rien. Deux
autres artefacts suivent le même motif, `-maj-notices` et `-maj-schemas`.

### Cas limites des règles

Ceux qui se cassent en silence, à écrire en même temps que la règle :

- fugitif sans déplacement légal en début de phase ;
- contact avec trois inspecteurs et plus, plafond appliqué ;
- extraction interrompue par un inspecteur occupant la case visée ;
- zone scellée qui se ferme pendant l'étranglement ;
- tour de révélation tombant le même tour qu'un silence acheté ;
- dernier tour atteint avec l'extraction engagée mais non achevée.

### Un test se vérifie en le faisant échouer

Un test qui n'a jamais échoué ne prouve rien. Écrire l'invariant plutôt que le
cas nominal — pour un effet, qu'appliquer puis annuler rende un état identique à
l'original — et l'éprouver une fois en cassant volontairement ce qu'il garde.

C'est ce qui a révélé qu'une annulation laissait une tranche vide non nulle là
où il y avait `nil`, ce qui aurait fait diverger le rejeu du journal en JSON
sans que rien ne le signale.

### Conformité du protocole de bot

Le bot minimal de [`protocole-bot.md`](protocole-bot.md) est un cas de test : il
joue cent parties au hasard, aucune ne doit s'interrompre sur un coup illégal ou
un message mal formé. L'IA livrée passe par le même chemin — si le protocole
manquait de quelque chose, ce test échouerait.

### Ce qu'on ne teste pas

Pas de bouchon simulant Ebitengine, pas de comparaison de rendu pixel à pixel.
Un faux moteur ne testerait que la fidélité du faux.
