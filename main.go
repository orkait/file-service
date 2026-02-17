package main

import (
	"context"
	"file-service/config"
	"file-service/pkg/cache"
	"file-service/pkg/database"
	mailerpkg "file-service/pkg/mailer"
	mailerproviders "file-service/pkg/mailer/providers"
	mailerstrategies "file-service/pkg/mailer/strategies"
	"file-service/pkg/middleware"
	"file-service/pkg/repository"
	"file-service/pkg/s3"
	"file-service/routes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	} else {
		port = ":" + port
	}

	return port
}

func parseMailProviders(raw string) []string {
	parts := strings.Split(raw, ",")
	seen := make(map[string]bool, len(parts))
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	return result
}

func buildProvider(cfg *config.Config, providerName string) (mailerproviders.EmailProvider, bool) {
	switch providerName {
	case "resend":
		apiKey := strings.TrimSpace(cfg.ResendAPIKey)
		apiURL := strings.TrimSpace(cfg.ResendAPIURL)
		if apiKey == "" {
			log.Println("Skipping resend provider: missing RESEND_API_KEY")
			return nil, false
		}
		return mailerpkg.NewResendProvider(mailerproviders.ResendConfig{
			APIKey: apiKey,
			APIURL: apiURL,
		}), true
	case "sendgrid":
		apiKey := strings.TrimSpace(cfg.SendGridAPIKey)
		apiURL := strings.TrimSpace(cfg.SendGridAPIURL)
		if apiKey == "" {
			log.Println("Skipping sendgrid provider: missing SENDGRID_API_KEY")
			return nil, false
		}
		return mailerpkg.NewSendGridProvider(mailerproviders.SendGridConfig{
			APIKey: apiKey,
			APIURL: apiURL,
		}), true
	default:
		log.Printf("Skipping unsupported mail provider %q", providerName)
		return nil, false
	}
}

func buildEmailService(cfg *config.Config) *mailerpkg.EmailService {
	if strings.TrimSpace(cfg.MailFrom) == "" {
		log.Println("Mailer disabled: set MAIL_FROM to enable email")
		return nil
	}

	providerNames := parseMailProviders(cfg.MailProviders)
	if len(providerNames) == 0 {
		log.Println("Mailer disabled: set MAIL_PROVIDERS to enable email")
		return nil
	}

	providerList := make([]mailerproviders.EmailProvider, 0, len(providerNames))
	for _, providerName := range providerNames {
		provider, ok := buildProvider(cfg, providerName)
		if ok {
			providerList = append(providerList, provider)
		}
	}
	if len(providerList) == 0 {
		log.Println("Mailer disabled: no valid providers configured")
		return nil
	}

	serviceConfig := mailerpkg.EmailServiceConfig{
		Providers:   providerList,
		DefaultFrom: cfg.MailFrom,
	}
	if len(providerList) > 1 {
		serviceConfig.Strategy = &mailerstrategies.FailoverStrategy{}
		log.Printf("Mailer enabled with failover chain (%d providers)", len(providerList))
	} else {
		log.Println("Mailer enabled with one provider in chain")
	}

	service, err := mailerpkg.NewEmailService(mailerpkg.EmailServiceConfig{
		Providers:   serviceConfig.Providers,
		Strategy:    serviceConfig.Strategy,
		DefaultFrom: serviceConfig.DefaultFrom,
	})
	if err != nil {
		log.Printf("Mailer disabled: %v", err)
		return nil
	}

	return service
}

func main() {
	e := echo.New()

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading environment variables")
	}

	log.SetOutput(os.Stderr)

	rateLimiterConfig := echomiddleware.RateLimiterConfig{
		Skipper: echomiddleware.DefaultSkipper,
		Store: echomiddleware.NewRateLimiterMemoryStoreWithConfig(
			echomiddleware.RateLimiterMemoryStoreConfig{Rate: 10, Burst: 30, ExpiresIn: 3 * time.Minute},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			id := ctx.RealIP()
			return id, nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusForbidden, nil)
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, nil)
		},
	}

	e.Use(echomiddleware.RateLimiterWithConfig(rateLimiterConfig))
	e.Use(echomiddleware.CORS())

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %s", err)
	}
	defer db.Close()

	s3Client, err := s3.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create S3 client: %s", err)
	}

	urlCache := cache.NewURLCache()

	clientRepo := repository.NewClientRepository(db.DB)
	projectRepo := repository.NewProjectRepository(db.DB)
	apiKeyRepo := repository.NewAPIKeyRepository(db.DB)
	assetRepo := repository.NewAssetRepository(db.DB)
	memberRepo := repository.NewMemberRepository(db.DB)

	emailService := buildEmailService(cfg)

	authRoutes := routes.NewAuthRoutes(clientRepo, cfg.JWTSecret, emailService, cfg.AppBaseURL, cfg.AppName)
	clientRoutes := routes.NewClientRoutes(clientRepo, assetRepo, s3Client)
	projectRoutes := routes.NewProjectRoutes(projectRepo)
	apiKeyRoutes := routes.NewAPIKeyRoutes(apiKeyRepo)
	assetRoutes := routes.NewAssetRoutes(s3Client, assetRepo, projectRepo, memberRepo, urlCache)
	memberRoutes := routes.NewMemberRoutes(memberRepo, projectRepo, clientRepo, emailService, cfg.AppBaseURL, cfg.AppName)

	jwtMiddleware := middleware.JWTAuth(cfg.JWTSecret, clientRepo)
	apiKeyMiddleware := middleware.APIKeyAuth(apiKeyRepo, clientRepo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if deleted, err := clientRoutes.CleanupDuePausedClients(); err != nil {
			log.Printf("Client cleanup finished with errors: %v", err)
		} else if deleted > 0 {
			log.Printf("Client cleanup deleted %d paused account(s)", deleted)
		}

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				deleted, err := clientRoutes.CleanupDuePausedClients()
				if err != nil {
					log.Printf("Client cleanup finished with errors: %v", err)
					continue
				}
				if deleted > 0 {
					log.Printf("Client cleanup deleted %d paused account(s)", deleted)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				urlCache.Clear()
			case <-ctx.Done():
				return
			}
		}
	}()

	routes.RegisterRoutes(e, s3Client, urlCache)
	routes.RegisterMultiTenantRoutes(e, authRoutes, clientRoutes, projectRoutes, apiKeyRoutes, assetRoutes, memberRoutes, jwtMiddleware, apiKeyMiddleware)

	go func() {
		if err := e.Start(getPort()); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %s", err)
		}
	}()

	log.Println("Server Started!!!")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %s", err)
	}

	log.Println("Server exited")
}
