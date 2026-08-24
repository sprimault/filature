# Filature

English: [README.md](README.md)

Un jeu de plateau au tour par tour, à déplacement caché. Un fugitif contre cinq
inspecteurs, sur une ville de rues et de bâtiments.

Le fugitif se déplace dans huit directions, les inspecteurs dans quatre. Il est
plus rapide, mais seul et invisible la plupart du temps. Ils sont nombreux et
voient loin, mais ne peuvent pas couvrir six zones d'extraction à cinq, et ne
savent presque jamais où il est.

L'asymétrie fait le jeu : le fugitif sait où il va, les inspecteurs doivent le
deviner.

## État

Début de projet. Les règles sont figées, le moteur n'est pas écrit.

- [`docs/regles.md`](docs/regles.md) — la spécification complète
- [`ROADMAP.md`](ROADMAP.md) — les étapes, et ce qui est hors périmètre v1

## Installation

Télécharger un binaire depuis la page des versions, extraire, lancer. Aucun
runtime, aucune dépendance système.

Le binaire Windows n'est pas signé : SmartScreen affichera un avertissement au
premier lancement.

## Étendre le jeu

Filature est fait pour être modifié. Quatre niveaux, du moins cher au plus
engageant :

| Niveau | Ce que c'est | Format |
|---|---|---|
| Données | Capacités, dépenses, préréglages, modes de jeu | TOML |
| Apparence | Formes et palettes | TOML |
| Bots | Une IA de remplacement, dans n'importe quel langage | processus séparé |
| WebAssembly | Un générateur de plateau ou une IA embarquée | `.wasm` |

Un greffon d'apparence est de la **géométrie, jamais une image**. Voir
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
- [`docs/greffons.md`](docs/greffons.md) — le format d'un greffon, fichier par
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

Apache 2.0 — voir [`LICENSE`](LICENSE).

Le nom Filature, son identité visuelle et sa palette ne sont pas couverts par
cette licence. Un fork est bienvenu sous un autre nom.
