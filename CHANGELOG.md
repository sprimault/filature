# Journal des versions

Le format suit [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/).

SemVer avec la clause du zéro : **en `0.x`, rien n'est imposé**. Le mineur
marque un jalon de la feuille de route, pas une rupture d'API — tout le reste
s'accumule en correctif, correctifs, fonctionnalités et ruptures confondus.

Quatre numéros à ne pas confondre :

| Numéro | Où | Ce qu'il suit |
|---|---|---|
| version du dépôt | tag git | le binaire |
| `shapes_version` | chaque fichier de formes | le contrat d'apparence |
| `protocol` | échanges avec un bot | le contrat de bot |
| `effects_version` | manifeste d'un plugin de règles | le vocabulaire d'effets |

Les trois derniers sont des entiers sans rapport avec SemVer. Une version peut
sortir sans qu'ils bougent ; ils ne bougent jamais sans version.

Un titre de section s'écrit `## [version] — date — titre`. La publication en
tire le nom et les notes de la version : ce qui est relu ici est ce qui sera
lu sur la page des versions, et il n'y a rien à recopier ensuite. Le titre est
facultatif ; sans lui, la version se nomme par son tag.

Chaque section est **bilingue, français d'abord, séparé par `***`** — les notes
de version sont ce que lit un auteur de plugin étranger avant de savoir s'il
doit reprendre son travail. Ce préambule reste en français : il n'est jamais
publié, et explique les conventions du dépôt à qui y contribue.

## [Non publié]

**Le protocole de bot finit de passer à l'anglais.** Quatre descriptions du
schéma publié citaient encore des noms retirés — dont deux fois « bonjour » pour
un message que le même fichier nomme `hello` —, et l'exemple d'un coup montrait
`pion`, `Colonne` et `Ligne` là où le jeu envoie `piece`, `Column` et `Row`. Un
bot écrit sur cet exemple produisait un coup qui n'égalait aucune entrée de
`legal_moves`, alors que la ligne suivante exige qu'il y figure tel quel.
L'exemple de `hello` annonçait treize réglages pour dix-sept, les quatre absents
étant ceux du ressourcement et de l'étranglement.

Les exemples du document sont désormais rapprochés des structures que le jeu
sérialise, comme les énumérations du schéma l'étaient déjà.

**L'aperçu montre ce que le contrat décrit.** Il appliquait le grain du sol en
pourcents, ce que `docs/contrat-formes.md` §8 écarte nommément : sur la palette
livrée, ça suffisait à faire passer la rue sous un lieu actif et un lieu actif
sous une zone ouverte, soit les deux inversions que le refus au chargement
existe pour empêcher. Une forme d'opacité nulle s'y affichait pleine, et le
liseré s'épaississait de l'épaisseur d'un contour absent — jusqu'à six unités
au lieu de deux, son épaisseur passant sous le contrôle du plugin.

La cause commune est nommée : la constante du grain se documentait en pourcents
et se lisait en niveaux de luminance selon l'endroit. Le décalage vit désormais
dans un point unique du moteur, que le rendu et l'aperçu partagent.

**Le contrat de formes, son schéma et le chargeur disent enfin la même chose.**
Ils divergeaient à trois voix sur les variantes d'état : le schéma déclarait
`highlighted` et `out_of_sight`, le document les montrait, et le jeu décodait
une table sous une clé `variant` que ni l'un ni l'autre n'accepte. Le jeu suit
désormais le contrat publié, et une variante sous l'ancienne clé est refusée au
lieu d'être perdue en silence. Le message de refus nomme la clé telle qu'elle
s'écrit — il désignait jusqu'ici un chemin qu'aucun fichier chargeable ne
pouvait contenir.

**`role` devient obligatoire, y compris en surcharge.** C'est lui qui désigne le
gabarit, donc ce que la validation refuse : le déduire d'un nom ferait dépendre
un contrôle de jeu d'une table de noms que le contrat ne tient pas pour fermée.
L'exemple de surcharge du §4 le déclare désormais — il était refusé au
chargement pour cette raison, et pour une seconde : un de ses cercles débordait
sous le sol.

**Et `outline_thickness = 0` est refusé** au lieu d'être traité comme le défaut.
Le schéma le refusait déjà et le message d'erreur annonçait « de 1 à 4 » ; seul
le chargeur l'acceptait, faute de distinguer une valeur absente d'un zéro écrit.

**Trois déclenchements que rien ne déclenchait quittent le vocabulaire.** Fin de
tour, contact et révélation étaient ouverts aux capacités comme aux modes par le
schéma et décrits un par un dans la documentation, sans qu'aucune ligne du noyau
ne les produise : un plugin qui s'y accrochait restait inerte sans un message.
Les rétablir demandera de trancher ce qu'ils portent — lequel des pions au
contact, une révélation avant ou après la dépense du silence —, et la première
mécanique périodique retenue paiera ce coût en le sachant. Restent les deux
phases pour ce qu'un joueur déclenche, et l'étranglement pour ce que le jeu
déclenche lui-même, réservé aux modes.

**Et la phase d'une dépense compte enfin.** Les cinq dépenses livrées la
déclaraient, rien ne la lisait : une dépense annoncée sur la phase des
inspecteurs se serait proposée au fugitif.

**Le chargeur applique quatre familles de refus que le contrat publiait sans
lui.** Une durée ou un rayon négatifs, une annonce posée sur autre chose qu'un
effet différé, une échéance nulle, un mode sans nom ou sans effet, un
déclenchement inventé sur une capacité passive : tous entraient. Le schéma les
refusait déjà, ce qui est le pire des deux mondes — l'auteur qui valide son JSON
voyait un refus que le jeu ne faisait pas.

**Et deux traducteurs de la même langue se heurtent au lieu de s'écraser.** Le
code de langue était décodé puis abandonné : il n'entrait pas au registre, donc
rien ne le comparait, et celui qui chargeait en second gagnait dans l'ordre
alphabétique des dossiers.

**Rouvrir la dernière zone fermée laissait un état qui se relit autrement.** Le
retrait au milieu d'une liste ne rendait pas la tranche à nul quand elle se
vidait, alors que la troncature le faisait : `[]` au journal là où `null` était
attendu, sur un champ qui décide du rejeu octet pour octet. Le geste est
désormais écrit une seule fois.

