# Construction

Le binaire est le mode d'installation : quelqu'un télécharge un fichier et
l'exécute, sans Go, sans runtime, sans dépendance système. Tout ce qui suit en
découle.

---

## La matrice

**Une seule machine ne produit pas toutes les cibles.** Ebitengine ne se compile
en croisé que vers Windows et WebAssembly ; Linux et macOS exigent une
compilation native, parce que le moteur y passe par les API graphiques du
système.

| Cible | Où | cgo |
|---|---|---|
| windows/amd64 | Linux, croisé | `0` |
| js/wasm | Linux, croisé | `0` |
| linux/amd64 | Linux, natif | `1` |
| linux/arm64 | Linux ARM, natif | `1` |
| darwin/arm64 | macOS Apple Silicon, natif | `1` |
| darwin/amd64 | macOS Intel, natif | `1` |

La cible Intel a une date de péremption connue : l'image correspondante est
annoncée comme la dernière x86_64 disponible en intégration continue, jusqu'en
août 2027.

## cgo n'est pas uniforme

C'est la particularité du projet, et elle se contourne sans bruit : rien
n'avertit qu'on vient de la perdre.

Le mettre à `0` sur darwin ne produit pas une erreur claire mais un échec de
linkage obscur. À poser en variable de matrice, jamais en valeur globale.

Conséquence à connaître : le binaire Linux est lié à la glibc. Il n'y a pas de
construction statique musl, donc pas d'image `scratch` possible — sans
importance pour un jeu de bureau.

**Aucune autre dépendance n'a le droit d'introduire cgo.** `modernc.org/sqlite`
et `wazero` sont choisis pour ça : une seconde source rendrait les cibles
Windows et WebAssembly impossibles.

## Le Makefile ne porte pas la matrice

`make binary` prend `OS`, `ARCH` et `CGO` en paramètres. La liste des couples
vit dans le workflow d'intégration, seul endroit où elle peut réellement
s'exécuter. Deux définitions finiraient par diverger, et c'est celle du workflow
qu'on ne peut pas essayer avant de poser un tag.

En local, `make binaries` ne produit que ce qui se croise — Windows et
WebAssembly — et le dit.

## Options de compilation

```
CGO_ENABLED=$CGO GOOS=$OS GOARCH=$ARCH go build -trimpath \
  -ldflags "-s -w -X main.version=$VERSION" -o dist/filature_${OS}_${ARCH}
```

`-trimpath` toujours : les chemins absolus de la machine de construction n'ont
rien à faire dans un artefact public.

Sous Windows, `-H windowsgui` supprime la console au lancement — mais alors
`filature version` n'écrit nulle part, la sortie standard n'existant pas. Le
contrôle de version de la chaîne de publication en dépend : soit on renonce au
drapeau, soit on attache une console à la demande.

## Empaquetage

`.zip` pour Windows, `.tar.gz` ailleurs : c'est ce que chaque système ouvre sans
rien installer, et le tar préserve le bit exécutable qu'un binaire téléchargé nu
perdrait.

`LICENSE` et `NOTICE` vont **dans** l'archive et non à côté : Apache 2.0 les
exige à la redistribution, et personne ne télécharge un fichier séparé pour
accompagner un binaire.

`THIRD-PARTY-NOTICES` les accompagne, au titre des licences des bibliothèques
liées au binaire. La MIT, comme la plupart des licences permissives, exige que
sa notice suive « toute copie ou partie substantielle » du logiciel — un binaire
qui la compile en est une.

**Ce fichier se génère, il ne se tient pas à la main** : `make notices` le
réécrit depuis les dépendances que `go list -deps` trouve dans la commande, donc
sans l'outillage ni les dépendances de test. Un test le compare aux dépendances
réelles à chaque exécution de la suite et échoue quand il vieillit, faute de quoi
une bibliothèque ajoutée entrerait dans les archives sans sa notice, et personne
ne s'en apercevrait.

**L'archive ne contient rien d'autre que le binaire et ces trois fichiers.** Le
contenu livré — règles, formes, palette, français et anglais — est embarqué dans
l'exécutable par `//go:embed` : un binaire déplacé continue de fonctionner, et
personne ne casse ses règles en éditant un fichier qu'il n'a pas écrit.

Ce contenu reste accessible à qui veut s'en inspirer : `filature examples
<dossier>` l'écrit sur le disque, sans jamais écraser un fichier existant. C'est
par là que passe un traducteur qui recopie l'anglais pour en faire sa langue.

Le nom de l'archive porte version, système et architecture ; le binaire qu'elle
contient s'appelle `filature` tout court — une fois extrait, on sait sur quelle
machine on est.

Trois opérations ne peuvent pas se faire cible par cible et vont dans une étape
de rassemblement : **`SHA256SUMS`, calculé en un seul endroit sur toutes les
archives**, l'attestation de provenance, et la création de la version.

Les sommes portent sur les archives et non sur les binaires : c'est l'archive
que la personne télécharge, donc ce qu'elle peut vérifier.

## Signature

Le binaire Windows n'est pas signé : SmartScreen affichera un avertissement au
premier lancement. C'est dit dans le README plutôt que laissé à découvrir.

L'attestation de provenance lie chaque archive au commit et à la chaîne qui l'a
produite, et se vérifie avec `gh attestation verify`. C'est le seul contrepoids
crédible à des binaires non signés.

## Pas d'image Docker

Le mode réseau est de pair à pair entre deux joueurs : personne ne déploie un
conteneur pour faire une partie, et le binaire ouvre une fenêtre.

Un serveur dédié sans écran serait un binaire séparé, avec une étiquette de
compilation excluant le rendu. Le noyau n'importe rien de graphique, donc c'est
faisable — mais rien ne le réclame aujourd'hui.

## Tests et intégration continue

Les machines d'intégration sont sans écran. `internal/core` et `internal/ai`
n'importent pas Ebitengine, ce qui rend la contrainte facile à tenir : aucun
test de la suite par défaut n'ouvre de fenêtre.

Un test qui exigerait `xvfb-run` sous Linux est le signe qu'il n'a pas sa place
dans cette suite. Le rendu se vérifie à l'œil.

`make race` demande cgo, y compris sous Windows où Ebitengine s'en passe : sans
compilateur C installé, le message d'erreur parle de cgo et non du détecteur de
courses, ce qui envoie chercher au mauvais endroit.
