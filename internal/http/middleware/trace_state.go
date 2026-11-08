package middleware

import (
	"context"
	"net/http"
)

type traceStateKey struct{}

type traceState struct {
	ctx context.Context
}

func attachTraceState(r *http.Request) {
	*r = *r.WithContext(context.WithValue(r.Context(), traceStateKey{}, &traceState{}))
}

func tracedContext(r *http.Request) context.Context {
	state, ok := r.Context().Value(traceStateKey{}).(*traceState)
	if ok && state.ctx != nil {
		return state.ctx
	}
	return r.Context()
}
