// Package auth 提供认证相关的 DCI 上下文（注册、登录）。
package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lms/server/internal/config"
	"github.com/lms/server/internal/dci/data"
	"github.com/lms/server/internal/dci/tx"
	"github.com/lms/server/internal/middleware"
	"github.com/lms/server/internal/model"
	"github.com/lms/server/internal/runtimecfg"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ---- 公共错误 ----

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrCaptchaRequired    = errors.New("captcha required")
	ErrCaptchaInvalid     = errors.New("invalid captcha")
	ErrBlocked            = errors.New("too many attempts, try again later")
)

// ---- RegisterContext ----

// RegisterContext 处理用户注册的 DCI 上下文。
type RegisterContext struct {
	db       *gorm.DB
	userRepo data.UserRepo
	cfg      *config.Config
	rtEngine *runtimecfg.Engine

	Username string
	Password string
	Email    string

	result *model.User
	token  string
}

func NewRegisterContext(db *gorm.DB, userRepo data.UserRepo, cfg *config.Config, rtEngine *runtimecfg.Engine, username, password, email string) *RegisterContext {
	return &RegisterContext{
		db:       db,
		userRepo: userRepo,
		cfg:      cfg,
		rtEngine: rtEngine,
		Username: username,
		Password: password,
		Email:    email,
	}
}

// Execute 执行注册交互：(1) 查重 (2) 哈希密码 (3) 事务内创建用户 (4) 签发 JWT。
func (c *RegisterContext) Execute() (*model.User, string, error) {
	existing, err := c.userRepo.FindByUsername(c.db, c.Username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", err
	}
	if existing != nil {
		return nil, "", errors.New("username already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	u := tx.NewUnit(c.db)
	if err := u.Begin(); err != nil {
		return nil, "", err
	}

	user := &model.User{
		Username:     c.Username,
		PasswordHash: string(hash),
		Email:        c.Email,
		Role:         model.RoleUser,
	}
	if err := c.userRepo.Create(u, user); err != nil {
		u.Rollback()
		return nil, "", err
	}

	if err := u.Commit(); err != nil {
		return nil, "", err
	}

	token, err := c.generateToken(user)
	if err != nil {
		return nil, "", err
	}

	c.result = user
	c.token = token
	return user, token, nil
}

func (c *RegisterContext) jwtExpireHours() int {
	if c.rtEngine != nil {
		if v := c.rtEngine.GetSet(runtimecfg.TargetJWT); v != nil {
			if h, err := strconv.Atoi(v[runtimecfg.FieldExpireTime]); err == nil && h > 0 {
				return h
			}
		}
	}
	return c.cfg.JWT.ExpireHour
}

func (c *RegisterContext) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		middleware.JWTClaimSub:  user.ID,
		middleware.JWTClaimRole: user.Role,
		"iat":                   time.Now().Unix(),
		"exp":                   time.Now().Add(time.Duration(c.jwtExpireHours()) * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.cfg.JWT.Secret))
}

// ---- LoginContext ----

// LoginContext 处理用户登录的 DCI 上下文（只读，无需事务）。
type LoginContext struct {
	db       *gorm.DB
	userRepo data.UserRepo
	cfg      *config.Config
	rtEngine *runtimecfg.Engine

	Username string
	Password string

	result *model.User
	token  string
}

func NewLoginContext(db *gorm.DB, userRepo data.UserRepo, cfg *config.Config, rtEngine *runtimecfg.Engine, username, password string) *LoginContext {
	return &LoginContext{
		db:       db,
		userRepo: userRepo,
		cfg:      cfg,
		rtEngine: rtEngine,
		Username: username,
		Password: password,
	}
}

// Execute 执行登录交互：(1) 查用户 (2) 验密码 (3) 签发 JWT。
func (c *LoginContext) Execute() (*model.User, string, error) {
	user, err := c.userRepo.FindByUsername(c.db, c.Username)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(c.Password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := c.generateToken(user)
	if err != nil {
		return nil, "", err
	}

	c.result = user
	c.token = token
	return user, token, nil
}

func (c *LoginContext) jwtExpireHours() int {
	if c.rtEngine != nil {
		if v := c.rtEngine.GetSet(runtimecfg.TargetJWT); v != nil {
			if h, err := strconv.Atoi(v[runtimecfg.FieldExpireTime]); err == nil && h > 0 {
				return h
			}
		}
	}
	return c.cfg.JWT.ExpireHour
}

func (c *LoginContext) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		middleware.JWTClaimSub:  user.ID,
		middleware.JWTClaimRole: user.Role,
		"iat":                   time.Now().Unix(),
		"exp":                   time.Now().Add(time.Duration(c.jwtExpireHours()) * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.cfg.JWT.Secret))
}
