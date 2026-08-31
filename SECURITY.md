# Security policy

Français : [SECURITY.fr.md](SECURITY.fr.md)

## Reporting

Use GitHub's private vulnerability reporting on this repository. Never open a
public issue for a security flaw.

## Scope

Evasion is a game. Two areas actually matter:

**Hidden information.** The whole game rests on one camp not knowing what the
other knows. A way to read the fugitive's position or sealed zone — through the
network traffic, a save file, a plugin, or a bot — is a real defect, not a
curiosity. Report it.

**Plugin sandbox.** A WebAssembly plugin must not reach the filesystem, the
network, the clock, or system entropy. An escape is a vulnerability.

## Out of scope

- Anything a player does on their own machine. Loading an unvetted plugin
  locally is allowed by design.
- The host of a networked game holding the full state in memory. This is known
  and documented, not a flaw.
- Third-party bots. They run as ordinary processes the player launched.
