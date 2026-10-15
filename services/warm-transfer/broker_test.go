package warmtransfer

import (
	"sync"
	"testing"
	"time"
)

func TestSignalDeliversJoinToWatcher(t *testing.T) {
	broker := NewBroker()

	joined := broker.Watch("WT123")
	defer broker.Release("WT123")

	broker.Signal("WT123", "CA456")

	select {
	case callSid := <-joined:
		if callSid != "CA456" {
			t.Fatalf("expected participant call sid 'CA456', got %q", callSid)
		}
	case <-time.After(time.Second):
		t.Fatal("expected a join signal, got none")
	}
}

// The agent's leg can join before the transfer task creation call even returns, so a
// signal that arrives before the watcher reads must still be delivered.
func TestSignalBeforeReceiveIsNotDropped(t *testing.T) {
	broker := NewBroker()

	joined := broker.Watch("WT123")
	defer broker.Release("WT123")

	broker.Signal("WT123", "CA456")
	broker.Signal("WT123", "CA789") // a second participant must not displace the first

	select {
	case callSid := <-joined:
		if callSid != "CA456" {
			t.Fatalf("expected the first join 'CA456', got %q", callSid)
		}
	case <-time.After(time.Second):
		t.Fatal("expected a join signal, got none")
	}
}

// Most conference callbacks are for calls nobody is transferring.
func TestSignalWithoutWatcherIsNoop(t *testing.T) {
	broker := NewBroker()

	broker.Signal("WT-nobody-is-watching", "CA456")
}

func TestReleaseStopsDelivery(t *testing.T) {
	broker := NewBroker()

	joined := broker.Watch("WT123")
	broker.Release("WT123")

	broker.Signal("WT123", "CA456")

	select {
	case callSid := <-joined:
		t.Fatalf("expected no signal after release, got %q", callSid)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestWatchesAreIsolatedPerConference(t *testing.T) {
	broker := NewBroker()

	first := broker.Watch("WT111")
	second := broker.Watch("WT222")
	defer broker.Release("WT111")
	defer broker.Release("WT222")

	broker.Signal("WT222", "CA222")

	select {
	case callSid := <-second:
		if callSid != "CA222" {
			t.Fatalf("expected 'CA222', got %q", callSid)
		}
	case <-time.After(time.Second):
		t.Fatal("expected the second conference to receive its join")
	}

	select {
	case callSid := <-first:
		t.Fatalf("expected no signal on the other conference, got %q", callSid)
	default:
	}
}

// Signal runs on Twilio's webhook goroutines while Watch/Release run on call goroutines.
// Run with -race to make this meaningful.
func TestConcurrentWatchSignalRelease(t *testing.T) {
	broker := NewBroker()

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			name := "WT-concurrent"

			joined := broker.Watch(name)
			broker.Signal(name, "CA1")

			select {
			case <-joined:
			case <-time.After(time.Second):
			}

			broker.Release(name)
		}()
	}

	wg.Wait()
}
