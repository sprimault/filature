# Contrat de formes

Version du contrat : **4**

Tout ce qui se dessine sur le plateau — pions, sol, bâtiments, marqueurs — est
décrit en géométrie, jamais en image. Un plugin d'apparence est un fichier
TOML.

Ce n'est pas une limitation de moyens, c'est ce qui rend un plugin publiable :
du texte se relit en diff, ne pose aucune question de provenance, et respecte le
gabarit par construction plutôt que par contrôle.

Le contenu livré avec le jeu est écrit dans ce format, au même titre qu'un
plugin tiers. Si une forme de base ne s'exprime pas avec les primitives
ci-dessous, c'est qu'il en manque une — pas qu'il faut un cas particulier.

---

## 1. Les deux plans

Une case fait **64 unités de large et 32 de haut** : c'est le losange
isométrique en rapport 2:1, et il n'est pas modifiable.

Les coordonnées se lisent dans l'un de deux plans, selon ce qu'on décrit. Les
confondre est l'erreur à ne pas faire.

**Plan du sol** — pour les marqueurs posés à plat sur une case. `x` et `y`
couvrent le losange vu en projection. Rien ne s'élève, tout est à plat.

```
         (0,-16)
           /\
          /  \
 (-32,0) /    \ (32,0)      losange de la case
         \    /
          \  /
           \/
         (0,16)
```

**Plan vertical** — pour les formes de rôle `piece` et `marker`, et pour la
`height` d'un prisme. L'origine est le **point d'ancrage au sol**, centre du
losange sur lequel la forme repose. `x` va vers la droite de l'écran, `y` vers
le haut. Le sol est donc `y = 0`, et une forme s'élève en `y` positif.

```
              y
              ^
         +----|----+          gabarit d'un pion
         |    |    |          x de -24 à 24
         |    |    |          y de 0 à 40
   ------+----o----+------ y = 0   ancrage, centre de la case
        -24   |    24
```

### Ce qu'un plugin peut changer, par rôle

| Rôle | Géométrie | Ce qui reste au moteur |
|---|---|---|
| le sol — rue, zones | **aucune** | le losange, figé |
| `building` | hauteur seule | l'emprise, qui est le losange |
| `piece` — fugitif, inspecteur | libre, dans le gabarit | rien |
| `marker` — trace, barrage, scène | libre, dans le gabarit | rien |

**Le sol ne se dessine pas.** Le losange est tracé par le moteur ; un plugin
n'en change que la couleur, par la palette. Il n'existe donc pas de
`shape.street` : les cinq noms de sol sont des noms de couleurs, pas des
formes.

La raison est mécanique. Un losange identique pour toutes les cases, c'est une
géométrie construite une fois pour les seize cents, un tri en profondeur qui
reste `colonne + ligne`, et une conversion écran vers plateau qui reste une
formule fermée. Avec une tuile de forme libre, il faudrait une boîte englobante
par case, un test de survol forme par forme, et le tri par ordre de peintre
cesserait d'être valide : tout le rendu repasserait en calcul, à chaque image.

S'y ajoute que la projection resterait sinon une propriété du plugin actif
plutôt que du jeu, et que deux joueurs ne verraient plus la même grille.

**Un bâtiment occupe exactement une case.** Son emprise est le losange, un
plugin ne déclare que sa hauteur et sa couleur. Cette contrainte n'est pas
esthétique : une emprise libre permettrait de déborder sur les cases voisines,
et de masquer ce que l'adversaire doit voir.

### Gabarits

| Rôle | Plan | x | y | Traits max |
|---|---|---|---|---|
| Pion | vertical | -24 à 24 | 0 à 40 | 24 |
| Marqueur | sol | -24 à 24 | -12 à 12 | 8 |
| Bâtiment | vertical | — | hauteur de 1 à 24 | 1 |

Un bâtiment plus haut serait plus joli et masquerait les cases derrière lui —
sur un jeu qui porte entièrement sur ce qu'on voit, ce n'est pas un choix
esthétique. Le plafond de 24 vaut une demi-case.

