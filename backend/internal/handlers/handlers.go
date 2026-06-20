package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/api"
	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/Ksalgotra1/Marshal/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handlers bundles all HTTP handler dependencies.
type Handlers struct {
	Pool      *pgxpool.Pool
	ServerCtx context.Context
}

// HandleCreateRequest handles POST /api/requests.
// Validates the payload, persists the request (H3 cells computed in the store),
// then triggers a synchronous grouper run so the caller sees near-instant grouping.
func (h *Handlers) HandleCreateRequest(w http.ResponseWriter, r *http.Request) {
	var payload models.CreateRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := validatePayload(&payload); err != nil {
		api.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := h.Pool.Begin(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback(r.Context())

	reqStore := &store.RequestStore{DB: tx}
	id, err := reqStore.Create(r.Context(), &payload)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "failed to create request")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		api.WriteError(w, http.StatusInternalServerError, "commit failed")
		return
	}

	// Enqueue a grouper job and wake the worker via Postgres NOTIFY
	js := &store.JobStore{DB: h.Pool}
	js.Enqueue(r.Context(), "group_pending", struct{}{}, time.Now())
	worker.Notify(r.Context(), h.Pool, "grouper_wakeup")

	api.WriteJSON(w, http.StatusCreated, api.JSON{
		"id":     id,
		"status": "pending",
	})
}

// HandleGetRequest handles GET /api/requests/{id}.
func (h *Handlers) HandleGetRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s := &store.RequestStore{DB: h.Pool}
	req, err := s.GetByID(r.Context(), id)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "request not found")
		return
	}
	api.WriteJSON(w, http.StatusOK, req)
}

// HandleListGroups handles GET /api/groups with optional ?status= and ?limit= filters.
func (h *Handlers) HandleListGroups(w http.ResponseWriter, r *http.Request) {
	filter := store.GroupFilter{
		Status: r.URL.Query().Get("status"),
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		filter.Limit, _ = strconv.Atoi(l)
	}

	s := &store.GroupStore{DB: h.Pool}
	groups, err := s.ListFiltered(r.Context(), filter)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "failed to list groups")
		return
	}
	api.WriteJSON(w, http.StatusOK, groups)
}

// HandleGetGroup handles GET /api/groups/{id} — returns group + its member details.
func (h *Handlers) HandleGetGroup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s := &store.GroupStore{DB: h.Pool}
	detail, err := s.GetByIDWithMembers(r.Context(), id)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "group not found")
		return
	}
	api.WriteJSON(w, http.StatusOK, detail)
}

// HandleListOpenGroups handles GET /api/groups/open (student browse).
func (h *Handlers) HandleListOpenGroups(w http.ResponseWriter, r *http.Request) {
	s := &store.GroupStore{DB: h.Pool}
	groups, err := s.ListOpen(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "failed to list open groups")
		return
	}
	api.WriteJSON(w, http.StatusOK, groups)
}

// HandleJoinGroup handles POST /api/groups/{id}/join.
// A student manually joins an open group.
func (h *Handlers) HandleJoinGroup(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("id")

	var body struct {
		RequestID string `json:"request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RequestID == "" {
		api.WriteError(w, http.StatusBadRequest, "request_id is required")
		return
	}

	tx, err := h.Pool.Begin(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback(r.Context())

	gs := &store.GroupStore{DB: tx}
	if err := gs.AddMember(r.Context(), groupID, body.RequestID); err != nil {
		api.WriteError(w, http.StatusInternalServerError, "failed to join group")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		api.WriteError(w, http.StatusInternalServerError, "commit failed")
		return
	}

	api.WriteJSON(w, http.StatusOK, api.JSON{"joined": true, "group_id": groupID})
}

// validatePayload checks the create request payload for basic sanity.
func validatePayload(p *models.CreateRequestPayload) error {
	if p.RequesterName == "" {
		return fmt.Errorf("requester_name is required")
	}
	if p.PickupLat < -90 || p.PickupLat > 90 {
		return fmt.Errorf("pickup_lat out of range")
	}
	if p.PickupLng < -180 || p.PickupLng > 180 {
		return fmt.Errorf("pickup_lng out of range")
	}
	if p.DropoffLat < -90 || p.DropoffLat > 90 {
		return fmt.Errorf("dropoff_lat out of range")
	}
	if p.DropoffLng < -180 || p.DropoffLng > 180 {
		return fmt.Errorf("dropoff_lng out of range")
	}
	if p.PickupLat == p.DropoffLat && p.PickupLng == p.DropoffLng {
		return fmt.Errorf("pickup and dropoff cannot be the same location")
	}
	if p.ArriveBy.Before(time.Now().Add(10 * time.Minute)) {
		return fmt.Errorf("arrive_by must be at least 10 minutes from now")
	}
	return nil
}