**Un effet différé annoncé par le fugitif ne livre plus sa position.** Son
contexte porte sa case exacte, et la vue le servait tel quel aux inspecteurs :
un plugin de règles qui aurait annoncé une dépense aurait donné la position
dans le champ le plus discret du programme. Le camp caché annonce désormais
qu'un effet vient et quand, jamais où — les inspecteurs, dont les positions
sont publiques, gardent le leur.

**Le silence se voit, se compte et se perd.** Trois défauts du même canal
d'information. Les inspecteurs n'apprenaient jamais qu'un silence avait été
payé, le drapeau retombant avant qu'ils reprennent la main : la vue porte
désormais le tour de l'achat. Le compte à rebours annonçait une période entière
au tour même de la révélation, c'est-à-dire à l'instant où le fugitif décide de
payer. Et un silence sans emploi se reportait sans borne, ce qui en faisait une
assurance qu'il valait toujours mieux prendre tôt — il couvre le tour de son
achat, et se perd s'il n'a rien couvert.

**Naître sur un lieu de ressourcement n'est pas y entrer.** Le fugitif dont le
tirage de départ tombait sur un lieu y gagnait deux points à la première
résolution sans avoir bougé, et consommait le lieu pour huit tours. La règle
attache le gain à l'entrée : il n'a rien payé pour être là, donc le lieu reste
actif jusqu'à ce qu'il en reparte et y revienne.

**Le Barreur ferme la case que le joueur choisit.** Sa capacité n'en portait
aucune, et le barrage tombait invariablement dans le coin du plateau, sur la
zone d'extraction qui s'y trouve. Il pose désormais son obstacle sur l'une des
huit cases voisines de son pion : barrer à distance retirerait une case d'entrée
d'une zone sans dégarnir la couverture, que la règle fait payer par un pion
posté dessus.

**Les sols et les pions se distinguent par leur luminance, et le chargeur le
vérifie.** Trois couleurs se tenaient en trois niveaux de gris — une zone
ouverte, un lieu actif et le fugitif — et deux autres en neuf. Les dix sont
reposées d'un coup, étagées de dix-sept niveaux au moins, trente autour du
fugitif : un sol confondu avec un autre se rattrape sur la carte à plat, un
camp confondu avec l'autre fausse la lecture du plateau.

Le grain du sol déplaçant chaque case de cinq niveaux, deux sols voisins qui
s'en approchent échangeraient leur rang à l'affichage. Toute palette qui les
resserre sous dix est désormais refusée au chargement, la paire fautive nommée.

**Le fugitif voit ce que les inspecteurs surveillent.** Les cases tenues par une
ligne de vue ne partaient qu'à un camp, alors que les positions des pions et le
terrain sont publics : ce calcul se déduit, mais le dérouler de tête à chaque
coup adverse ne handicape que celui des deux joueurs qui fatigue.

**Le Guetteur double sa portée au lieu de la tripler.** Il déclarait huit cases
en absolu, juste tant qu'un seul préréglage existait : sur un Quartier, la portée
vaut quatre et la capacité la portait à douze. Les effets acceptent désormais un
`mode`, absent pour additionner comme avant, `multiply` pour multiplier — une
valeur absolue ne peut plus dire une intention relative depuis que la portée
dérive de la taille du plateau.

**Deux refus de plus au chargement, et une promesse retirée.** Une cible qui ne
désigne personne — `other_piece` ou `all_pieces` dans une dépense du fugitif,
qui est seul — et un code de langue hors de l'étiquette BCP 47 étaient tous deux
acceptés. En revanche, la promesse qu'une cible « incompatible avec le camp
déclarant » serait refusée disparaît : viser le camp adverse est le cas
ordinaire, et distinguer ce qui avantage le fugitif de ce qui le gêne
demanderait au chargeur de juger l'intention de chaque effet. Un plugin qui
déclare côté inspecteurs un effet lui rendant de la résistance se charge : il
est mal écrit, pas invalide.

**Le Chef ordonne un coup de filet.** Sa capacité partageait la vue d'un
coéquipier, ce qui ne voulait rien dire : les cinq inspecteurs sont un joueur
unique, et la vue du camp unit déjà ce que chacun voit. Elle rend désormais la
position du fugitif publique deux tours plus tard, annoncée aux deux camps —
seule capacité d'information à côté de quatre capacités de terrain, elle donne
aux inspecteurs le moyen d'en acheter que le fugitif avait déjà trois fois pour
la brouiller.

**Un silence couvre le tour, pas une révélation.** Un coup de filet peut tomber
le même tour que la révélation périodique, et le fugitif paie avant que les
inspecteurs jouent : il ne peut pas anticiper la coïncidence, donc il n'a pas à
la payer deux fois. Il s'achète contre être trouvé, jamais contre se montrer —
une révélation qu'il s'inflige lui-même reste en vigueur.

**Le meurtre disparaît, et la scène de crime avec lui.** Le leurre fait le même
travail en mentant au lieu de se dénoncer, ce qui laisse au fugitif quelque
chose à faire de son information au lieu de la sacrifier. Le jeu quitte le
registre du polar noir, et c'est le prix. Un plugin de règles peut le rétablir —
le coût et le plafond sont déclaratifs — mais il devra livrer la forme et la
couleur de sa scène. `shapes_version` passe à 4 pour cette raison, et
`mark_crime_scene` quitte le vocabulaire.

**Défaire une dépense rendait un état équivalent, pas identique.** Le compteur
d'emplois gardait sa clé à zéro, et la table restait allouée là où elle était
nulle : sans effet sur les règles, puisqu'une clé absente se lit zéro, mais
l'IA compare des états pour reconnaître une position déjà explorée et en aurait
vu deux. Le geste qui manquait était écrit deux fois ailleurs dans le noyau ; il
est désormais écrit une seule.

