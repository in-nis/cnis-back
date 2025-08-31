package auth

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
	"github.com/in-nis/cnis-back/internal/config"
)

func RefreshHandler(cfg *config.Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req struct {
            RefreshToken string `json:"refresh_token"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Missing refresh token"})
            return
        }

		jwtSecret := []byte(cfg.JWT_SECRET)

        token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, jwt.ErrSignatureInvalid
            }
            return jwtSecret, nil
        })
        if err != nil || !token.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
            return
        }

        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok || claims["type"] != "refresh" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token type"})
            return
        }

        email, ok := claims["email"].(string)
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid claims"})
            return
        }

        newAccessClaims := jwt.MapClaims{
            "email": email,
            "exp":   time.Now().Add(15 * time.Minute).Unix(),
        }
        newAccess := jwt.NewWithClaims(jwt.SigningMethodHS256, newAccessClaims)
        signedAccess, _ := newAccess.SignedString(jwtSecret)

        newRefreshClaims := jwt.MapClaims{
            "email": email,
            "exp":   time.Now().Add(7 * 24 * time.Hour).Unix(),
            "type":  "refresh",
        }
        newRefresh := jwt.NewWithClaims(jwt.SigningMethodHS256, newRefreshClaims)
        signedRefresh, _ := newRefresh.SignedString(jwtSecret)

        c.JSON(200, gin.H{
            "access_token":  signedAccess,
            "refresh_token": signedRefresh,
        })
    }
}