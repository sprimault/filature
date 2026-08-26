# Filature

Français : [README.fr.md](README.fr.md)

![CI](https://github.com/sprimault/filature/actions/workflows/ci.yml/badge.svg)
![License](https://img.shields.io/badge/license-Apache%202.0-blue)

A turn-based board game of hidden movement. One fugitive against five
inspectors, on a city of streets and buildings.

The fugitive moves in eight directions, the inspectors in four. He is faster,
but alone and invisible most of the time. They are many and see far, but cannot
cover six extraction zones with five pieces, and rarely know where he is.

The asymmetry is the game: the fugitive knows where he is going, the inspectors
have to guess.

## Status

**The binary plays, in text.** Isometric rendering is step 7, network play step
12, and the opponent still draws its moves at random — the real AI comes with
steps 9 and 10.

- [`docs/regles.md`](docs/regles.md) — the full specification (French)
- [`ROADMAP.md`](ROADMAP.md) — the steps, and what is out of scope for v1

## Install

Download a binary from the releases page, extract, run. No runtime, no system
dependency.

**The archive holds the executable and the licence, nothing else.** Rules,
shapes and labels are inside it: nothing to install alongside, and moving the
file breaks nothing.

The Windows binary is not signed: SmartScreen will warn on first launch.

### Commands

```
filature                      plays a game, in text
filature version              the installed version number
filature examples <folder>    writes the shipped plugins, as templates
filature validate <folder>    checks a plugin and prints its fingerprint
```

The last three belong in a terminal, not a double-click — their output is text,
and a window opened from the file explorer closes before anything can be read.
`--version` is accepted as an alias.

`validate` runs exactly the checks the loader runs, and lists everything wrong at
once rather than one fault at a time. A plugin it accepts will load for whoever
you give it to.

`examples` refuses to write into the active plugin folder, and says so: what it
writes out is already inside the binary, and putting it back there would declare
it twice. These are templates to copy under another name.

### Flags

```
--side fugitive     play the fugitive; inspectors is the default,
                    watch lets two machines play
--preset ville      board size: quartier, faubourg or ville
--seed 1            the game's seed; the same one replays the same game
--plugins <folder>  where to look for plugins
--host              host a network game
--join <address>    join one
--game <name>       resume a saved game
```

The game looks for plugins in a `plugins` folder **next to the executable**,
not in the current directory — so a shortcut behaves like a direct launch. Use
`--plugins` to point elsewhere.

## Extending

Filature is meant to be modified. Four levels, from cheapest to most involved:

| Level | What it is | Format |
|---|---|---|
| Data | Abilities, resistance costs, board presets, game modes | TOML |
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
- [`docs/construction.md`](docs/construction.md) — build matrix, packaging,
  signing
- [`schemas/`](schemas/) — the public contracts, versioned separately
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — what a pull request is judged against

## Where the name comes from

*Filature* is French for a tail — following someone without being seen. It names
both camps at once: you tail, and you are tailed.

## License

Apache 2.0 — see [`LICENSE`](LICENSE).

The name Filature, its visual identity and its palette are not covered by that
license. Forks are welcome under a different name.
