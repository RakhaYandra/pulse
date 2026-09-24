package auth

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/RakhaYandra/pulse/pkg/response"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

func secret() string {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "dev-secret-change-me"
	}
	return s
}

func (h *Handler) Register(c *gin.Context) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, http.StatusBadRequest, "invalid body")
		return
	}
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.Email == "" || !strings.Contains(in.Email, "@") {
		response.Err(c, http.StatusBadRequest, "email invalid")
		return
	}
	if len(in.Password) < 8 {
		response.Err(c, http.StatusBadRequest, "password min 8 chars")
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		response.Err(c, http.StatusBadRequest, "name required")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "hash failed")
		return
	}
	id := uuid.NewString()
	_, err = h.db.Exec(`INSERT INTO users(id,email,password_hash,name) VALUES($1,$2,$3,$4)`,
		id, in.Email, string(hash), strings.TrimSpace(in.Name))
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			response.Err(c, http.StatusConflict, "email already registered")
			return
		}
		response.Err(c, http.StatusInternalServerError, "register failed")
		return
	}
	token, err := sign(id)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "token failed")
		return
	}
	response.OK(c, gin.H{"token": token, "user": User{ID: id, Email: in.Email, Name: strings.TrimSpace(in.Name)}})
}

func (h *Handler) Login(c *gin.Context) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, http.StatusBadRequest, "invalid body")
		return
	}
	var u User
	var hash string
	err := h.db.QueryRow(`SELECT id,email,password_hash,name FROM users WHERE email=$1`,
		strings.TrimSpace(strings.ToLower(in.Email))).Scan(&u.ID, &u.Email, &hash, &u.Name)
	if err != nil {
		response.Err(c, http.StatusUnauthorized, "email/password salah")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)); err != nil {
		response.Err(c, http.StatusUnauthorized, "email/password salah")
		return
	}
	token, err := sign(u.ID)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "token failed")
		return
	}
	response.OK(c, gin.H{"token": token, "user": u})
}

func (h *Handler) Me(c *gin.Context) {
	uid, _ := c.Get("userID")
	var u User
	err := h.db.QueryRow(`SELECT id,email,name FROM users WHERE id=$1`, uid.(string)).Scan(&u.ID, &u.Email, &u.Name)
	if err != nil {
		response.Err(c, http.StatusNotFound, "user not found")
		return
	}
	response.OK(c, u)
}

func sign(userID string) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})
	return t.SignedString([]byte(secret()))
}

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			response.Err(c, http.StatusUnauthorized, "missing bearer token")
			c.Abort()
			return
		}
		tok, err := jwt.Parse(strings.TrimPrefix(h, "Bearer "), func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("bad alg")
			}
			return []byte(secret()), nil
		})
		if err != nil || !tok.Valid {
			response.Err(c, http.StatusUnauthorized, "invalid token")
			c.Abort()
			return
		}
		sub, _ := tok.Claims.(jwt.MapClaims)["sub"].(string)
		if sub == "" {
			response.Err(c, http.StatusUnauthorized, "invalid claims")
			c.Abort()
			return
		}
		c.Set("userID", sub)
		c.Next()
	}
}
