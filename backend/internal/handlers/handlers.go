package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/api"
	"github.com/Ksalgotra1/Marshal/internal/dispatch"
	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/Ksalgotra1/Marshal/internal/realtime"
	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/Ksalgotra1/Marshal/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handlers bundles all HTTP handler dependencies.
type Handlers struct {
	Pool          *pgxpool.Pool
	ServerCtx     context.Context
	Events        EventPublisher
	WebSocket     WebSocketUpgrader
	MessageSender MessageSender
}

type EventPublisher interface {
	BroadcastMulti([]string, realtime.Event)
}

type WebSocketUpgrader interface {
	HandleUpgrade(http.ResponseWriter, *http.Request)
}

type MessageSender interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
}

// HandleCreateRequest handles POST /api/requests.
// Validates the payload, persists the request (H3 cells computed in the store),
// then triggers a synchronous grouper run so the caller sees near-instant grouping.
func (h *Handlers) HandleCreateRequest(w http.ResponseWriter, r *http.Request) {
	var payload models.CreateRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		api.WriteRequestError(w, r, http.StatusBadRequest, "invalid JSON", err)
		return
	}
	if err := validatePayload(&payload); err != nil {
		api.WriteRequestError(w, r, http.StatusBadRequest, err.Error(), nil)
		return
	}

	tx, err := h.Pool.Begin(r.Context())
	if err != nil {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "database unavailable", err)
		return
	}
	defer tx.Rollback(r.Context())

	reqStore := &store.RequestStore{DB: tx}
	id, err := reqStore.Create(r.Context(), &payload)
	if err != nil {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "failed to create request", err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "failed to create request", err)
		return
	}

	if h.Events != nil {
		h.Events.BroadcastMulti([]string{"global"}, realtime.RequestCreated(id, payload.RequesterName))
	}

	// Enqueue a grouper job and wake the worker via Postgres NOTIFY
	js := &store.JobStore{DB: h.Pool}
	js.Enqueue(r.Context(), "group_pending", struct{}{}, models.PriorityNormal, time.Now())
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
		api.WriteRequestError(w, r, http.StatusNotFound, "request not found", nil, "lookup_id", id)
		return
	}
	api.WriteJSON(w, http.StatusOK, req)
}

// HandleListRequests handles GET /api/requests with optional ?status= and ?limit= filters.
func (h *Handlers) HandleListRequests(w http.ResponseWriter, r *http.Request) {
	filter := store.RequestFilter{
		Status: r.URL.Query().Get("status"),
		Offset: store.ParseOffset(r.URL.Query().Get("offset")),
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		filter.Limit, _ = strconv.Atoi(l)
	}

	s := &store.RequestStore{DB: h.Pool}
	requests, err := s.ListFiltered(r.Context(), filter)
	if err != nil {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "failed to list requests", err)
		return
	}
	api.WriteJSON(w, http.StatusOK, requests)
}

// HandleListGroups handles GET /api/groups with optional ?status= and ?limit= filters.
func (h *Handlers) HandleListGroups(w http.ResponseWriter, r *http.Request) {
	filter := store.GroupFilter{
		Status: r.URL.Query().Get("status"),
		Offset: store.ParseOffset(r.URL.Query().Get("offset")),
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		filter.Limit, _ = strconv.Atoi(l)
	}

	s := &store.GroupStore{DB: h.Pool}
	groups, err := s.ListFiltered(r.Context(), filter)
	if err != nil {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "failed to list groups", err)
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
		api.WriteRequestError(w, r, http.StatusNotFound, "group not found", nil, "lookup_id", id)
		return
	}
	api.WriteJSON(w, http.StatusOK, detail)
}

