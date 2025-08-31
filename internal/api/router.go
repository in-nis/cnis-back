package api

import (
    "github.com/gin-gonic/gin"
    "github.com/in-nis/cnis-back/internal/auth"
	"github.com/in-nis/cnis-back/internal/db"
	"github.com/in-nis/cnis-back/internal/config"
	_ "github.com/in-nis/cnis-back/docs"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
)

// @title           CNIS API
// @version         1.0
// @description     This is the API documentation for CNIS backend.
// @host            localhost:8000
// @BasePath        /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func SetupRouter(cfg *config.Config) *gin.Engine {
	auth.InitGoogle(cfg)

    r := gin.Default()

    // Public routes
    r.GET("/health", func(c *gin.Context) {
		dbPingError := db.PingDB()
		if dbPingError != nil {
			c.JSON(500, gin.H{"status": "db_ping_error"})
			return 
		}
        c.JSON(200, gin.H{"status": "ok"})
    })

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    // Google login
    r.GET("/auth/google/login", auth.GoogleLoginHandler())
    r.GET("/auth/google/callback", auth.GoogleCallbackHandler(cfg))
	r.POST("/auth/refresh", auth.RefreshHandler(cfg))
	r.GET("/lessons/filter", GetLessonsByClassAndGroups)

	// Protected
    authGroup := r.Group("/user")
    authGroup.Use(auth.AuthMiddleware(cfg))
    {
        authGroup.PATCH("/grade", UpdateUserGrade)
        authGroup.GET("/groups", GetUserGroups)
        authGroup.POST("/groups", AddUserGroup)
        authGroup.DELETE("/groups/:id", DeleteUserGroup)
		authGroup.GET("/me", GetMe)
		authGroup.POST("/lessons/reload", ParseLessons)
    }

    return r
}