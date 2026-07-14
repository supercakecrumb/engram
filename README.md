# engram

A storage-agnostic spaced-repetition engine for Go, wrapping the FSRS algorithm.

## Why

`github.com/open-spaced-repetition/go-fsrs` implements only the FSRS scheduling *algorithm* — one card in, a schedule out. Everything an app actually needs around it — decks, due queues, daily new-item caps, review logs, statistics, multiple-choice quiz generation — everyone hand-rolls. engram is that missing layer. It has exactly one dependency: go-fsrs itself.

## Design principles

- **Deterministic.** Time is always an explicit `now time.Time` parameter; randomness is always an injected `*rand.Rand`. No hidden clock reads, no global state. Fully reproducible in tests.
- **Storage-agnostic.** engram defines `SkillStore` and `ReviewStore` interfaces; the application owns persistence. An in-memory `memstore` ships for tests and small apps.
- **Generic over "skill".** A skill is whatever unit you track (a language to recognise, a chord, a capital city) — engram schedules it and samples fresh content each review.
- **v0.x until the API settles.** The surface is pinned but may evolve from real use before v1.0.

## Install

```
go get github.com/supercakecrumb/engram
```

Requires Go 1.26 (the module targets `go 1.26`; the toolchain is fetched automatically).

## Quickstart

Play a deterministic Romance-languages flashcard session:

```
go run ./examples/cli
```

It demonstrates the scheduler, due queue, quiz generation, and stats working together over the in-memory store.

Here's the core loop:

```go
package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/supercakecrumb/engram"
	"github.com/supercakecrumb/engram/quiz"
)

func main() {
	sched := engram.NewScheduler() // retention 0.9, max interval 365d, deterministic
	now := time.Now()

	// A never-reviewed skill starts due immediately.
	card := sched.NewCardState(now)

	// Build a multiple-choice exercise from a deck of skills.
	deck := []engram.Skill{
		{ID: "sk-por", DeckID: "romance", Key: "por", Label: "Portuguese"},
		{ID: "sk-spa", DeckID: "romance", Key: "spa", Label: "Spanish"},
	}
	rng := rand.New(rand.NewSource(1))
	ex := quiz.Generate(rng, deck[0], deck, quiz.Content{ID: "c1", Payload: "Olá, tudo bem?"})

	// Grade an answer and advance the schedule.
	correct, rating := ex.Grade("por")
	next, review := sched.Next(card, now, rating)
	review.SkillID = ex.SkillID

	fmt.Printf("correct=%v next due in %s\n", correct, next.Due.Sub(now))
	_ = context.Background()
	_ = review
}
```

## API tour

- Root package `engram`: types `Deck`, `Skill`, `CardState`, `Review`, `State`, `Rating`; `Scheduler` (via `NewScheduler` with options `WithRetention`, `WithMaxInterval`, `WithWeights`) exposing `NewCardState(now)` and `Next(cs, now, rating)`; the answer→rating policy `RatingForAnswer(correct)`.
- Queue: `NextDue(items, newIntroducedToday, cfg, now)` (due-first ordering, `MaxNewPerDay` cap) and `CountNewIntroduced(log, now, loc)`.
- Stats (pure functions over review logs): `Accuracy`, `Streak`, `DueForecast`.
- Storage interfaces: `SkillStore`, `ReviewStore`; in-memory `memstore.Store` implements both.
- `quiz` subpackage: `Generate` (seeded, shuffled, exactly one correct option), `Exercise.Grade`, and `Confusion` for "you mistake Portuguese for Spanish 40% of the time"-style analysis.

## Status

v0.x, first consumer is a Telegram GeoGuessr language trainer. Local development targets Go 1.26.

## License

MIT © Aurora Kiel — see [LICENSE](LICENSE).
