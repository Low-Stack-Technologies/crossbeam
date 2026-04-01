package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/low-stack-technologies/crossbeam/server/internal/db"
	"github.com/low-stack-technologies/crossbeam/server/internal/db/generated"
	"github.com/low-stack-technologies/crossbeam/server/internal/middleware"
)

type UserHandler struct {
	q *generated.Queries
}

func NewUserHandler(q *generated.Queries) *UserHandler {
	return &UserHandler{q: q}
}

type userResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func toUserResponse(u *generated.User) userResponse {
	return userResponse{
		ID:        db.PgToUUID(u.ID),
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: u.CreatedAt.Time,
	}
}

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	user, err := h.q.GetUserByID(r.Context(), db.UUIDToPg(userID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(&user))
}
