// Package reservation guards against acting on the same Task Router reservation twice.
//
// Task Router can present one reservation to this service more than once: a workflow's
// assignment callback and the workspace event stream both fire for it, and the event
// stream re-delivers when a handler does not answer 2xx. Connecting the assistant is not
// idempotent - it dials a new leg onto a live call - so the work is claimed once and later
// arrivals stand down.
package reservation

import (
	"sync"
	"time"
)

// How long a claim is remembered. Long enough to outlast Task Router's redelivery window,
// short enough that the map does not grow without bound on a long-running process.
const claimTTL = 30 * time.Minute

type Guard struct {
	mu      sync.Mutex
	claimed map[string]time.Time
	nowFunc func() time.Time
}

func NewGuard() *Guard {
	return &Guard{claimed: make(map[string]time.Time), nowFunc: time.Now}
}

// Claim reports whether the caller is the first to take this reservation. A caller that
// gets false should do nothing: someone else is already connecting the assistant.
func (g *Guard) Claim(reservationSid string) bool {
	if reservationSid == "" {
		return true // nothing to deduplicate on; better to act than to drop the call
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.nowFunc()
	g.sweep(now)

	if _, taken := g.claimed[reservationSid]; taken {
		return false
	}

	g.claimed[reservationSid] = now

	return true
}

// Release gives a reservation back so a later delivery can try again. Used when the work
// failed before the assistant was actually connected, so there is nothing to duplicate.
func (g *Guard) Release(reservationSid string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.claimed, reservationSid)
}

// sweep drops expired claims. Called under the lock on every claim, which is often enough
// for a map that only ever holds reservations seen in the last half hour.
func (g *Guard) sweep(now time.Time) {
	for sid, claimedAt := range g.claimed {
		if now.Sub(claimedAt) > claimTTL {
			delete(g.claimed, sid)
		}
	}
}
