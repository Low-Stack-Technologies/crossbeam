package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/low-stack-technologies/crossbeam/server/internal/service"
)

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type authResponse struct {
	User  userResponse `json:"user"`
	Token string       `json:"token"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		writeError(w, http.StatusUnprocessableEntity, "email, password, and name are required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusUnprocessableEntity, "password must be at least 8 characters")
		return
	}

	user, token, err := h.authSvc.Register(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{
		User:  toUserResponse(user),
		Token: token,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusUnprocessableEntity, "email and password are required")
		return
	}

	user, token, err := h.authSvc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCreds) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		User:  toUserResponse(user),
		Token: token,
	})
}
