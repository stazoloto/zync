package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
	"unicode"

	"github.com/stazoloto/zync/internal/repository"
	"github.com/stazoloto/zync/pkg/email"
	"github.com/stazoloto/zync/pkg/jwt"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	users       repository.UserRepository
	verifyCodes repository.VerifyCodeRepository
	tokens      jwt.TokenManager
	email       email.Sender
	logger      *zap.Logger
}

func NewAuthService(
	users repository.UserRepository,
	verifyCodes repository.VerifyCodeRepository,
	tokens jwt.TokenManager,
	email email.Sender,
	logger *zap.Logger,
) UserService {
	return &authService{
		users:       users,
		verifyCodes: verifyCodes,
		tokens:      tokens,
		email:       email,
		logger:      logger,
	}
}

// Register создаёт нового пользователя и возвращает JWT-токен для него.
func (s *authService) Register(ctx context.Context, verifiedToken, firstName, lastName, password string) (string, error) {
	email, err := s.tokens.ValidateVerified(verifiedToken)
	if err != nil {
		return "", errors.New("invalid or expired verification token")
	}

	if err = validatePassword(password); err != nil {
		return "", err
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	userID, err := s.users.Register(email, firstName, lastName, string(hashedPassword))
	if err != nil {
		s.logger.Error("registration failed", zap.String("email", email), zap.Error(err))
		return "", err
	}

	return s.tokens.Generate(userID, firstName+" "+lastName, "user")
}

// Login проверяет учетные данные пользователя и возвращает JWT-токен при успешной аутентификации.
func (s *authService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.users.GetByEmail(email)
	if err != nil {
		s.logger.Warn("login failed: user not found", zap.String("email", email), zap.Error(err))
		return "", errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.logger.Warn("login failed: invalid password", zap.String("email", email), zap.Error(err))
		return "", errors.New("invalid email or password")
	}

	return s.tokens.Generate(user.ID, user.FullName(), user.Role)
}

// SendVerificationCode генерирует код, сохраняет его в репозитории и отправляет пользователю на email.
func (s *authService) SendVerificationCode(ctx context.Context, email string) error {
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	err := s.verifyCodes.SaveCode(ctx, email, code, 10*time.Minute)
	if err != nil {
		s.logger.Error("failed to save verification code", zap.String("email", email), zap.Error(err))
		return err
	}
	if err = s.email.SendVerificationCode(email, code); err != nil {
		s.logger.Error("failed to send verification email", zap.String("email", email), zap.Error(err))
		return err
	}
	return nil
}

// VerifyCode проверяет код из email и возвращает JWT-токен для пользователя.
func (s *authService) VerifyCode(ctx context.Context, email, code string) (string, error) {
	storedCode, err := s.verifyCodes.GetCode(ctx, email)
	if err != nil {
		s.logger.Warn("verification code not found", zap.String("email", email), zap.Error(err))
		return "", errors.New("invalid verification code")
	}
	if code != storedCode {
		s.logger.Warn("verification code mismatch", zap.String("email", email), zap.String("provided", code), zap.String("expected", storedCode))
		return "", errors.New("invalid verification code")
	}

	if err = s.verifyCodes.DeleteCode(ctx, email); err != nil {
		s.logger.Warn("failed to delete verify code", zap.String("email", email), zap.Error(err))
	}
	return s.tokens.GenerateVerified(email)
}

func (s *authService) EmailExists(_ context.Context, email string) (bool, error) {
	u, err := s.users.GetByEmail(email)
	if err != nil {
		return false, err
	}
	return u != nil, nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("пароль должен содержать минимум 8 символов")
	}

	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}

	if !hasUpper {
		return errors.New("пароль должен содержать хотя бы одну заглавную букву")
	}
	if !hasLower {
		return errors.New("пароль должен содержать хотя бы одну строчную букву")
	}
	if !hasDigit {
		return errors.New("пароль должен содержать хотя бы одну цифру")
	}
	return nil
}
