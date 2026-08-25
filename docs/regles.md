# Règles

Les règles complètes et leurs chiffres. Ce document fait foi : le code s'y
conforme, pas l'inverse.

---

## 1. Ce qu'est le jeu

Deux joueurs, tour par tour, information imparfaite asymétrique.

Un **fugitif** doit rejoindre l'un des six points d'extraction de la ville et s'y
maintenir un tour complet. Cinq **inspecteurs** doivent l'en empêcher, en le
piégeant ou en épuisant sa résistance.

Le fugitif est plus rapide (8 directions contre 4) mais seul et invisible la
plupart du temps. Les inspecteurs sont nombreux et voient loin, mais ne peuvent
pas couvrir six zones à cinq, et ne savent presque jamais où il est.

L'asymétrie centrale : le fugitif sait où il va, les inspecteurs doivent le
deviner.

---

## 2. Décisions tranchées

Ces points étaient contradictoires ou ouverts dans l'énoncé initial. Ils sont
figés ici.

**Ordre du tour.** Les inspecteurs jouent avant le fugitif. L'énoncé initial
disait l'inverse, mais les inspecteurs disposent de trois déplacements contre un :
donner au fugitif la position réactive est ce qui compense. Il voit le barrage se
former avant de choisir.

**Téléportation supprimée.** Un inspecteur qui repère le fugitif ne saute plus sur
sa case : il gagne un déplacement supplémentaire immédiat, hors quota. La pression
est conservée, la partie ne se termine plus sur un seul coup.

**Décompte des contacts.** Un point de résistance par inspecteur orthogonalement
adjacent, évalué une seule fois par tour, à la fin du tour complet, plafonné à
trois. Être encerclé doit faire très mal sans être instantanément fatal.

**Trois pions par tour, pas cinq.** L'énoncé initial laissait les inspecteurs
déplacer tous leurs pions. Le quota est ce qui fait de leur phase une décision :
arbitrer entre garder les zones et resserrer le filet, au lieu d'avancer tout le
monde et de regarder. L'objection — cinq déplacements contre un serait écrasant
— est plus faible qu'elle n'en a l'air, puisque les inspecteurs ignorent où
viser : leur problème est l'information, pas la vitesse.

C'est un paramètre et non une constante. Trois par défaut, mesuré à l'étape 11,
et le premier chiffre à corriger si l'équilibrage penche.

**Placement.** Le fugitif est placé aléatoirement dans le noyau central (rayon 5).
Les inspecteurs se placent ensuite librement sur n'importe quelle case claire,
sans connaître sa position exacte — seulement la zone. Le premier vrai choix de
la partie : verrouiller les sorties ou resserrer sur le centre.

**Diagonales.** Le fugitif ne peut pas franchir un angle fermé : le déplacement
diagonal exige qu'au moins une des deux cases orthogonales intermédiaires soit une
rue. Même règle pour les lignes de vue diagonales.

**Bords.** La v1 est bornée. Le plateau infini par tuiles reste l'objectif v2 :
la mécanique d'extraction le rend jouable, et l'interface `Plateau` est conçue
pour que le passage ne touche aucune règle.

---

## 3. Le plateau

Grille de rues et de bâtiments. Les rues sont praticables, les bâtiments bloquent
déplacement et vision.

Génération déterministe depuis une graine :

1. Trame de rues orthogonales à intervalle irrégulier (3 à 6 cases).
2. Remplissage des îlots en bâtiments.
3. Perçage aléatoire de passages et de cours, pour casser la régularité.
4. Creusement de quelques impasses — sans bords, ce sont elles qui permettent le
   piégeage.
5. Placement des six zones d'extraction, en périphérie, réparties angulairement.
6. **Validation** : toutes les cases claires forment une seule composante
   connexe, chaque zone est atteignable, le taux de rues tombe entre 35 % et 50 %.
   Un plateau qui échoue est rejeté et régénéré avec la graine suivante.

Une zone d'extraction est un bloc de 3×3 dont au moins 5 cases sont des rues.
Les six zones sont visibles des deux joueurs dès le début.

---

## 4. Mise en place

1. Génération du plateau depuis la graine.
2. Placement aléatoire du fugitif dans le noyau central. Position cachée.
3. Le fugitif choisit secrètement sa zone d'extraction. **Le choix est scellé** :
   en changer coûtera 2 points de résistance.
4. Les inspecteurs placent leurs cinq pions, librement, sur des cases claires.
5. Le tour 1 commence.

---

## 5. Déroulement d'un tour

**Phase inspecteurs**

- Jusqu'à 3 pions **parmi les 5** déplacés, d'une case, en orthogonal uniquement.
  Le joueur choisit lesquels : c'est là qu'est sa décision de tour.
- Un pion qui repère le fugitif pendant la phase gagne un déplacement
  supplémentaire immédiat, hors quota.
- Une capacité peut être déclenchée, une seule par tour, une seule fois par pion
  et par partie.

**Phase fugitif**

- Un déplacement d'une case, dans les 8 directions.
- Ou une dépense de résistance (voir §7), qui peut se cumuler avec le déplacement.
- Le fugitif peut passer son tour.

