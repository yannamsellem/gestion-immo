package application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/infrastructure"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/infrastructure/memory"
	authhttp "github.com/yannamsellem/gestion-immo/internal/modules/auth/interfaces/http"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

func TestSignup_RequiresSuperAdminApprovalBeforeLogin(t *testing.T) {
	users := memory.NewUserRepo()
	refreshTokens := memory.NewRefreshTokenRepo()
	mcpTokens := memory.NewMCPTokenRepo()
	uow := memory.NewUnitOfWork(users, refreshTokens, mcpTokens)
	hasher := infrastructure.NewArgon2Hasher()
	authority, err := domain.GenerateTokenAuthority("kid_unit_test")
	if err != nil {
		t.Fatal(err)
	}
	jwtSvc, err := infrastructure.NewJwtService(authority, "gestion-immo", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	svc := application.NewService(users, refreshTokens, mcpTokens, uow, jwtSvc, hasher)
	ctx := context.Background()

	// 1. Bootstrap Super Admin (you)
	superAdmin, err := svc.BootstrapSuperAdmin(ctx, "yann@investissimo.fr", secret.New("SuperAdminPass123!"))
	if err != nil || superAdmin.Role != domain.RoleSuperAdmin || superAdmin.Status != domain.StatusApproved {
		t.Fatalf("expected approved SUPER_ADMIN, got %+v (err=%v)", superAdmin, err)
	}

	// 2. New user signs up -> status is PENDING_APPROVAL
	applicant, err := svc.Signup(ctx, application.SignupCommand{
		FullName: "Collaborateur Immo",
		Email:    "collab@investissimo.fr",
		Password: secret.New("CollabPass123!"),
	})
	if err != nil || applicant.Status != domain.StatusPendingApproval || applicant.Role != domain.RoleManager {
		t.Fatalf("expected PENDING_APPROVAL MANAGER, got %+v (err=%v)", applicant, err)
	}

	// 3. Login before approval MUST fail with ErrAccountPendingApproval
	_, err = svc.Login(ctx, application.LoginCommand{
		Email:    "collab@investissimo.fr",
		Password: secret.New("CollabPass123!"),
	})
	if !errors.Is(err, domain.ErrAccountNotApproved) {
		t.Fatalf("expected ErrAccountNotApproved when logging in before approval, got %v", err)
	}

	// 4. Non-super-admin cannot approve users
	_, err = svc.ApproveUser(ctx, applicant.ID, applicant.ID)
	if !errors.Is(err, domain.ErrForbiddenSuperAdminOnly) {
		t.Fatalf("expected ErrForbiddenSuperAdminOnly, got %v", err)
	}

	// 5. Super Admin approves the user -> Login now succeeds
	approved, err := svc.ApproveUser(ctx, superAdmin.ID, applicant.ID)
	if err != nil || approved.Status != domain.StatusApproved {
		t.Fatalf("expected APPROVED status after super admin approval, got %+v (err=%v)", approved, err)
	}
	tokens, err := svc.Login(ctx, application.LoginCommand{
		Email:    "collab@investissimo.fr",
		Password: secret.New("CollabPass123!"),
	})
	if err != nil || tokens.AccessToken.Value() == "" {
		t.Fatalf("expected login to succeed after approval, got err=%v", err)
	}
}

func TestRefreshTokenRotation_DetectsReuseAndRevokesFamily(t *testing.T) {
	users := memory.NewUserRepo()
	refreshTokens := memory.NewRefreshTokenRepo()
	mcpTokens := memory.NewMCPTokenRepo()
	uow := memory.NewUnitOfWork(users, refreshTokens, mcpTokens)
	hasher := infrastructure.NewArgon2Hasher()
	authority, err := domain.GenerateTokenAuthority("kid_rotation_test")
	if err != nil {
		t.Fatal(err)
	}
	jwtSvc, err := infrastructure.NewJwtService(authority, "gestion-immo", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	svc := application.NewService(users, refreshTokens, mcpTokens, uow, jwtSvc, hasher)
	ctx := context.Background()

	if _, err := svc.BootstrapSuperAdmin(ctx, "yann@investissimo.fr", secret.New("SuperAdminPass123!")); err != nil {
		t.Fatal(err)
	}

	// 1. Initial login issues RT1
	initialTokens, err := svc.Login(ctx, application.LoginCommand{
		Email:    "yann@investissimo.fr",
		Password: secret.New("SuperAdminPass123!"),
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	rt1 := initialTokens.RefreshToken

	// 2. First refresh with RT1 succeeds and rotates to RT2
	rotatedTokens1, err := svc.Refresh(ctx, rt1)
	if err != nil {
		t.Fatalf("first refresh failed: %v", err)
	}
	rt2 := rotatedTokens1.RefreshToken
	if rt2.Value() == "" || rt2.Value() == rt1.Value() {
		t.Fatalf("expected new rotated refresh token, got %q", rt2.Value())
	}

	// 3. Legitimate refresh with RT2 succeeds and rotates to RT3
	rotatedTokens2, err := svc.Refresh(ctx, rt2)
	if err != nil {
		t.Fatalf("second refresh with RT2 failed: %v", err)
	}
	rt3 := rotatedTokens2.RefreshToken

	// 4. Replay of already-rotated RT1 triggers Reuse Detection -> revokes entire token family (including RT3!)
	_, err = svc.Refresh(ctx, rt1)
	if !errors.Is(err, domain.ErrRefreshTokenReuseDetected) {
		t.Fatalf("expected ErrRefreshTokenReuseDetected on RT1 replay, got %v", err)
	}

	// 5. Subsequent use of RT3 must now fail because the family was revoked upon RT1 reuse
	_, err = svc.Refresh(ctx, rt3)
	if !errors.Is(err, domain.ErrRefreshTokenReuseDetected) {
		t.Fatalf("expected RT3 to be revoked after family compromise, got %v", err)
	}
}

func TestAuthHTTPAndMCPWorkflow_EndToEnd(t *testing.T) {
	users := memory.NewUserRepo()
	refreshTokens := memory.NewRefreshTokenRepo()
	mcpTokens := memory.NewMCPTokenRepo()
	uow := memory.NewUnitOfWork(users, refreshTokens, mcpTokens)
	hasher := infrastructure.NewArgon2Hasher()
	authority, err := domain.GenerateTokenAuthority("kid_e2e_test")
	if err != nil {
		t.Fatal(err)
	}
	jwtSvc, err := infrastructure.NewJwtService(authority, "gestion-immo", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	svc := application.NewService(users, refreshTokens, mcpTokens, uow, jwtSvc, hasher)
	handler := authhttp.NewHandler(svc, jwtSvc)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	ctx := context.Background()
	_, err = svc.BootstrapSuperAdmin(ctx, "yann@investissimo.fr", secret.New("SuperAdminPass123!"))
	if err != nil {
		t.Fatal(err)
	}

	// 1. POST /api/v1/auth/signup -> 202 Accepted
	signupBody := `{"full_name":"Gestionnaire Test","email":"mgr@investissimo.fr","password":"ManagerPassword123!"}`
	reqSignup := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBufferString(signupBody))
	recSignup := httptest.NewRecorder()
	mux.ServeHTTP(recSignup, reqSignup)
	if recSignup.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted on signup, got %d: %s", recSignup.Code, recSignup.Body.String())
	}
	var pendingUser application.UserDTO
	if err := json.Unmarshal(recSignup.Body.Bytes(), &pendingUser); err != nil {
		t.Fatal(err)
	}
	if pendingUser.Status != domain.StatusPendingApproval {
		t.Fatalf("expected PENDING_APPROVAL, got %s", pendingUser.Status)
	}

	// 2. POST /api/v1/auth/login before approval -> 403 Forbidden
	loginBody := `{"email":"mgr@investissimo.fr","password":"ManagerPassword123!"}`
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	recLogin := httptest.NewRecorder()
	mux.ServeHTTP(recLogin, reqLogin)
	if recLogin.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden before approval, got %d", recLogin.Code)
	}

	// 3. SuperAdmin logs in and approves user via HTTP
	adminLoginBody := `{"email":"yann@investissimo.fr","password":"SuperAdminPass123!"}`
	reqAdminLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(adminLoginBody))
	recAdminLogin := httptest.NewRecorder()
	mux.ServeHTTP(recAdminLogin, reqAdminLogin)
	if recAdminLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for super admin login, got %d", recAdminLogin.Code)
	}
	var adminToks struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.Unmarshal(recAdminLogin.Body.Bytes(), &adminToks)

	reqApprove := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+pendingUser.ID+"/approve", nil)
	reqApprove.Header.Set("Authorization", "Bearer "+adminToks.AccessToken)
	recApprove := httptest.NewRecorder()
	mux.ServeHTTP(recApprove, reqApprove)
	if recApprove.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on approve, got %d: %s", recApprove.Code, recApprove.Body.String())
	}

	// 4. Approved manager logs in -> 200 OK, cannot call /api/v1/admin/users/pending -> 403 Forbidden
	reqMgrLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	recMgrLogin := httptest.NewRecorder()
	mux.ServeHTTP(recMgrLogin, reqMgrLogin)
	if recMgrLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK after approval, got %d", recMgrLogin.Code)
	}
	var mgrToks struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.Unmarshal(recMgrLogin.Body.Bytes(), &mgrToks)

	reqForbiddenAdmin := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/pending", nil)
	reqForbiddenAdmin.Header.Set("Authorization", "Bearer "+mgrToks.AccessToken)
	recForbiddenAdmin := httptest.NewRecorder()
	mux.ServeHTTP(recForbiddenAdmin, reqForbiddenAdmin)
	if recForbiddenAdmin.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for MANAGER on admin route, got %d", recForbiddenAdmin.Code)
	}

	// 5. Create and validate MCP token, then reject user -> MCP & refresh blocked
	mcpTok, err := svc.CreateMCPToken(ctx, pendingUser.ID, "antigravity")
	if err != nil {
		t.Fatalf("expected MCP token creation to succeed, got %v", err)
	}
	if _, err := svc.ValidateMCPBearer(ctx, mcpTok.Value()); err != nil {
		t.Fatalf("expected MCP token validation to succeed, got %v", err)
	}

	reqReject := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+pendingUser.ID+"/reject", nil)
	reqReject.Header.Set("Authorization", "Bearer "+adminToks.AccessToken)
	recReject := httptest.NewRecorder()
	mux.ServeHTTP(recReject, reqReject)
	if recReject.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on reject, got %d", recReject.Code)
	}

	if _, err := svc.ValidateMCPBearer(ctx, mcpTok.Value()); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for revoked MCP token of rejected user, got %v", err)
	}
	storedMCP, err := mcpTokens.FindMCPTokenByHash(ctx, hasher.HashToken(mcpTok))
	if err != nil || storedMCP.IsActive() {
		t.Fatalf("expected MCP token to be marked revoked in repository after RejectUser, got active=%v err=%v", storedMCP != nil && storedMCP.IsActive(), err)
	}
}

