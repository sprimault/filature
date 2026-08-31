# Politique de sécurité

English: [SECURITY.md](SECURITY.md)

## Signalement

Passer par le signalement privé de vulnérabilité de GitHub sur ce dépôt. Jamais
par une issue publique.

## Périmètre

Evasion est un jeu. Deux domaines comptent vraiment :

**L'information cachée.** Tout le jeu repose sur le fait qu'un camp ignore ce
que sait l'autre. Un moyen de lire la position du fugitif ou sa zone scellée —
par le trafic réseau, un fichier de sauvegarde, un plugin ou un bot — est un
vrai défaut, pas une curiosité. À signaler.

**Le bac à sable des plugins.** Un module WebAssembly ne doit atteindre ni le
disque, ni le réseau, ni l'horloge, ni l'entropie système. En sortir est une
vulnérabilité.

## Hors périmètre

- Ce qu'un joueur fait sur sa propre machine. Charger un plugin non vérifié en
  local est permis par conception.
- L'hôte d'une partie en réseau qui détient l'état complet en mémoire. C'est
  connu et documenté, pas un défaut.
- Les bots tiers. Ce sont des processus ordinaires que le joueur a lancés.
