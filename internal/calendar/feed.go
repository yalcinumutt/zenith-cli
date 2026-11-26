package calendar

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	feedTTL     = 5 * time.Minute
	feedTimeout = 10 * time.Second
	maxFeedSize = 8 << 20 // a year of busy calendar is well under this
)

// Feed fetches and caches one ICS URL. A failed refresh keeps serving the last
// good copy, so a flaky network never empties the planner.
type Feed struct {
	mu      sync.Mutex
	url     string
	cal     *Calendar
	fetched time.Time
	lastErr error
	client  *http.Client
	now     func() time.Time
	loc     *time.Location // zone for floating (zone-less) event times
}

// NewFeed builds a Feed. A nil client uses a default with a short timeout.
func NewFeed(client *http.Client) *Feed {
	if client == nil {
		client = &http.Client{Timeout: feedTimeout}
	}
	return &Feed{client: client, now: time.Now, loc: time.Local}
}

// Normalize turns what a user pastes into a fetchable URL: webcal:// becomes
// https://, and only http(s) is accepted.
func Normalize(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if strings.HasPrefix(strings.ToLower(raw), "webcal://") {
		raw = "https://" + raw[len("webcal://"):]
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return "", errors.New("the calendar link must be an http(s) or webcal address")
	}
	return u.String(), nil
}

// SetURL points the feed somewhere else and drops the cache. An empty URL disables it.
func (f *Feed) SetURL(raw string) error {
	u, err := Normalize(raw)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if u != f.url {
		f.url, f.cal, f.fetched, f.lastErr = u, nil, time.Time{}, nil
	}
	return nil
}

// SetLocation sets the zone that zone-less event times are read in; it should
// match the zone the planner shows days in. The cache is dropped.
func (f *Feed) SetLocation(loc *time.Location) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loc, f.cal, f.fetched = loc, nil, time.Time{}
}

// Configured reports whether a feed URL is set.
func (f *Feed) Configured() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.url != ""
}

// Status is the last refresh error, or nil when the feed is fine or unset.
func (f *Feed) Status() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastErr
}

// Refresh fetches now, ignoring the cache age.
func (f *Feed) Refresh(ctx context.Context) error {
	f.mu.Lock()
	u := f.url
	f.mu.Unlock()
	if u == "" {
		return nil
	}
	cal, err := f.fetch(ctx, u)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.url != u { // reconfigured while fetching
		return nil
	}
	f.lastErr = err
	if err == nil {
		f.cal, f.fetched = cal, f.now()
	}
	return err
}

// Busy returns the busy events on a day in the given zone, refreshing the feed
// when its cache is stale. An error means the data may be old or missing; the
// events returned are still the best known.
func (f *Feed) Busy(ctx context.Context, day string, local *time.Location) ([]Event, error) {
	f.mu.Lock()
	configured := f.url != ""
	stale := f.cal == nil || f.now().Sub(f.fetched) > feedTTL
	f.mu.Unlock()
	if !configured {
		return nil, nil
	}
	var err error
	if stale {
		err = f.Refresh(ctx)
	}
	f.mu.Lock()
	cal := f.cal
	f.mu.Unlock()
	if cal == nil {
		return nil, err
	}
	return cal.On(day, local), err
}

func (f *Feed) fetch(ctx context.Context, u string) (*Calendar, error) {
	ctx, cancel := context.WithTimeout(ctx, feedTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/calendar, */*;q=0.5")
	resp, err := f.client.Do(req)
	if err != nil {
		// url.Error text embeds the (secret) feed URL; report only the cause.
		cause := errors.Unwrap(err)
		if cause == nil {
			cause = errors.New("network error")
		}
		return nil, fmt.Errorf("could not reach the calendar: %w", cause)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the calendar link returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedSize+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxFeedSize {
		return nil, errors.New("the calendar feed is too large")
	}
	text := string(body)
	if !strings.Contains(text, "BEGIN:VCALENDAR") {
		return nil, errors.New("that link is not an iCalendar feed")
	}
	f.mu.Lock()
	loc := f.loc
	f.mu.Unlock()
	return Parse(text, loc), nil
}
