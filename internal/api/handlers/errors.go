package handlers

import "github.com/Agora-Calyce/spx-tf-api/pkg/client/spx"

// spxStatus renvoie le statut à retourner au client pour une erreur de l'API SPX :
// les erreurs 4xx de SPX (401, 403, 400…) sont transmises telles quelles,
// les autres erreurs (5xx, timeout, réseau) utilisent le statut fallback.
func spxStatus(err error, fallback int) int {
	if code := spx.StatusCode(err); code >= 400 && code < 500 {
		return code
	}
	return fallback
}