Le calcul est celui des marqueurs, pas celui des pions. Une case cachée par le
bâtiment devant elle l'est sur `[0, h]` d'un losange qui va de 0 à 32 : un pion
dépasse dans tous les cas, mais le sommet d'un marqueur est à 28. À 24 il en
reste 4 de visible, à 32 plus rien — et selon le préréglage, 64 à 72 % des cases
de rue ont du bâti juste devant. Trace, barrage et scène y disparaîtraient.

**Tous les bâtiments d'une partie ont la même hauteur, et rien ne s'empile.**
C'est ce qui rend l'occlusion prévisible : un joueur doit pouvoir dire d'un coup
d'œil ce qui est caché, sans mesurer.

Le gabarit est vérifié au chargement, pas seulement à la publication : un
plugin local qui déborde est refusé de la même façon. Une forme qui masque les
cases voisines est un avantage de jeu déguisé en habillage.

---

## 2. Les quatre primitives

Les traits sont dessinés dans l'ordre de déclaration, le dernier par-dessus.

### polygon

Surface pleine. De 3 à 32 sommets, fermée automatiquement.

```toml
[[shape.fugitive.stroke]]
type = "polygon"
points = [[-14, 0], [14, 0], [12, 10], [6, 10], [3, 16], [-8, 16], [-12, 10]]
color = "fugitive_main"
```

### circle

```toml
[[shape.fugitive.stroke]]
type = "circle"
center = [-8, 4]
radius = 4
color = "fugitive_detail"
```

### segment

Trait épais, extrémités arrondies.

```toml
[[shape.inspector.stroke]]
type = "segment"
from = [0, 20]
to = [0, 34]
thickness = 3
color = "inspector_detail"
```

### prism

Le losange de la case, extrudé. Réservé au rôle `building`, et seul trait qu'il
accepte.

```toml
[[shape.building.stroke]]
type = "prism"
height = 12
color = "building"
```

L'emprise n'est pas déclarable : c'est le losange, toujours. Le moteur dérive
les trois faces de la couleur donnée — un plugin ne gère ni l'éclairage, ni
l'ordre des faces, ni la forme au sol.

| Face | Coefficient |
|---|---|
| dessus | × 1,50 |
| droite | × 1,14 |
| gauche | × 0,72 |

Le coefficient s'applique aux trois canaux, ce qui préserve la teinte. Il est
écrit ici parce que c'est exactement le genre de détail que deux implémentations
règlent différemment sans que personne ne s'en aperçoive avant de comparer deux
captures.

### Attributs communs

| Clé | Valeur | Par défaut |
|---|---|---|
| `color` | nom de palette | obligatoire |
| `outline` | nom de palette | aucun |
| `outline_thickness` | 1 à 4 | 1 |
| `opacity` | 0 à 100 | 100 |

**`opacity` se paie.** Une valeur inférieure à 100 compose la couleur avec la
case en dessous, contour compris : la même forme devient claire sur la rue et
sombre sur une zone fermée, donc invisible dans les deux cas. C'est pour cette
raison qu'aucune forme livrée n'en pose. Ce qui tient un marqueur à sa place
d'indice, c'est sa finesse et le fait qu'il reste au sol, pas son effacement.

### Le liseré, posé par le moteur

Sous le contour de toute forme de rôle `piece` ou `marker`, le moteur pose un
liseré clair d'une unité. **Rien à déclarer, et rien à retirer.**

Un contour seul ne suffit pas, et c'est le point le moins évident du contrat. Il
tient contre le sol, qui est clair. Mais un pion se dessine par-dessus les cubes
situés devant lui : sa moitié supérieure est en permanence sur du bâti sombre,
où un contour sombre ne se voit plus. L'inverse vaut pour un contour clair, qui
meurt sur la rue. Les fonds possibles vont de 15 à 210 en luminance ; aucune
couleur unique ne les couvre.

| Fond | Contour seul | Avec liseré |
|---|---|---|
| hors plateau | 1,10 | **14,62** |
| face latérale d'un bâtiment | 1,25 | **10,69** |
| dessus d'un bâtiment | 2,65 | **5,03** |
| zone fermée | 2,21 | **6,01** |
| rue | 11,41 | 11,41 |

