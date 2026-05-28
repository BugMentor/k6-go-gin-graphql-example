package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/enterprise/payment-service/internal/application/usecase"
	"github.com/enterprise/payment-service/internal/infrastructure/persistence"
	"github.com/enterprise/payment-service/internal/presentation/graphql"
	"github.com/enterprise/payment-service/internal/presentation/rest"
	"github.com/enterprise/payment-service/internal/telemetry"
)

func main() {
	cfg := loadConfig()

	shutdown, err := telemetry.InitTracer(cfg.OTLPEndpoint, cfg.ServiceName)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer: %v", err)
		}
	}()

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	if err := persistence.RunMigrations(ctx, pool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	repo := persistence.NewPostgresRepository(pool)

	ucs := graphql.UseCases{
		ProcessPayment:       usecase.NewProcessPayment(repo),
		ProcessBatchPayments: usecase.NewProcessBatchPayments(repo),
		RefundPayment:        usecase.NewRefundPayment(repo),
		GetPayment:           usecase.NewGetPayment(repo),
		ListUserPayments:     usecase.NewListUserPayments(repo),
		SearchPayments:       usecase.NewSearchPayments(repo),
		GetPaymentSummary:    usecase.NewGetPaymentSummary(repo),
		WalletTransfer:       usecase.NewWalletTransfer(repo),
		TopUpWallet:          usecase.NewTopUpWallet(repo),
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(otelgin.Middleware(cfg.ServiceName))
	router.Use(telemetry.PrometheusMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	router.GET("/metrics", telemetry.PrometheusHandler())

	rest.RegisterRoutes(router, repo)

	graphql.RegisterRoutes(router, ucs, repo)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Payment Service starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

type config struct {
	Port         string
	ServiceName  string
	DatabaseURL  string
	OTLPEndpoint string
}

func loadConfig() config {
	return config{
		Port:         getEnv("PORT", "8080"),
		ServiceName:  getEnv("SERVICE_NAME", "payment-service"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://payment:payment@localhost:5432/payments?sslmode=disable"),
		OTLPEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
