package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

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

// SignupRequest represents the JSON request body for user registration.
type SignupRequest struct {
	FullName string `json:"full_name" required:"true"`
	Email    string `json:"email" required:"true"`
	Password string `json:"password" required:"true"`
}

// LoginRequest represents the JSON request body for email/password authentication.
type LoginRequest struct {
	Email    string `json:"email" required:"true"`
	Password string `json:"password" required:"true"`
}

// RefreshRequest represents the JSON request body for token refresh and logout.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" required:"true"`
}

// CreateMCPTokenRequest represents the JSON request body for creating an MCP bearer token.
type CreateMCPTokenRequest struct {
	Label string `json:"label,omitempty"`
}

// AuthTokensResponse represents the JSON response containing access/refresh tokens and user profile.
type AuthTokensResponse struct {
	AccessToken  string              `json:"access_token"`
	RefreshToken string              `json:"refresh_token"`
	TokenType    string              `json:"token_type"`
	User         application.UserDTO `json:"user"`
}

// MCPTokenResponse represents the JSON response containing a newly generated MCP token.
type MCPTokenResponse struct {
	Token string `json:"token"`
	Label string `json:"label"`
}

type signupInput struct {
	Body SignupRequest
}

type userOutput struct {
	Body application.UserDTO
}

type loginInput struct {
	Body LoginRequest
}

type authTokensOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         AuthTokensResponse
}

type refreshInput struct {
	Body RefreshRequest
}

type createMCPTokenInput struct {
	Authorization string `header:"Authorization"`
	RawBody       []byte
	Body          *CreateMCPTokenRequest `required:"false"`
}

type mcpTokenOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         MCPTokenResponse
}

type authHeaderInput struct {
	Authorization string `header:"Authorization"`
}

type userListOutput struct {
	Body []application.UserDTO
}

type userDecisionInput struct {
	Authorization string `header:"Authorization"`
	ID            string `path:"id" format:"uuid"`
}

// RegisterRoutes mounts all auth and admin user-approval endpoints onto mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	h.RegisterHumaRoutes(httpx.NewHumaAPI(mux))
}

// RegisterHumaRoutes registers all auth and admin user-approval endpoints onto a Huma API.
func (h *Handler) RegisterHumaRoutes(api huma.API) {
	bearerSecurity := []map[string][]string{{"bearerAuth": {}}}

	huma.Register(api, huma.Operation{
		OperationID:   "signup",
		Method:        http.MethodPost,
		Path:          "/api/v1/auth/signup",
		Summary:       "Request a new user account (created in PENDING_APPROVAL status)",
		DefaultStatus: http.StatusAccepted,
		Tags:          []string{"Auth"},
	}, h.handleSignup)

	huma.Register(api, huma.Operation{
		OperationID:   "login",
		Method:        http.MethodPost,
		Path:          "/api/v1/auth/login",
		Summary:       "Authenticate with email and password",
		DefaultStatus: http.StatusOK,
		Tags:          []string{"Auth"},
	}, h.handleLogin)

	huma.Register(api, huma.Operation{
		OperationID:   "refresh",
		Method:        http.MethodPost,
		Path:          "/api/v1/auth/refresh",
		Summary:       "Rotate refresh token and issue a new access token",
		DefaultStatus: http.StatusOK,
		Tags:          []string{"Auth"},
	}, h.handleRefresh)

	huma.Register(api, huma.Operation{
		OperationID:   "logout",
		Method:        http.MethodPost,
		Path:          "/api/v1/auth/logout",
		Summary:       "Revoke refresh token session",
		DefaultStatus: http.StatusNoContent,
		Tags:          []string{"Auth"},
	}, h.handleLogout)

	huma.Register(api, huma.Operation{
		OperationID:   "createMcpToken",
		Method:        http.MethodPost,
		Path:          "/api/v1/auth/mcp-tokens",
		Summary:       "Create a personal MCP bearer token",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Auth"},
	}, h.handleCreateMCPToken)

	huma.Register(api, huma.Operation{
		OperationID:   "listPendingUsers",
		Method:        http.MethodGet,
		Path:          "/api/v1/admin/users/pending",
		Summary:       "List user accounts awaiting SUPER_ADMIN approval",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Auth"},
	}, h.handleListPendingUsers)

	huma.Register(api, huma.Operation{
		OperationID:   "approveUser",
		Method:        http.MethodPost,
		Path:          "/api/v1/admin/users/{id}/approve",
		Summary:       "Approve a pending user account",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Auth"},
	}, h.handleApproveUser)

	huma.Register(api, huma.Operation{
		OperationID:   "rejectUser",
		Method:        http.MethodPost,
		Path:          "/api/v1/admin/users/{id}/reject",
		Summary:       "Reject a user account and revoke active refresh tokens",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Auth"},
	}, h.handleRejectUser)
}

