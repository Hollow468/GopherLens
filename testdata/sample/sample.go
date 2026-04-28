package sample

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

// UserRepository wraps database access.
type UserRepository struct {
	DB *sql.DB
}

// GetUser fetches a user by ID. Returns nil, 404 if not found.
func (r *UserRepository) GetUser(id int) (*User, error) {
	row := r.DB.QueryRow("SELECT id, name, email FROM users WHERE id = ?", id)

	var user User
	err := row.Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// User represents a user record.
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var ErrNotFound = errors.New("user not found")

// UserHandler handles HTTP requests for user resources.
type UserHandler struct {
	Repo *UserRepository
}

// HandleGetUser processes GET /users/{id}.
func (h *UserHandler) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, `{"error":"missing id"}`, http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	user, err := h.Repo.GetUser(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
