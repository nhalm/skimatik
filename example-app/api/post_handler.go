package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nhalm/skimatik/example-app/domain"
)

// PostHandler handles HTTP requests for post operations
type PostHandler struct {
	postService PostService // Use service interface, not API interface
}

// NewPostHandler creates a new post handler
func NewPostHandler(postService PostService) *PostHandler {
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

	servicePosts, err := h.postService.GetPublishedPosts(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert service types to API types
	apiPosts := make([]domain.PostSummary, len(servicePosts))
	for i, post := range servicePosts {
		apiPosts[i] = domain.PostSummary{
			ID:          post.ID,
			Title:       post.Title,
			Content:     post.Content,
			AuthorID:    post.AuthorID,
			IsPublished: post.IsPublished,
			PublishedAt: post.PublishedAt,
			CreatedAt:   post.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"posts": apiPosts,
		"count": len(apiPosts),
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

	servicePost, err := h.postService.GetPostWithAuthor(r.Context(), postID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Convert service type to API type
	apiPost := domain.PostDetail{
		ID:          servicePost.ID,
		Title:       servicePost.Title,
		Content:     servicePost.Content,
		AuthorID:    servicePost.AuthorID,
		AuthorName:  servicePost.AuthorName,
		AuthorEmail: servicePost.AuthorEmail,
		IsPublished: servicePost.IsPublished,
		PublishedAt: servicePost.PublishedAt,
		CreatedAt:   servicePost.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiPost)
}

// GetUserPosts handles GET /api/users/{id}/posts
func (h *PostHandler) GetUserPosts(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	servicePosts, err := h.postService.GetUserPosts(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get user posts", http.StatusInternalServerError)
		return
	}

	// Convert service types to API types
	apiPosts := make([]domain.PostSummary, len(servicePosts))
	for i, post := range servicePosts {
		apiPosts[i] = domain.PostSummary{
			ID:          post.ID,
			Title:       post.Title,
			Content:     post.Content,
			AuthorID:    post.AuthorID,
			IsPublished: post.IsPublished,
			PublishedAt: post.PublishedAt,
			CreatedAt:   post.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"posts": apiPosts,
		"count": len(apiPosts),
	})
}

// GetPostsWithStats handles GET /api/posts/with-stats?limit=10
func (h *PostHandler) GetPostsWithStats(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	serviceStats, err := h.postService.GetPostsWithStats(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert service types to API types
	apiStats := make([]domain.PostWithStats, len(serviceStats))
	for i, stat := range serviceStats {
		apiStats[i] = domain.PostWithStats{
			ID:           stat.ID,
			Title:        stat.Title,
			AuthorID:     stat.AuthorID,
			AuthorName:   stat.AuthorName,
			CommentCount: stat.CommentCount,
			CreatedAt:    stat.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"posts": apiStats,
		"count": len(apiStats),
	})
}

// Custom handler methods that demonstrate the layered architecture

// GetFeaturedPosts handles GET /api/posts/featured
func (h *PostHandler) GetFeaturedPosts(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 5 // default for featured posts

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 20 {
			limit = parsedLimit
		}
	}

	servicePosts, err := h.postService.GetFeaturedPosts(r.Context(), limit)
	if err != nil {
		http.Error(w, "Failed to get featured posts", http.StatusInternalServerError)
		return
	}

	// Convert service types to API types
	apiPosts := make([]domain.PostSummary, len(servicePosts))
	for i, post := range servicePosts {
		apiPosts[i] = domain.PostSummary{
			ID:          post.ID,
			Title:       post.Title,
			Content:     post.Content,
			AuthorID:    post.AuthorID,
			IsPublished: post.IsPublished,
			PublishedAt: post.PublishedAt,
			CreatedAt:   post.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"posts": apiPosts,
		"count": len(apiPosts),
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

	servicePosts, err := h.postService.GetPostsByTag(r.Context(), tag, limit)
	if err != nil {
		http.Error(w, "Failed to get posts by tag", http.StatusInternalServerError)
		return
	}

	// Convert service types to API types
	apiPosts := make([]domain.PostSummary, len(servicePosts))
	for i, post := range servicePosts {
		apiPosts[i] = domain.PostSummary{
			ID:          post.ID,
			Title:       post.Title,
			Content:     post.Content,
			AuthorID:    post.AuthorID,
			IsPublished: post.IsPublished,
			PublishedAt: post.PublishedAt,
			CreatedAt:   post.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"posts": apiPosts,
		"tag":   tag,
		"count": len(apiPosts),
	})
}

// GetPostStatistics handles GET /api/posts/statistics
func (h *PostHandler) GetPostStatistics(w http.ResponseWriter, r *http.Request) {
	serviceStats, err := h.postService.GetPostStatistics(r.Context())
	if err != nil {
		http.Error(w, "Failed to get post statistics", http.StatusInternalServerError)
		return
	}

	// Convert service type to API type
	apiStats := domain.PostStats{
		TotalPosts:     serviceStats.TotalPosts,
		PublishedPosts: serviceStats.PublishedPosts,
		DraftPosts:     serviceStats.DraftPosts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiStats)
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
