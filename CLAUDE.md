# engram

Storage- and domain-agnostic spaced-repetition engine for Go — decks, due queues, daily caps, review logs, and stats layered over [go-fsrs](https://github.com/open-spaced-repetition/go-fsrs). Owner: Aurora (`supercakecrumb`). First consumer: geodrill (`../../PersonalProjects/geodrill`), co-developed via geodrill's `go.work`.

## Git workflow

- Remote: `https://github.com/supercakecrumb/engram` (**public**), branch `master`. Semver tags (`v0.1.0`, `v0.2.0` pushed).
- **One atomic commit per logical change, pushed immediately.** Never batch. Every commit builds, `go vet`-clean, tests green.
- **changie** manages the changelog: fragment per user-facing change in the same commit; `changie batch` + `changie merge` at each version tag; push tags to origin.
- Never rewrite pushed history.
- **Tell Aurora everything, in Telegram.** Every commit auto-notifies her via the post-commit hook (`scripts/githooks/`, self-installed by `scripts/pre-commit.sh`; token read from the sibling geodrill `.env`). Milestones get a manual `./scripts/notify.sh "<text>"`.

## Rules

- **Domain-agnostic, always.** No Telegram/Postgres/GeoGuessr concepts may leak into the API. Content behind provider interfaces, persistence behind store interfaces, clock/RNG injected (deterministic tests).
- Public repo — no secrets (none should exist here at all).
- v0.x until the API is proven by real consumers; breaking changes allowed but must be changelogged.

## Build

`go build ./... && go vet ./... && go test ./...` (race-clean). CI (being introduced): GitHub Actions lint/test/build on push; GitHub Release on tag — it's a library, no container image. Validate workflow changes locally with `act` before pushing.
