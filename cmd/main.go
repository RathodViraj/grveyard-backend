package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
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
	"grveyard/pkg/chat"
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

	// Chat setup
	chatManager := chat.NewConnectionManager()
	chatHandler := chat.NewHandler(chatManager)
	// Inject message store for persistence
	msgRepo := chat.NewPostgresMessageStore(pool)
	chatHandler.SetRepository(msgRepo)

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

	// WebSocket chat endpoint (uses UUID for user_id)
	router.GET("/ws/chat", chatHandler.HandleWebSocketGin)

	// Status endpoint for online users (proxy to handler)
	router.GET("/chat/status", chatHandler.GetStatusGin)

	router.GET("/messages", chatHandler.GetMessagesGin)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	settings := loadTLSSettingsFromEnv()
	if err := settings.Validate(); err != nil {
		log.Fatalf("TLS settings invalid: %v", err)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		if settings.EnableTLS {
			port = "8443"
		} else {
			port = "8080"
		}
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start HTTP or HTTPS based on settings
	go func() {
		if !settings.EnableTLS {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen (HTTP): %v", err)
			}
			return
		}

		tlsConfig, certFile, keyFile, err := buildTLSConfigWithSettings(settings)
		if err != nil {
			log.Fatalf("TLS setup error: %v", err)
		}
		srv.TLSConfig = tlsConfig

		if certFile != "" && keyFile != "" {
			if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen (TLS files): %v", err)
			}
			return
		}
		if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen (TLS config): %v", err)
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

// TLSSettings holds environment-driven TLS configuration.
type TLSSettings struct {
	EnableTLS       bool
	CertPath        string
	KeyPath         string
	Env             string // "production" or "development"
	AllowSelfSigned bool   // allow generating self-signed in dev when files are missing
}

// loadTLSSettingsFromEnv reads TLS settings from environment variables.
// Vars:
// - ENABLE_TLS: true/false
// - TLS_CERT_PATH / TLS_KEY_PATH: file paths to PEM cert/key
// - APP_ENV or ENV: "production" or "development"
// - TLS_SELF_SIGNED: true/false (dev convenience)
func loadTLSSettingsFromEnv() TLSSettings {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if env == "" {
		env = strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))
	}
	if env == "" {
		env = "development"
	}

	enableTLS := !strings.EqualFold(os.Getenv("ENABLE_TLS"), "false")
	// Enforce TLS in production
	if env == "production" {
		enableTLS = true
	}

	return TLSSettings{
		EnableTLS:       enableTLS,
		CertPath:        os.Getenv("TLS_CERT_PATH"),
		KeyPath:         os.Getenv("TLS_KEY_PATH"),
		Env:             env,
		AllowSelfSigned: !strings.EqualFold(os.Getenv("TLS_SELF_SIGNED"), "false"),
	}
}

// Validate ensures TLS settings are safe for the selected environment.
func (s TLSSettings) Validate() error {
	if s.Env == "production" {
		if !s.EnableTLS {
			return fmt.Errorf("TLS must be enabled in production")
		}
		if s.CertPath == "" || s.KeyPath == "" {
			return fmt.Errorf("TLS_CERT_PATH and TLS_KEY_PATH are required in production")
		}
	}
	return nil
}

// buildTLSConfigWithSettings constructs a tls.Config based on TLSSettings.
// Prefers file paths; falls back to inline PEM (TLS_CERT/TLS_KEY) or self-signed in development.
func buildTLSConfigWithSettings(s TLSSettings) (*tls.Config, string, string, error) {
	var cert tls.Certificate
	var err error

	// Prefer explicit file paths
	if s.CertPath != "" && s.KeyPath != "" {
		cert, err = tls.LoadX509KeyPair(s.CertPath, s.KeyPath)
		if err != nil {
			return nil, "", "", err
		}
		return &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}, s.CertPath, s.KeyPath, nil
	}

	// Try inline PEM from env (backward compatibility)
	certPEM := os.Getenv("TLS_CERT")
	keyPEM := os.Getenv("TLS_KEY")
	if certPEM != "" && keyPEM != "" {
		cert, err = tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
		if err != nil {
			return nil, "", "", err
		}
		return &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}, "", "", nil
	}

	// Development fallback: self-signed
	if s.Env != "production" && s.AllowSelfSigned {
		genCert, genErr := generateSelfSignedCert()
		if genErr != nil {
			return nil, "", "", genErr
		}
		return &tls.Config{Certificates: []tls.Certificate{genCert}, MinVersion: tls.VersionTLS12}, "", "", nil
	}

	return nil, "", "", fmt.Errorf("no TLS certificates available")
}

// generateSelfSignedCert creates a minimal self-signed certificate for localhost usage.
func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return tls.Certificate{}, err
	}

	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return tls.X509KeyPair(certPEM, keyPEM)
}
