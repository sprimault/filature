# Protocole de bot

Version du protocole : **2**

Un bot **remplace** l'IA du jeu, il ne l'étend pas. Le jeu lui transmet une vue
de la partie, il renvoie un coup. Aucune API interne à respecter, aucun plugin
à compiler, n'importe quel langage.

L'IA livrée avec le jeu parle ce protocole comme les autres. C'est ce qui
garantit qu'il est suffisant : s'il manquait quelque chose, le jeu ne pourrait
pas jouer contre lui-même.

---

## 1. Transport

Le jeu lance l'exécutable et parle par ses **entrée et sortie standard**, en
JSON Lines : un objet JSON par ligne, terminé par un saut de ligne, sans
indentation.

Ni socket, ni longueur préfixée, ni négociation. Un bot s'écrit en trente lignes
de Python, se déverminne avec un fichier et une redirection, et se rejoue à la
main.

La **sortie d'erreur est libre** : le bot y écrit ce qu'il veut, le jeu la
capture et l'affiche dans son journal. C'est là que va le débogage, jamais sur
la sortie standard.

Le bot reçoit exactement la même `View` que l'interface. Il ne voit donc jamais
la position du fugitif quand celui-ci est caché, ni sa zone d'extraction. Ce
n'est pas une politesse : c'est la même projection qui protège le mode réseau, et
elle est appliquée au même endroit.

---

## 2. Séquence

```
jeu  -> bot   bonjour     version du protocole, réglages, graine, camp
bot  -> jeu   pret        nom, version, déterminisme
jeu  -> bot   joue        vue de la partie, budget en millisecondes
bot  -> jeu   coup        le coup choisi
              ... répété jusqu'à la fin ...
jeu  -> bot   fin         résultat
```

Le jeu ferme l'entrée standard après `fin`. Le bot termine ; s'il est encore
vivant au bout d'une seconde, il est tué.

---

## 3. Messages du jeu

### bonjour

```json
{"type":"bonjour","protocol":2,"side":"inspectors","seed":178342119,
 "settings":{"size":41,"range":8,"turns":40,"centre_radius":10,"stamina":10,
 "inspectors":5,"pieces_per_turn":3,"reveal_period":4,"zones":6,
 "trail_lifetime":6,"strangling_start":30,"strangling_period":2},
 "plugins":[{"name":"base","version":"0.1.0","fingerprint":"…","rules":true}]}
```

`seed` est fournie pour qu'un bot puisse être déterministe s'il le souhaite.
**Un bot ne tire jamais de l'entropie système ni de l'horloge s'il se déclare
déterministe** : c'est la seule chose que le jeu lui demande, et elle est
vérifiée.

`plugins` liste les extensions de règles actives. Un bot qui ne les connaît pas
peut refuser de jouer plutôt que de jouer faux.

### joue

```json
{"type":"joue","turn":7,"budget_ms":2000,"view":{ … }}
```

`view` est l'objet décrit dans
[`schemas/view.schema.json`](../schemas/view.schema.json), identique à celui que
reçoit l'interface. Il porte `legal_moves` : un bot minimal en choisit un et
s'arrête là.

Ce schéma est **généré depuis la structure Go**, jamais écrit à la main : un
test compare les deux et échoue tant qu'ils divergent. Ce qu'il décrit est donc
ce que le jeu envoie, sans décalage possible.

Toute liste y est un tableau, jamais `null` : une vue sans barrage porte un
tableau vide. Un bot n'a pas à traiter deux formes pour la même absence.

`legal_moves` contient tout ce que le camp peut faire ce tour-ci —
déplacements, capacités, dépenses de résistance, meurtre compris. Un bot n'a
donc rien à savoir des règles pour être correct : il ne peut pas jouer un coup
illégal s'il choisit dans cette liste. C'est aussi ce qui fait qu'un plugin de
règles ajoutant une dépense est utilisable par un bot qui l'ignore.

### fin

```json
{"type":"fin","winner":"fugitive","reason":"extraction","turn":34}
```

Le jeu de base produit cinq motifs : `extraction`, `captured`, `stamina_spent`,
`fugitive_cornered`, `time_up`. **La liste n'est pas fermée** — un plugin de
règles qui invente une condition de victoire produit son propre motif, et un bot
qui ne le connaît pas l'a appris dans `bonjour`, qui annonce les plugins
actifs. Traiter le champ comme une chaîne, jamais comme une énumération.

