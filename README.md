# Evasion

Français : [README.fr.md](README.fr.md)

[![CI](https://github.com/sprimault/evasion/actions/workflows/ci.yml/badge.svg)](https://github.com/sprimault/evasion/actions/workflows/ci.yml)

A turn-based board game of hidden movement. One fugitive after a way out, five
inspectors closing it off, on a city of streets and buildings.

Apache 2.0 — [`LICENSE`](LICENSE), and [`THIRD-PARTY-NOTICES`](THIRD-PARTY-NOTICES)
for the libraries linked into the binary.

The fugitive moves in eight directions, the inspectors in four. He is faster,
but alone and invisible most of the time. They are many and see far, but cannot
cover six extraction zones with five pieces, and rarely know where he is.

The asymmetry is the game: the fugitive knows where he is going, the inspectors
have to guess.

## Status

**The binary plays, in text.** Isometric rendering is step 7, network play step
12, and the opponent still draws its moves at random — the real AI comes with
steps 9 and 10.

Appearance plugins already load and validate, even though nothing draws them
yet: a shape overflowing its template or naming a colour that does not exist is
refused at startup.

- [`docs/regles.md`](docs/regles.md) — the full specification (French)
- [`ROADMAP.md`](ROADMAP.md) — the steps, and what is out of scope for v1

## How this game is written

*Evasion* is also an experiment in method: the game is written alongside an
assistant, under explicit repository rules versioned with the code.

Design decisions are argued and measured rather than guessed —
[`docs/regles.md`](docs/regles.md) carries, for each rule, what it rules out and
what remains to be checked. What this changes for a contribution is in
[`CONTRIBUTING.md`](CONTRIBUTING.md): nothing substantive, the same checks apply
to everyone.

## Install

Download a binary from the releases page, extract, run. No runtime, no system
dependency.

**The archive holds the executable and its licence notices, nothing else.**
Rules, shapes and labels are inside the binary: nothing to install alongside,
and moving the file breaks nothing.

The Windows binary is not signed: SmartScreen will warn on first launch.

### Commands

```
evasion                      plays a game, in text
evasion version              the installed version number
evasion examples <folder>    writes the shipped plugins, as templates
evasion validate <folder>    checks a plugin and prints its fingerprint
evasion preview <folder>     renders its shapes and a board as SVG
```

The last three belong in a terminal, not a double-click — their output is text,
and a window opened from the file explorer closes before anything can be read.
`--version` is accepted as an alias.

`validate` runs exactly the checks the loader runs, and lists everything wrong at
once rather than one fault at a time. A plugin it accepts will load for whoever
you give it to.

`preview` writes two files: the shape sheet, each shape on every ground, and a
board in situation. The plugin is merged onto shipped content
before rendering — it only declares what it replaces — and the sheet marks what
comes from it, which also shows when a misspelled key overrode nothing. A second
argument says where to write.

`examples` refuses to write into the active plugin folder, and says so: what it
writes out is already inside the binary, and putting it back there would declare
it twice. These are templates to copy under another name.

### Flags

```
--side fugitive     play the fugitive; inspectors is the default,
                    watch lets two machines play
--preset city       board size: district, outskirts or city
--seed 1            the game's seed; the same one replays the same game
--delay 800ms       pause between turns when nobody plays; without it
                    the whole game scrolls past at once
--plugins <folder>  where to look for plugins
--host              host a network game
--join <address>    join one
--game <name>       resume a saved game
```

The game looks for plugins in a `plugins` folder **next to the executable**,
not in the current directory — so a shortcut behaves like a direct launch. Use
`--plugins` to point elsewhere.

## Extending

Evasion is meant to be modified. Four levels, from cheapest to most involved:

| Level | What it is | Format |
|---|---|---|
| Data | Abilities, resistance expenses, game modes | TOML |
| Appearance | Shapes and palettes | TOML |
| Bots | A replacement AI, in any language | separate process |
| WebAssembly | A board generator or an embedded AI | `.wasm` |

Appearance plugins are **geometry, never images**. See
[`docs/contrat-formes.md`](docs/contrat-formes.md) — it is text, so it reads as a
diff, raises no provenance question, and fits the sprite budget by construction.
The trade-off is stated there: hand-drawn art will never be publishable through
the catalogue.

A bot replaces the game's AI rather than extending it: the game sends a view, the
bot returns a move. The shipped AI speaks the same protocol, which is what proves
it sufficient. See [`docs/protocole-bot.md`](docs/protocole-bot.md).

Outside the catalogue, nothing is restricted. Your machine, your rules.

## Going further

- [`docs/architecture.md`](docs/architecture.md) — the core, the filtered view,
  determinism
- [`docs/regles.md`](docs/regles.md) — the full rules and their numbers
- [`docs/plugins.md`](docs/plugins.md) — a plugin's format, file by file and
  field by field (French)
- [`docs/vocabulaire-effets.md`](docs/vocabulaire-effets.md) — the primitives
  abilities, costs and game modes are composed from (French)
- [`docs/go.md`](docs/go.md) — code conventions and testing doctrine (French)
- [`docs/construction.md`](docs/construction.md) — build matrix, packaging,
  signing
- [`schemas/`](schemas/) — the public contracts, versioned separately
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — what a pull request is judged against

## Where the name comes from

In French, an evasion is a breakout: what the fugitive is after, and what the
five inspectors exist to prevent. The English word carries the other half —
slipping away, never being where they look. The game asks for both.

## License

Apache 2.0. **The name Evasion, its visual identity and its palette are not
part of it** — forks are welcome under a different name.

The binary embeds third-party libraries whose licences require their notices to
travel with it: `THIRD-PARTY-NOTICES` ships in every archive, next to
`LICENSE`.
