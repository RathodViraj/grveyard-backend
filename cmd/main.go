package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "grveyard/docs"
	"grveyard/pkg/assets"
	"grveyard/pkg/buy"
	"grveyard/pkg/db"
	"grveyard/pkg/startups"
	"grveyard/pkg/users"
)

// @title           Graveyard API
// @version         1.0
// @description     REST API for failed startup marketplace - buy and sell startup assets

// @contact.email  virajrathod631@gmail.com

// @host      localhost:8080
// @BasePath  /

// @schemes   http https

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	pool := db.Connect()
	defer pool.Close()

	startupsRepo := startups.NewPostgresStartupRepository(pool)
	startupsService := startups.NewStartupService(startupsRepo)
	startupsHandler := startups.NewStartupHandler(startupsService)

	assetsRepo := assets.NewPostgresAssetRepository(pool)
	assetsService := assets.NewAssetService(assetsRepo)
	assetsHandler := assets.NewAssetHandler(assetsService)

	buyRepo := buy.NewPostgresBuyRepository(pool)
	buyService := buy.NewBuyService(buyRepo)
	buyHandler := buy.NewBuyHandler(buyService)

	usersRepo := users.NewPostgresUserRepository(pool)
	usersService := users.NewUserService(usersRepo)
	usersHandler := users.NewUserHandler(usersService)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// CORS configuration
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	var origins []string
	if allowedOrigins == "" {
		origins = []string{"*"}
	} else {
		// comma-separated list
		parts := strings.Split(allowedOrigins, ",")
		origins = make([]string, 0, len(parts))
		for _, p := range parts {
			o := strings.TrimSpace(p)
			if o != "" {
				origins = append(origins, o)
			}
		}
		if len(origins) == 0 {
			origins = []string{"*"}
		}
	}

	allowCreds := strings.EqualFold(os.Getenv("CORS_ALLOW_CREDENTIALS"), "true")

	corsCfg := cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: allowCreds,
		MaxAge:           12 * time.Hour,
	}
	// If wildcard '*' is used with credentials=false, it's valid; otherwise list explicit origins
	router.Use(cors.New(corsCfg))

	startupsHandler.RegisterRoutes(router)
	assetsHandler.RegisterRoutes(router)
	buyHandler.RegisterRoutes(router)
	usersHandler.RegisterRoutes(router)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
