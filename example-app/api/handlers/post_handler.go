package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nhalm/skimatik/example-app/service"
)

// PostHandler handles HTTP requests for post operations
type PostHandler struct {
	postService service.PostService
}

// NewPostHandler creates a new post handler
func NewPostHandler(postService service.PostService) *PostHandler {
	return &PostHandler{
		postService: postService,
	}
}

// GetPublishedPosts handles GET /api/posts?limit=10
func (h *PostHandler) GetPublishedPosts(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	posts, err := h.postService.GetPublishedPosts(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"posts": posts,
		"count": len(posts),
	})
}

// GetPost handles GET /api/posts/{id}
func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	postID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	post, err := h.postService.GetPostWithAuthor(r.Context(), postID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(post)
}

// GetUserPosts handles GET /api/users/{id}/posts
func (h *PostHandler) GetUserPosts(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	posts, err := h.postService.GetUserPosts(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get user posts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"posts": posts,
		"count": len(posts),
	})
}

// GetPostsWithStats handles GET /api/posts/stats?limit=10
func (h *PostHandler) GetPostsWithStats(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	posts, err := h.postService.GetPostsWithStats(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"posts": posts,
		"count": len(posts),
	})
}

// PublishPost handles PUT /api/posts/{id}/publish
func (h *PostHandler) PublishPost(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "id")

	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post ID format", http.StatusBadRequest)
		return
	}

	err = h.postService.PublishPost(r.Context(), postID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Post published successfully",
	})
}

// GetFeaturedPosts handles GET /api/posts/featured
func (h *PostHandler) GetFeaturedPosts(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 5 // default for featured posts

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 20 {
			limit = parsedLimit
		}
	}

	posts, err := h.postService.GetFeaturedPosts(r.Context(), limit)
	if err != nil {
		http.Error(w, "Failed to get featured posts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"posts": posts,
		"count": len(posts),
	})
}

// GetPostsByTag handles GET /api/posts/tag/{tag}
func (h *PostHandler) GetPostsByTag(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	if tag == "" {
		http.Error(w, "Tag parameter is required", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	posts, err := h.postService.GetPostsByTag(r.Context(), tag, limit)
	if err != nil {
		http.Error(w, "Failed to get posts by tag", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"posts": posts,
		"tag":   tag,
		"count": len(posts),
	})
}

// GetPostStatistics handles GET /api/posts/statistics
func (h *PostHandler) GetPostStatistics(w http.ResponseWriter, r *http.Request) {
	stats, err := h.postService.GetPostStatistics(r.Context())
	if err != nil {
		http.Error(w, "Failed to get post statistics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// writeJSON writes a JSON response
func (h *PostHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}