C'est de l'éclairage et non une couleur de plugin, au même titre que les
coefficients de faces. Une palette qui pourrait le déplacer pourrait rendre les
pions illisibles, ce que le liseré existe précisément pour empêcher.

**Toute épaisseur de contour est encadrée par deux bornes**, celle d'un plugin
comme le liseré du moteur : jamais moins d'**un pixel d'écran**, jamais plus
d'**un sixième de la plus petite dimension de la forme**. Entre les deux, elle
suit le zoom comme le reste de la géométrie.

Les deux bornes traitent le même défaut par ses deux bouts. Sans plancher, une
épaisseur mise à l'échelle passe sous le pixel au dézoom : l'antialiasing la mêle
à ce qu'elle devait séparer, et le contraste réel s'effondre bien avant la valeur
calculée — or c'est au plateau entier qu'on cherche où sont les pions. Sans
plafond, une épaisseur fixe finit par occuper la forme entière : à 24 pixels par
case, la tête d'un pion en fait quatre et demi, et deux pixels de liseré n'en
détachent plus rien. Ils avalent la couleur, qui est le seul signal
d'appartenance à un camp.

Le plafond porte sur le trait et non sur la case ni sur la forme entière : c'est
ce trait-là que le contour doit laisser voir, et un trait long et fin serait
avalé si l'on bornait par sa plus grande dimension. Chaque trait est encadré
pour lui-même, jamais leur somme — borner le total reviendrait à écraser le
liseré dès que le plafond mord, alors que c'est lui qui porte le contraste sur
le bâti.

Ce que les deux bornes garantissent, c'est qu'**un pion garde un remplissage
majoritaire** : 64 % de sa tête à 64 et 32 pixels par case, 57 % à 24. C'est la
couleur qui dit à quel camp il appartient, et elle ne doit pas se faire manger
par ce qui sert à la détacher.

**La propriété ne vaut que pour le rôle `piece`.** Un marqueur peut être plus
fin que ses bordures — la trace l'est, à 27 % de son épaisseur à 24 pixels par
case — et ce n'est pas un défaut : sa lisibilité vient précisément de ses
bordures, et il ne porte aucune identité de camp.

**En dessous de 24 pixels par case, le rendu ne garantit plus rien.** Le
plancher étant un minimum en pixels, il ne dépend pas de la taille du trait mais
de l'échelle, et finit par commander partout : à 16 pixels par case, la tête
d'un pion tombe à 47 % de remplissage. Le plateau entier tient dans 984 sur 492
pixels à cette limite — la projection est en deux pour un — ; en dessous, la vue
défile plutôt que de continuer à réduire.

La marge est mince et il vaut mieux le savoir : le plus grand préréglage fait 41
cases de côté, ce qui donne moins de cinquante pixels par case sur un écran de
1920 de large, et une trentaine sur un 1280.

Conséquence à connaître avant l'étape 7 : **le halo clair autour d'une pièce est
désormais pris.** C'est la convention habituelle pour « sélectionné » ou
« jouable », et elle n'est plus disponible. Les surcouches `cell_visible` et
`cell_playable` marquent donc le sol, jamais le pourtour d'un pion.

---

## 3. Les couleurs sont des noms

**Aucune valeur hexadécimale n'est acceptée dans une forme.** Un trait référence
un nom de palette, et rien d'autre.

C'est ce qui rend les deux catégories indépendantes : changer une palette
reteinte tout le jeu sans toucher aux formes, et une forme tierce suit
automatiquement la palette active. Un plugin de palette seule est le mod le
moins cher qui existe — un fichier de quinze lignes, aucun éditeur, aucune
géométrie.

`shapes_version` est à la racine du fichier, hors de la table `[palette]` : il
qualifie le fichier, pas la palette.

