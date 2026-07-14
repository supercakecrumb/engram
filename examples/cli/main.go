// Command cli is a small, deterministic, self-playing demo of the engram
// spaced-repetition engine: it builds a "Romance languages" deck, runs a
// simulated study session through the scheduler + queue + quiz + memstore
// packages, and prints a stats summary at the end. Nothing here reads from
// stdin or the real clock, so the output is fully reproducible.
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/supercakecrumb/engram"
	"github.com/supercakecrumb/engram/memstore"
	"github.com/supercakecrumb/engram/quiz"
)

// sentenceBank holds a couple of short sample sentences per language, keyed
// by ISO-639-3 code. This stands in for a real quiz.ContentProvider: the
// demo just samples an index with the injected rng.
var sentenceBank = map[string][]string{
	"spa": {"Buenos días, ¿cómo estás?", "Me gusta mucho la música.", "El sol brilla hoy."},
	"por": {"Bom dia, como você está?", "Eu gosto muito de música.", "O sol brilha hoje."},
	"ita": {"Buongiorno, come stai?", "Mi piace molto la musica.", "Il sole splende oggi."},
	"fra": {"Bonjour, comment ça va ?", "J'aime beaucoup la musique.", "Le soleil brille aujourd'hui."},
	"ron": {"Bună ziua, ce mai faci?", "Îmi place foarte mult muzica.", "Soarele strălucește azi."},
}

func main() {
	ctx := context.Background()
	loc := time.UTC

	// Fixed clock and seeded rng: same output every run.
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	rng := rand.New(rand.NewSource(7))

	// --- 1-2. Build the deck, register skills, and seed the store. ---
	deck := engram.Deck{ID: engram.DeckID("romance"), Name: "Romance languages"}
	skills := []engram.Skill{
		{ID: engram.SkillID("spa"), DeckID: deck.ID, Key: "spa", Label: "Spanish"},
		{ID: engram.SkillID("por"), DeckID: deck.ID, Key: "por", Label: "Portuguese"},
		{ID: engram.SkillID("ita"), DeckID: deck.ID, Key: "ita", Label: "Italian"},
		{ID: engram.SkillID("fra"), DeckID: deck.ID, Key: "fra", Label: "French"},
		{ID: engram.SkillID("ron"), DeckID: deck.ID, Key: "ron", Label: "Romanian"},
	}

	store := memstore.New()
	for _, sk := range skills {
		store.AddSkill(sk)
	}

	sched := engram.NewScheduler()
	cfg := engram.QueueConfig{MaxNewPerDay: 3}

	fmt.Printf("Deck: %s (%d skills)\n", deck.Name, len(skills))

	// --- 3-4. Run a simulated study session. ---
	const sessionLen = 12
	reviewLog := make([]engram.Review, 0, sessionLen)

	for i := 0; i < sessionLen; i++ {
		items, err := buildQueueItems(ctx, store, sched, skills, now)
		if err != nil {
			log.Fatalf("build queue items: %v", err)
		}

		newToday := engram.CountNewIntroduced(reviewLog, now, loc)
		item, ok := engram.NextDue(items, newToday, cfg, now)
		if !ok {
			next, found := earliestFutureDue(items, now)
			if !found {
				fmt.Println("\nNothing due right now, and nothing scheduled later today — ending session.")
				break
			}
			fmt.Printf("\nNothing due right now (new-card cap reached) — fast-forwarding from %s to %s.\n",
				now.Format("2006-01-02 15:04:05"), next.Format("2006-01-02 15:04:05"))
			now = next
			item, ok = engram.NextDue(items, newToday, cfg, now)
			if !ok {
				fmt.Println("Still nothing due after fast-forwarding — ending session.")
				break
			}
		}

		skill := item.Skill
		card := item.Card

		// Sample a sentence for the target skill's language.
		sentences := sentenceBank[skill.Key]
		idx := rng.Intn(len(sentences))
		content := quiz.Content{
			ID:      engram.ContentID(fmt.Sprintf("%s-%d", skill.Key, idx)),
			Payload: sentences[idx],
		}

		ex := quiz.Generate(rng, skill, skills, content)

		// Simulate the learner: correct ~75% of the time, driven
		// deterministically by the seeded rng.
		chosenKey := skill.Key
		if rng.Float64() >= 0.75 {
			chosenKey = pickWrongKey(rng, ex.Options, skill.Key)
		}
		correct, rating := ex.Grade(chosenKey)

		mark := "✓"
		if !correct {
			mark = "✗"
		}
		fmt.Printf("\n--- Review #%d (session clock %s) ---\n", i+1, now.Format("2006-01-02 15:04:05"))
		fmt.Printf("Target skill: %s (%s)\n", skill.Label, skill.Key)
		fmt.Printf("Sentence: %q\n", content.Payload)
		fmt.Print("Options:")
		for _, opt := range ex.Options {
			fmt.Printf(" [%s]%s", opt.Key, opt.Label)
		}
		fmt.Println()
		fmt.Printf("Chose %q %s (correct answer: %s / %q) -> rating %s\n",
			chosenKey, mark, skill.Key, skill.Label, ratingName(rating))

		newCard, rev := sched.Next(card, now, rating)
		rev.SkillID = skill.ID
		if err := store.PutCard(ctx, skill.ID, newCard); err != nil {
			log.Fatalf("put card: %v", err)
		}
		if err := store.Append(ctx, rev); err != nil {
			log.Fatalf("append review: %v", err)
		}
		reviewLog = append(reviewLog, rev)

		fmt.Printf("Next due: %s (scheduled %dd, state %s)\n",
			newCard.Due.Format("2006-01-02 15:04:05"), rev.ScheduledDays, stateName(newCard.State))

		// Distinct timestamps for successive reviews, same day.
		now = now.Add(30 * time.Second)
	}

	// --- 5. Stats summary. ---
	fmt.Println("\n=== Session summary ===")
	fmt.Printf("Total reviews this session: %d\n", len(reviewLog))
	fmt.Printf("Accuracy: %.1f%%\n", engram.Accuracy(reviewLog)*100)
	fmt.Printf("Streak: %d day(s)\n", engram.Streak(reviewLog, now, loc))
	fmt.Printf("New cards introduced today: %d\n", engram.CountNewIntroduced(reviewLog, now, loc))

	cards := make([]engram.CardState, 0, len(skills))
	for _, sk := range skills {
		cs, ok, err := store.Card(ctx, sk.ID)
		if err != nil {
			log.Fatalf("card: %v", err)
		}
		if !ok {
			cs = sched.NewCardState(now)
		}
		cards = append(cards, cs)
	}
	forecast := engram.DueForecast(cards, 7, now, loc)
	fmt.Println("Due forecast (next 7 days, day 0 = today):")
	for i, n := range forecast {
		fmt.Printf("  day %d: %d card(s) due\n", i, n)
	}
}

