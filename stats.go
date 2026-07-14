package engram

import "time"

// dayNumber returns the calendar-day number of t in loc (nil defaults to
// time.UTC): t's local year/month/day is rebuilt as a UTC midnight and
// expressed as whole days since the Unix epoch. Differencing two dayNumbers
// gives the exact number of calendar days between them regardless of loc or
// DST, which raw 24h-duration arithmetic cannot guarantee. Shared by
// CountNewIntroduced (queue.go), Streak, and DueForecast.
func dayNumber(t time.Time, loc *time.Location) int64 {
	if loc == nil {
		loc = time.UTC
	}
	lt := t.In(loc)
	y, m, d := lt.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / 86400
}

// Accuracy is the share of reviews in log with Rating != Again (Hard, Good,
// and Easy all count as correct). Returns 0 for an empty log.
func Accuracy(log []Review) float64 {
	if len(log) == 0 {
		return 0
	}
	correct := 0
	for _, r := range log {
		if r.Rating != Again {
			correct++
		}
	}
	return float64(correct) / float64(len(log))
}

// Streak returns the number of consecutive calendar days (in loc, nil
// defaulting to time.UTC) with at least one review, counting back from now's
// day. If now's day itself has no review yet, the count still starts from
// yesterday instead of resetting to 0 — a day that "just hasn't been
// studied yet" doesn't erase an otherwise-current streak. Returns 0 if log
// is empty or neither today nor yesterday (relative to now) has a review.
func Streak(log []Review, now time.Time, loc *time.Location) int {
	if loc == nil {
		loc = time.UTC
	}
	if len(log) == 0 {
		return 0
	}

	days := make(map[int64]bool, len(log))
	for _, r := range log {
		days[dayNumber(r.ReviewedAt, loc)] = true
	}

	today := dayNumber(now, loc)
	cursor := today
	if !days[today] {
		cursor = today - 1
	}

	streak := 0
	for days[cursor] {
		streak++
		cursor--
	}
	return streak
}

// DueForecast buckets cards by the calendar day (in loc, nil defaulting to
// time.UTC) their Due falls on, relative to now's day. Bucket 0 is "due
// today or earlier" — overdue cards clamp into today's bucket — bucket i
// (i>0) is now's day + i. Cards due beyond now's day + days - 1 are not
// counted. Returns a slice of length days, or an empty non-nil slice if
// days <= 0.
func DueForecast(cards []CardState, days int, now time.Time, loc *time.Location) []int {
	if days <= 0 {
		return []int{}
	}
	if loc == nil {
		loc = time.UTC
	}

	buckets := make([]int, days)
	today := dayNumber(now, loc)
	for _, c := range cards {
		idx := int(dayNumber(c.Due, loc) - today)
		if idx < 0 {
			idx = 0
		}
		if idx >= days {
			continue
		}
		buckets[idx]++
	}
	return buckets
}
