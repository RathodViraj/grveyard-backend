package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"grveyard/db"
	_ "grveyard/docs"
	"grveyard/pkg/assets"
	"grveyard/pkg/buy"
	"grveyard/pkg/otp"
	"grveyard/pkg/sendemail"
	"grveyard/pkg/startups"
	"grveyard/pkg/users"
)

// @title           Graveyard API
// @version         1.0
// @description     REST API for failed startup marketplace - buy and sell startup assets

// @contact.email  virajrathod631@gmail.com

// @host      grveyard-backend.onrender.com
// @BasePath  /

// @schemes   http https

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	pool := db.Connect()
	defer pool.Close()

	emailService := sendemail.NewEmailService()

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

	otpRepo := otp.NewPostgresOTPRepository(pool)
	otpService := otp.NewOTPService(otpRepo, usersRepo, emailService)
	otpHandler := otp.NewOTPHandler(otpService)

	/*
		// Chat setup
		chatManager := chat.NewConnectionManager()
		chatHandler := chat.NewHandler(chatManager)
		// Inject message repository for persistence
		msgRepo := chat.NewPostgresMessageRepository(pool)
		chatHandler.SetRepository(msgRepo)
	*/

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// CORS configuration
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	var origins []string
	if allowedOrigins == "" {
		origins = []string{"*"}
	} else {
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
	otpHandler.RegisterRoutes(router)

	/*
		// WebSocket chat endpoint (uses UUID for user_id)
		router.GET("/ws/chat", func(c *gin.Context) {
			uid := c.Query("user_id")
			if _, err := uuid.Parse(uid); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id, must be UUID"})
				return
			}

			// Inject user_id into request context for the chat handler
			ctx := context.WithValue(c.Request.Context(), "user_id", uid)
			req := c.Request.WithContext(ctx)
			chatHandler.HandleWebSocket(c.Writer, req)
		})

		// Status endpoint for online users (proxy to handler)
		router.GET("/chat/status", func(c *gin.Context) {
			chatHandler.GetStatusGin(c)
		})

		router.GET("/messages", func(c *gin.Context) {
			uid := c.Query("user_id")
			if _, err := uuid.Parse(uid); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id, must be UUID"})
				return
			}

			// Inject authenticated user_id into context for handler validation
			ctx := context.WithValue(c.Request.Context(), "user_id", uid)
			c.Request = c.Request.WithContext(ctx)

			chatHandler.GetMessagesGin(c)
		})
	*/
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