func TestRejectUser_RevokesAllActiveMCPTokensAndBlocksApprovedMiddleware(t *testing.T) {
	users := memory.NewUserRepo()
	refreshTokens := memory.NewRefreshTokenRepo()
	mcpTokens := memory.NewMCPTokenRepo()
	uow := memory.NewUnitOfWork(users, refreshTokens, mcpTokens)
	hasher := infrastructure.NewArgon2Hasher()
	authority, err := domain.GenerateTokenAuthority("kid_reject_mcp_test")
	if err != nil {
		t.Fatal(err)
	}
	jwtSvc, err := infrastructure.NewJwtService(authority, "gestion-immo", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	svc := application.NewService(users, refreshTokens, mcpTokens, uow, jwtSvc, hasher)
	ctx := context.Background()

	superAdmin, err := svc.BootstrapSuperAdmin(ctx, "yann@investissimo.fr", secret.New("SuperAdminPass123!"))
	if err != nil {
		t.Fatal(err)
	}
	user, err := svc.Signup(ctx, application.SignupCommand{
		FullName: "Gestionnaire MCP",
		Email:    "mcpuser@investissimo.fr",
		Password: secret.New("ValidPassword123!"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ApproveUser(ctx, superAdmin.ID, user.ID); err != nil {
		t.Fatal(err)
	}

	tok1, err := svc.CreateMCPToken(ctx, user.ID, "cli-1")
	if err != nil {
		t.Fatal(err)
	}
	tok2, err := svc.CreateMCPToken(ctx, user.ID, "cli-2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ValidateMCPBearer(ctx, tok1.Value()); err != nil {
		t.Fatalf("expected tok1 valid before rejection, got %v", err)
	}
	if _, err := svc.ValidateMCPBearer(ctx, tok2.Value()); err != nil {
		t.Fatalf("expected tok2 valid before rejection, got %v", err)
	}

	if _, err := svc.RejectUser(ctx, superAdmin.ID, user.ID); err != nil {
		t.Fatalf("RejectUser failed: %v", err)
	}

	if _, err := svc.ValidateMCPBearer(ctx, tok1.Value()); err == nil {
		t.Fatal("expected tok1 validation to fail immediately after RejectUser")
	}
	if _, err := svc.ValidateMCPBearer(ctx, tok2.Value()); err == nil {
		t.Fatal("expected tok2 validation to fail immediately after RejectUser")
	}
	if _, err := svc.RequireApprovedUser(ctx, user.ID); !errors.Is(err, domain.ErrAccountNotApproved) {
		t.Fatalf("expected ErrAccountNotApproved from RequireApprovedUser, got %v", err)
	}
}

func TestSignup_ValidatesPasswordLengthBounds(t *testing.T) {
	users := memory.NewUserRepo()
	refreshTokens := memory.NewRefreshTokenRepo()
	mcpTokens := memory.NewMCPTokenRepo()
	uow := memory.NewUnitOfWork(users, refreshTokens, mcpTokens)
	hasher := infrastructure.NewArgon2Hasher()
	authority, err := domain.GenerateTokenAuthority("kid_pw_bounds_test")
	if err != nil {
		t.Fatal(err)
	}
	jwtSvc, err := infrastructure.NewJwtService(authority, "gestion-immo", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	svc := application.NewService(users, refreshTokens, mcpTokens, uow, jwtSvc, hasher)
	ctx := context.Background()

	// Shorter than 8 bytes -> ErrValidation
	_, err = svc.Signup(ctx, application.SignupCommand{
		FullName: "Too Short",
		Email:    "short@investissimo.fr",
		Password: secret.New("1234567"),
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation for 7-byte password, got %v", err)
	}

	// Longer than 128 bytes -> ErrValidation
	_, err = svc.Signup(ctx, application.SignupCommand{
		FullName: "Too Long",
		Email:    "long@investissimo.fr",
		Password: secret.New(strings.Repeat("a", 129)),
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation for 129-byte password, got %v", err)
	}
}

func TestAuth_UUIDv7IdentitiesAnd256BitTokenFormats(t *testing.T) {
	users := memory.NewUserRepo()
	refreshTokens := memory.NewRefreshTokenRepo()
	mcpTokens := memory.NewMCPTokenRepo()
	uow := memory.NewUnitOfWork(users, refreshTokens, mcpTokens)
	hasher := infrastructure.NewArgon2Hasher()
	authority, err := domain.GenerateTokenAuthority("kid_uuidv7_test")
	if err != nil {
		t.Fatal(err)
	}
	jwtSvc, err := infrastructure.NewJwtService(authority, "gestion-immo", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	svc := application.NewService(users, refreshTokens, mcpTokens, uow, jwtSvc, hasher)
	ctx := context.Background()

	superAdmin, err := svc.BootstrapSuperAdmin(ctx, "admin@investissimo.fr", secret.New("SuperAdminPass123!"))
	if err != nil {
		t.Fatalf("BootstrapSuperAdmin: %v", err)
	}
	if !id.IsValidUUID(superAdmin.ID) || superAdmin.ID[14] != '7' {
		t.Fatalf("expected UUIDv7 superAdmin.ID, got %q", superAdmin.ID)
	}

	applicant, err := svc.Signup(ctx, application.SignupCommand{
		FullName: "Manager UUID",
		Email:    "manager-uuid@investissimo.fr",
		Password: secret.New("ManagerPass123!"),
	})
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}
	if !id.IsValidUUID(applicant.ID) || applicant.ID[14] != '7' {
		t.Fatalf("expected UUIDv7 applicant.ID, got %q", applicant.ID)
	}

	if _, err := svc.ApproveUser(ctx, superAdmin.ID, "not-a-uuid"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation on malformed target user ID in ApproveUser, got %v", err)
	}
	if _, err := svc.RejectUser(ctx, superAdmin.ID, "not-a-uuid"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation on malformed target user ID in RejectUser, got %v", err)
	}
	if _, err := svc.CreateMCPToken(ctx, "not-a-uuid", "cli"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation on malformed user ID in CreateMCPToken, got %v", err)
	}

	if _, err := svc.ApproveUser(ctx, superAdmin.ID, applicant.ID); err != nil {
		t.Fatalf("ApproveUser: %v", err)
	}

	pair, err := svc.Login(ctx, application.LoginCommand{
		Email:    "manager-uuid@investissimo.fr",
		Password: secret.New("ManagerPass123!"),
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	rawRT := pair.RefreshToken.Value()
	if len(rawRT) != 64 || strings.Contains(rawRT, "_") {
		t.Fatalf("expected unprefixed 64-char hex refresh token, got len=%d %q", len(rawRT), rawRT)
	}
	rtRec, err := refreshTokens.FindRefreshTokenByHash(ctx, hasher.HashToken(pair.RefreshToken))
	if err != nil {
		t.Fatalf("FindRefreshTokenByHash: %v", err)
	}
	if !id.IsValidUUID(rtRec.ID) || rtRec.ID[14] != '7' {
		t.Fatalf("expected UUIDv7 refresh token record ID, got %q", rtRec.ID)
	}
	if !id.IsValidUUID(rtRec.FamilyID) || rtRec.FamilyID[14] != '7' {
		t.Fatalf("expected UUIDv7 refresh token family ID, got %q", rtRec.FamilyID)
	}

	mcpTok, err := svc.CreateMCPToken(ctx, applicant.ID, "antigravity")
	if err != nil {
		t.Fatalf("CreateMCPToken: %v", err)
	}
	if !strings.HasPrefix(mcpTok.Value(), "mcp_live_") || len(mcpTok.Value()) != 73 {
		t.Fatalf("expected 73-char mcp_live_ token, got len=%d %q", len(mcpTok.Value()), mcpTok.Value())
	}
	mcpRec, err := mcpTokens.FindMCPTokenByHash(ctx, hasher.HashToken(mcpTok))
	if err != nil {
		t.Fatalf("FindMCPTokenByHash: %v", err)
	}
	if !id.IsValidUUID(mcpRec.ID) || mcpRec.ID[14] != '7' {
		t.Fatalf("expected UUIDv7 MCP token record ID, got %q", mcpRec.ID)
	}
}
