// Package warmtransfer coordinates a warm transfer between the HTTP handler that
// receives Twilio conference status callbacks and the long-lived goroutine that is
// driving the OpenAI realtime session for the same call.
//
// The realtime session registers a watch on the conference it is sitting in before it
// creates the transfer task. When the human agent's leg joins that conference, Twilio
// posts a participant-join callback and the handler signals the watch, which is how the
// assistant learns it is time to brief the agent and drop off.
package warmtransfer

import "sync"

type Broker struct {
	mu      sync.Mutex
	watches map[string]chan string
}

func NewBroker() *Broker {
	return &Broker{watches: make(map[string]chan string)}
}

// Watch registers interest in the next participant to join conferenceName and returns
// the channel that receives that participant's call sid. Release must be called when the
// caller is done, whether or not the join ever arrived.
func (b *Broker) Watch(conferenceName string) <-chan string {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Buffered so a join that lands before the watcher selects is not dropped, and so
	// Signal never blocks the HTTP handler.
	joined := make(chan string, 1)
	b.watches[conferenceName] = joined

	return joined
}

// Signal reports that participantCallSid joined conferenceName. It is a no-op when
// nothing is watching, which is the common case: most conference callbacks are for calls
// that are not being transferred.
func (b *Broker) Signal(conferenceName string, participantCallSid string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	joined, ok := b.watches[conferenceName]
	if !ok {
		return
	}

	select {
	case joined <- participantCallSid:
	default: // a join is already queued; the first one is the agent we care about
	}
}

func (b *Broker) Release(conferenceName string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.watches, conferenceName)
}
