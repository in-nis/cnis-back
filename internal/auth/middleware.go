package auth

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
	"github.com/in-nis/cnis-back/internal/config"
)

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
            return
        }

        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header"})
            return
        }

		jwtSecret := []byte(cfg.JWT_SECRET)

        tokenStr := parts[1]
        token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
            return jwtSecret, nil
        })
        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            return
        }

        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid claims"})
            return
        }

        // Attach email to context
        c.Set("email", claims["email"])
        c.Next()
    }
}