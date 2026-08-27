# Contribuer

English: [CONTRIBUTING.md](CONTRIBUTING.md)

## Comment ce projet est écrit

Le code est écrit en binôme avec un assistant, sous les règles de ce dépôt. Ce
n'est pas une note de bas de page : la méthode est le second objet du projet, et
la façon dont les règles sont posées en découle.

- **Les règles précèdent le code**, elles n'en sont pas déduites.
  [`docs/regles.md`](docs/regles.md) fait foi, et un désaccord entre le document
  et le code est un défaut du code.
- **Une décision porte ce qu'elle écarte.** Chaque arbitrage garde le motif des
  options rejetées, pour qu'on puisse le rouvrir sans le rejouer.
- **Ce qui se mesure se mesure.** Plusieurs règles ont été corrigées par une
  mesure qui contredisait un raisonnement ; c'est la mesure qui tranche.
- **Les messages de commit ne racontent pas la fabrication.** Ils disent ce qui
  change et pourquoi — le reste est dans le diff, et le récit de la façon dont
  le travail a été mené n'apprend rien à qui l'a sous les yeux.

Rien de cela ne s'applique différemment à une contribution extérieure : mêmes
contrôles, mêmes règles de style, même bilingue.

## Avant d'écrire du code

Ouvrir une issue d'abord, pour tout ce qui dépasse une correction. Les règles
sont figées et écrites dans [`docs/regles.md`](docs/regles.md) ; les changer est
une discussion de conception, pas un correctif.

## Ce qui se discute avant d'être écrit

Une pull request qui touche l'un de ces chemins sans discussion préalable sera
renvoyée à une issue, quelle que soit sa qualité — non par principe, mais parce
que ce sont les endroits où une modification en fait basculer d'autres.

- **[`docs/regles.md`](docs/regles.md)** fait foi. Le code s'y conforme, donc en
  changer une ligne change ce que le code doit faire, et périme les parties
  enregistrées.
- **[`docs/contrat-formes.md`](docs/contrat-formes.md),
  [`docs/vocabulaire-effets.md`](docs/vocabulaire-effets.md),
  [`docs/protocole-bot.md`](docs/protocole-bot.md), [`schemas/`](schemas/)** sont
  des contrats publics. Ce qui y est écrit engage les plugins et les bots déjà
  publiés.
- **`.github/workflows/`** décide de ce qui est vérifié. La protection de
  branche exige les contrôles par leur nom, pas par leur contenu : un workflow
  modifié peut rendre vert un contrôle qui ne vérifie plus rien.

Le reste — code, tests, documentation d'accompagnement — se propose directement.

## Ce sur quoi une contribution est jugée

Les conventions de code et la doctrine de test sont dans
[`docs/go.md`](docs/go.md). Ce qui suit en est le résumé exigible.

- `make lint && make test && make race && make vulncheck && make sec` passent.
- Toute déclaration a sa documentation. Les commentaires disent *pourquoi*, ils
  ne paraphrasent jamais la ligne suivante.
- Pas de bannière, pas d'emoji décoratif, ni dans le code, ni dans les logs, ni
  dans les messages de commit.
- Le déterminisme est préservé. Rien ne lit l'horloge ni l'entropie système,
  l'aléatoire vient du générateur alimenté par la graine de la partie. Un
  changement qui fait diverger un rejeu de son journal est un défaut même si
  tous les tests passent.
- Une dépendance ajoutée passe par `make notices` : sa licence entre dans
  `THIRD-PARTY-NOTICES`, qui accompagne le binaire dans chaque archive.
- Rien dans `internal/core` ni `internal/ai` n'importe le rendu. Les runners
  sont sans écran ; un test qui exige une fenêtre n'a pas sa place dans la suite
  par défaut.

## Livraison

**Un lot, une branche, un commit.** La branche part de `master` à jour et se
nomme `<type>/<sujet>`, où le type est le préfixe conventionnel de son commit :
`feat/`, `fix/`, `docs/`, `chore/`, `test/`, `refactor/`. Ne pas enchaîner deux
lots sur la même branche — chacun doit rester relisible et annulable seul.

