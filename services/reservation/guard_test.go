package reservation

import (
	"sync"
	"testing"
	"time"
)

func TestFirstClaimWinsAndLaterOnesStandDown(t *testing.T) {
	guard := NewGuard()

	if !guard.Claim("WR123") {
		t.Fatal("the first caller must win the claim")
	}

	// The assignment callback and the event stream both fire for one reservation, and the
	// event stream re-delivers. Only one of them may dial the assistant in.
	if guard.Claim("WR123") {
		t.Fatal("a second caller must stand down")
	}
}

func TestReleaseLetsALaterDeliveryRetry(t *testing.T) {
	guard := NewGuard()

	guard.Claim("WR123")
	guard.Release("WR123") // the work failed before anything was dialled

	if !guard.Claim("WR123") {
		t.Fatal("a released reservation must be claimable again")
	}
}

func TestClaimsAreIndependentPerReservation(t *testing.T) {
	guard := NewGuard()

	if !guard.Claim("WR111") || !guard.Claim("WR222") {
		t.Fatal("different reservations must not block each other")
	}
}

// Better to risk acting twice than to silently drop a call because an identifier was
// missing from the payload.
func TestAnEmptySidIsAlwaysClaimable(t *testing.T) {
	guard := NewGuard()

	if !guard.Claim("") || !guard.Claim("") {
		t.Fatal("an empty reservation sid must never block the work")
	}
}

func TestClaimsExpire(t *testing.T) {
	guard := NewGuard()

	now := time.Now()
	guard.nowFunc = func() time.Time { return now }

	guard.Claim("WR123")

	guard.nowFunc = func() time.Time { return now.Add(claimTTL + time.Minute) }

	if !guard.Claim("WR123") {
		t.Fatal("a claim older than the ttl must be swept")
	}

	if len(guard.claimed) != 1 {
		t.Fatalf("expected the swept map to hold only the new claim, got %d entries", len(guard.claimed))
	}
}

// The assignment callback and the event stream arrive on different goroutines.
func TestConcurrentClaimsElectExactlyOneWinner(t *testing.T) {
	guard := NewGuard()

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		winners int
	)

	for i := 0; i < 50; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if guard.Claim("WR-contended") {
				mu.Lock()
				winners++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if winners != 1 {
		t.Fatalf("expected exactly one winner, got %d", winners)
	}
}
