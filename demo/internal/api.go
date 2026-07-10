package demo

import (
	"encoding/json"
	"net/http"
)

// API serves the marketing demo HTTP endpoints.
type API struct {
	Engine *Engine
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (a *API) Start(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req StartRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	resp, err := a.Engine.Start(r.Context(), req)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) Next(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req NextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SessionID == "" {
		writeErr(w, http.StatusBadRequest, "session_id required")
		return
	}
	resp, err := a.Engine.Next(r.Context(), req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) Probe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req ProbeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SessionID == "" {
		writeErr(w, http.StatusBadRequest, "session_id required")
		return
	}
	resp, err := a.Engine.Probe(r.Context(), req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) Reset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var req ResetRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	_ = a.Engine.Reset(req)
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

// Mount registers demo API routes on mux.
func (a *API) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/api/demo/start", a.Start)
	mux.HandleFunc("/api/demo/next", a.Next)
	mux.HandleFunc("/api/demo/probe", a.Probe)
	mux.HandleFunc("/api/demo/reset", a.Reset)
}
