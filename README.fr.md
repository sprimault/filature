# Filature

English: [README.md](README.md)

[![CI](https://github.com/sprimault/filature/actions/workflows/ci.yml/badge.svg)](https://github.com/sprimault/filature/actions/workflows/ci.yml)

Un jeu de plateau au tour par tour, à déplacement caché. Un fugitif contre cinq
inspecteurs, sur une ville de rues et de bâtiments.

Apache 2.0 — [`LICENSE`](LICENSE), et [`THIRD-PARTY-NOTICES`](THIRD-PARTY-NOTICES)
pour les bibliothèques liées au binaire.

Le fugitif se déplace dans huit directions, les inspecteurs dans quatre. Il est
plus rapide, mais seul et invisible la plupart du temps. Ils sont nombreux et
voient loin, mais ne peuvent pas couvrir six zones d'extraction à cinq, et ne
savent presque jamais où il est.

L'asymétrie fait le jeu : le fugitif sait où il va, les inspecteurs doivent le
deviner.

## État

**Le binaire joue, en texte.** Le rendu isométrique est l'étape 7, le mode
réseau l'étape 12, et l'adversaire choisit encore au hasard — la véritable
IA vient aux étapes 9 et 10.

Les plugins d'apparence se chargent et se valident déjà, bien que rien ne les
dessine encore : une forme qui déborde de son gabarit ou référence une couleur
absente est refusée au démarrage.

- [`docs/regles.md`](docs/regles.md) — la spécification complète
- [`ROADMAP.md`](ROADMAP.md) — les étapes, et ce qui est hors périmètre v1

## Installation

Télécharger un binaire depuis la page des versions, extraire, lancer. Aucun
runtime, aucune dépendance système.

**L'archive ne contient que l'exécutable**, avec la licence. Les règles, les
formes et les libellés sont dedans : rien à installer à côté, et déplacer le
fichier ne casse rien.

Le binaire Windows n'est pas signé : SmartScreen affichera un avertissement au
premier lancement.

### Les commandes

Le jeu et ses commandes se lancent depuis un terminal, pas en double-cliquant — leur
sortie est du texte, et une fenêtre ouverte par l'explorateur se referme avant
qu'on ait pu lire quoi que ce soit.

```
filature                      joue une partie, en texte
filature version              le numéro de la version installée
filature examples <dossier>   écrit les plugins livrés, pour servir de modèle
filature validate <dossier>   contrôle un plugin et affiche son empreinte
filature preview <dossier>    rend ses formes et un plateau en SVG
```

`--version` est accepté comme équivalent de `version`.

`validate` applique exactement les contrôles du chargement, et liste tout ce qui
cloche d'un coup plutôt qu'une faute à la fois. Un plugin qu'il accepte se
chargera chez ceux à qui vous le donnez.

`preview` écrit deux fichiers : la planche des formes, chacune sur les trois
sols possibles, et un plateau en situation. Le plugin est fusionné sur le
contenu livré avant d'être rendu — il ne déclare que ce qu'il remplace — et la
planche marque ce qui vient de lui, ce qui montre du même coup qu'une clé mal
orthographiée n'a rien surchargé. Un second argument dit où écrire.

`examples` ne peut pas écrire dans le dossier des plugins actifs, et le refuse
en le disant : ce qu'il en sort est déjà dans le binaire, l'y remettre le
déclarerait deux fois. Ce sont des modèles à recopier sous un autre nom.

### Les drapeaux

Lancé sans argument, le jeu vous donne le camp des inspecteurs face à un
adversaire.

```
--side fugitive     prend l'autre rôle ; inspectors par défaut,
                    watch regarde deux machines s'affronter
--preset ville      taille du plateau : quartier, faubourg ou ville
--seed 1            la graine ; la même rejoue la même partie
--delay 800ms       pause entre deux tours quand personne ne joue ; sans
                    elle la partie entière défile d'un bloc
--plugins <dossier> où chercher les plugins
--host              héberger une partie en réseau
--join <adresse>    en rejoindre une
--game <nom>        reprendre une partie enregistrée
```

Le jeu cherche ses plugins dans un dossier `plugins` **à côté de
l'exécutable**, et non dans le répertoire courant — un raccourci fonctionne donc
comme un lancement direct. `--plugins` pointe ailleurs si besoin.

## Étendre le jeu

Filature est fait pour être modifié. Quatre niveaux, du moins cher au plus
engageant :

| Niveau | Ce que c'est | Format |
|---|---|---|
| Données | Capacités, dépenses, préréglages, modes de jeu | TOML |
| Apparence | Formes et palettes | TOML |
| Bots | Une IA de remplacement, dans n'importe quel langage | processus séparé |
| WebAssembly | Un générateur de plateau ou une IA embarquée | `.wasm` |

Un plugin d'apparence est de la **géométrie, jamais une image**. Voir
[`docs/contrat-formes.md`](docs/contrat-formes.md) : du texte se relit en diff,
ne pose aucune question de provenance, et respecte le gabarit par construction.
La contrepartie y est écrite — le dessin à la main ne sera jamais publiable par
le catalogue.

Un bot remplace l'IA du jeu plutôt que de l'étendre : le jeu envoie une vue, le
bot renvoie un coup. L'IA livrée parle le même protocole, ce qui prouve qu'il
suffit. Voir [`docs/protocole-bot.md`](docs/protocole-bot.md).

Hors catalogue, rien n'est restreint. C'est ta machine.

## Pour aller plus loin

- [`docs/architecture.md`](docs/architecture.md) — le noyau, la vue filtrée, le
  déterminisme
- [`docs/regles.md`](docs/regles.md) — les règles complètes et leurs chiffres
- [`docs/plugins.md`](docs/plugins.md) — le format d'un plugin, fichier par
  fichier et champ par champ
- [`docs/vocabulaire-effets.md`](docs/vocabulaire-effets.md) — les primitives
  dont se composent capacités, dépenses et modes de jeu
- [`docs/construction.md`](docs/construction.md) — matrice de compilation,
  empaquetage, signature
- [`schemas/`](schemas/) — les contrats publics, versionnés à part
- [`CONTRIBUTING.fr.md`](CONTRIBUTING.fr.md) — ce sur quoi une contribution est
  jugée

## D'où vient le nom

Une filature, c'est l'acte de suivre quelqu'un sans être vu. Le mot nomme les
deux camps à la fois : on file, et on est filé.

## Licence

Apache 2.0. **Le nom Filature, son identité visuelle et sa palette n'en font
pas partie** — un fork est bienvenu sous un autre nom.

Le binaire embarque des bibliothèques tierces dont les licences exigent que
leurs notices l'accompagnent : `THIRD-PARTY-NOTICES` est livré dans chaque
archive, à côté de `LICENSE`.
