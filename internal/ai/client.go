package ai

import "net/http"

// httpClient is a shared HTTP client for all AI provider calls.
// No client-level timeout — we rely on per-request context deadlines
// (passed via http.NewRequestWithContext) so streaming SSE responses
// aren't killed mid-response by a fixed wall-clock timeout.
// Callers that need a deadline should use context.WithTimeout.
var httpClient = &http.Client{}
