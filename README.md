# Filature

Français : [README.fr.md](README.fr.md)

A turn-based board game of hidden movement. One fugitive against five
inspectors, on a city of streets and buildings.

The fugitive moves in eight directions, the inspectors in four. He is faster,
but alone and invisible most of the time. They are many and see far, but cannot
cover six extraction zones with five pieces, and rarely know where he is.

The asymmetry is the game: the fugitive knows where he is going, the inspectors
have to guess.

## Status

Early. The rules are settled, the engine is not written yet.

- [`docs/regles.md`](docs/regles.md) — the full specification (French)
- [`ROADMAP.md`](ROADMAP.md) — the steps, and what is out of scope for v1

## Install

Download a binary from the releases page, extract, run. No runtime, no system
dependency.

The Windows binary is not signed: SmartScreen will warn on first launch.

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
