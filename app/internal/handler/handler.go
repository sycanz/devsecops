package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"employee-api/internal/crypto"
	"employee-api/internal/store"
)

type Handler struct {
	store   store.Store
	crypter crypto.Crypter
}

func New(s store.Store, c crypto.Crypter) *Handler {
	return &Handler{store: s, crypter: c}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}
