package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/devices", s.handleRegister)
	mux.HandleFunc("GET /api/v1/devices/{id}/shadow", s.handleGetShadow)
	mux.HandleFunc("PUT /api/v1/devices/{id}/desired", s.handleDesired)
	mux.HandleFunc("POST /api/v1/devices/{id}/move", s.handleMove)
	mux.HandleFunc("POST /api/v1/devices/{id}/commands", s.handleCommand)
	mux.HandleFunc("GET /api/v1/devices/{id}/history", s.handleHistory)
	mux.HandleFunc("GET /api/v1/devices/{id}/diff", s.handleDiff)
	mux.HandleFunc("POST /api/v1/devices/{id}/telemetry", s.handleTelemetry)
	mux.HandleFunc("POST /api/v1/devices/batch/desired", s.handleBatchDesired)
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /healthz/ready", s.handleReady)
	mux.HandleFunc("GET /healthz/history", s.handleHealthHistory)
	mux.HandleFunc("GET /api/v1/stats", s.handleStats)
}

func (s *Server) Serve(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	s.Routes(mux)
	srv := &http.Server{Addr: addr, Handler: withLogging(mux)}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	return srv.ListenAndServe()
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID       string   `json:"id"`
		Group    string   `json:"group"`
		Template string   `json:"template"`
		Caps     []string `json:"capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	d, err := s.RegisterDevice(in.ID, in.Group, in.Template, in.Caps)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (s *Server) handleGetShadow(w http.ResponseWriter, r *http.Request) {
	sh, err := s.GetShadow(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, sh)
}

func (s *Server) handleDesired(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Version int64             `json:"version"`
		Patch   map[string]string `json:"patch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	sh, err := s.UpdateDesired(r.Context(), r.PathValue("id"), in.Version, in.Patch)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, sh)
}

func (s *Server) handleMove(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Group string `json:"group"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	d, err := s.MoveDevice(r.PathValue("id"), in.Group)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Payload []byte `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	cmd, err := s.SendCommand(r.Context(), r.PathValue("id"), in.Payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, cmd)
}

func (s *Server) handleBatchDesired(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DeviceIDs []string          `json:"device_ids"`
		Version   int64             `json:"version"`
		Patch     map[string]string `json:"patch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	shadows, err := s.SubmitBatchDesired(r.Context(), in.DeviceIDs, in.Version, in.Patch)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, shadows)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	entries, err := s.History(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	diff, err := s.Diff(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"diff": diff})
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Seq    int64              `json:"seq"`
		Fields map[string]float64 `json:"fields"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.SubmitTelemetry(r.PathValue("id"), in.Seq, in.Fields); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if s.health == nil {
		writeJSON(w, http.StatusOK, s.Health())
		return
	}
	writeJSON(w, http.StatusOK, s.health.Check(r.Context()))
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if s.health == nil || !s.health.Ready(r.Context()) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleHealthHistory(w http.ResponseWriter, r *http.Request) {
	if s.health == nil {
		writeJSON(w, http.StatusOK, map[string]any{"recent": []any{}, "summary": map[string]any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"recent":  s.health.Recent(10),
		"summary": s.health.Summary(),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Stats())
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