**Le quota compte les actions, pas les résultats.** Un pion qui se déplace puis
revient sur sa case a consommé son déplacement, sinon un joueur sonderait le
terrain gratuitement.

**Plusieurs inspecteurs peuvent tenir la même case**, et n'y gagnent rien : une
ligne de vue s'arrête au premier d'entre eux, et le contact plafonne à trois
quel que soit leur nombre. **Le fugitif, lui, ne peut pas entrer sur une case
occupée.** Il y serait à l'abri de tout, puisque le contact exige l'adjacence et
qu'une case partagée n'est pas adjacente à elle-même.

**Une phase se termine quand son camp rend la main.** Les inspecteurs ne sont
pas tenus de déplacer trois pions, le fugitif n'est pas tenu de bouger, et
aucun des deux n'est mis dehors parce qu'il a épuisé son quota — un joueur qui
garde un déplacement en réserve fait un choix, pas une erreur.

**Le tour n'est résolu qu'une fois les deux phases jouées.** Rien de ce qui
suit n'a lieu entre elles.

**Résolution de fin de tour**

1. Calcul de la visibilité. Si le fugitif est dans une ligne de vue, son pion
   devient visible et le reste jusqu'à ce qu'il en sorte.
2. Contacts : perte de résistance selon §7.
3. Traces : dépôt sur la case quittée, vieillissement des traces existantes.
4. Révélation périodique si le tour est multiple de 4.
5. Fermeture de zone si le tour l'impose (§8).
6. Test des conditions de fin (§8).

---

## 6. Vision et information

Chaque inspecteur voit dans 8 directions, jusqu'à 8 cases, la ligne s'arrêtant au
premier bâtiment. La vision est **calculée avant le déplacement du fugitif comme
après** : un fugitif qui sort d'une ligne de vue redevient invisible.

**Révélation périodique.** Tous les 4 tours, la position exacte du fugitif est
affichée aux deux joueurs. Sans ce battement, l'incertitude des inspecteurs ne
converge jamais et la partie devient une loterie. C'est aussi ce qui borne la
carte de croyance de l'IA quelle que soit la taille du plateau.

Le fugitif peut acheter le silence (3 points) pour sauter une révélation — mais
les inspecteurs sont informés qu'il a payé. Ils ne savent pas où il est, ils
savent qu'il s'est appauvri.

---

## 7. Résistance

10 points au départ. C'est à la fois une jauge de survie et une monnaie.

| Dépense | Coût | Effet |
|---|---|---|
| Double déplacement | 2 | Deux cases au lieu d'une, ce tour |
| Silence | 3 | Annule la prochaine révélation périodique |
| Effacement | 1 | Supprime toutes ses traces de moins de 3 tours |
| Changement de zone | 2 | Rescelle une autre zone d'extraction |

**Contact** : 1 point par inspecteur orthogonalement adjacent en fin de tour,
plafond de 3 par tour. Les diagonales ne comptent pas.

Un fugitif à 2 points est acculé même s'il est encore libre : il n'a plus les
moyens de percer un barrage. La jauge cesse d'être subie, chaque point devient
un arbitrage.

---

## 8. Traces

Le fugitif dépose une trace sur chaque case qu'il quitte. Une trace porte le
numéro du tour et la direction prise.

Elle est **invisible à distance**. Un inspecteur ne la découvre qu'en occupant la
case ou une case orthogonalement adjacente. Elle s'efface au bout de 6 tours.

C'est ce qui donne aux inspecteurs quelque chose à chercher entre deux
révélations, ce qui récompense la patrouille plutôt que le campement, et ce qui
fournit à l'IA un objet de jeu concret.

---

## 9. Les cinq inspecteurs

Une capacité par pion, une seule utilisation par partie.

| Pion | Capacité |
|---|---|
| Guetteur | Portée de vue doublée pendant un tour |
| Coureur | Déplacement de deux cases ce tour |
| Traqueur | Perçoit les traces à deux cases, en permanence (passif) |
| Barreur | Ferme une case de rue pendant 3 tours |
| Chef | Voit ce que voit un autre inspecteur pendant deux tours |

Le Barreur mérite attention : c'est le seul moyen de créer une impasse là où il
n'y en a pas. Sur un plateau ouvert, c'est lui qui rend le piégeage possible.

---

## 10. Fin de partie

**Le fugitif gagne** s'il se trouve dans sa zone scellée à la fin de son tour et
qu'il y est toujours à la fin du tour suivant.

Une zone occupée par un inspecteur est **neutralisée** tant qu'il y reste : le
compte d'extraction ne démarre pas et s'interrompt s'il était en cours. Camper est
une stratégie valide — mais un inspecteur assis sur une zone est un inspecteur qui
ne cherche pas.

**Les inspecteurs gagnent** dans trois cas :

- résistance du fugitif tombée à 0 ;
- fugitif sans déplacement légal en début de sa phase ;
- tour 40 atteint sans extraction.

