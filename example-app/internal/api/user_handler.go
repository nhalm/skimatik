package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	userService UserService
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetActiveUsers handles GET /api/users?limit=10
func (h *UserHandler) GetActiveUsers(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	domainUsers, err := h.userService.GetActiveUsers(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert domain types to API response types
	apiUsers := make([]UserSummaryResponse, len(domainUsers))
	for i, user := range domainUsers {
		apiUsers[i] = UserSummaryResponse{
			ID:       user.ID,
			Name:     user.Name,
			Email:    user.Email,
			IsActive: user.IsActive,
		}
	}

	writeJSON(w, map[string]any{
		"users": apiUsers,
		"count": len(apiUsers),
	})
}

// GetUser handles GET /api/users/{id}
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	domainUser, err := h.userService.GetUser(r.Context(), userID)
	if err != nil {
		// Note: We'll need to define domain errors later
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Convert domain type to API response type
	apiUser := UserDetailResponse{
		ID:          domainUser.ID,
		Name:        domainUser.Name,
		Email:       domainUser.Email,
		IsActive:    domainUser.IsActive,
		PostCount:   domainUser.PostCount,
		CreatedAt:   domainUser.CreatedAt,
		LastLoginAt: domainUser.LastLoginAt,
	}

	writeJSON(w, apiUser)
}

// GetUserStats handles GET /api/users/{id}/stats
func (h *UserHandler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	domainStats, err := h.userService.GetUserStats(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get user stats", http.StatusInternalServerError)
		return
	}

	// Convert domain type to API response type
	apiStats := UserStatsResponse{
		UserID:       domainStats.UserID,
		PostCount:    domainStats.PostCount,
		CommentCount: domainStats.CommentCount,
		LastActivity: domainStats.LastActivity,
	}

	writeJSON(w, apiStats)
}

// SearchUsers handles GET /api/users/search?q=query
func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	domainUsers, err := h.userService.SearchUsers(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert domain types to API response types
	apiUsers := make([]UserSummaryResponse, len(domainUsers))
	for i, user := range domainUsers {
		apiUsers[i] = UserSummaryResponse{
			ID:       user.ID,
			Name:     user.Name,
			Email:    user.Email,
			IsActive: user.IsActive,
		}
	}

	writeJSON(w, map[string]any{
		"users": apiUsers,
		"query": query,
		"count": len(apiUsers),
	})
}

// CreateUserRequest is the JSON body for POST /api/users.
type CreateUserRequest struct {
	Name  string  `json:"name"`
	Email string  `json:"email"`
	Bio   *string `json:"bio,omitempty"`
}

// UpdateUserNameRequest is the JSON body for PATCH /api/users/{id}/name.
type UpdateUserNameRequest struct {
	Name string `json:"name"`
}

// CreateUser handles POST /api/users.
//
// This is the entry point exercised by the example-app curl smoke test that
// proves skimatik's audited Create CTE works end-to-end: a 201-equivalent
// response here means a row was written to both `users` and `users_audit` in
// a single statement.
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Email == "" {
		http.Error(w, "name and email are required", http.StatusBadRequest)
		return
	}

	user, err := h.userService.CreateUser(r.Context(), req.Name, req.Email, req.Bio)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, UserSummaryResponse{
		ID:       user.ID,
		Name:     user.Name,
		Email:    user.Email,
		IsActive: user.IsActive,
	})
}

// UpdateUserName handles PATCH /api/users/{id}/name.
//
// Pairs with CreateUser to demonstrate the audit Update CTE: the previously
// open audit row is closed and a new one is opened, sharing a single
// statement-level NOW() timestamp.
func (h *UserHandler) UpdateUserName(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	var req UpdateUserNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	user, err := h.userService.UpdateUserName(r.Context(), userID, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, UserSummaryResponse{
		ID:       user.ID,
		Name:     user.Name,
		Email:    user.Email,
		IsActive: user.IsActive,
	})
}

// GetUserAuditHistory handles GET /api/users/{id}/audit.
//
// Surfaces the audit trail kept by skimatik's CTE-based Create/Update.
// Useful for the smoke test to assert "after one Create + one Update we
// have exactly two audit rows, one closed and one open."
func (h *UserHandler) GetUserAuditHistory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	entries, err := h.userService.GetUserAuditHistory(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"audit": entries,
		"count": len(entries),
	})
}

// DeactivateUser handles DELETE /api/users/{id}
func (h *UserHandler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	err = h.userService.DeactivateUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{
		"message": "User deactivated successfully",
	})
}
