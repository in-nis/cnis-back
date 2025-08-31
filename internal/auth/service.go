package auth

import (
    "context"
    "net/http"
	"encoding/json"
	"time"
	"log"

    "github.com/gin-gonic/gin"
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
	"github.com/in-nis/cnis-back/internal/config"
	"github.com/in-nis/cnis-back/internal/db"
	"github.com/golang-jwt/jwt/v5"
	"github.com/in-nis/cnis-back/internal/models"
)

var googleOauthConfig *oauth2.Config

func InitGoogle(cfg *config.Config) {
    googleOauthConfig = &oauth2.Config{
        RedirectURL:  "http://localhost:8000/auth/google/callback",
        ClientID:     cfg.GoogleClientID,
        ClientSecret: cfg.GoogleSecret,
        Scopes: []string{
			"openid",
            "https://www.googleapis.com/auth/userinfo.email",
            "https://www.googleapis.com/auth/userinfo.profile",
            "https://www.googleapis.com/auth/calendar",         
            "https://www.googleapis.com/auth/calendar.readonly", 
        },
        Endpoint: google.Endpoint,
    }
}

// @Summary      Login with Google 
// @Description  Responds with pong
// @Tags         auth
// @Produce      json
// @Success      200 {string} string "pong"
// @Router       /auth/google/login [get]
func GoogleLoginHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        url := googleOauthConfig.AuthCodeURL("state")
        c.Redirect(http.StatusTemporaryRedirect, url)
    }
}

// @Summary      Google Callback 
// @Description  Responds with pong
// @Tags         auth
// @Produce      json
// @Success      200 {string} string "pong"
// @Router       /auth/google/callback [get]
func GoogleCallbackHandler(cfg *config.Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        code := c.Query("code")
        token, err := googleOauthConfig.Exchange(context.Background(), code)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to exchange token"})
            return
        }

        // Fetch user info
        resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user info"})
            return
        }
        defer resp.Body.Close()

        var userInfo struct {
            Email string `json:"email"`
        }
        if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse user info"})
            return
        }

        // Save user to DB
        u := models.User{
            Email:        userInfo.Email,
            AccessToken:  token.AccessToken,
            RefreshToken: token.RefreshToken,
            TokenType:    token.TokenType,
            Expiry:       token.Expiry,
        }
        if err := db.SaveOrUpdateUser(context.Background(), u); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user"})
            return
        }

		jwtSecret := []byte(cfg.JWT_SECRET)
		log.Println(jwtSecret)

        // Create Access Token (short-lived)
		accessClaims := jwt.MapClaims{
			"email": u.Email,
			"exp":   time.Now().Add(15 * time.Minute).Unix(),
		}
		accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
		signedAccess, err := accessToken.SignedString(jwtSecret)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid"})
            return
		}

		// Create Refresh Token (long-lived)
		refreshClaims := jwt.MapClaims{
			"email": u.Email,
			"exp":   time.Now().Add(7 * 24 * time.Hour).Unix(),
			"type":  "refresh",
		}
		refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
		signedRefresh, _ := refreshToken.SignedString(jwtSecret)

		c.JSON(200, gin.H{
			"access_token":  signedAccess,
			"refresh_token": signedRefresh,
			"email":         u.Email,
		})
    }
}