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

// GetUserByEmail handles GET /api/users/email/{email}
func (h *UserHandler) GetUserByEmail(w http.ResponseWriter, r *http.Request) {
	email := chi.URLParam(r, "email")

	if email == "" {
		http.Error(w, "Email parameter is required", http.StatusBadRequest)
		return
	}

	domainUser, err := h.userService.GetUserByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiUser)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiUser)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiStats)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": apiUsers,
		"query": query,
		"count": len(apiUsers),
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User deactivated successfully",
	})
}

// writeJSON writes a JSON response
func (h *UserHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}
