package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lms/server/internal/config"
	"github.com/lms/server/internal/model"
	"github.com/lms/server/internal/repository"
	"github.com/lms/server/internal/runtimecfg"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	userRepo *repository.UserRepo
	cfg      *config.Config
	rtEngine *runtimecfg.Engine
}

func NewService(userRepo *repository.UserRepo, cfg *config.Config, rtEngine *runtimecfg.Engine) *Service {
	return &Service{userRepo: userRepo, cfg: cfg, rtEngine: rtEngine}
}

type RegisterInput struct {
	Username string `json:"username" binding:"required,min=2,max=64"`
	Password string `json:"password" binding:"required,min=6,max=128"`
	Email    string `json:"email"`
}

type LoginInput struct {
	Username      string `json:"username" binding:"required"`
	Password      string `json:"password" binding:"required"`
	CaptchaID     string `json:"captcha_id"`
	CaptchaAnswer string `json:"captcha_answer"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrCaptchaRequired    = errors.New("captcha required")
	ErrCaptchaInvalid     = errors.New("invalid captcha")
	ErrBlocked            = errors.New("too many attempts, try again later")
)

func (s *Service) Register(input RegisterInput) (*AuthResponse, error) {
	existing, err := s.userRepo.FindByUsername(input.Username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("username already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     input.Username,
		PasswordHash: string(hash),
		Email:        input.Email,
		Role:         "user",
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: user}, nil
}

func (s *Service) Login(input LoginInput) (*AuthResponse, error) {
	user, err := s.userRepo.FindByUsername(input.Username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: user}, nil
}

func (s *Service) jwtExpireHours() int {
	if s.rtEngine != nil {
		if v := s.rtEngine.GetSet("JWT"); v != nil {
			if h, err := strconv.Atoi(v["EXPIRETIME"]); err == nil && h > 0 {
				return h
			}
		}
	}
	return s.cfg.JWT.ExpireHour
}

func (s *Service) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Duration(s.jwtExpireHours()) * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}