Elle retourne dans `master` **par une pull request**, jamais par une fusion
locale : c'est la PR qui laisse la trace de ce qui a été livré, et sa fusion qui
supprime la branche des deux côtés.

**Vérifier avant de pousser, pas après :**

```
make lint && make test && make race && make vulncheck && make sec
```

`govulncheck` interroge sa base d'avis **en direct** : un job vert le matin peut
être rouge l'après-midi sur exactement le même code. Ne pas se reposer sur
l'intégration continue seule, qui valide une fois la branche déjà poussée.

**La documentation part avec le changement.** Avant de commiter, vérifier ce que
le changement rend faux ailleurs : l'état annoncé dans le README, une règle de
[`docs/regles.md`](docs/regles.md), un exemple, un schéma de
[`schemas/`](schemas/). Cas propre à ce projet : une règle du jeu qui change rend
`docs/regles.md` faux, et ce document fait foi — le code ne prend jamais de
l'avance sur lui.

**Un message dit ce qui change et pourquoi**, en quelques lignes. Le défaut est
le titre seul : un corps n'existe que s'il porte quelque chose que le titre ne
dit pas et que le diff ne montre pas. Une description de pull request n'existe
que si elle porte ce qu'un relecteur ne peut pas déduire du diff — une mesure,
un cas reproduit, une rupture pour l'utilisateur. Sans cela, elle reste vide.

## Corriger une vulnérabilité sans en créer une autre

Ne pas adopter une version publiée **le jour même**, même corrective. Chercher
la plus ancienne qui suffit :

```
go list -m -versions <module>
```

Une version parue dans l'heure est le profil type d'une compromission de compte
mainteneur.

Un épinglage s'explique : un `require` figé plus bas que le dernier disponible
porte un commentaire de fin de ligne disant pourquoi, et **quand le retirer**.

## Quatre numéros, à ne pas confondre

| Numéro | Où | Ce qu'il suit |
|---|---|---|
| version du dépôt | tag git | le binaire |
| `shapes_version` | chaque fichier de formes | le contrat d'apparence |
| `protocol` | échanges avec un bot | le contrat de bot |
| `effects_version` | manifeste d'un plugin de règles | le vocabulaire d'effets |

Les trois derniers sont des entiers sans rapport avec SemVer : ajouter un champ
optionnel ne les incrémente pas, tout le reste les incrémente. Une version peut
sortir sans qu'ils bougent ; ils ne bougent jamais sans version.

**Un `shapes_version` qui change périme tous les plugins d'apparence publiés.**
C'est l'événement le plus coûteux du projet, à annoncer en tête des notes de
version. Un `effects_version` qui change périme les plugins de règles, ce qui
touche moins de monde mais casse aussi les sauvegardes qui les portent.

Le dépôt suit SemVer avec la clause du zéro : **en `0.x`, rien n'est imposé.**
Le mineur marque un jalon de [`ROADMAP.md`](ROADMAP.md), pas une rupture d'API ;
tout le reste s'accumule en correctif, qu'il s'agisse d'un correctif, d'une
fonctionnalité ou d'une rupture. Conséquence directe : **le numéro ne prévient
de rien**, et ce sont les notes de version qui doivent dire ce qu'un auteur de
plugin doit reprendre.

## Langue

**Les identifiants sont en anglais** — répertoires, fichiers, paquets, types,
fonctions, champs : `Fugitive`, `Board`, `Trail`. **La documentation est en
français** : godoc, commentaires, messages d'erreur et journaux. L'API se lit en
anglais parce que c'est du code ; le raisonnement se lit en français parce que
c'est de la pensée.

Messages de commit en français d'abord, anglais ensuite, dans un seul texte
séparé par `***`. Jamais `---` : `git am` le traite comme un séparateur de patch
et tronque tout ce qui suit.

Les contributions en anglais sont bienvenues et ne sont pas soumises à la règle
bilingue.

## Plugins

Les plugins vivent dans leurs propres dépôts. Le catalogue les indexe, il
n'héberge aucun exécutable, et il n'accepte **aucun fichier binaire, sous aucune
extension**. C'est une règle mécanique et non un jugement : elle supprime toute
question de provenance.