// HandleListOpenGroups handles GET /api/groups/open (student browse).
func (h *Handlers) HandleListOpenGroups(w http.ResponseWriter, r *http.Request) {
	s := &store.GroupStore{DB: h.Pool}
	groups, err := s.ListOpen(r.Context())
	if err != nil {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "failed to list open groups", err)
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
		api.WriteRequestError(w, r, http.StatusBadRequest, "request_id is required", err)
		return
	}

	tx, err := h.Pool.Begin(r.Context())
	if err != nil {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "database unavailable", err)
		return
	}
	defer tx.Rollback(r.Context())

	gs := &store.GroupStore{DB: tx}

	// Enforce capacity limit of 4
	ids, err := gs.GetMemberRequestIDs(r.Context(), groupID)
	if err != nil {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "failed to check group capacity", err)
		return
	}
	if len(ids) >= 4 {
		api.WriteRequestError(w, r, http.StatusBadRequest, "group is full", nil)
		return
	}

	if err := gs.AddMember(r.Context(), groupID, body.RequestID); err != nil {
		if errors.Is(err, store.ErrGroupNotJoinable) {
			api.WriteRequestError(w, r, http.StatusConflict, "group is no longer open", err, "group_id", groupID)
			return
		}
		api.WriteRequestError(w, r, http.StatusInternalServerError, "failed to join group", err, "group_id", groupID)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "failed to join group", err, "group_id", groupID)
		return
	}

	// Broadcast member:joined event
	if h.Events != nil {
		event := realtime.MemberJoined(groupID, body.RequestID)
		h.Events.BroadcastMulti([]string{"global", groupID}, event)
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

// ─── Chat Handlers ────────────────────────────────────────────────────────

// HandleListMessages handles GET /api/groups/{id}/messages.
func (h *Handlers) HandleListMessages(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("id")
	
	// Check if group exists to 404 properly instead of returning empty list
	gs := &store.GroupStore{DB: h.Pool}
	if _, err := gs.GetByID(r.Context(), groupID); err != nil {
		api.WriteRequestError(w, r, http.StatusNotFound, "group not found", nil, "lookup_id", groupID)
		return
	}

	cs := &store.ChatStore{DB: h.Pool}
	messages, err := cs.ListMessages(r.Context(), groupID)
	if err != nil {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "failed to list messages", err)
		return
	}

	// Make sure we return an empty array instead of null for empty history
	if messages == nil {
		messages = []models.ChatMessage{}
	}
	api.WriteJSON(w, http.StatusOK, messages)
}

// HandleCreateMessage handles POST /api/groups/{id}/messages.
func (h *Handlers) HandleCreateMessage(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("id")

	var body struct {
		Content   string `json:"content"`
		RequestID string `json:"request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Content == "" {
		api.WriteRequestError(w, r, http.StatusBadRequest, "content is required", err)
		return
	}

	// Verify group exists
	gs := &store.GroupStore{DB: h.Pool}
	group, err := gs.GetByID(r.Context(), groupID)
	if err != nil {
		api.WriteRequestError(w, r, http.StatusNotFound, "group not found", nil, "lookup_id", groupID)
		return
	}

	senderName := "Unknown"
	if body.RequestID != "" {
		rs := &store.RequestStore{DB: h.Pool}
		if req, err := rs.GetByID(r.Context(), body.RequestID); err == nil {
			senderName = strings.Split(req.RequesterName, " ")[0]
		}
	}

	cs := &store.ChatStore{DB: h.Pool}
	msg, err := cs.AddMessage(r.Context(), groupID, "student", senderName, body.Content)
	if err != nil {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "failed to create message", err)
		return
	}

	if h.Events != nil {
		h.Events.BroadcastMulti([]string{groupID}, realtime.ChatMessageEvent(*msg))
	}

	deliveredToDriver := false
	if group.DriverID != nil {
		ds := &store.DriverStore{DB: h.Pool}
		if driver, err := ds.GetByID(r.Context(), *group.DriverID); err == nil {
			if driver.TelegramChat != nil && h.MessageSender != nil {
				driverMsg := body.Content
				if body.RequestID != "" {
					rs := &store.RequestStore{DB: h.Pool}
					if req, err := rs.GetByID(r.Context(), body.RequestID); err == nil {
						driverMsg = fmt.Sprintf("👤 %s\n💬 %s", req.RequesterName, body.Content)
					}
				}
				
				err := h.MessageSender.SendMessage(r.Context(), *driver.TelegramChat, driverMsg)
				if err != nil {
					slog.Error("failed to send message to driver", "error", err, "driver_id", driver.ID)
				} else {
					deliveredToDriver = true
				}
			}
		}
	}

	api.WriteJSON(w, http.StatusOK, api.JSON{
		"message":             msg,
		"delivered_to_driver": deliveredToDriver,
	})
}

// ─── Driver Handlers ────────────────────────────────────────────────────────

// HandleRegisterDriver handles POST /api/drivers.
func (h *Handlers) HandleRegisterDriver(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name       string `json:"name"`
		TelegramID int64  `json:"telegram_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.TelegramID == 0 {
		api.WriteRequestError(w, r, http.StatusBadRequest, "name and telegram_id are required", err)
		return
	}

	ds := &store.DriverStore{DB: h.Pool}
	id, err := ds.Register(r.Context(), body.Name, body.TelegramID)
	if err != nil {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "failed to register driver", err)
		return
	}

	if h.Events != nil {
		h.Events.BroadcastMulti([]string{"global"}, realtime.DriverRegistered(id, body.Name))
	}

	api.WriteJSON(w, http.StatusCreated, api.JSON{"id": id, "name": body.Name})
}

