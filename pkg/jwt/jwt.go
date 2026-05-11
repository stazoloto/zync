package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TTL         = 24 * time.Hour
	verifiedTTL = 10 * time.Minute
)

type TokenManager interface {
	Generate(userID, fullName, role string) (string, error)
	Validate(token string) (userID, fullName, role string, err error)
	GenerateVerified(email string) (string, error)
	ValidateVerified(token string) (email string, err error)
}

type Manager struct {
	secret []byte
}

func NewManager(secret []byte) *Manager {
	return &Manager{secret: secret}
}

type claims struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type verifiedClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (m *Manager) Generate(userID, fullName, role string) (string, error) {
	c := claims{
		UserID:   userID,
		FullName: fullName,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}

func (m *Manager) Validate(token string) (string, string, string, error) {
	parsed, err := jwt.ParseWithClaims(token, &claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return "", "", "", err
	}
	c, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid {
		return "", "", "", errors.New("invalid token")
	}
	return c.UserID, c.FullName, c.Role, nil
}

func (m *Manager) GenerateVerified(email string) (string, error) {
	c := verifiedClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(verifiedTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}

func (m *Manager) ValidateVerified(token string) (string, error) {
	parsed, err := jwt.ParseWithClaims(token, &verifiedClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return "", err
	}
	c, ok := parsed.Claims.(*verifiedClaims)
	if !ok || !parsed.Valid {
		return "", errors.New("invalid token")
	}
	return c.Email, nil
}