**Étranglement.** À partir du tour 30, une zone se ferme tous les 2 tours. L'ordre
est déterminé par la graine et **annoncé 2 tours à l'avance** aux deux joueurs. Si
la zone scellée se ferme, le fugitif doit payer 2 points pour en choisir une autre.

---

## 11. Paramètres

Tous exposés dans l'interface, groupés en préréglages. Ce sont les leviers
d'équilibrage.

| Paramètre | v1 | Note |
|---|---|---|
| Taille du plateau | 41×41 | Préréglages Quartier 21, Faubourg 31, Ville 41 |
| Portée de vue | 8 | Un cinquième du côté, jamais moins de 3 |
| Durée | 40 tours | Environ le côté du plateau |
| Résistance | 10 | |
| Inspecteurs | 5 | |
| Pions déplaçables par tour | 3 sur 5 | **Premier levier à ajuster** |
| Période de révélation | 4 tours | |
| Zones d'extraction | 6 | Doit rester supérieur au nombre d'inspecteurs |
| Durée d'une trace | 6 tours | |
| Coût d'un meurtre | 3 points | |
| Meurtres par partie | 2 | |
| Début de l'étranglement | Tour 30 | |

Suspicion à vérifier en simulation : trois déplacements contre un reste peut-être
trop favorable aux inspecteurs. C'est là qu'il faudra corriger en premier.

---

## 12. Le meurtre

Le fugitif peut tuer. C'est sa carte la plus forte, et elle se paie en
information.

**Coût : 3 points de résistance. Deux fois par partie au maximum.**

Au moment du meurtre, sa position exacte est révélée aux inspecteurs, et la
scène reste marquée sur le plateau. Il ne gagne rien d'autre — pas de
déplacement, pas d'avantage direct.

Ce qu'il achète, c'est **du mouvement adverse**. Cinq inspecteurs qui convergent
sur une scène de crime sont cinq inspecteurs qui ne gardent plus les zones
d'extraction. Un meurtre commis loin de sa zone scellée, suivi d'une course dans
l'autre sens, est le seul moyen dont il dispose pour déplacer le dispositif
entier.

Rien n'oblige les inspecteurs à s'y rendre. C'est justement l'intérêt : ils
savent où il *était*, ils doivent parier sur ce que ça dit de sa destination. Un
joueur qui ignore la scène refuse le leurre, et prend le risque que ce n'en soit
pas un.

### Pourquoi ce coût

Trois points, c'est plus cher que le silence. Un fugitif qui tue deux fois a
dépensé six de ses dix points et ne peut plus se permettre d'être approché. La
mécanique ne doit pas devenir un réflexe d'ouverture.

Le plafond de deux existe pour la même raison : trois leurres suffiraient à
promener les inspecteurs d'un bout à l'autre du plateau sans qu'ils puissent
jamais se replacer.

### Ce qui n'est pas retenu

**Le meurtre n'est pas involontaire.** Une pulsion qui monte tour après tour et
finit par se déclencher toute seule est thématiquement juste, mais le fugitif la
subit au lieu de la décider — et il n'en tire aucun avantage, seulement la
convergence adverse. La version volontaire garde la tension et ajoute
l'arbitrage.

Un greffon de règles peut rétablir la pulsion : compteur croissant, meurtre
imposé au-delà d'un seuil. Le coût et le plafond sont déclaratifs, ils se
changent sans toucher au code.

---

## 13. Un inspecteur bloque la vue

Une ligne de vue s'arrête au premier bâtiment **ou au premier autre inspecteur**.

La règle punit l'alignement des pions et force la dispersion, ce qui va dans le
sens de tout le reste : cinq inspecteurs en file indienne ne voient qu'avec le
premier d'entre eux.

Conséquence technique à connaître : la table de vision précalculée ne dépend que
du terrain. L'occlusion par un pion s'applique **à la lecture**, en tronquant la
ligne à la première case occupée. Précalculer les deux ensemble supposerait de
recalculer à chaque déplacement.

Les barrages du Barreur bloquent la vue de la même façon, sans quoi la capacité
ne serait qu'un mur de déplacement sans effet sur l'information.

---

## 14. Ce qui reste ouvert

- Le rapport de trois déplacements contre un est à mesurer avant d'être
  défendu. C'est le premier paramètre à corriger si l'équilibrage penche.
- La distance minimale entre le noyau de départ et les zones d'extraction n'est
  pas fixée ; elle dépend du taux de rues obtenu par la génération.
- Le coût du silence, trois points, est le chiffre le plus arbitraire du
  document.
- Les capacités des inspecteurs n'ont jamais été essayées ensemble. Le Coureur
  et le Barreur sont les deux suspects.
- Le meurtre à 3 points, deux fois par partie, n'a jamais été mesuré. S'il n'est
  jamais joué, il est trop cher ; s'il est joué systématiquement au premier
  tour, il l'est trop peu.
- Le taux de bâtiments visé, 35 à 50 %, vient d'une estimation. Un prototype
  antérieur tournait à 28 % et produisait des plateaux jouables : la fourchette
  est peut-être à revoir vers le bas.

L'architecture, les structures et la feuille de route ont leur propre document :
[`architecture.md`](architecture.md).
