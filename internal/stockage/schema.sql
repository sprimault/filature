-- Schéma de la base de parties.
--
-- Le manifeste des greffons part avec la partie : une sauvegarde faite avec un
-- jeu de greffons ne se recharge pas sans eux, et le jeu le dit au lieu de
-- planter ou, pire, de rejouer faux.

CREATE TABLE IF NOT EXISTS partie (
    id           INTEGER PRIMARY KEY,
    nom          TEXT UNIQUE NOT NULL,
    graine       INTEGER NOT NULL,
    parametres   TEXT NOT NULL,
    greffons     TEXT NOT NULL,
    instantane   TEXT NOT NULL,
    tour         INTEGER NOT NULL,
    terminee     INTEGER NOT NULL DEFAULT 0,
    resultat     TEXT,
    modifiee_le  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS coup (
    partie_id  INTEGER NOT NULL REFERENCES partie(id) ON DELETE CASCADE,
    numero     INTEGER NOT NULL,
    tour       INTEGER NOT NULL,
    acteur     TEXT NOT NULL,
    charge     TEXT NOT NULL,
    PRIMARY KEY (partie_id, numero)
);

CREATE TABLE IF NOT EXISTS poids_ia (
    profil     TEXT PRIMARY KEY,
    niveau     INTEGER NOT NULL,
    poids      TEXT NOT NULL,
    parties    INTEGER NOT NULL DEFAULT 0,
    victoires  INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS coup_partie ON coup(partie_id, tour);