**Le fugitif peut mentir avec ses traces.** Un leurre coûte un point, trois fois
par partie, et substitue une fausse trace à toutes celles que son déplacement
aurait laissées ce tour-là — il ne s'y ajoute pas, faute de quoi un inspecteur
qui en découvrirait deux incompatibles saurait qu'il y a eu leurre. Effacer
laisse un trou, et un trou est déjà une information ; une fausse trace attaque
la déduction elle-même.

Le joueur choisit la case et la direction parmi ce que le terrain autorise : un
leurre a la forme d'un pas qui n'a pas eu lieu, donc il ne peut produire qu'une
trace qu'un vrai déplacement aurait pu laisser. `effects_version` passe à 3.

**Le chargeur applique trois refus qu'il annonçait sans les faire.** Un nom de
plugin hors du motif publié, une licence absente de la liste fermée, un
déclenchement inventé ou `strangling` sur une capacité étaient tous acceptés.
La licence est le cas le plus net : la liste blanche existe pour écarter les
mentions qu'on ne peut pas juger, et « à voir » entrait sans un mot.

**Les schémas de `schemas/` sont désormais exécutés.** Un test valide le
manifeste livré contre le contrat publié, et une série de manifestes ne portant
chacun qu'une faute, pour que chaque refus soit exercé séparément. Tant que
personne ne les exécutait, ils pouvaient mentir comme un commentaire.

**Les préréglages s'appellent `district`, `outskirts` et `city`.** Leurs clés
étaient en français quand les dictionnaires livrés déclaraient déjà les libellés
sous les noms anglais : aucun préréglage n'aurait trouvé son libellé, et le
repli sur le français n'aurait pas joué non plus, la clé manquant des deux
côtés. `filature -preset quartier` devient `filature -preset district`.

**Le protocole de bot passe à la version 3, tout en anglais.** Ses six types de
message étaient en français sauf un, que le document et le schéma nommaient
d'ailleurs différemment : un auteur de bot ne pouvait deviner ni la règle ni
l'exception. Ils sont désormais `hello`, `ready`, `play`, `move`, `over` et
`error`, comme tous les identifiants publics du projet. Un bot qui annonce une
autre version est écarté dès sa réponse à `hello`, sur un message qui porte les
deux numéros.

**Les trois numéros de contrat concordent à nouveau avec ce que le code
applique.** Le schéma de manifeste figeait `effects_version` à 1 quand le noyau
appliquait la 2, et celui du protocole restait à 1 dans ses deux messages. Le
même schéma annonçait quatre motifs de fin francisés là où le jeu en produit
cinq en anglais, et sa clause conditionnelle sur les effets différés testait un
nom de primitive qui n'existe pas — elle refusait donc la seule forme valide de
celle qu'elle prétendait contraindre.

