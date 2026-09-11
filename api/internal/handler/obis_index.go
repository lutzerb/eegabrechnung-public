package handler

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	edaxml "github.com/lutzerb/eegabrechnung/internal/eda/xml"
	"github.com/lutzerb/eegabrechnung/internal/repository"
)

// obisIndexCacheTTL bounds how long a per-(eeg,zaehlpunkt) OBIS index is reused before
// being rebuilt from the DB — long enough to make paging/filtering through the
// "Messwerte-Debug" view cheap, short enough that a reimport (e.g. after fixing a
// transition-date bug and reprocessing a message) shows up without a service restart.
const obisIndexCacheTTL = 5 * time.Minute

// obisIndexMessageLimit caps how many raw DATEN_CRMSG messages are parsed per
// Zählpunkt. A debug tool does not need perfect completeness over a Zählpunkt's
// entire multi-year history — the most recent messages cover what an admin is
// realistically debugging, and capping keeps a cold-cache request bounded.
const obisIndexMessageLimit = 300

// OBISEntry is one OBIS-code value for a single 15-minute slot, as reported by one
// raw EDA message. Multiple entries for the same slot+code can occur when a
// Zählpunkt was reported more than once (e.g. a correction) — all are kept so the
// admin can see the history, not just the latest value.
type OBISEntry struct {
	MeterCode        string    `json:"meter_code"`
	Value            float64   `json:"value"`
	Quality          string    `json:"quality"`
	MessageID        uuid.UUID `json:"message_id"`
	MessageCreatedAt time.Time `json:"message_created_at"`
}

// obisIndexCacheEntry keys its index by Unix second rather than time.Time — Go's
// time.Time equality (used implicitly for map keys) also compares the Location
// pointer and any monotonic reading, so two time.Time values representing the exact
// same instant (one from pgx, one from edaxml.ParseCRMsg's .UTC()) can silently fail
// to match as map keys even though they are the same reading. Unix() sidesteps that.
type obisIndexCacheEntry struct {
	index   map[int64][]OBISEntry
	builtAt time.Time
}

// obisIndexCache is a small in-process cache mapping (eegID, zaehlpunkt) to a parsed
// OBIS index, avoiding re-parsing raw EDA XML on every request of the "Messwerte-Debug"
// view. Not shared across API replicas — acceptable for an admin debug tool.
type obisIndexCache struct {
	mu      sync.Mutex
	entries map[string]obisIndexCacheEntry
}

func newOBISIndexCache() *obisIndexCache {
	return &obisIndexCache{entries: map[string]obisIndexCacheEntry{}}
}

// get returns the OBIS index for one Zählpunkt within one EEG, rebuilding it from the
// DB when missing or stale. The eegID is part of the cache key so a recycled Zählpunkt
// (migration 070) never leaks another EEG's cached data.
func (c *obisIndexCache) get(ctx context.Context, edaMsgRepo *repository.EDAMessageRepository, eegID uuid.UUID, zaehlpunkt string) (map[int64][]OBISEntry, error) {
	key := eegID.String() + "|" + zaehlpunkt

	c.mu.Lock()
	entry, ok := c.entries[key]
	c.mu.Unlock()
	if ok && time.Since(entry.builtAt) < obisIndexCacheTTL {
		return entry.index, nil
	}

	messages, err := edaMsgRepo.ListCRMsgPayloadsByZaehlpunkt(ctx, eegID, zaehlpunkt, obisIndexMessageLimit)
	if err != nil {
		return nil, err
	}

	index := map[int64][]OBISEntry{}
	for _, msg := range messages {
		record, err := edaxml.ParseCRMsg(msg.XMLPayload)
		if err != nil {
			// Malformed/unrelated payload — skip it rather than fail the whole view.
			continue
		}
		for _, block := range record.Energies {
			for _, ed := range block.Data {
				for _, pos := range ed.Positions {
					key := pos.From.Unix()
					index[key] = append(index[key], OBISEntry{
						MeterCode:        ed.MeterCode,
						Value:            pos.Value,
						Quality:          pos.Quality,
						MessageID:        msg.ID,
						MessageCreatedAt: msg.CreatedAt,
					})
				}
			}
		}
	}

	c.mu.Lock()
	c.entries[key] = obisIndexCacheEntry{index: index, builtAt: time.Now()}
	c.mu.Unlock()

	return index, nil
}
