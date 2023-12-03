package sifters

import (
	"fmt"
	"time"

	evsifter "github.com/jiftechnify/strfry-evsifter"
	"github.com/jiftechnify/strfry-evsifter/sifters/internal/utils"
	"github.com/nbd-wtf/go-nostr"
)

func MatchesFilters(filters []nostr.Filter, mode Mode) *sifterUnit {
	matchInput := func(input *evsifter.Input) (inputMatchResult, error) {
		return matchResultFromBool(nostr.Filters(filters).Match(input.Event)), nil
	}
	defaultRejFn := rejectWithMsgPerMode(
		mode,
		"blocked: event must match filters to be accepted",
		"blocked: event is denied by filters",
	)
	return newSifterUnit(matchInput, mode, defaultRejFn)
}

func AuthorMatcher(matcher func(string) bool, mode Mode) *sifterUnit {
	matchInput := func(input *evsifter.Input) (inputMatchResult, error) {
		return matchResultFromBool(matcher(input.Event.PubKey)), nil
	}
	defaultRejFn := rejectWithMsgPerMode(
		mode,
		"blocked: event author is not in the whitelist",
		"blocked: event author is in the blacklist",
	)
	return newSifterUnit(matchInput, mode, defaultRejFn)
}

func AuthorList(authors []string, mode Mode) *sifterUnit {
	authorSet := utils.SliceToSet(authors)
	matchInput := func(input *evsifter.Input) (inputMatchResult, error) {
		_, ok := authorSet[input.Event.PubKey]
		return matchResultFromBool(ok), nil
	}
	defaultRejFn := rejectWithMsgPerMode(
		mode,
		"blocked: event author is not in the whitelist",
		"blocked: event author is in the blacklist",
	)
	return newSifterUnit(matchInput, mode, defaultRejFn)
}

var (
	// Regular events: kind < 1000 (excluding 0, 3, 41)
	KindsAllRegular = func(k int) bool {
		return k == 1 || k == 2 || (3 < k && k < 41) || (41 < k && k < 10000)
	}
	// Replaceable events: kind 0, 3, 41 or 10000 <= kind < 20000
	KindsAllReplaceable = func(k int) bool {
		return k == 0 || k == 3 || k == 41 || (10000 <= k && k < 20000)
	}
	KindsAllEphemeral = func(k int) bool {
		return 20000 <= k && k < 30000
	}
	KindsAllParameterizedReplaceable = func(k int) bool {
		return 30000 <= k && k < 40000
	}
)

func KindMatcher(matcher func(int) bool, mode Mode) *sifterUnit {
	matchInput := func(input *evsifter.Input) (inputMatchResult, error) {
		return matchResultFromBool(matcher(input.Event.Kind)), nil
	}
	defaultRejFn := rejectWithMsgPerMode(
		mode,
		"blocked: the kind of the event is not in the whitelist",
		"blocked: the kind of the event is in the blacklist",
	)
	return newSifterUnit(matchInput, mode, defaultRejFn)
}

func KindList(kinds []int, mode Mode) *sifterUnit {
	kindSet := utils.SliceToSet(kinds)
	matchInput := func(input *evsifter.Input) (inputMatchResult, error) {
		_, ok := kindSet[input.Event.Kind]
		return matchResultFromBool(ok), nil
	}
	defaultRejFn := rejectWithMsgPerMode(
		mode,
		"blocked: the kind of the event is not in the whitelist",
		"blocked: the kind of the event is in the blacklist",
	)
	return newSifterUnit(matchInput, mode, defaultRejFn)
}

type fakeableClock struct {
	fakeNow time.Time
}

var (
	clock fakeableClock
)

func (c fakeableClock) now() time.Time {
	if c.fakeNow.IsZero() {
		return time.Now()
	}
	return c.fakeNow
}

func (c *fakeableClock) setFake(t time.Time) {
	c.fakeNow = t
}

func (c *fakeableClock) reset() {
	c.fakeNow = time.Time{}
}

type RelativeTimeRange struct {
	maxPastDelta   time.Duration
	maxFutureDelta time.Duration
}

func (r RelativeTimeRange) Contains(t time.Time) bool {
	now := clock.now()

	okPast := r.maxPastDelta == 0 || !t.Before(now.Add(-r.maxPastDelta))
	okFuture := r.maxFutureDelta == 0 || !t.After(now.Add(r.maxFutureDelta))

	return okPast && okFuture
}

func (r RelativeTimeRange) String() string {
	left := "-∞"
	if r.maxPastDelta != 0 {
		left = r.maxFutureDelta.String() + " ago"
	}
	right := "+∞"
	if r.maxFutureDelta != 0 {
		right = r.maxFutureDelta.String() + " after"
	}
	return fmt.Sprintf("[%s, %s]", left, right)
}

func CreatedAtRange(timeRange RelativeTimeRange, mode Mode) *sifterUnit {
	matchInput := func(input *evsifter.Input) (inputMatchResult, error) {
		createdAt := input.Event.CreatedAt.Time()
		return matchResultFromBool(timeRange.Contains(createdAt)), nil
	}
	defaultRejFn := rejectWithMsgPerMode(
		mode,
		fmt.Sprintf("invalid: event timestamp is out of the range: %v", timeRange),
		fmt.Sprintf("blocked: event timestamp must be out of the range: %v", timeRange),
	)
	return newSifterUnit(matchInput, mode, defaultRejFn)
}