**Les cinq sols entrent tous dans le contrat d'apparence.** Les deux couleurs
des lieux de ressourcement étaient livrées dans la palette sans être exigées
d'un plugin : une palette tierce qui les omettait était acceptée, et le rendu
cherchait un nom absent sur toutes les cases d'abri. La planche de `filature
preview` les montre désormais — elle n'en affichait que trois, et laissait donc
de côté les deux fonds que la palette livrée signale elle-même comme trop
proches l'un de l'autre.

**Les zones se ferment aux tours annoncés.** Le préavis de deux tours était
déclaré par le mode d'étranglement du contenu livré, et s'ajoutait à une cadence
que le noyau calculait déjà en tours de fermeture : tout glissait de deux tours,
et la dernière zone tombait au dernier tour de la partie au lieu de laisser de
quoi en profiter. Le préavis devient un paramètre de partie, refusé s'il dépasse
la période — voir une fermeture venir est une règle, et une règle ne se négocie
pas par manifeste.

**Le taux de rues ne juge plus que la trame.** Les blocs percés par-dessus —
zones d'extraction et lieux de ressourcement — n'entrent plus dans la mesure :
ce ne sont pas du terrain mais des dispositifs, publics et désignés, et les
agréger faisait dépendre le plafond d'une composition qui varie avec la taille
du plateau. Le Quartier passait ainsi la borne haute par ses seules zones, si
bien que la génération y jetait quatre-vingt-douze tirages sur cent et ne
retenait que les trames les plus denses. Il en sort plus ouvert qu'il ne
l'était, un tirage suffit là où il en fallait quatorze, et quelques-uns de ses
plateaux portent moins de cinq impasses.

**Cinq règles ne s'appliquaient pas.** L'étranglement ne fermait aucune zone, un
barrage et un percement duraient jusqu'à la fin de la partie, les capacités
passives n'étaient jamais posées, le déplacement rendu par un repérage
n'existait pas, et un fugitif immobile sur un lieu de ressourcement s'y
rechargeait chaque fois que celui-ci redevenait actif.

**La résistance se récupère, sur quatre lieux du plateau.** Un lieu de
ressourcement est un bloc au même format qu'une zone d'extraction, posé dans la
couronne entre le noyau de départ et les sorties. Le fugitif y regagne deux
points en fin de tour, et le lieu entier passe en recharge pour huit tours — les
deux camps voyant lequel a servi et quand il revient. Il paie donc ses points en
information, comme pour ses autres dépenses.

**La trame de rues ne se resserre plus sur le dernier îlot.** Le quatrième bord
était ajouté comme avenue sans regarder la distance au dernier axe tiré.

**La capture entre dans le jeu.** Un inspecteur qui reste au contact du fugitif
deux résolutions de suite le rattrape. En terrain ouvert celui-ci rompt le
contact d'une diagonale ; dans un couloir d'une case il ne le peut pas, et le
double déplacement redevient une dépense de survie. Un inspecteur qui occupe sa
case compte désormais comme au contact — l'en empêcher lui aurait appris où il
se trouve.

**Une zone ne se neutralise plus, ses cases s'occupent.** Un pion posté dessus
n'en retire qu'une case d'entrée : fermer une zone demande autant de pions
qu'elle a de cases de rue. L'étranglement s'arrête en laissant trois zones
ouvertes, et sa cadence suit la durée de la partie au lieu d'être figée.

**Quand deux conditions de fin tombent ensemble, celle du fugitif l'emporte.**
Les inspecteurs en ont quatre, il n'en a qu'une.

**Le noyau de départ suit la taille du plateau, et les inspecteurs ne s'y
placent plus.** Son rayon valait cinq quelle que soit la ville, pendant que la
portée de vue, elle, suivait le côté : cinq pions posés autour du centre y
voyaient le fugitif dès le premier tour dans la quasi-totalité des parties, et
d'autant mieux que le plateau était grand. `Settings` porte donc un
`centre_radius`, que les bots reçoivent dans `hello`.

**Les plugins d'apparence se chargent et se valident.** Un plugin déclare ce
qu'il remplace, le reste retombe sur le contenu livré. Une forme qui déborde de
son gabarit, un polygone hors de ses trois à trente-deux sommets, une couleur
absente de la palette, une valeur hexadécimale écrite dans une forme, un prisme
hors du rôle bâtiment : chacun arrête le chargement en nommant le chemin
complet de la clé fautive, et tous sont listés d'un coup.

Le contrôle vaut **au démarrage**, plugin local compris, et pas seulement pour
`filature validate` — une forme qui déborde masque les cases voisines, ce qui
est un avantage de jeu déguisé en habillage.

Deux plugins qui redéfinissent la même forme sont un conflit, nommé des deux
côtés. Surcharger le contenu livré n'en est pas un.

**`filature preview <dossier>` rend un plugin d'apparence en SVG**, sans lancer
le jeu : la planche de ses formes sur les trois sols possibles, et un plateau en
situation. Le plugin est fusionné sur le contenu livré avant d'être rendu, et la
planche marque ce qui vient de lui — une clé mal orthographiée passe la
validation sans rien surcharger, et rien d'autre ne le dirait.

**L'archive `js/wasm` n'est plus publiée.** Elle ne contenait que le `.wasm`,
inutilisable sans `wasm_exec.js` ni page HTML. La cible continue d'être
compilée : c'est le seul contrôle qui empêche une dépendance d'introduire du
cgo sans que rien ne le signale.

**Les archives portent `THIRD-PARTY-NOTICES`**, à côté de `LICENSE` et
`NOTICE`. Les licences des bibliothèques liées au binaire l'exigent à la
redistribution, et les versions précédentes ne le faisaient pas. Le fichier est
généré depuis les dépendances réelles ; un test échoue quand il vieillit.

***

**The bot protocol finishes its move to English.** Four descriptions in the
published schema still cited retired names — twice "bonjour" for a message the
same file calls `hello` — and the move example showed `pion`, `Colonne` and
`Ligne` where the game sends `piece`, `Column` and `Row`. A bot written from
that example produced a move matching no entry in `legal_moves`, though the very
next line requires it to appear there as is. The `hello` example announced
thirteen settings out of seventeen, the four missing ones belonging to respites
and strangling.

The document's examples are now checked against the structures the game
serialises, as the schema's enumerations already were.

**The preview shows what the contract describes.** It applied the ground grain
as a percentage, which `docs/contrat-formes.md` §8 rules out by name: on the
shipped palette that was enough to push the street below an active respite and
an active respite below an open zone — the very two rank swaps the load-time
refusal exists to prevent. A stroke with zero opacity drew solid, and the rim
thickened by the width of an absent outline, up to six units instead of two,
putting its width under the plugin's control.

The common cause is named: the grain constant documented itself in percent and
was read in luminance levels depending on the caller. The shift now lives in one
place in the engine, shared by the renderer and the preview.

**The shape contract, its schema and the loader now say the same thing.** They
diverged three ways on state variants: the schema declared `highlighted` and
`out_of_sight`, the document showed them, and the game decoded a table under a
`variant` key neither accepts. The game now follows the published contract, and
a variant under the old key is refused instead of silently lost. The rejection
message names the key as it is written — until now it pointed at a path no
loadable file could contain.

**`role` becomes mandatory, overrides included.** It designates the template,
hence what validation refuses: deriving it from a name would make a gameplay
check depend on a name table the contract does not hold closed. The §4 override
example now declares it — it was refused at load time for that reason, and for a
second one: one of its circles dipped below the ground.

**And `outline_thickness = 0` is refused** instead of being treated as the
default. The schema already refused it and the error message announced "1 to 4";
only the loader accepted it, unable to tell an absent value from a written zero.

**Three triggers that nothing triggered leave the vocabulary.** Turn end,
contact and reveal were open to both abilities and modes in the schema and
described one by one in the documentation, without a single line of the core
producing them: a plugin hooking onto them stayed inert without a message.
Restoring them will mean deciding what they carry — which of the pieces in
contact, a reveal before or after the silence is spent — and the first periodic
mechanic kept will pay that cost knowingly. What remains: the two phases for
what a player triggers, and strangling for what the game triggers itself,
reserved to modes.

**And an expense's phase finally counts.** All five shipped expenses declared
it, nothing read it: an expense announced on the inspectors' phase would have
been offered to the fugitive.

**The loader applies four families of rejection the contract published without
it.** A negative duration or radius, an announcement on something other than a
deferred effect, a zero deadline, a mode with no name or no effect, an invented
trigger on a passive ability: all went through. The schema already refused them,
which is the worst of both worlds — an author validating their JSON saw a
rejection the game did not make.

**And two translators of the same language now collide instead of overwriting.**
The language code was decoded then dropped: it never reached the registry, so
nothing compared it, and whoever loaded second won by folder order.

**Reopening the last closed zone left a state that reads back differently.**
Removing from the middle of a list did not return the slice to nil when it
emptied, where truncation did: `[]` in the journal where `null` was expected, on
a field that decides byte-for-byte replay. The gesture is now written once.

**A deferred effect announced by the fugitive no longer gives away his
position.** Its context carries his exact cell, and the view served it to the
inspectors as is: a rules plugin announcing a fugitive expense would have handed
over the position through the quietest field in the program. The hidden side now
announces that an effect is coming and when, never where — inspectors, whose
positions are public, keep theirs.

**Silence is seen, counted and lost.** Three faults on the same information
channel. Inspectors never learnt that a silence had been paid, the flag dropping
before they took their turn: the view now carries the turn it was bought on. The
countdown announced a full period on the very turn of the reveal, that is, at
the moment the fugitive decides whether to pay. And an unused silence carried
over indefinitely, making it insurance always worth buying early — it now covers
the turn it is bought on, and is lost if it covered nothing.

**Being born on a respite is not entering it.** A fugitive whose starting draw
landed on a respite gained two points at the first resolution without having
moved, and spent the place for eight turns. The rule ties the gain to entering:
he paid nothing to be there, so the respite stays active until he leaves and
comes back.

**The Blocker closes the cell the player picks.** Its ability carried none, so
the roadblock always landed in the board's corner, on the extraction zone that
sits there. It now drops the obstacle on one of the eight cells around its
piece: blocking at range would take an entry cell away from a zone without
thinning the cover, which the rules make you pay for with a piece posted on it.

**Grounds and pieces are told apart by luminance, and the loader checks it.**
Three colours sat within three grey levels of each other — an open zone, an
active respite and the fugitive — and two more within nine. All ten are reset at
once, spaced at least seventeen levels apart and thirty around the fugitive: one
ground mistaken for another is caught on the flat map, one side mistaken for the
other misreads the whole board.

Since the ground grain shifts each cell by five levels, two neighbouring grounds
closer than that would swap ranks on screen. Any palette packing them under ten
is now refused at load time, with the offending pair named.

**The fugitive now sees what the inspectors watch.** Cells held by a line of
sight went to one side only, though piece positions and terrain are public: the
calculation follows from them, but running it in your head on every opposing
move only handicaps whichever of the two players gets tired.

**The Lookout doubles his range instead of tripling it.** He declared eight cells
outright, correct while a single preset existed: on a Small board range is four
and the ability took it to twelve. Effects now accept a `mode` — absent to add as
before, `multiply` to multiply — since an absolute value can no longer express a
relative intent once range derives from board size.

**Two more rejections at load time, and one promise withdrawn.** A target that
designates nobody — `other_piece` or `all_pieces` in a fugitive expense, he being
alone — and a language code outside the BCP 47 tag were both accepted. The
promise that a target "incompatible with the declaring side" would be refused, on
the other hand, is gone: targeting the opposing side is the ordinary case, and
telling what helps the fugitive from what hinders him would require the loader to
judge each effect's intent. A plugin declaring an inspector-side effect that
restores his stamina loads: it is badly written, not invalid.

**The Chief calls a dragnet.** His ability shared a teammate's view, which meant
nothing: the five inspectors are a single player, and the side's view already
merges what each of them sees. It now makes the fugitive's position public two
turns later, announced to both sides — the only information ability next to four
terrain ones, giving the inspectors a way to buy what the fugitive already had
three ways to blur.

**Silence covers the turn, not a reveal.** A dragnet can land on the same turn as
the periodic reveal, and the fugitive pays before the inspectors play: he cannot
anticipate the coincidence, so he does not pay for it twice. It is bought against
being found, never against showing oneself — a reveal he inflicts on himself
still stands.

**Murder is gone, and the crime scene with it.** The decoy does the same work by
lying instead of giving itself away, which leaves the fugitive something to do
with his information rather than sacrificing it. The game leaves the noir
register, and that is the price. A rules plugin can bring it back — cost and cap
are declarative — but it will have to ship the shape and the colour of its
scene. `shapes_version` moves to 4 for that reason, and `mark_crime_scene`
leaves the vocabulary.

**Undoing an expense returned an equivalent state, not an identical one.** The
use counter kept its key at zero, and the table stayed allocated where it had
been nil: no effect on the rules, since a missing key reads as zero, but the AI
compares states to recognise an already-explored position and would have seen
two. The missing gesture was written twice elsewhere in the core; it is now
written once.

**The fugitive can now lie with his trails.** A decoy costs one point, three
times a game, and substitutes a false trail for every one his movement would
have left that turn — it does not add to them, otherwise an inspector finding
two incompatible ones would know a decoy had been played. Erasing leaves a gap,
and a gap is already information; a false trail attacks the deduction itself.

The player picks the cell and the direction from what the terrain allows: a
decoy has the shape of a step that never happened, so it can only produce a
trail a real move could have left. `effects_version` moves to 3.

**The loader now applies three rejections it announced without performing.** A
plugin name outside the published pattern, a licence missing from the closed
list, an invented trigger or `strangling` on an ability were all accepted. The
licence is the clearest case: the allow-list exists to keep out entries nobody
can judge, and "à voir" walked straight in.

**The schemas in `schemas/` are now executed.** A test validates the shipped
manifest against the published contract, along with a series of manifests each
carrying a single fault, so every rejection is exercised on its own. As long as
nothing ran them, they could lie like a comment.

**The presets are now called `district`, `outskirts` and `city`.** Their keys
were in French while the shipped dictionaries already declared the labels under
the English names: no preset would have found its label, and the fallback to
French would not have helped either, the key being missing on both sides.
`filature -preset quartier` becomes `filature -preset district`.

**The bot protocol moves to version 3, entirely in English.** Its six message
types were in French except one, which the document and the schema happened to
name differently: a bot author could guess neither the rule nor the exception.
They are now `hello`, `ready`, `play`, `move`, `over` and `error`, like every
other public identifier in the project. A bot announcing any other version is
turned away on its answer to `hello`, with both numbers in the message.

**The three contract numbers agree again with what the code applies.** The
manifest schema pinned `effects_version` to 1 while the core applied 2, and the
protocol schema stayed at 1 in both its messages. That same schema announced
four French end reasons where the game produces five English ones, and its
conditional clause on deferred effects tested a primitive name that does not
exist — so it rejected the only valid form of the very primitive it claimed to
constrain.

**All five grounds are now part of the appearance contract.** The two respite
colours shipped in the palette without being required of a plugin: a
third-party palette that omitted them was accepted, and rendering then looked up
a missing name on every respite cell. The `filature preview` board now shows
them — it displayed only three, and so left out the very two backgrounds the
shipped palette itself flags as too close to each other.

**Zones now close on the turns announced.** The two-turn notice was declared by
the shipped strangling mode, and added itself to a cadence the core already
computed as closing turns: everything slid by two, and the last zone fell on the
final turn of the game instead of leaving room to use it. The notice becomes a
game setting, refused if it exceeds the period — seeing a closure coming is a
rule, and a rule is not negotiable by manifest.

**The street ratio now judges the grid alone.** The blocks carved on top —
extraction zones and respites — no longer count towards it: they are not
terrain but fixtures, public and marked, and lumping them together made the
ceiling depend on a composition that varies with board size. The Small preset
was therefore over the upper bound on its zones alone, so generation threw away
ninety-two draws out of every hundred and kept only the densest grids. It comes
out more open than it was, one draw now suffices where fourteen were needed,
and a few of its boards carry fewer than five dead ends.

**Five rules did not apply.** Strangling closed no zone, a roadblock and an
opening lasted until the end of the game, passive abilities were never applied,
the step granted by a sighting did not exist, and a fugitive standing still on a
respite recovered there again every time it came back.

**Stamina can be recovered, at four places on the board.** A respite is a block
in the same format as an extraction zone, set in the ring between the starting
core and the exits. The fugitive regains two points there at the end of a turn,
and the whole place goes into an eight-turn cooldown — both sides seeing which
one was used and when it returns. He therefore pays for those points in
information, as he does for every other expense.

**The street grid no longer tightens on the last block.** The fourth edge was
added as an avenue without checking its distance to the last drawn axis.

**Capture enters the game.** An inspector who stays in contact with the fugitive
across two resolutions catches him. In open ground the fugitive breaks contact
with a diagonal step; in a one-cell corridor he cannot, and the double step
becomes a survival cost again. An inspector standing on his cell now counts as
contact — refusing him that move would have told him where the fugitive is.

**A zone is no longer neutralised, its cells are occupied.** A piece posted on
one only removes an entry cell: closing a zone takes as many pieces as it has
street cells. Strangling stops with three zones left open, and its cadence
follows the length of the game instead of being fixed.

**When two end conditions fall together, the fugitive's wins.** Inspectors have
four of them, he has one.

**The starting core follows the board size, and inspectors can no longer be
placed inside it.** Its radius was five whatever the city, while sight range
followed the board's side: five pieces set around the centre spotted the
fugitive on the very first turn in nearly every game, and all the more easily
on a large board. `Settings` therefore carries a `centre_radius`, which bots
receive in `hello`.

**Appearance plugins now load and validate.** A plugin declares what it
replaces; everything else falls back on shipped content. A shape overflowing its
template, a polygon outside its three-to-thirty-two vertices, a colour missing
from the palette, a hexadecimal value written inside a shape, a prism outside
the building role: each stops the load naming the full path of the offending
key, and all are listed at once.

The check applies **at startup**, local plugins included, not only to
`filature validate` — a shape that overflows hides neighbouring tiles, which is
a gameplay advantage dressed up as decoration.

Two plugins redefining the same shape are a conflict, named on both sides.
Overriding shipped content is not.

**`filature preview <folder>` renders an appearance plugin as SVG**, without
launching the game: the sheet of its shapes on all three possible grounds, and a
board in situation. The plugin is merged onto shipped content before rendering,
and the sheet marks what comes from it — a misspelled key passes validation
while overriding nothing, and nothing else would say so.

**The `js/wasm` archive is no longer published.** It held only the `.wasm`,
unusable without `wasm_exec.js` and an HTML page. The target is still built:
it is the one check that keeps a dependency from pulling in cgo unnoticed.

**Archives now carry `THIRD-PARTY-NOTICES`**, alongside `LICENSE` and `NOTICE`.
The licences of the libraries linked into the binary require it on
redistribution, and earlier releases did not comply. The file is generated from
the actual dependencies; a test fails when it goes stale.

## [0.5.0] — 2026-08-26 — La boucle de jeu

Étape 5 de la feuille de route. **Le binaire joue** : une partie se déroule
entièrement en texte, du placement à la fin, contre un adversaire qui choisit au
hasard parmi les coups légaux. Jusqu'ici le jeu se vérifiait en test ; il se
vérifie maintenant en jouant, et une règle a déjà bougé pour cette raison.

**`shapes_version` passe à 3 : les plugins d'apparence publiés sont à
reprendre.** Deux couleurs entrent au socle obligatoire — `backdrop`, ce qu'on
voit autour du plateau, et `marker_outline`, le contour des quatre marqueurs.
Une palette qui ne les déclare pas est refusée.

Le moteur pose désormais un liseré clair sous le contour des pions et des
marqueurs, et encadre toute épaisseur de contour — jamais moins d'un pixel
d'écran, jamais plus d'un sixième de la plus petite dimension du trait. Rien à
déclarer, rien à retirer. En dessous de 24 pixels par case, le rendu ne garantit
plus la lisibilité des pions et la vue défile plutôt que de réduire.

Les faces d'un bâtiment sont dérivées de sa couleur par trois coefficients fixes
— 1,50 pour le dessus, 1,14 et 0,72 pour les côtés — que le contrat énonce.

**La validation de ce contrat n'est pas encore écrite** : `Validate` et `Merge`
sont l'étape 6. Un plugin d'apparence peut donc être accepté aujourd'hui et
refusé demain — `docs/contrat-formes.md` fait foi, pas ce que le binaire laisse
passer.

**Le jeu cherche ses extensions dans un dossier `plugins`**, à côté de
l'exécutable, et non plus dans `greffons` — le renommer suffit. Le drapeau
`--plugins` remplace `--greffons`.

Les deux sous-commandes suivent : `filature examples` et `filature validate`,
là où l'on écrivait `exemples` et `valide`.

**Tout passe à l'anglais, contrats publics compris.** Les trois numéros
l'annoncent, et changent de nom au passage : `version_effets` devient
`effects_version`, `version_formes` devient `shapes_version`, `protocole`
devient `protocol`. `effects_version` et `protocol` valent 2, `shapes_version`
vaut 3. Un plugin écrit contre une version antérieure est refusé au chargement
plutôt qu'appliqué de travers.

Ce qu'un auteur de plugin doit reprendre : les clés de son manifeste — `nom`
devient `name`, `regles` devient `rules`, `capacite` devient `ability` — et les
valeurs qu'il y écrit — `"deplacer"` devient `"step"`, `"phase_inspecteurs"`
devient `"inspectors_phase"`. Un bot doit relire `schemas/view.schema.json`,
dont toutes les clés changent. Les quatre schémas sont renommés en conséquence.

**Une graine ne rend plus le même plateau.** Les flux d'aléa portent des noms
anglais, et ces noms entrent dans le tirage : la ville de la graine 7 n'est plus
celle d'hier. Aucune partie enregistrée n'existait, mais c'est la dernière
version où ce changement était sans conséquence.

Le mot « greffon » disparaît du projet au profit de « plugin », partout — code,
documentation et libellés. Il est exclusivement français, absent de l'usage
courant, et il ne se cherche pas.

**Ce qu'un auteur de plugin d'apparence doit reprendre.** Deux couleurs entrent
au socle obligatoire : `backdrop`, ce qu'on voit autour du plateau, et
`marker_outline`, le contour des quatre marqueurs. Une palette qui ne les
déclare pas est refusée.

Le moteur pose désormais un liseré clair sous le contour des pions et des
marqueurs, et encadre toute épaisseur de contour — jamais moins d'un pixel
d'écran, jamais plus d'un sixième de la plus petite dimension du trait. Rien à
déclarer, rien à retirer. En dessous de 24 pixels par case, le rendu ne garantit
plus la lisibilité des pions et la vue défile plutôt que de réduire.

Les faces d'un bâtiment sont dérivées de sa couleur par trois coefficients fixes
— 1,50 pour le dessus, 1,14 et 0,72 pour les côtés — que le contrat énonce.

### Ajouté
- **Le binaire joue.** Lancé sans argument, il donne la main aux inspecteurs
  face à un adversaire, en texte. `--side fugitive` prend l'autre rôle,
  `--side watch` regarde deux machines s'affronter, `--preset` et `--seed`
  choisissent la partie. En spectateur, chaque tour s'affiche ; `--delay`
  espace les tours et redessine à la même place plutôt que de les empiler.
- Un adversaire qui choisit au hasard parmi les coups légaux, en
  attendant l'IA des étapes 9 et 10. Il est déterministe : une graine donnée
  rejoue la même partie.
- Chargement des plugins : capacités, dépenses et modes entrent au registre
  depuis les manifestes. Le contenu livré emprunte le même chemin qu'un plugin
  posé sur le disque, si bien qu'il est exercé à chaque démarrage.
- Les refus de `docs/plugins.md` §9 sont appliqués : un champ inconnu, une
  primitive que ce binaire ne sait pas appliquer, `rules = false` accompagné
  d'une capacité, un `differer` imbriqué, un bot mêlé à des effets. Chacun
  arrête le chargement entier en nommant le fichier et la clé fautive.
- Empreinte de contenu par plugin, qui distingue deux plugins se disant
  identiques sans l'être.
- `filature validate <dossier>`, annoncée à la version 0.2.0 : le même contrôle
  que le chargement, tous les manquements listés d'un coup avec leur fichier et
  leur chemin de clé, et l'empreinte quand le plugin tient.

### Corrigé
- La vue du fugitif annonçait une zone scellée numéro -1 tant qu'il n'avait pas
  choisi. Le champ reste maintenant absent, ce qui est ce qu'il faut lire.
- Le fugitif pouvait dépenser son dernier point de résistance, ce qui le faisait
  perdre sur-le-champ sans que l'effet acheté serve jamais. La règle lui en fait
  garder un.

***

Step 5 of the roadmap. **The binary plays**: a game runs through entirely in
text, from setup to the end, against an opponent picking at random among the
legal moves. Until now the game was checked in tests; it is now checked by
playing, and one rule has already moved because of it.

**`shapes_version` goes to 3: published appearance plugins must be revisited.**
Two colours join the required set — `backdrop`, what surrounds the board, and
`marker_outline`, the outline of all four markers. A palette declaring neither
is refused.

The engine now lays a light rim beneath the outline of pieces and markers, and
bounds every outline width — never below one screen pixel, never above one sixth
of the stroke's smallest dimension. Nothing to declare, nothing to remove. Below
24 pixels per tile, rendering no longer guarantees piece legibility and the view
scrolls rather than shrinking further.

A building's faces are derived from its colour by three fixed coefficients —
1.50 for the top, 1.14 and 0.72 for the sides — which the contract states.

**Validation of that contract is not written yet**: `Validate` and `Merge` are
step 6. An appearance plugin may therefore be accepted today and refused
tomorrow — `docs/contrat-formes.md` is what holds, not what the binary lets
through.

**The game looks for extensions in a `plugins` folder**, next to the executable,
rather than in `greffons` — renaming it is enough. The `--plugins` flag replaces
`--greffons`.

Both subcommands follow: `filature examples` and `filature validate`, where one
used to write `exemples` and `valide`.

**Everything moves to English, public contracts included.** The three numbers
say so, and are renamed on the way: `version_effets` becomes `effects_version`,
`version_formes` becomes `shapes_version`, `protocole` becomes `protocol`.
`effects_version` and `protocol` are now 2, `shapes_version` is 3. A plugin
written against an earlier version is refused at load rather than applied
wrongly.

What a plugin author must revisit: their manifest keys — `nom` becomes `name`,
`regles` becomes `rules`, `capacite` becomes `ability` — and the values written
there — `"deplacer"` becomes `"step"`, `"phase_inspecteurs"` becomes
`"inspectors_phase"`. A bot must re-read `schemas/view.schema.json`, whose every
key changes. All four schemas are renamed accordingly.

**A seed no longer yields the same board.** Random streams carry English names,
and those names feed the draw: seed 7's city is not yesterday's. No saved game
existed, but this is the last release where that change came free.

The word « greffon » is gone from the project in favour of « plugin », everywhere —
code, documentation and labels. It is exclusively French, absent from common
usage, and nobody searches for it.

**What an appearance plugin author must revisit.** Two colours join the required
set: `backdrop`, what surrounds the board, and `marker_outline`, the outline of
all four markers. A palette that declares neither is refused.

The engine now lays a light rim beneath the outline of pieces and markers, and
bounds every outline width — never below one screen pixel, never above one sixth
of the stroke's smallest dimension. Nothing to declare, nothing to remove. Below
24 pixels per tile, rendering no longer guarantees piece legibility and the view
scrolls rather than shrinking further.

A building's faces are derived from its colour by three fixed coefficients —
1.50 for the top, 1.14 and 0.72 for the sides — which the contract states.

### Added
- **The binary plays.** Run with no argument, it hands you the inspectors
  against an opponent, in text. `--side fugitive` takes the other side,
  `--side watch` watches two machines play, `--preset` and `--seed` choose
  the game. When watching, every turn is shown; `--delay` spaces them out and
  redraws in place rather than stacking them.
- An opponent drawing its moves at random among the legal ones, standing in
  until the AI of steps 9 and 10. It is deterministic: a given seed replays the
  same game.
- Plugin loading: abilities, expenses and modes now enter the registry from
  manifests. Shipped content takes the same path as a plugin dropped on disk,
  so that path is exercised on every start.
- The refusals in `docs/plugins.md` §9 are enforced: an unknown field, a
  primitive this binary cannot apply, `rules = false` alongside an ability, a
  nested `differer`, a bot mixed with effects. Each aborts the whole load,
  naming the file and the offending key.
- Per-plugin content fingerprint, telling apart two plugins that claim to be
  identical without being so.
- `filature validate <folder>`, announced back in 0.2.0: the same checks the
  loader runs, every failure listed at once with its file and key path, and the
  fingerprint when the plugin holds.

### Fixed
- The fugitive's view reported a sealed zone numbered -1 until he had chosen
  one. The field is now simply absent, which is what should be read.
- The fugitive could spend his last stamina point, losing on the spot without
  the purchased effect ever serving. The rule makes him keep one.

## [0.4.0] — 2026-08-26 — La vision

Étape 4 de la feuille de route. Le fugitif peut désormais être repéré : jusqu'à
cette version il ne l'était jamais, quoi qu'il fasse, et ni la portée de vue ni
les capacités des inspecteurs ne changeaient quoi que ce soit.

**Le binaire ne joue toujours pas** : la boucle de jeu est l'étape 5.

Rien à reprendre pour un auteur de greffon : `version_formes`, `protocole` et
`version_effets` sont inchangés. Un bot recevra en revanche des vues plus
fournies — le champ des cases couvertes n'était jamais rempli.

### Ajouté
- Vision : un inspecteur voit dans les huit directions jusqu'à sa portée, la
  ligne s'arrêtant au premier bâtiment, au premier barrage ou au premier autre
  inspecteur. Le fugitif est repéré en entrant dans une ligne de vue et cesse de
  l'être en sortant — jusqu'ici il ne l'était jamais, quoi qu'il fasse.
- La vue filtrée des inspecteurs porte les cases qu'ils couvrent réellement,
  occlusion comprise : rien de ce qui se trouve derrière un collègue ou un
  barrage n'y figure, et chacun voit la case où il se tient.

***

Step 4 of the roadmap. The fugitive can now be spotted: until this release he
never was, whatever he did, and neither sight range nor the inspectors'
abilities changed anything at all.

**The binary still does not play**: the game loop is step 5.

Nothing for plugin authors to update: `version_formes`, `protocole` and
`version_effets` are unchanged. A bot will however receive fuller views — the
covered-cells field was never populated.

### Added
- Sight: an inspector sees in all eight directions up to their range, the line
  stopping at the first building, roadblock or fellow inspector. The fugitive is
  spotted on entering a line of sight and stops being so on leaving it — until
  now he never was, whatever he did.
- The inspectors' filtered view carries the cells they actually cover, occlusion
  included: nothing behind a fellow inspector or a roadblock appears in it, and
  each of them sees the cell he stands on.

## [0.3.0] — 2026-08-25 — Le plateau

Étape 3 de la feuille de route. Le plateau se fabrique au lieu d'être fourni :
une graine donne une ville, et la même graine la redonne à l'identique sur
Windows, Linux et macOS.

**Le binaire ne joue toujours pas** : la boucle de jeu est l'étape 5.

Rien à reprendre pour un auteur de greffon : `version_formes`, `protocole` et
`version_effets` sont inchangés.

### Ajouté
- Génération de plateau : trame irrégulière, cours percées, impasses et six
  zones d'extraction réparties sur le pourtour. Une graine donnée produit
  toujours le même plateau, vérifié sur Windows, Linux et macOS.
- Trois préréglages de partie — Quartier, Faubourg et Ville — là où seule la
  Ville existait. La portée de vue et la durée suivent la taille du plateau.

### Corrigé
- Les bornes des paramètres avaient des valeurs sans nom ni justification, et
  les refus ne disaient ni la valeur reçue ni ce qui était attendu. Le nombre de
  pions déplaçables n'était pas contrôlé.
- Le README annonçait un moteur non écrit alors qu'une partie complète se joue
  depuis des appels Go.
- Le creusement des impasses n'en produisait aucune : les couloirs rejoignaient
  l'avenue suivante au lieu de rester borgnes, et le piégeage sur lequel repose
  le Barreur n'existait pas. Leur nombre suit désormais la surface du plateau et
  non son côté, qui rendait une Ville deux fois moins piégeuse qu'un Quartier.

***

Step 3 of the roadmap. The board is now built rather than supplied: a seed gives
a city, and the same seed gives it back identically on Windows, Linux and macOS.

**The binary still does not play**: the game loop is step 5.

Nothing for plugin authors to update: `version_formes`, `protocole` and
`version_effets` are unchanged.

### Added
- Board generation: irregular street grid, punched courtyards, dead ends and six
  extraction zones spread around the perimeter. A given seed always produces the
  same board, verified on Windows, Linux and macOS.
- Three game presets — District, Outskirts and City — where only the City
  existed. Sight range and duration follow the board size.

### Fixed
- Parameter bounds were unnamed magic numbers, and rejections stated neither the
  value received nor what was expected. The number of movable pieces was not
  checked at all.
- The README claimed the engine was not written yet, when a full game already
  plays out from Go calls.
- Dead-end carving produced none: corridors reached the next avenue instead of
  staying blind, and the trapping the Blocker relies on did not exist. Their
  count now follows the board's area rather than its side, which made a City
  half as trap-ridden as a District.

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
