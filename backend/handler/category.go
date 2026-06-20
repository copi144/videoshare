package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"videoshare/middleware"
	"videoshare/model"
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
		DisplayName string `json:"display_name"`
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

	if !model.IsValidName(req.Name) {
		respondJSONError(w, "Category name must only contain letters, numbers, and hyphens", http.StatusBadRequest)
		return
	}

	name := middleware.GetUsername(r.Context(), h.sm)

	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Name
	}

	cat := &model.Category{
		Name:        req.Name,
		DisplayName: displayName,
		Description: req.Description,
		CreatedBy:   name,
	}

	if err := h.categoryStore.Insert(cat); err != nil {
		slog.Error("failed to create category", "error", err)
		respondJSONError(w, "Failed to create category", http.StatusInternalServerError)
		return
	}

	slog.Info("category created via API", "name", cat.Name, "display_name", cat.DisplayName)
	respondJSONOK(w, map[string]interface{}{
		"redirect": "/admin/categories",
	})
}

// DeleteCategoryAPI handles JSON category deletion.
// DELETE /api/categories/{id}
func (h *CategoryHandler) DeleteCategoryAPI(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "id")
	if name == "" {
		respondJSONError(w, "Missing category ID", http.StatusBadRequest)
		return
	}

	if model.IsGlobal(name) {
		respondJSONError(w, "Cannot delete the Global category", http.StatusBadRequest)
		return
	}

	if err := h.categoryStore.Delete(name); err != nil {
		slog.Error("failed to delete category", "name", name, "error", err)
		respondJSONError(w, "Failed to delete category", http.StatusInternalServerError)
		return
	}

	slog.Info("category deleted via API", "name", name)
	respondJSONOK(w, nil)
}

// AssignUploadersAPI handles JSON assignment of uploaders to a category.
// POST /api/categories/{id}/uploaders
func (h *CategoryHandler) AssignUploadersAPI(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "id")
	if name == "" {
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

	if err := h.categoryStore.AssignUploaders(name, req.UserIDs); err != nil {
		slog.Error("failed to assign uploaders", "category_name", name, "error", err)
		respondJSONError(w, "Failed to assign uploaders", http.StatusInternalServerError)
		return
	}

	slog.Info("uploaders assigned via API", "category_name", name, "count", len(req.UserIDs))
	respondJSONOK(w, nil)
}

// ListCategoriesAPI returns all categories as JSON (for dropdowns).
// GET /api/categories
func (h *CategoryHandler) ListCategoriesAPI(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context(), h.sm)
	isAdmin := middleware.GetIsAdmin(r.Context(), h.sm)

	// Parse pagination parameters at the boundary.
	const defaultLimit = 50
	const maxLimit = 100

	limit := defaultLimit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			if l <= 0 {
				limit = defaultLimit
			} else if l > maxLimit {
				limit = maxLimit
			} else {
				limit = l
			}
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			if o < 0 {
				offset = 0
			} else {
				offset = o
			}
		}
	}

	var categories []*model.Category
	var total int
	var err error
	if isAdmin {
		categories, err = h.categoryStore.ListPaginated(limit, offset)
		if err == nil {
			total, err = h.categoryStore.Count()
		}
	} else {
		categories, err = h.categoryStore.ListByUploaderPaginated(userID, limit, offset)
		if err == nil {
			total, err = h.categoryStore.CountByUploader(userID)
		}
	}
	if err != nil {
		slog.Error("failed to list categories", "error", err)
		respondJSONError(w, "Failed to list categories", http.StatusInternalServerError)
		return
	}

	// Ensure the Global category is always included (it may not be in ListByUploader results
	// since Global was only inserted into categories, not category_uploaders).
	if !isAdmin {
		globalCat, globalErr := h.categoryStore.GetByName(model.GlobalCategoryName)
		if globalErr == nil && globalCat != nil {
			// Check if Global is already in the list
			hasGlobal := false
			for _, c := range categories {
				if model.IsGlobal(c.Name) {
					hasGlobal = true
					break
				}
			}
			if !hasGlobal {
				categories = append([]*model.Category{globalCat}, categories...)
			}
		}
	}

	respondJSONOK(w, map[string]interface{}{
		"categories": categories,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}
