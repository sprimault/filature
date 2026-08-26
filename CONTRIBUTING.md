# Contributing

Français : [CONTRIBUTING.fr.md](CONTRIBUTING.fr.md)

## Before writing code

Open an issue first for anything beyond a fix. The rules are settled and written
down in [`docs/regles.md`](docs/regles.md); a change to them is a design
discussion, not a patch.

## What a pull request is judged against

- `make lint && make test && make race && make vulncheck && make sec` pass.
- Every declaration has documentation. Comments explain *why*, never restate the
  line below.
- No banners, no decorative emoji, in code, logs or commit messages.
- Determinism is preserved. Nothing reads the clock or system entropy; random
  numbers come from the game's seeded generator. A change that makes a replay
  diverge from its journal is a defect even if every test passes.
- A new dependency goes through `make notices`: its licence enters
  `THIRD-PARTY-NOTICES`, which ships with the binary in every archive.
- Nothing in `internal/core` or `internal/ai` imports the renderer. CI runners
  are headless; a test that needs a window does not belong in the default suite.

## Language

**Identifiers are in English** — directories, files, packages, types, functions,
fields: `Fugitive`, `Board`, `Trail`. **Documentation is in French**: godoc,
comments, error messages and logs. The API reads in English because it is code;
the reasoning reads in French because it is thought.

Commit messages are French first, then English, in one text separated by `***`.
Never `---`: `git am` treats it as a patch separator and truncates everything
after it.

Contributions in English are welcome and are not held to the bilingual rule.

## Plugins

Plugins live in their own repositories. The catalogue indexes them; it does not
host executables, and it accepts **no binary files under any extension**. That is
a mechanical rule, not a judgement call — it removes every provenance question.
