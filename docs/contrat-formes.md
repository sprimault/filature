# Contrat de formes

Version du contrat : **1**

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

**Plan vertical** — pour les formes de rôle `pion` et `marqueur`, et pour la
`hauteur` d'un prisme. L'origine est le **point d'ancrage au sol**, centre du
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
| `sol` — rue, zones | **aucune** | le losange, figé |
| `batiment` | hauteur seule | l'emprise, qui est le losange |
| `pion` — fugitif, inspecteur | libre, dans le gabarit | rien |
| `marqueur` — trace, barrage, scène | libre, dans le gabarit | rien |

**Le sol ne se dessine pas.** Le losange est tracé par le moteur ; un plugin
n'en change que la couleur, par la palette. Il n'existe donc pas de
`forme.rue` : `rue`, `zone_ouverte` et `zone_fermee` sont des noms de couleurs,
pas des formes.

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

Le gabarit est vérifié au chargement, pas seulement à la publication : un
plugin local qui déborde est refusé de la même façon. Une forme qui masque les
cases voisines est un avantage de jeu déguisé en habillage.

---

## 2. Les quatre primitives

Les traits sont dessinés dans l'ordre de déclaration, le dernier par-dessus.

### polygone

Surface pleine. De 3 à 32 sommets, fermée automatiquement.

```toml
[[forme.fugitif.trait]]
type = "polygone"
points = [[-14, 0], [14, 0], [12, 10], [6, 10], [3, 16], [-8, 16], [-12, 10]]
couleur = "fugitif_principal"
```

### cercle

```toml
[[forme.fugitif.trait]]
type = "cercle"
centre = [-8, 4]
rayon = 4
couleur = "fugitif_detail"
```

### segment

Trait épais, extrémités arrondies.

```toml
[[forme.inspecteur.trait]]
type = "segment"
de = [0, 20]
a = [0, 34]
epaisseur = 3
couleur = "inspecteur_detail"
```

### prisme

Le losange de la case, extrudé. Réservé au rôle `batiment`, et seul trait qu'il
accepte.

```toml
[[forme.batiment.trait]]
type = "prisme"
hauteur = 12
couleur = "batiment"
```

L'emprise n'est pas déclarable : c'est le losange, toujours. Le moteur assombrit
les faces latérales selon leur orientation à partir de la couleur donnée — un
plugin ne gère ni l'éclairage, ni l'ordre des faces, ni la forme au sol.

### Attributs communs

| Clé | Valeur | Par défaut |
|---|---|---|
| `couleur` | nom de palette | obligatoire |
| `contour` | nom de palette | aucun |
| `epaisseur_contour` | 1 à 4 | 1 |
| `opacite` | 0 à 100 | 100 |

---

## 3. Les couleurs sont des noms

**Aucune valeur hexadécimale n'est acceptée dans une forme.** Un trait référence
un nom de palette, et rien d'autre.

C'est ce qui rend les deux catégories indépendantes : changer une palette
reteinte tout le jeu sans toucher aux formes, et une forme tierce suit
automatiquement la palette active. Un plugin de palette seule est le mod le
moins cher qui existe — un fichier de quinze lignes, aucun éditeur, aucune
géométrie.

`version_formes` est à la racine du fichier, hors de la table `[palette]` : il
qualifie le fichier, pas la palette.

```toml
version_formes = 1

[palette]
rue = "#d8d2c4"
batiment = "#3a3f4a"
zone_ouverte = "#7fa86b"
zone_fermee = "#6b4a4a"
fugitif_principal = "#c85a3c"
fugitif_detail = "#8a3a26"
inspecteur_principal = "#2f5f8f"
inspecteur_detail = "#1c3d5c"
trace = "#a89c84"
barrage = "#8a7a4a"
```

Les noms ci-dessus sont **obligatoires** : ils constituent le socle sur lequel
toute forme peut compter. `rue`, `zone_ouverte` et `zone_fermee` n'ont d'ailleurs
pas d'autre existence — ce sont les seules prises qu'un plugin a sur le sol. Une palette peut en ajouter, et une forme qui
référence un nom ajouté doit livrer la palette qui le définit.

---

## 4. Surcharge partielle

Un plugin déclare **uniquement ce qu'il remplace**. Tout le reste retombe sur
le contenu de base.

Changer l'allure du fugitif tient donc en un dossier et deux fichiers :

```
mes-vehicules/
  manifeste.toml
  formes.toml
```

```toml
# manifeste.toml
nom = "mes-vehicules"
version = "1.0.0"
regles = false
licence = "CC0-1.0"
description = "Le fugitif en voiture, les inspecteurs en gyrophare"
```

```toml
# formes.toml
version_formes = 1

[forme.fugitif]
[[forme.fugitif.trait]]
type = "polygone"
points = [[-16, 0], [16, 0], [14, 8], [8, 8], [4, 15], [-10, 15], [-14, 8]]
couleur = "fugitif_principal"

[[forme.fugitif.trait]]
type = "cercle"
centre = [-9, 2]
rayon = 3
couleur = "fugitif_detail"

[[forme.fugitif.trait]]
type = "cercle"
centre = [9, 2]
rayon = 3
couleur = "fugitif_detail"
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
[forme.fugitif.surligne]
```

En son absence, la variante automatique s'applique. C'est le défaut recommandé :
une forme qui déclare tous ses états sera fausse à la première évolution du
rendu.

---

## 6. Noms de formes

| Nom | Rôle |
|---|---|
| `batiment` | case bloquante |
| `fugitif` | pion du fugitif |
| `inspecteur` | pion d'inspecteur, teinté par numéro |
| `inspecteur_1` … `inspecteur_5` | surcharge par pion, facultative |
| `trace` | passage découvert |
| `barrage` | case fermée par le Barreur |
| `scene` | lieu d'un meurtre, connu des deux camps |
| `case_visible`, `case_jouable` | marqueurs de surcouche |

`inspecteur_1` à `5` n'existent que pour qui veut distinguer les cinq
capacités. En leur absence, `inspecteur` sert aux cinq, teinté.

Les cases de rue et les zones d'extraction n'apparaissent pas : ce sont des
couleurs, pas des formes.

---

## 7. Validation

Contrôles appliqués au chargement comme à la publication :

- schéma respecté, `version_formes` connue ;
- tout point à l'intérieur du gabarit du rôle ;
- nombre de traits sous le plafond, polygones de 3 à 32 sommets ;
- toute `couleur` et tout `contour` résolus dans la palette active ;
- aucune valeur hexadécimale dans une forme ;
- pour un plugin d'apparence, `regles = false` **et** absence de toute
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

Il se coupe entièrement en vue de débogage, où l'uniformité aide à lire les
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
de dessus seraient une rupture du contrat, avec un `version_formes` incrémenté,
pas une extension.
