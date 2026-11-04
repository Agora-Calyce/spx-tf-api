package handlers

import "github.com/Agora-Calyce/spx-tf-api/pkg/client/spx"

// spxStatus returns the status to send back to the client for an SPX API error:
// 4xx errors from SPX (401, 403, 400…) are passed through as is,
// other errors (5xx, timeout, network) use the fallback status.
func spxStatus(err error, fallback int) int {
	if code := spx.StatusCode(err); code >= 400 && code < 500 {
		return code
	}
	return fallback
}
