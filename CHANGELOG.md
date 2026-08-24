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

## [Non publié]

### Ajouté
- Squelette du projet, contrats de formes et de bot.
- Vocabulaire d'effets documenté, avec les primitives `differer` et
  `ouvrir_case`.
- File des effets différés dans l'état, et les différés annoncés dans la vue.
- Modes de jeu déclaratifs, et `version_effets` au manifeste.
