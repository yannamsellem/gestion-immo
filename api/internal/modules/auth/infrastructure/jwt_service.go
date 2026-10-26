package infrastructure

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

// JwtService is the strictly isolated service responsible for generating and validating JWT access tokens
// using an asymmetric Ed25519 TokenAuthority (private key, public key, and kid).
type JwtService struct {
	authority *domain.TokenAuthority
	issuer    string
	accessTTL time.Duration
}

type accessTokenClaims struct {
	Email string          `json:"email"`
	Role  domain.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// NewJwtService constructs an isolated JwtService backed by a domain.TokenAuthority using EdDSA (Ed25519).
func NewJwtService(authority *domain.TokenAuthority, issuer string, accessTTL time.Duration) (*JwtService, error) {
	if authority == nil || authority.KID() == "" {
		return nil, errors.New("jwt_service: valid TokenAuthority with kid is required")
	}
	trimmedIssuer := strings.TrimSpace(issuer)
	if trimmedIssuer == "" {
		return nil, errors.New("jwt_service: issuer must not be empty")
	}
	if accessTTL <= 0 {
		return nil, errors.New("jwt_service: accessTTL must be positive")
	}
	return &JwtService{
		authority: authority,
		issuer:    trimmedIssuer,
		accessTTL: accessTTL,
	}, nil
}

// Authority returns the underlying domain.TokenAuthority.
func (s *JwtService) Authority() *domain.TokenAuthority {
	return s.authority
}

// GenerateAccessToken creates a signed EdDSA JWT access token carrying the TokenAuthority's kid header,
// wrapped in secret.String to prevent log leakage.
func (s *JwtService) GenerateAccessToken(userID string, email string, role domain.UserRole) (secret.String, error) {
	trimmedUserID := strings.TrimSpace(userID)
	trimmedEmail := strings.TrimSpace(email)
	if trimmedUserID == "" || trimmedEmail == "" || role == "" {
		return secret.String{}, errors.New("jwt_service: userID, email, and role are required")
	}

	now := time.Now().UTC()
	claims := accessTokenClaims{
		Email: trimmedEmail,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   trimmedUserID,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = s.authority.KID()

	signed, err := token.SignedString(s.authority.PrivateKey())
	if err != nil {
		return secret.String{}, fmt.Errorf("jwt_service: sign access token: %w", err)
	}

	return secret.New(signed), nil
}

// ValidateAccessToken verifies the EdDSA signature, kid header, algorithm, issuer, and expiration of a JWT access token.
func (s *JwtService) ValidateAccessToken(rawToken string) (*domain.Claims, error) {
	trimmed := strings.TrimSpace(rawToken)
	if trimmed == "" {
		return nil, domain.ErrInvalidToken
	}

	var parsedClaims accessTokenClaims
	token, err := jwt.ParseWithClaims(
		trimmed,
		&parsedClaims,
		func(t *jwt.Token) (any, error) {
			kid, ok := t.Header["kid"].(string)
			if !ok || strings.TrimSpace(kid) == "" || kid != s.authority.KID() {
				return nil, domain.ErrInvalidToken
			}
			return s.authority.PublicKey(), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithIssuer(s.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || token == nil || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	sub := strings.TrimSpace(parsedClaims.Subject)
	if sub == "" {
		return nil, domain.ErrInvalidToken
	}

	var iat, exp int64
	if parsedClaims.IssuedAt != nil {
		iat = parsedClaims.IssuedAt.Unix()
	}
	if parsedClaims.ExpiresAt != nil {
		exp = parsedClaims.ExpiresAt.Unix()
	}

	return &domain.Claims{
		UserID:    sub,
		Subject:   sub,
		Email:     parsedClaims.Email,
		Role:      parsedClaims.Role,
		Issuer:    parsedClaims.Issuer,
		IssuedAt:  iat,
		ExpiresAt: exp,
	}, nil
}