func (h *Handler) handleSignup(ctx context.Context, in *signupInput) (*userOutput, error) {
	userDTO, err := h.svc.Signup(ctx, application.SignupCommand{
		FullName: in.Body.FullName,
		Email:    in.Body.Email,
		Password: secret.New(in.Body.Password),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &userOutput{Body: userDTO}, nil
}

func (h *Handler) handleLogin(ctx context.Context, in *loginInput) (*authTokensOutput, error) {
	tokens, err := h.svc.Login(ctx, application.LoginCommand{
		Email:    in.Body.Email,
		Password: secret.New(in.Body.Password),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &authTokensOutput{
		CacheControl: "no-store",
		Body: AuthTokensResponse{
			AccessToken:  tokens.AccessToken.Value(),
			RefreshToken: tokens.RefreshToken.Value(),
			TokenType:    tokens.TokenType,
			User:         tokens.User,
		},
	}, nil
}

func (h *Handler) handleRefresh(ctx context.Context, in *refreshInput) (*authTokensOutput, error) {
	tokens, err := h.svc.Refresh(ctx, secret.New(in.Body.RefreshToken))
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &authTokensOutput{
		CacheControl: "no-store",
		Body: AuthTokensResponse{
			AccessToken:  tokens.AccessToken.Value(),
			RefreshToken: tokens.RefreshToken.Value(),
			TokenType:    tokens.TokenType,
			User:         tokens.User,
		},
	}, nil
}

func (h *Handler) handleLogout(ctx context.Context, in *refreshInput) (*struct{}, error) {
	if err := h.svc.Logout(ctx, secret.New(in.Body.RefreshToken)); err != nil {
		return nil, mapDomainError(err)
	}
	return &struct{}{}, nil
}

func (h *Handler) handleCreateMCPToken(ctx context.Context, in *createMCPTokenInput) (*mcpTokenOutput, error) {
	claims, err := h.authenticate(ctx, in.Authorization, false)
	if err != nil {
		return nil, err
	}

	label := ""
	if in.Body != nil {
		label = in.Body.Label
	} else if len(strings.TrimSpace(string(in.RawBody))) > 0 {
		var parsed CreateMCPTokenRequest
		if err := json.Unmarshal(in.RawBody, &parsed); err != nil {
			return nil, httpx.StatusError(http.StatusBadRequest, "invalid JSON payload")
		}
		label = parsed.Label
	}

	rawToken, err := h.svc.CreateMCPToken(ctx, claims.UserID, label)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &mcpTokenOutput{
		CacheControl: "no-store",
		Body: MCPTokenResponse{
			Token: rawToken.Value(),
			Label: label,
		},
	}, nil
}

func (h *Handler) handleListPendingUsers(ctx context.Context, in *authHeaderInput) (*userListOutput, error) {
	claims, err := h.authenticate(ctx, in.Authorization, true)
	if err != nil {
		return nil, err
	}

	users, err := h.svc.ListPendingUsers(ctx, claims.UserID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	if users == nil {
		users = []application.UserDTO{}
	}
	return &userListOutput{Body: users}, nil
}

func (h *Handler) handleApproveUser(ctx context.Context, in *userDecisionInput) (*userOutput, error) {
	claims, err := h.authenticate(ctx, in.Authorization, true)
	if err != nil {
		return nil, err
	}

	approved, err := h.svc.ApproveUser(ctx, claims.UserID, in.ID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &userOutput{Body: approved}, nil
}

func (h *Handler) handleRejectUser(ctx context.Context, in *userDecisionInput) (*userOutput, error) {
	claims, err := h.authenticate(ctx, in.Authorization, true)
	if err != nil {
		return nil, err
	}

	rejected, err := h.svc.RejectUser(ctx, claims.UserID, in.ID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &userOutput{Body: rejected}, nil
}

func (h *Handler) authenticate(ctx context.Context, authHeader string, requireSuperAdmin bool) (*domain.Claims, error) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		rawToken := extractBearer(authHeader)
		if rawToken == "" || h.jwtSvc == nil {
			return nil, httpx.StatusError(http.StatusUnauthorized, "missing or invalid Authorization Bearer token")
		}
		validated, err := h.jwtSvc.ValidateAccessToken(rawToken)
		if err != nil {
			return nil, httpx.StatusError(http.StatusUnauthorized, "invalid or expired access token")
		}
		claims = validated
	}
	if requireSuperAdmin && claims.Role != domain.RoleSuperAdmin {
		return nil, httpx.StatusError(http.StatusForbidden, domain.ErrForbiddenSuperAdminOnly.Error())
	}
	return claims, nil
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return httpx.StatusError(http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		return httpx.StatusError(http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrInvalidToken),
		errors.Is(err, domain.ErrRefreshTokenReuseDetected):
		return httpx.StatusError(http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrAccountNotApproved),
		errors.Is(err, domain.ErrForbiddenSuperAdminOnly):
		return httpx.StatusError(http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		return httpx.StatusError(http.StatusNotFound, err.Error())
	default:
		return httpx.StatusError(http.StatusInternalServerError, "internal authentication error")
	}
}
