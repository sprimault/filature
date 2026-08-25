# Journal des versions

Le format suit [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/).

SemVer avec la clause du zéro : **en `0.x`, rien n'est imposé**. Le mineur
marque un jalon de la feuille de route, pas une rupture d'API — tout le reste
s'accumule en correctif, correctifs, fonctionnalités et ruptures confondus.

Quatre numéros à ne pas confondre :

| Numéro | Où | Ce qu'il suit |
|---|---|---|
| version du dépôt | tag git | le binaire |
| `version_formes` | chaque fichier de formes | le contrat d'apparence |
| `protocole` | échanges avec un bot | le contrat de bot |
| `version_effets` | manifeste d'un greffon de règles | le vocabulaire d'effets |

Les trois derniers sont des entiers sans rapport avec SemVer. Une version peut
sortir sans qu'ils bougent ; ils ne bougent jamais sans version.

Un titre de section s'écrit `## [version] — date — titre`. La publication en
tire le nom et les notes de la version : ce qui est relu ici est ce qui sera
lu sur la page des versions, et il n'y a rien à recopier ensuite. Le titre est
facultatif ; sans lui, la version se nomme par son tag.

## [Non publié]

## [0.1.1] — 2026-08-25 — Correctifs du binaire

### Corrigé
- Le dossier des greffons est cherché à côté de l'exécutable et non dans le
  répertoire courant. Lancé par un raccourci, le jeu ignorait les greffons
  installés sans rien dire.
- `filature exemples` refuse d'écrire dans le dossier des greffons actifs. Le
  contenu livré y aurait été déclaré deux fois — une fois depuis le binaire, une
  fois depuis le disque — et deux greffons qui définissent la même clé sont un
  conflit.

### Ajouté
- Le README décrit les deux commandes du binaire et l'emplacement des greffons.

## [0.1.0] — 2026-08-25 — Le noyau

Étape 1 de la feuille de route : le noyau. Une partie complète se joue depuis
des appels Go, sans interface.

**Le binaire ne joue pas encore.** `filature` lancé répond qu'il reste à
implémenter : la boucle de jeu est l'étape 5. Cette version marque un jalon de
développement, pas une version jouable.

Rien à reprendre pour un auteur de greffon : `version_formes`, `protocole` et
`version_effets` valent 1, et c'est leur première publication.

### Ajouté
- Squelette du projet, contrats de formes et de bot.
- Vocabulaire d'effets documenté, avec les primitives `differer` et
  `ouvrir_case`.
- File des effets différés dans l'état, et les différés annoncés dans la vue.
- Modes de jeu déclaratifs, et `version_effets` au manifeste.
- Étranglement exprimé en effets dans `greffons/base`, plus codé en dur.
- Documentation du format d'un greffon, et table `bot` au schéma de manifeste.
- Application et annulation des dix-sept primitives d'effets.
- Greffons de langue, avec le français en repli et l'anglais livré.
- Contenu livré embarqué dans le binaire, et `filature exemples` pour l'en
  ressortir.
- Énumération des coups légaux, et la dépense de changement de zone qui
  manquait au contenu livré.
- Application et annulation d'un coup, avec l'enchaînement des phases.
- Générateur déterministe à flux nommés, seul aléatoire du jeu.
- Résolution de fin de tour : contacts plafonnés, traces, révélation
  périodique, effets différés et étranglement.
- Arbitrage : extraction sur deux tours, zone neutralisée par un inspecteur ou
  par sa fermeture, et les trois victoires des inspecteurs.
- Mise en place d'une partie, plateau reçu et non fabriqué.