```toml
shapes_version = 4

[palette]
street = "#d8d2c4"
building = "#3a3f4a"
zone_open = "#7fa86b"
zone_closed = "#6b4a4a"
shelter_open = "#b5895a"
shelter_used = "#6b5a4a"
backdrop = "#0f1116"
fugitive_main = "#e07a45"
fugitive_detail = "#2a1a10"
inspector_main = "#2a5580"
inspector_detail = "#101c2a"
marker_outline = "#241d16"
trail = "#f0e6c8"
roadblock = "#8a7a4a"
```

**Deux sols se distinguent d'abord par la luminance, la teinte ne vient
qu'ensuite.** C'est la contrainte la plus facile à enfreindre de bonne foi :
deux bruns qu'un écran calibré sépare très bien deviennent une seule couleur en
niveaux de gris, sur un écran mal réglé, ou pour qui distingue mal les rouges et
les verts. Un plateau se lit en balayant, jamais en comparant deux cases côte à
côte — et c'est la luminance qui survit à ce balayage.

La règle vaut d'autant plus que les sols sont nombreux. Une palette qui les
range tous dans le tiers sombre de l'échelle les rend interchangeables, quelles
que soient leurs teintes ; les écarter en luminosité coûte moins qu'un jeu de
teintes savant, et se vérifie sur une capture désaturée.

**L'écart se juge après le grain, pas sur les valeurs déclarées.** Le moteur
déplace chaque case de sol de quelques pourcents (§8), donc deux couleurs
séparées de douze niveaux de gris se recouvrent partiellement une fois posées
sur le plateau. Lire la palette ne suffit pas : c'est le rendu qu'il faut
regarder, ce que `filature preview` donne sans lancer le jeu.

Les noms en `_detail` et `marker_outline` sont des **contours**, pas des nuances
d'accompagnement, et c'est la contrainte la moins évidente de la palette. Une
forme se pose indifféremment sur n'importe lequel des cinq sols, dont les
luminances vont de 210 à 84 : aucun remplissage ne se détache de tous. Une
palette qui remonterait ses contours au niveau de ses remplissages rendrait
pions et marqueurs illisibles sur les sols sombres, sans qu'aucun contrôle ne
puisse le voir.

`backdrop` est ce qu'on voit autour du plateau, et n'appartient à aucune forme.
Il doit être franchement plus sombre que `building` : à égalité, les blocs du
pourtour perdent leur silhouette et la ville se dissout sur ses bords, avec les
pièces qui s'y trouvent.

Les noms ci-dessus sont **obligatoires** : ils constituent le socle sur lequel
toute forme peut compter. Les cinq sols — `street`, `zone_open`, `zone_closed`,
`shelter_open`, `shelter_used` — n'ont d'ailleurs pas d'autre existence : ce
sont les prises qu'un plugin a sur le losange, et il n'en a pas d'autres. Une
palette peut ajouter des noms, et une forme qui référence un nom ajouté doit
livrer la palette qui le définit.

---

## 4. Surcharge partielle

Un plugin déclare **uniquement ce qu'il remplace**. Tout le reste retombe sur
le contenu de base.

Changer l'allure du fugitif tient donc en un dossier et deux fichiers :

```
mes-vehicules/
  manifest.toml
  shapes.toml
```

```toml
# manifest.toml
name = "mes-vehicules"
version = "1.0.0"
rules = false
license = "CC0-1.0"
description = "Le fugitif en voiture, les inspecteurs en gyrophare"
```

```toml
# shapes.toml
shapes_version = 4

[shape.fugitive]
[[shape.fugitive.stroke]]
type = "polygon"
points = [[-16, 0], [16, 0], [14, 8], [8, 8], [4, 15], [-10, 15], [-14, 8]]
color = "fugitive_main"

[[shape.fugitive.stroke]]
type = "circle"
center = [-9, 2]
radius = 3
color = "fugitive_detail"

[[shape.fugitive.stroke]]
type = "circle"
center = [9, 2]
radius = 3
color = "fugitive_detail"
```

Rien d'autre n'est nécessaire. Le sol, les bâtiments et les inspecteurs restent
ceux du jeu.

Deux plugins qui redéfinissent la même forme sont un conflit signalé au
chargement, pas un écrasement silencieux.

---

## 5. États

