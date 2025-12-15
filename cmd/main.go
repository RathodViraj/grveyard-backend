package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "grveyard/docs"
	"grveyard/pkg/assets"
	"grveyard/pkg/buy"
	"grveyard/pkg/db"
	"grveyard/pkg/startups"
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

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	startupsHandler.RegisterRoutes(router)
	assetsHandler.RegisterRoutes(router)
	buyHandler.RegisterRoutes(router)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