// buildQueueItems loads every skill's current card state from store,
// falling back to a fresh card via sched.NewCardState when the skill has
// never been reviewed.
func buildQueueItems(ctx context.Context, store *memstore.Store, sched *engram.Scheduler, skills []engram.Skill, now time.Time) ([]engram.QueueItem, error) {
	items := make([]engram.QueueItem, 0, len(skills))
	for _, sk := range skills {
		cs, ok, err := store.Card(ctx, sk.ID)
		if err != nil {
			return nil, err
		}
		if !ok {
			cs = sched.NewCardState(now)
		}
		items = append(items, engram.QueueItem{Skill: sk, Card: cs})
	}
	return items, nil
}

// earliestFutureDue returns the earliest Due time among items that lies
// strictly after now, so the demo can fast-forward its clock when nothing is
// studyable yet (e.g. the new-card cap was reached for the day).
func earliestFutureDue(items []engram.QueueItem, now time.Time) (time.Time, bool) {
	var best time.Time
	found := false
	for _, it := range items {
		if it.Card.Due.After(now) && (!found || it.Card.Due.Before(best)) {
			best = it.Card.Due
			found = true
		}
	}
	return best, found
}

// pickWrongKey deterministically picks one incorrect option's key using rng,
// simulating a wrong answer.
func pickWrongKey(rng *rand.Rand, options []quiz.Option, correctKey string) string {
	candidates := make([]string, 0, len(options))
	for _, o := range options {
		if o.Key != correctKey {
			candidates = append(candidates, o.Key)
		}
	}
	if len(candidates) == 0 {
		return correctKey
	}
	return candidates[rng.Intn(len(candidates))]
}

func ratingName(r engram.Rating) string {
	switch r {
	case engram.Again:
		return "Again"
	case engram.Hard:
		return "Hard"
	case engram.Good:
		return "Good"
	case engram.Easy:
		return "Easy"
	default:
		return "Unknown"
	}
}

func stateName(s engram.State) string {
	switch s {
	case engram.StateNew:
		return "New"
	case engram.StateLearning:
		return "Learning"
	case engram.StateReview:
		return "Review"
	case engram.StateRelearning:
		return "Relearning"
	default:
		return "Unknown"
	}
}
