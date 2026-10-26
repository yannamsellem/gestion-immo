package http

import (
	"errors"
	"net/http"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/infrastructure"
	"github.com/yannamsellem/gestion-immo/internal/platform/httpx"
	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

// Handler exposes the REST HTTP endpoints for authentication and SUPER_ADMIN user approval.
type Handler struct {
	svc    *application.Service
	jwtSvc *infrastructure.JwtService
}

// NewHandler constructs a new auth HTTP Handler via constructor injection.
func NewHandler(svc *application.Service, jwtSvc *infrastructure.JwtService) *Handler {
	return &Handler{
		svc:    svc,
		jwtSvc: jwtSvc,
	}
}

type signupRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type createMCPTokenRequest struct {
	Label string `json:"label"`
}

type authTokensResponse struct {
	AccessToken  string              `json:"access_token"`
	RefreshToken string              `json:"refresh_token"`
	TokenType    string              `json:"token_type"`
	User         application.UserDTO `json:"user"`
}

type mcpTokenResponse struct {
	Token string `json:"token"`
	Label string `json:"label"`
}

// RegisterRoutes mounts all auth and admin user-approval endpoints onto mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/signup", h.handleSignup)
	mux.HandleFunc("POST /api/v1/auth/login", h.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.handleRefresh)
	mux.HandleFunc("POST /api/v1/auth/logout", h.handleLogout)
	mux.HandleFunc("POST /api/v1/auth/mcp-tokens", RequireAuthFunc(h.jwtSvc, h.handleCreateMCPToken))

	mux.HandleFunc("GET /api/v1/admin/users/pending", RequireSuperAdminFunc(h.jwtSvc, h.handleListPendingUsers))
	mux.HandleFunc("POST /api/v1/admin/users/{id}/approve", RequireSuperAdminFunc(h.jwtSvc, h.handleApproveUser))
	mux.HandleFunc("POST /api/v1/admin/users/{id}/reject", RequireSuperAdminFunc(h.jwtSvc, h.handleRejectUser))
}

func (h *Handler) handleSignup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	userDTO, err := h.svc.Signup(r.Context(), application.SignupCommand{
		FullName: req.FullName,
		Email:    req.Email,
		Password: secret.New(req.Password),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	// 202 Accepted: account created in PENDING_APPROVAL status, awaiting SUPER_ADMIN approval.
	httpx.Accepted(w, userDTO)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	tokens, err := h.svc.Login(r.Context(), application.LoginCommand{
		Email:    req.Email,
		Password: secret.New(req.Password),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.OK(w, authTokensResponse{
		AccessToken:  tokens.AccessToken.Value(),
		RefreshToken: tokens.RefreshToken.Value(),
		TokenType:    tokens.TokenType,
		User:         tokens.User,
	})
}

func (h *Handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	tokens, err := h.svc.Refresh(r.Context(), secret.New(req.RefreshToken))
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.OK(w, authTokensResponse{
		AccessToken:  tokens.AccessToken.Value(),
		RefreshToken: tokens.RefreshToken.Value(),
		TokenType:    tokens.TokenType,
		User:         tokens.User,
	})
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	if err := h.svc.Logout(r.Context(), secret.New(req.RefreshToken)); err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.NoContent(w)
}

func (h *Handler) handleCreateMCPToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		httpx.Unauthorized(w, "authentication required")
		return
	}

	var req createMCPTokenRequest
	if r.ContentLength != 0 {
		if err := httpx.DecodeJSON(r, &req); err != nil {
			httpx.BadRequest(w, err.Error())
			return
		}
	}

	rawToken, err := h.svc.CreateMCPToken(r.Context(), claims.UserID, req.Label)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, mcpTokenResponse{
		Token: rawToken.Value(),
		Label: req.Label,
	})
}

func (h *Handler) handleListPendingUsers(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		httpx.Unauthorized(w, "authentication required")
		return
	}

	users, err := h.svc.ListPendingUsers(r.Context(), claims.UserID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if users == nil {
		users = []application.UserDTO{}
	}

	httpx.OK(w, users)
}

func (h *Handler) handleApproveUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		httpx.Unauthorized(w, "authentication required")
		return
	}

	targetID := r.PathValue("id")
	approved, err := h.svc.ApproveUser(r.Context(), claims.UserID, targetID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.OK(w, approved)
}

func (h *Handler) handleRejectUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		httpx.Unauthorized(w, "authentication required")
		return
	}

	targetID := r.PathValue("id")
	rejected, err := h.svc.RejectUser(r.Context(), claims.UserID, targetID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.OK(w, rejected)
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		httpx.UnprocessableEntity(w, err.Error())
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		httpx.Conflict(w, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrInvalidToken), errors.Is(err, domain.ErrRefreshTokenReuseDetected):
		httpx.Unauthorized(w, err.Error())
	case errors.Is(err, domain.ErrAccountNotApproved), errors.Is(err, domain.ErrForbiddenSuperAdminOnly):
		httpx.Forbidden(w, err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		httpx.NotFound(w, err.Error())
	default:
		httpx.InternalError(w, "internal authentication error")
	}
}