---

## 4. Messages du bot

### pret

```json
{"type":"pret","name":"traqueur-glouton","version":"0.3.1",
 "protocol":2,"deterministic":true,"author":"…"}
```

### coup

```json
{"type":"move","move":{"turn":7,"side":"inspectors","type":"step",
 "pion":2,"from":{"Colonne":18,"Ligne":9},"to":{"Colonne":19,"Ligne":9}}}
```

Le coup doit figurer tel quel dans `legal_moves`. Le jeu ne corrige rien et
n'interprète rien.

### erreur

```json
{"type":"erreur","message":"cas non traité : phase de placement"}
```

Le bot déclare qu'il ne peut pas jouer. La partie s'arrête proprement, avec le
message affiché.

---

## 5. Ce que le jeu fait des écarts

| Écart | Réaction |
|---|---|
| Coup absent de `legal_moves` | partie interrompue, coup fautif journalisé |
| JSON invalide, type inconnu | idem |
| `budget_ms` dépassé une fois | coup légal choisi au sort, incident journalisé |
| Budget dépassé trois fois | partie interrompue |
| Processus mort | partie interrompue |
| `protocol` inconnu dans `pret` | refus avant le début |

Un bot lent est toléré une fois puis écarté : laisser une partie se figer sur un
processus tiers est pire qu'un coup au hasard, et le journal garde la trace des
deux.

**Le budget n'est pas une horloge de tournoi.** Il vaut 2000 ms par défaut, et
il est levé pendant une simulation d'équilibrage, où seul le déterminisme
compte.

---

## 6. Déterminisme

Un bot non déterministe est parfaitement jouable. Le journal enregistre les
**coups**, pas l'état interne du bot : la reprise d'une partie sauvegardée et le
rejeu pas à pas fonctionnent quoi qu'il fasse.

Le déterminisme n'est requis que pour deux usages :

- reproduire un défaut à l'identique ;
- entrer dans une passe d'équilibrage, où des milliers de parties doivent être
  comparables entre deux versions.

D'où le drapeau `deterministic` dans `pret`, et le contrôle correspondant : deux
parties sur la même graine, comparaison des coups. Un bot qui se déclare
déterministe sans l'être est refusé au catalogue.

---

## 7. Déclaration

Un bot se déclare dans son manifeste :

```toml
name = "traqueur-glouton"
version = "0.3.1"
rules = false
license = "MIT"

[bot]
side = "inspectors"
command = "traqueur"
arguments = ["--niveau", "3"]
deterministic = true
```

`rules = false` : un bot ne modifie pas les règles, il joue avec. Un plugin
qui déclare un bot **et** des effets est refusé — ce sont deux choses, et les
mélanger casserait la poignée de main réseau, où les plugins de règles doivent
être identiques des deux côtés.

`command` est cherchée dans le dossier du plugin puis dans le `PATH`. Aucun
interpréteur n'est fourni : un bot Python se livre avec son lanceur, ou en
paquet autonome.

---

## 8. Catalogue

Le catalogue **ne distribue aucun exécutable**. L'entrée d'index porte le nom,
la licence, le camp, le drapeau de déterminisme et l'URL du dépôt de l'auteur.
L'installation se fait depuis chez lui, comme n'importe quel logiciel.

Les contrôles — protocole, légalité des coups sur cent parties, déterminisme
déclaré — tournent chez l'auteur, via un workflow réutilisable publié avec le
jeu. Un bot arrive au catalogue validé, ou n'y arrive pas.

Hors catalogue, aucune restriction : un exécutable posé dans le dossier des
plugins est lancé tel quel.

---

## 9. Bot minimal

Il tient en dix lignes, et sert de test de conformité du protocole lui-même :

```python
import json, random, sys

for ligne in sys.stdin:
    message = json.loads(ligne)
    if message["type"] == "bonjour":
        reponse = {"type": "pret", "name": "hasard", "version": "1.0.0",
                   "protocol": 2, "deterministic": False}
    elif message["type"] == "joue":
        reponse = {"type": "move",
                   "move": random.choice(message["view"]["legal_moves"])}
    else:
        break
    print(json.dumps(reponse), flush=True)
```

`flush=True` est la seule chose qui se rate systématiquement : sans lui, la
sortie reste dans le tampon et le jeu attend un coup qui ne vient pas, jusqu'au
dépassement de budget.