// HandleListDrivers handles GET /api/drivers (admin view).
func (h *Handlers) HandleListDrivers(w http.ResponseWriter, r *http.Request) {
	ds := &store.DriverStore{DB: h.Pool}
	filter := store.DriverFilter{
		Offset: store.ParseOffset(r.URL.Query().Get("offset")),
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		filter.Limit, _ = strconv.Atoi(l)
	}
	drivers, err := ds.List(r.Context(), filter)
	if err != nil {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "failed to list drivers", err)
		return
	}
	api.WriteJSON(w, http.StatusOK, drivers)
}

// ─── Claim Handler ──────────────────────────────────────────────────────────

// HandleClaimGroup handles POST /api/groups/{id}/claim.
// A driver claims a dispatching group. Row-level WHERE guard ensures only one wins.
// Returns 409 Conflict if the group was already claimed or is not in 'dispatching' state.
func (h *Handlers) HandleClaimGroup(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("id")

	var body struct {
		DriverID string `json:"driver_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.DriverID == "" {
		api.WriteRequestError(w, r, http.StatusBadRequest, "driver_id is required", err)
		return
	}

	gs := &store.GroupStore{DB: h.Pool}
	rowsAffected, err := gs.ClaimGroup(r.Context(), groupID, body.DriverID)
	if err != nil {
		api.WriteRequestError(w, r, http.StatusInternalServerError, "claim failed", err, "group_id", groupID)
		return
	}

	if rowsAffected == 0 {
		api.WriteRequestError(w, r, http.StatusConflict, "group already claimed or not available", nil, "group_id", groupID)
		return
	}

	// Mark driver as busy
	ds := &store.DriverStore{DB: h.Pool}
	ds.SetStatus(r.Context(), body.DriverID, "busy")

	// Look up driver name for the event
	driverName := "unknown"
	if driver, err := ds.GetByID(r.Context(), body.DriverID); err == nil {
		driverName = driver.Name
	}

	var mapsLink, msg string
	detail, err := gs.GetByIDWithMembers(r.Context(), groupID)
	if err == nil {
		if _, link, m, err := dispatch.GenerateMessage(detail.Members); err == nil {
			mapsLink = link
			msg = m
		}
	}

	slog.Info("driver claimed group", "group_id", groupID, "driver_id", body.DriverID)

	// Broadcast assignment event
	if h.Events != nil {
		event := realtime.GroupAssigned(groupID, body.DriverID, driverName)
		h.Events.BroadcastMulti([]string{"global", groupID}, event)
	}

	api.WriteJSON(w, http.StatusOK, api.JSON{
		"claimed":          true,
		"group_id":         groupID,
		"driver_id":        body.DriverID,
		"maps_link":        mapsLink,
		"dispatch_message": msg,
	})
}

// HandleWebSocket upgrades HTTP to WebSocket. Query: ?room=global or ?room={group_id}
func (h *Handlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if h.WebSocket == nil {
		http.Error(w, "websocket unavailable", http.StatusServiceUnavailable)
		return
	}
	h.WebSocket.HandleUpgrade(w, r)
}