Les états d'une forme — surlignée, hors de vue, sélectionnée — sont produits par
le moteur : variation de teinte et contour. **Un plugin n'a rien à en
déclarer.**

Il peut le faire, à titre optionnel, en nommant la variante :

```toml
[shape.fugitive.highlighted]
```

En son absence, la variante automatique s'applique. C'est le défaut recommandé :
une forme qui déclare tous ses états sera fausse à la première évolution du
rendu.

---

## 6. Noms de formes

| Nom | Rôle |
|---|---|
| `building` | case bloquante |
| `fugitive` | pion du fugitif |
| `inspector` | pion d'inspecteur, teinté par numéro |
| `inspector_1` … `inspector_5` | surcharge par pion, facultative |
| `trail` | passage découvert |
| `roadblock` | case fermée par le Barreur |
| `cell_visible`, `cell_playable` | marqueurs de surcouche |

`inspector_1` à `5` n'existent que pour qui veut distinguer les cinq
capacités. En leur absence, `inspector` sert aux cinq, teinté.

Les cases de rue et les zones d'extraction n'apparaissent pas : ce sont des
couleurs, pas des formes.

---

## 7. Validation

Contrôles appliqués au chargement comme à la publication :

- schéma respecté, `shapes_version` connue ;
- tout point à l'intérieur du gabarit du rôle ;
- nombre de traits sous le plafond, polygones de 3 à 32 sommets ;
- toute `color` et tout `outline` résolus dans la palette active ;
- aucune valeur hexadécimale dans une forme ;
- pour un plugin d'apparence, `rules = false` **et** absence de toute
  capacité, dépense, effet ou module exécutable. La déclaration ne suffit pas,
  elle est vérifiée.

Le catalogue public ajoute une seule règle, mécanique : **aucun fichier binaire,
sous aucune extension.** Pas d'image, pas de son, pas d'exécutable. C'est ce qui
supprime toute question de provenance, donc toute relecture humaine.

Hors catalogue, rien de tout cela ne s'applique au-delà du gabarit : un dossier
posé à la main se charge tel quel.

---

## 8. Le grain du sol

Un plateau de couleurs pleines est plat à l'œil sur seize cents cases. Le moteur
applique donc à chaque case de sol un écart de luminosité de quelques pourcents,
dérivé de sa position et de la graine du plateau.

**Ce n'est pas une primitive et ça ne se déclare pas.** Aucun plugin n'a à en
tenir compte, et tous en bénéficient — y compris ceux qui ne changent que la
palette. C'est la raison d'avoir écarté une cinquième primitive de motif : elle
aurait alourdi le contrat public de façon permanente pour un besoin décoratif,
en déplaçant la responsabilité du fini visuel vers les auteurs de plugins.

Trois contraintes, qui sont ce qui fait la différence entre du grain et du
bruit :

- **luminosité seule**, jamais la teinte — au-delà, les cases cessent de se lire
  comme un sol continu ;
- **stable** : dérivé de la position et de la graine, jamais retiré au sort à
  l'affichage. Un scintillement au défilement serait pire que l'uni ;
- **le sol seulement.** Pions et bâtiments gardent leur couleur exacte, qui doit
  rester identifiable d'un coup d'œil.

Il se coupe entièrement en vue à plat, où l'uniformité aide à lire les
surcouches.

Conséquence à accepter : deux plateaux de graines différentes n'ont pas le même
grain, donc une capture d'écran n'est pas reproductible sans sa graine.

---

## 9. Ce que ce contrat ferme

**Le pixel-art et le dessin à la main ne sont pas publiables.** C'est le prix du
refus de vérifier les provenances, assumé.

**Le losange est figé, et le sol n'est pas une forme.** Aucun plugin ne
redéfinit la tuile de sol ni l'emprise d'un bâtiment, et rien n'est prévu pour
l'ouvrir : entrebâiller la porte coûterait de la complexité
dans tout le rendu pour un besoin hypothétique. Une grille hexagonale ou une vue
de dessus seraient une rupture du contrat, avec un `shapes_version` incrémenté,
pas une extension.
