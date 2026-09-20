package router

import (
	"net/http"
	"strings"
	"time"

	"data-server/internal/dashboard"
	"data-server/internal/database"
	"data-server/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

type authClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func Login(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := strings.TrimSpace(viper.GetString("jwt.secret"))
		if secret == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "jwt secret is not configured"})
			return
		}
		var request struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if database.GetDatabase() == nil {
			store.RecordAudit(request.Username, "auth.login", request.Username, "unavailable", nil)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not initialized"})
			return
		}
		var user model.User
		err := database.FindOne("users", bson.M{"username": request.Username}).Decode(&user)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)) != nil {
			store.RecordAudit(request.Username, "auth.login", request.Username, "failed", nil)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
			return
		}
		expiresAt := time.Now().Add(time.Duration(viper.GetInt("jwt.expire_minutes")) * time.Minute)
		if expiresAt.Before(time.Now()) {
			expiresAt = time.Now().Add(time.Hour)
		}
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, authClaims{Username: user.Username, Role: user.Role, RegisteredClaims: jwt.RegisteredClaims{Subject: user.ID.Hex(), ExpiresAt: jwt.NewNumericDate(expiresAt), IssuedAt: jwt.NewNumericDate(time.Now())}}).SignedString([]byte(secret))
		if err != nil {
			store.RecordAudit(user.Username, "auth.login", user.Username, "failed", map[string]any{"reason": "token generation"})
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
			return
		}
		store.RecordAudit(user.Username, "auth.login", user.Username, "success", nil)
		c.JSON(http.StatusOK, gin.H{"token": token, "expires_at": expiresAt.Unix(), "user": gin.H{"username": user.Username, "role": user.Role}})
	}
}

func Logout(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, ok := c.Get("username")
		if !ok {
			actor = "unknown"
		}
		username, ok := actor.(string)
		if !ok || username == "" {
			username = "unknown"
		}
		store.RecordAudit(username, "auth.logout", username, "success", nil)
		c.SetCookie("auth-token", "", -1, "/", "", false, true)
		c.Status(http.StatusNoContent)
	}
}

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := strings.TrimSpace(viper.GetString("jwt.secret"))
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "jwt secret is not configured"})
			return
		}
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}
		token, err := jwt.ParseWithClaims(strings.TrimPrefix(header, "Bearer "), &authClaims{}, func(token *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization token"})
			return
		}
		claims, ok := token.Claims.(*authClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization claims"})
			return
		}
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}
