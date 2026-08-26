# Contribuer

English: [CONTRIBUTING.md](CONTRIBUTING.md)

## Avant d'écrire du code

Ouvrir une issue d'abord, pour tout ce qui dépasse une correction. Les règles
sont figées et écrites dans [`docs/regles.md`](docs/regles.md) ; les changer est
une discussion de conception, pas un correctif.

## Ce sur quoi une contribution est jugée

- `make lint && make test && make race && make vulncheck && make sec` passent.
- Toute déclaration a sa documentation. Les commentaires disent *pourquoi*, ils
  ne paraphrasent jamais la ligne suivante.
- Pas de bannière, pas d'emoji décoratif, ni dans le code, ni dans les logs, ni
  dans les messages de commit.
- Le déterminisme est préservé. Rien ne lit l'horloge ni l'entropie système,
  l'aléatoire vient du générateur alimenté par la graine de la partie. Un
  changement qui fait diverger un rejeu de son journal est un défaut même si
  tous les tests passent.
- Rien dans `internal/core` ni `internal/ai` n'importe le rendu. Les runners
  sont sans écran ; un test qui exige une fenêtre n'a pas sa place dans la suite
  par défaut.

## Langue

Les identifiants suivent le domaine, qui est français ici — `Fugitif`,
`Plateau`, `Trace`. Commentaires et documentation en français. Messages de
commit en français d'abord, anglais ensuite, dans un seul texte séparé par
`***`. Jamais `---` : `git am` le traite comme un séparateur de patch et tronque
tout ce qui suit.

Les contributions en anglais sont bienvenues et ne sont pas soumises à la règle
bilingue.

## Plugins

Les plugins vivent dans leurs propres dépôts. Le catalogue les indexe, il
n'héberge aucun exécutable, et il n'accepte **aucun fichier binaire, sous aucune
extension**. C'est une règle mécanique et non un jugement : elle supprime toute
question de provenance.
