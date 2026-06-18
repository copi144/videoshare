package handler

import (
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/csrf"

	"videoshare/internal/middleware"
	"videoshare/internal/model"
)

// CategoryHandler handles category management (admin only).
type CategoryHandler struct {
	categoryStore *model.CategoryStore
	userStore     *model.UserStore
	sm            *scs.SessionManager
	templates     fs.FS
}

// NewCategoryHandler creates a new CategoryHandler with injected dependencies.
func NewCategoryHandler(categoryStore *model.CategoryStore, userStore *model.UserStore, sm *scs.SessionManager, templates fs.FS) *CategoryHandler {
	return &CategoryHandler{
		categoryStore: categoryStore,
		userStore:     userStore,
		sm:            sm,
		templates:     templates,
	}
}

// ServeCategoriesPage lists all categories with management controls.
// GET /admin/categories
func (h *CategoryHandler) ServeCategoriesPage(w http.ResponseWriter, r *http.Request) {
	username := middleware.GetUsername(r.Context(), h.sm)
	userRole := middleware.GetUserRole(r.Context(), h.sm)

	categories, err := h.categoryStore.List()
	if err != nil {
		slog.Error("failed to list categories", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// For each category, load its assigned uploaders and video count.
	type categoryWithUploaders struct {
		*model.Category
		UploaderUsernames []string
		VideoCount        int
	}

	categoryData := make([]categoryWithUploaders, 0, len(categories))
	for _, cat := range categories {
		uploaderIDs, err := h.categoryStore.GetUploaders(cat.ID)
		if err != nil {
			slog.Error("failed to get uploaders for category", "category_id", cat.ID, "error", err)
			continue
		}
		usernames := make([]string, 0, len(uploaderIDs))
		for _, uid := range uploaderIDs {
			user, err := h.userStore.GetByID(uid)
			if err == nil {
				usernames = append(usernames, user.Username)
			}
		}

		videoCount, err := h.categoryStore.GetVideoCount(cat.ID)
		if err != nil {
			slog.Error("failed to get video count for category", "category_id", cat.ID, "error", err)
			videoCount = 0
		}

		categoryData = append(categoryData, categoryWithUploaders{
			Category:         cat,
			UploaderUsernames: usernames,
			VideoCount:       videoCount,
		})
	}

	// Get all users for the assign form.
	allUsers, err := h.userStore.List()
	if err != nil {
		slog.Error("failed to list users", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	errorMsg := r.URL.Query().Get("error")

	if err := parseAndRender(w, h.templates, "categories.html", &TemplateData{
		Title:      "Categories — VideoShare",
		IsLoggedIn: true,
		Username:   username,
		UserRole:   userRole,
		CSRFToken:  csrf.Token(r),
		Error:      errorMsg,
		Data: map[string]interface{}{
			"Categories": categoryData,
			"Users":      allUsers,
		},
	}); err != nil {
		slog.Error("failed to render categories template", "error", err)
	}
}

// CreateCategory creates a new category.
// POST /admin/categories
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	description := r.FormValue("description")

	if name == "" {
		http.Redirect(w, r, "/admin/categories?error="+url.QueryEscape("Category name is required"), http.StatusSeeOther)
		return
	}

	userID := middleware.GetUserID(r.Context(), h.sm)

	cat := &model.Category{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedBy:   userID,
	}

	if err := h.categoryStore.Insert(cat); err != nil {
		slog.Error("failed to create category", "error", err)
		http.Redirect(w, r, "/admin/categories?error="+url.QueryEscape("Failed to create category"), http.StatusSeeOther)
		return
	}

	slog.Info("category created", "id", cat.ID, "name", name)
	http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
}

// DeleteCategory deletes a category.
// POST /admin/categories/{id}/delete
func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Redirect(w, r, "/admin/categories?error="+url.QueryEscape("Missing category ID"), http.StatusSeeOther)
		return
	}

	if err := h.categoryStore.Delete(id); err != nil {
		slog.Error("failed to delete category", "id", id, "error", err)
		http.Redirect(w, r, "/admin/categories?error="+url.QueryEscape("Failed to delete category"), http.StatusSeeOther)
		return
	}

	slog.Info("category deleted", "id", id)
	http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
}

// AssignUploaders sets the uploaders assigned to a category.
// POST /admin/categories/{id}/uploaders
func (h *CategoryHandler) AssignUploaders(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Redirect(w, r, "/admin/categories?error="+url.QueryEscape("Missing category ID"), http.StatusSeeOther)
		return
	}

	userIDs := r.Form["user_ids"]

	if err := h.categoryStore.AssignUploaders(id, userIDs); err != nil {
		slog.Error("failed to assign uploaders", "category_id", id, "error", err)
		http.Redirect(w, r, "/admin/categories?error="+url.QueryEscape("Failed to assign uploaders"), http.StatusSeeOther)
		return
	}

	slog.Info("uploaders assigned to category", "category_id", id, "count", len(userIDs))
	http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
}
