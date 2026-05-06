package handler

import (
	"net/http"
	"strings"
	"time"

	"ansmeee-ai-agent/internal/middleware"
	"ansmeee-ai-agent/internal/models"
	"ansmeee-ai-agent/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const jwtSecret = "ai-agent-secret-key-change-in-production"

// AuthHandler handles user registration and login.
type AuthHandler struct {
	db *gorm.DB
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register creates a new user account.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
		response.BadRequest(c, "email and password are required")
		return
	}
	if len(req.Password) < 6 {
		response.BadRequest(c, "password must be at least 6 characters")
		return
	}

	// Check if user already exists.
	var existing models.User
	if err := h.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		response.Fail(c, http.StatusConflict, response.CodeBadRequest, "email already registered")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.InternalError(c, "failed to hash password")
		return
	}

	user := models.User{
		UUID:     strings.ReplaceAll(uuid.New().String(), "-", ""),
		Email:    req.Email,
		Password: string(hash),
		Status:   1,
	}
	if err := h.db.Create(&user).Error; err != nil {
		response.InternalError(c, "failed to create user")
		return
	}

	token, err := generateJWT(user.ID, user.UUID)
	if err != nil {
		response.InternalError(c, "failed to generate token")
		return
	}

	response.OK(c, gin.H{
		"token":    token,
		"user_id":  user.UUID,
		"email":    user.Email,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login authenticates a user and returns a JWT.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
		response.BadRequest(c, "email and password are required")
		return
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "invalid email or password")
		return
	}
	if user.Status != 1 {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "account is disabled")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "invalid email or password")
		return
	}

	token, err := generateJWT(user.ID, user.UUID)
	if err != nil {
		response.InternalError(c, "failed to generate token")
		return
	}

	response.OK(c, gin.H{
		"token":    token,
		"user_id":  user.UUID,
		"email":    user.Email,
	})
}

// Me returns the current user info.
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetInt64(middleware.CtxUserID)
	userUUID := c.GetString(middleware.CtxUserUUID)
	response.OK(c, gin.H{
		"user_id": userUUID,
		"id":      userID,
	})
}

func generateJWT(userID int64, userUUID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"user_uuid": userUUID,
		"exp":       time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}
