# engram — build progress (Phases 1–2)

Agent A. Built the **engram** spaced-repetition engine library end-to-end against the pinned
contract in `Projects/wiki/tech/geodrill-architecture.md` §3 (public API) and §9 (verification
matrix). Status: **complete and green.**

## What was built

```
engram/
├── engram.go            # types: DeckID/SkillID/ContentID, Deck, Skill, State, Rating, CardState, Review
├── scheduler.go         # Scheduler wrapping go-fsrs/v3; options, NewCardState, Next, RatingForAnswer
├── queue.go             # QueueItem, QueueConfig, NextDue (due-first + new-cap), CountNewIntroduced
├── stats.go             # Accuracy, Streak, DueForecast (+ shared DST-safe dayNumber helper)
├── store.go             # SkillStore + ReviewStore interfaces (user-scoped)
├── quiz/quiz.go         # Content, ContentProvider, Option, Exercise, Generate, Grade, Attempt, Confusion
├── memstore/memstore.go # concurrency-safe in-memory SkillStore+ReviewStore (+ AddSkill helper)
├── examples/cli/main.go # runnable deterministic Romance-deck flashcard demo
├── *_test.go            # table-driven tests per §9 (root, quiz, memstore)
├── README.md, LICENSE (MIT), PROGRESS.md, .gitignore
└── go.mod / go.sum      # module github.com/supercakecrumb/engram, go 1.26
```

Every pinned symbol in contract §3 is implemented verbatim (type names, field names/order, constant
values, function signatures). Additions only — see below.

## Environment / versions

- **Go 1.26 toolchain fetch SUCCEEDED.** Installed toolchain is 1.25.3; `go.mod` declares `go 1.26`
  and Go auto-downloaded `go1.26.0 (darwin/arm64)` on first `go get`. All build/test/vet run under
  1.26 via `GOTOOLCHAIN=auto`. No fallback to 1.25 was needed.
- Sole dependency: `github.com/open-spaced-repetition/go-fsrs/v3 v3.3.1` (pinned). No other deps.
- One environment quirk: `go test -cover ./...` prints
  `compile: version "go1.26.0" does not match go tool version "go1.25.3"` warnings while
  coverage-instrumenting the `examples/cli` `main` package (a GOTOOLCHAIN/-cover interaction). It
  does **not** affect plain `go test`/`go build`/`go vet`/`go run`, which are all clean, and coverage
  still reports for the library packages (root 90.8%, memstore 100%, quiz 88.1%).

## Key decisions (and why)

1. **CardState ↔ fsrs.Card mapping.** `fsrs.Card` uses `uint64` for Reps/Lapses and carries
   `ElapsedDays`/`ScheduledDays`; engram's pinned `CardState` uses `int` and omits those two fields.
   Verified in go-fsrs source that `FSRS.Next` recomputes elapsed days from `now - LastReview` and
   ignores the input card's `ElapsedDays`/`ScheduledDays`, so `CardState` round-trips losslessly for
   scheduling. Converters `toFSRSCard`/`fromFSRSCard` leave those two fields zero on the way in.
2. **Scheduler defaults.** Start from `fsrs.DefaultParam()` (retention 0.9, fuzz off) but override
   `MaximumInterval` to **365** per the contract default (go-fsrs defaults to 36500). `EnableFuzz` is
   forced `false` regardless of options — fuzz injects uncontrolled randomness that would break the
   determinism guarantee, and no option exposes it.
3. **`Review.ScheduledDays` semantics.** Set to the *newly scheduled* interval (`info.Card.ScheduledDays`),
   not the FSRS `ReviewLog.ScheduledDays`. Because `CardState` doesn't persist `ScheduledDays`, the
   FSRS review-log field would always read 0 here; the new card's scheduled interval is the meaningful,
   non-degenerate value (0 for Again/learning steps, N days for review transitions).
   `Review.ElapsedDays` = `info.ReviewLog.ElapsedDays` (days since last review, recomputed by FSRS).
4. **`Review.SkillID` left zero by `Scheduler.Next`** exactly as the contract comment specifies — the
   caller (geodrill) fills it. The examples/CLI and tests do so.
5. **DST-safe calendar-day math.** `CountNewIntroduced`, `Streak`, and `DueForecast` bucket by calendar
   day *in the passed `*time.Location`*, not by 24h arithmetic. Shared helper `dayNumber(t, loc)`
   converts `t.In(loc)`'s local Y/M/D to a UTC-midnight day number (`Unix()/86400`); differencing two
   day numbers gives exact calendar-day distance regardless of loc/DST. `loc == nil` defaults to UTC
   everywhere (no panics).
6. **`Streak` "today not yet studied" rule.** If `now`'s day has no review, the streak still counts
   from yesterday backwards (a day that just hasn't been studied yet doesn't zero a live streak).
   Returns 0 only if neither today nor yesterday has a review, or the log is empty.
