package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go-prisma-neon/db"
	"go-prisma-neon/middleware"
	"go-prisma-neon/services"
)

type PostHandler struct {
	client      *db.PrismaClient
	userService *services.UserService
}

func NewPostHandler(client *db.PrismaClient, userService *services.UserService) *PostHandler {
	return &PostHandler{
		client:      client,
		userService: userService,
	}
}

func (h *PostHandler) GetPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := h.client.Post.FindMany(
		db.Post.Published.Equals(true),
	).With(
		db.Post.Author.Fetch(),
	).Exec(r.Context())

	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, posts, http.StatusOK)
}

func (h *PostHandler) GetPostByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	post, err := h.client.Post.FindUnique(
		db.Post.ID.Equals(id),
	).With(
		db.Post.Author.Fetch(),
	).Exec(r.Context())

	if err != nil {
		middleware.ErrorResponse(w, "Post not found", http.StatusNotFound)
		return
	}

	middleware.JSONResponse(w, post, http.StatusOK)
}

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// Ensure user exists in database
	_, err := h.userService.GetOrCreateUser(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var req struct {
		Title     string `json:"title"`
		Content   string `json:"content"`
		Published bool   `json:"published"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.ErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		middleware.ErrorResponse(w, "Title is required", http.StatusBadRequest)
		return
	}

	post, err := h.client.Post.CreateOne(
		db.Post.Title.Set(req.Title),
		db.Post.Content.SetIfPresent(req.Content),
		db.Post.Published.Set(req.Published),
		db.Post.Author.Link(db.User.ID.Equals(userID)),
	).Exec(r.Context())

	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, post, http.StatusCreated)
}

func (h *PostHandler) GetMyPosts(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	posts, err := h.client.Post.FindMany(
		db.Post.AuthorID.Equals(userID),
	).With(
		db.Post.Author.Fetch(),
	).OrderBy(
		db.Post.CreatedAt.Order(db.DESC),
	).Exec(r.Context())

	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, posts, http.StatusOK)
}

func (h *PostHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	postID := vars["id"]

	var req struct {
		Title     string `json:"title"`
		Content   string `json:"content"`
		Published *bool  `json:"published"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.ErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if user owns the post
	post, err := h.client.Post.FindUnique(
		db.Post.ID.Equals(postID),
	).Exec(r.Context())

	if err != nil {
		middleware.ErrorResponse(w, "Post not found", http.StatusNotFound)
		return
	}

	if post.AuthorID != userID {
		middleware.ErrorResponse(w, "Not authorized to edit this post", http.StatusForbidden)
		return
	}

	// Update post
	updatedPost, err := h.client.Post.FindUnique(
		db.Post.ID.Equals(postID),
	).Update(
		db.Post.Title.SetIfPresent(req.Title),
		db.Post.Content.SetIfPresent(req.Content),
		db.Post.Published.SetIfPresent(req.Published),
	).Exec(r.Context())

	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, updatedPost, http.StatusOK)
}

func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	postID := vars["id"]

	// Check if user owns the post
	post, err := h.client.Post.FindUnique(
		db.Post.ID.Equals(postID),
	).Exec(r.Context())

	if err != nil {
		middleware.ErrorResponse(w, "Post not found", http.StatusNotFound)
		return
	}

	if post.AuthorID != userID {
		middleware.ErrorResponse(w, "Not authorized to delete this post", http.StatusForbidden)
		return
	}

	// Delete post
	_, err = h.client.Post.FindUnique(
		db.Post.ID.Equals(postID),
	).Delete().Exec(r.Context())

	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, map[string]string{"message": "Post deleted successfully"}, http.StatusOK)
}
