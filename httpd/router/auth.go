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

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if role, exists := c.Get("role"); !exists || role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "administrator access required"})
			return
		}
		c.Next()
	}
}

func UserList() gin.HandlerFunc {
	return func(c *gin.Context) {
		if database.GetDatabase() == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not initialized"})
			return
		}
		var users []model.User
		if err := database.FindAll("users", bson.M{}, &users); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to load users"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": users})
	}
}

func UserCreate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if database.GetDatabase() == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not initialized"})
			return
		}
		var request struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
			Email    string `json:"email"`
			Role     string `json:"role"`
		}
		if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Username) == "" || len(request.Password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username is required and password must contain at least 6 characters"})
			return
		}
		request.Username = strings.TrimSpace(request.Username)
		request.Role = strings.TrimSpace(request.Role)
		if request.Role == "" {
			request.Role = "user"
		}
		if request.Role != "admin" && request.Role != "user" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role must be admin or user"})
			return
		}
		if database.FindOne("users", bson.M{"username": request.Username}).Err() == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
			return
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to secure password"})
			return
		}
		now := time.Now()
		user := model.User{Username: request.Username, Password: string(hashed), Email: strings.TrimSpace(request.Email), Role: request.Role, CreatedAt: now, UpdatedAt: now}
		if _, err := database.InsertOne("users", user); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to create user"})
			return
		}
		user.Password = ""
		c.JSON(http.StatusCreated, user)
	}
}

func UserDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
		if database.GetDatabase() == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not initialized"})
			return
		}
		username := c.Param("username")
		actor, _ := c.Get("username")
		if username == actor {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete current user"})
			return
		}
		result, err := database.DeleteOne("users", bson.M{"username": username})
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to delete user"})
			return
		}
		if result.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

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
		c.SetCookie("auth-token", token, int(time.Until(expiresAt).Seconds()), "/", "", false, true)
		store.RecordAudit(user.Username, "auth.login", user.Username, "success", nil)
		c.JSON(http.StatusOK, gin.H{"token": token, "expires_at": expiresAt.Unix(), "user": gin.H{"username": user.Username, "role": user.Role}})
	}
}

func Me() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, _ := c.Get("username")
		role, _ := c.Get("role")
		c.JSON(http.StatusOK, gin.H{"username": username, "role": role})
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
		tokenString := ""
		if strings.HasPrefix(header, "Bearer ") {
			tokenString = strings.TrimPrefix(header, "Bearer ")
		} else if cookie, err := c.Cookie("auth-token"); err == nil {
			tokenString = cookie
		}
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}
		token, err := jwt.ParseWithClaims(tokenString, &authClaims{}, func(token *jwt.Token) (any, error) {
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