7. **`DueForecast` clamping.** Overdue and due-today cards both land in bucket 0; cards due beyond the
   window are dropped; `days <= 0` returns a non-nil empty slice.
8. **`Confusion.Share` denominator** = count of *all* attempts (correct or wrong) whose
   `CorrectKey == TargetKey`, i.e. "how often you saw this target", not just the wrong ones. Pairs are
   sorted `Count` desc, then `TargetKey` asc, then `ChosenKey` asc for a fully deterministic order.
9. **`quiz.Generate`** builds one option per `deckSkills` entry and shuffles with the injected `*rand.Rand`
   only (no global rand). If `target` isn't found in `deckSkills` by Key, an option for it is appended so
   there is always exactly one correct option. `Exercise.ID` is left empty for the caller to set.
10. **memstore** guards all state with a `sync.Mutex` (safe for concurrent use) and returns copies of all
    slices so callers can't mutate internal state. Added an **`AddSkill`** helper (not in the pinned
    contract — additive, allowed) so the store can be populated in tests/small apps.

## Deviations from the contract

None to pinned signatures/constants/fields. Additions only: exported helper `memstore.(*Store).AddSkill`,
unexported helpers (`dayNumber`, `toFSRSCard`, `fromFSRSCard`, sort/confusion helpers), and doc comments.

## Test summary (`go test ./...`)

29 tests (incl. subtests, table-driven), all passing under Go 1.26; `go vet ./...` clean; `gofmt -l .`
reports nothing. Coverage: root `engram` 90.8%, `memstore` 100%, `quiz` 88.1%.

```
$ GOTOOLCHAIN=auto go test ./...
ok  	github.com/supercakecrumb/engram	0.207s
?   	github.com/supercakecrumb/engram/examples/cli	[no test files]
ok  	github.com/supercakecrumb/engram/memstore	0.356s
ok  	github.com/supercakecrumb/engram/quiz	0.510s
```

Coverage of the §9 matrix:
- **scheduler monotonicity** — `NewCardState` due-now; Good grows the interval (strictly larger interval
  and stability each Review-state step); Again resets to minutes + increments Lapses from Review; Due
  always advances past `now` for every rating on a review-state card; construction determinism (fuzz off).
- **queue invariants** — due-first ordering (earliest Due wins over new cards), `MaxNewPerDay` respected
  (under/at/over cap; 0 = unlimited), `ok=false` on empty; due card preferred over new even with cap room.
- **CountNewIntroduced day boundary** — verified against `America/New_York`: instants that fall on a
  different UTC day than NY day count by the NY day; StateReview entries never counted; `nil` loc → UTC.
- **stats fixtures** — Accuracy (empty→0, mixed shares), Streak across day gaps incl. non-UTC and the
  today-not-yet-studied case, DueForecast bucketing incl. overdue clamp and `days<=0`.
- **quiz** — exactly one correct option and each deck key present once; seeded-shuffle determinism;
  Grade correctness + rating for every option; Confusion counts/shares with float tolerance; empty→[].
- **memstore round-trips** — Skills (insertion order, per-deck, unknown→[]), Card (miss→false, put/update),
  Log (since filter + append order), and copy semantics (mutating returned slices doesn't corrupt state).

Also runs clean under `go test -race ./...`.

## For the geodrill agent (Agent B)

- Import path `github.com/supercakecrumb/engram`; subpackages `.../quiz` and `.../memstore`. Module is
  `go 1.26`. In the `go.work` workspace, geodrill's Postgres adapters implement `engram.SkillStore` and
  `engram.ReviewStore` (both scoped to one user).
- `Scheduler.Next` returns `Review` with `SkillID` **zero** — set it yourself before persisting.
- `CardState` maps 1:1 onto the `user_skills` columns (contract §4); `Review` maps onto the `reviews`
  columns. `Review.ScheduledDays` is the newly-scheduled interval; `Review.ElapsedDays` is days since last
  review (see decision 3).
- Everything is deterministic: pass an explicit `now time.Time` and, for `quiz.Generate`, an injected
  `*rand.Rand`. Bucket stats with the user's `*time.Location` (tz) so day boundaries match the DB.
- Local tag **`v0.1.0`** created. Local repo only — no GitHub remote, no push (Aurora creates the remote).

## Known gaps / non-goals (per plan §10)

- Rating policy is v1 only (`RatingForAnswer`: wrong→Again, correct→Good). Hard/Easy by response time is
  future work. The scheduler already accepts all four ratings.
- No personalised FSRS weight optimisation (reviews export → offline optimiser is a later phase);
  `WithWeights` is provided to plug optimised weights in when available.
- Image/other content kinds are the app's concern via `quiz.ContentProvider`; engram stays content-agnostic.
