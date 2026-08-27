# Contributing

Français : [CONTRIBUTING.fr.md](CONTRIBUTING.fr.md)

## How this project is written

The code is written alongside an assistant, under this repository's rules. It is
not a footnote: the method is the project's second subject, and the way the rules
are laid down follows from it.

- **Rules come before code**, they are not inferred from it.
  [`docs/regles.md`](docs/regles.md) is authoritative, and a disagreement between
  the document and the code is a defect in the code.
- **A decision carries what it rules out.** Every trade-off keeps the reason the
  rejected options were rejected, so it can be reopened without being replayed.
- **What can be measured is measured.** Several rules were corrected by a
  measurement that contradicted an argument; the measurement decides.
- **Commit messages do not narrate how the work was done.** They say what changes
  and why — the rest is in the diff, and an account of the making teaches nothing
  to someone who has it in front of them.

None of this applies differently to an outside contribution: same checks, same
style rules, same bilingual format.

## Before writing code

Open an issue first for anything beyond a fix. The rules are settled and written
down in [`docs/regles.md`](docs/regles.md); a change to them is a design
discussion, not a patch.

## What gets discussed before it gets written

A pull request touching any of these paths without prior discussion will be sent
back to an issue, however good it is — not on principle, but because these are
the places where one change makes others fall over.

- **[`docs/regles.md`](docs/regles.md)** is authoritative. The code conforms to
  it, so changing a line there changes what the code must do, and invalidates
  recorded games.
- **[`docs/contrat-formes.md`](docs/contrat-formes.md),
  [`docs/vocabulaire-effets.md`](docs/vocabulaire-effets.md),
  [`docs/protocole-bot.md`](docs/protocole-bot.md), [`schemas/`](schemas/)** are
  public contracts. What is written there binds already-published plugins and
  bots.
- **`.github/workflows/`** decides what gets checked. Branch protection requires
  checks by name, not by content: a modified workflow can turn green a check
  that no longer verifies anything.

Everything else — code, tests, supporting documentation — can be proposed
directly.

## What a pull request is judged against

Code conventions and testing doctrine are in [`docs/go.md`](docs/go.md), in
French. What follows is the summary you are held to.

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

## Delivery

**One batch, one branch, one commit.** The branch starts from an up-to-date
`master` and is named `<type>/<subject>`, where the type is its commit's
conventional prefix: `feat/`, `fix/`, `docs/`, `chore/`, `test/`, `refactor/`.
Do not chain two batches on one branch — each must stay readable and revertable
on its own.

It goes back into `master` **through a pull request**, never through a local
merge: the PR is what records what was delivered, and merging it removes the
branch on both sides.

**Check before pushing, not after:**

```
make lint && make test && make race && make vulncheck && make sec
```

`govulncheck` queries its advisory database **live**: a job that is green in the
morning can be red in the afternoon on exactly the same code. Do not rely on
continuous integration alone, which validates once the branch is already pushed.

**Documentation ships with the change.** Before committing, check what the change
makes false elsewhere: the status announced in the README, a rule in
[`docs/regles.md`](docs/regles.md), an example, a schema in
[`schemas/`](schemas/). Specific to this project: a game rule that changes makes
`docs/regles.md` false, and that document is authoritative — the code never runs
ahead of it.

**A message says what changes and why**, in a few lines. The default is the
title alone: a body exists only if it carries something the title does not say
and the diff does not show. A pull request description exists only if it carries
what a reviewer cannot infer from the diff — a measurement, a reproduced case, a
break for users. Otherwise it stays empty.

## Fixing a vulnerability without creating another

Do not adopt a version published **the same day**, even a corrective one. Look
for the oldest one that suffices:

```
go list -m -versions <module>
```

A version released within the hour is the typical profile of a compromised
maintainer account.

A pin is explained: a `require` held below the latest available carries an
end-of-line comment saying why, and **when to remove it**.

## Four numbers, not to be confused

| Number | Where | What it tracks |
|---|---|---|
| repository version | git tag | the binary |
| `shapes_version` | every shapes file | the appearance contract |
| `protocol` | exchanges with a bot | the bot contract |
| `effects_version` | a rules plugin's manifest | the effects vocabulary |

The last three are integers unrelated to SemVer: adding an optional field does
not bump them, everything else does. A release can ship without them moving;
they never move without a release.

**A `shapes_version` change invalidates every published appearance plugin.** It
is the most expensive event in the project, to be announced at the top of the
release notes. An `effects_version` change invalidates rules plugins, which
affects fewer people but also breaks the saves that carry them.

The repository follows SemVer with the zero clause: **in `0.x`, nothing is
imposed.** The minor marks a milestone from [`ROADMAP.md`](ROADMAP.md), not an
API break; everything else accumulates as a patch, be it a fix, a feature or a
break. Direct consequence: **the number warns of nothing**, and it is the release
notes that must say what a plugin author has to revisit.

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
