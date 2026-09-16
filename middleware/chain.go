package middleware

import "net/http"

// Chain applies middleware to a handler in sequence, with RequestID always
// applied first (outermost). Middleware are applied in the order given,
// after RequestID.
//
// Example:
//   Chain(handler, auth, logging)
// produces:
//   RequestID(auth(logging(handler)))
// which executes as: RequestID → auth → logging → handler
func Chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	// Apply user middleware in reverse order (innermost to outermost)
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	// Always apply RequestID outermost
	h = RequestID(h)
	return h
}
