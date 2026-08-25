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

Chaque section est **bilingue, français d'abord, séparé par `***`** — les notes
de version sont ce que lit un auteur de greffon étranger avant de savoir s'il
doit reprendre son travail. Ce préambule reste en français : il n'est jamais
publié, et explique les conventions du dépôt à qui y contribue.

## [Non publié]

### Ajouté
- Génération de plateau : trame irrégulière, cours percées, impasses et six
  zones d'extraction réparties sur le pourtour. Une graine donnée produit
  toujours le même plateau, vérifié sur Windows et Linux.
- Trois préréglages de partie — Quartier, Faubourg et Ville — là où seule la
  Ville existait. La portée de vue et la durée suivent la taille du plateau.

### Corrigé
- Les bornes des paramètres avaient des valeurs sans nom ni justification, et
  les refus ne disaient ni la valeur reçue ni ce qui était attendu. Le nombre de
  pions déplaçables n'était pas contrôlé.

***

### Added
- Board generation: irregular street grid, punched courtyards, dead ends and six
  extraction zones spread around the perimeter. A given seed always produces the
  same board, verified on Windows and Linux.
- Three game presets — District, Outskirts and City — where only the City
  existed. Sight range and duration follow the board size.

### Fixed
- Parameter bounds were unnamed magic numbers, and rejections stated neither the
  value received nor what was expected. The number of movable pieces was not
  checked at all.

## [0.2.0] — 2026-08-25 — La vue filtrée

Étape 2 de la feuille de route. Les quatre invariants de l'architecture sont
désormais posés et gardés par des tests.

**Le binaire ne joue toujours pas** : la boucle de jeu est l'étape 5.

`schemas/vue.schema.json` paraît pour la première fois. Un bot peut s'écrire
contre lui — c'est exactement ce que le jeu enverra, puisque le schéma est
généré depuis la structure et qu'un test échoue s'ils divergent.

### Ajouté
- Vue filtrée : chaque camp ne reçoit que ce qu'il a le droit de savoir. La
  position du fugitif, sa zone scellée, sa résistance et les traces hors de
  portée ne franchissent pas la projection.
- `schemas/vue.schema.json`, généré depuis `noyau.Vue` et non écrit à la main.
- La validation d'un greffon avant installation est prévue pour l'étape 8 :
  `filature valide <chemin>`, avec le fichier, la ligne et le chemin de clé de
  chaque manquement.

### Corrigé
- Les listes de la vue rendaient `null` au lieu d'un tableau vide, ce qui
  obligeait un bot à traiter deux formes pour la même absence.
- Les traces se découvrent en distance de Manhattan, comme la règle l'énonce, et
  non de Tchebychev qui aurait ouvert les quatre diagonales.

***

Step 2 of the roadmap. All four architectural invariants are now in place and
guarded by tests.

**The binary still does not play**: the game loop is step 5.

`schemas/vue.schema.json` appears for the first time. A bot can be written
against it — it is exactly what the game will send, since the schema is
generated from the structure and a test fails if the two drift apart.

### Added
- Filtered view: each side receives only what it is entitled to know. The
  fugitive's position, sealed zone, stamina and out-of-range trails do not cross
  the projection.
- `schemas/vue.schema.json`, generated from `noyau.Vue` rather than written by
  hand.
- Validating a plugin before install is planned for step 8: `filature valide
  <path>`, reporting the file, line and key path of each failure.

### Fixed
- The view's lists returned `null` instead of an empty array, forcing a bot to
  handle two shapes for the same absence.
- Trails are found at Manhattan distance, as the rules state, not Chebyshev
  which would have opened the four diagonals.

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

***

Two fixes to the binary. The core is unchanged, and the game still does not
launch — the loop is step 5.

### Fixed
- The plugin folder is looked up next to the executable, not in the current
  directory. Launched from a shortcut, the game silently ignored installed
  plugins.
- `filature exemples` refuses to write into the active plugin folder. Shipped
  content would have been declared twice — once from the binary, once from
  disk — and two plugins defining the same key are a conflict.

### Added
- The README documents the binary's two commands and where plugins live.

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

***

Step 1 of the roadmap: the core. A full game plays out from Go calls, with no
interface.

**The binary does not play yet.** Running `filature` reports that it remains to
be implemented: the game loop is step 5. This release marks a development
milestone, not a playable version.

Nothing for plugin authors to update: `version_formes`, `protocole` and
`version_effets` are all 1, published here for the first time.

### Added
- Project skeleton, shape and bot contracts.
- Effect vocabulary documented, with the `differer` and `ouvrir_case`
  primitives.
- Deferred effect queue in the state, and announced deferrals in the view.
- Declarative game modes, and `version_effets` in the manifest.
- Strangling expressed as effects in `greffons/base`, no longer hardcoded.
- Documentation of a plugin's format, and a `bot` table in the manifest schema.
- Apply and undo for the seventeen effect primitives.
- Language plugins, with French as fallback and English shipped.
- Shipped content embedded in the binary, and `filature exemples` to write it
  back out.
- Legal move enumeration, and the zone-change expense missing from shipped
  content.
- Apply and undo for a move, with phase sequencing.
- Deterministic generator with named streams, the game's only randomness.
- End-of-turn resolution: capped contacts, trails, periodic reveal, deferred
  effects and strangling.
- Arbitration: extraction over two turns, zone neutralised by an inspector or by
  its closing, and the inspectors' three victories.
- Game setup, with the board received rather than built.
