package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"videoshare/internal/middleware"
	"videoshare/internal/model"
)

// CategoryHandler handles category management (admin only).
type CategoryHandler struct {
	categoryStore *model.CategoryStore
	userStore     *model.UserStore
	sm            *scs.SessionManager
}

// NewCategoryHandler creates a new CategoryHandler with injected dependencies.
func NewCategoryHandler(categoryStore *model.CategoryStore, userStore *model.UserStore, sm *scs.SessionManager) *CategoryHandler {
	return &CategoryHandler{
		categoryStore: categoryStore,
		userStore:     userStore,
		sm:            sm,
	}
}

// CreateCategoryAPI handles JSON category creation.
// POST /api/categories
func (h *CategoryHandler) CreateCategoryAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		respondJSONError(w, "Category name is required", http.StatusBadRequest)
		return
	}

	if !model.IsValidCategoryName(req.Name) {
		respondJSONError(w, "Category name must only contain letters, numbers, and hyphens", http.StatusBadRequest)
		return
	}

	userID := middleware.GetUserID(r.Context(), h.sm)

	cat := &model.Category{
		ID:          req.Name,
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   userID,
	}

	if err := h.categoryStore.Insert(cat); err != nil {
		slog.Error("failed to create category", "error", err)
		respondJSONError(w, "Failed to create category", http.StatusInternalServerError)
		return
	}

	slog.Info("category created via API", "id", cat.ID, "name", req.Name)
	respondJSONOK(w, map[string]interface{}{
		"redirect": "/admin/categories",
	})
}

// DeleteCategoryAPI handles JSON category deletion.
// DELETE /api/categories/{id}
func (h *CategoryHandler) DeleteCategoryAPI(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondJSONError(w, "Missing category ID", http.StatusBadRequest)
		return
	}

	if id == model.GlobalCategoryID {
		respondJSONError(w, "Cannot delete the Global category", http.StatusBadRequest)
		return
	}

	if err := h.categoryStore.Delete(id); err != nil {
		slog.Error("failed to delete category", "id", id, "error", err)
		respondJSONError(w, "Failed to delete category", http.StatusInternalServerError)
		return
	}

	slog.Info("category deleted via API", "id", id)
	respondJSONOK(w, nil)
}

// AssignUploadersAPI handles JSON assignment of uploaders to a category.
// POST /api/categories/{id}/uploaders
func (h *CategoryHandler) AssignUploadersAPI(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondJSONError(w, "Missing category ID", http.StatusBadRequest)
		return
	}

	var req struct {
		UserIDs []string `json:"user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.categoryStore.AssignUploaders(id, req.UserIDs); err != nil {
		slog.Error("failed to assign uploaders", "category_id", id, "error", err)
		respondJSONError(w, "Failed to assign uploaders", http.StatusInternalServerError)
		return
	}

	slog.Info("uploaders assigned via API", "category_id", id, "count", len(req.UserIDs))
	respondJSONOK(w, nil)
}

// ListCategoriesAPI returns all categories as JSON (for dropdowns).
// GET /api/categories
func (h *CategoryHandler) ListCategoriesAPI(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context(), h.sm)
	userRole := middleware.GetUserRole(r.Context(), h.sm)

	var categories []*model.Category
	var err error
	if userRole == "admin" {
		categories, err = h.categoryStore.List()
	} else {
		categories, err = h.categoryStore.ListByUploader(userID)
	}
	if err != nil {
		slog.Error("failed to list categories", "error", err)
		respondJSONError(w, "Failed to list categories", http.StatusInternalServerError)
		return
	}

	respondJSONOK(w, map[string]interface{}{
		"categories": categories,
	})
}
