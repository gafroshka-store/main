package main

import (
	"context"
	"database/sql"
	"fmt"
	"gafroshka-main/internal/analytics"
	"gafroshka-main/internal/kafka"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	_ "github.com/lib/pq"
)

const (
	cfgPath      = "config/analytics-config.yaml"
	KafkaBrokers = "kafka:9092"
	KafkaTopic   = "user-events"
	KafkaGroupID = "analytics-group"
)

func main() {
	// Init logger
	zapLogger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	logger := zapLogger.Sugar()
	defer func() { _ = zapLogger.Sync() }()

	// Parse config
	c, err := analytics.NewConfig(cfgPath)
	if err != nil {
		logger.Fatalf("Error parsing config: %v", err)
	}

	// Init DB
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.CfgDB.Host, c.CfgDB.Port, c.CfgDB.Login, c.CfgDB.Password, c.CfgDB.Database,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Fatalf("Error connecting to DB: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(c.MaxOpenConns)
	if err := db.Ping(); err != nil {
		logger.Errorf("DB ping failed: %v", err)
	}

	// Init Kafka Consumer
	consumer := kafka.NewConsumer(KafkaBrokers, KafkaTopic, KafkaGroupID, logger)
	defer consumer.Close()

	// Init analytics repository и service через интерфейсы
	repo := analytics.NewRepository(db, logger)
	service := analytics.NewService(repo, logger)

	// Start event processor
	go func() {
		consumer.Consume(context.Background(), func(ctx context.Context, event kafka.Event) error {
			return service.ProcessEvent(ctx, event)
		})
	}()

	// Init HTTP server
	handler := analytics.NewHandler(service, logger)
	r := mux.NewRouter()
	r.HandleFunc("/user/{user_id}/preferences", handler.GetUserPreferences).Methods("GET")

	srv := &http.Server{
		Addr:         ":8082",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("Starting analytics service on :8082")
	if err := srv.ListenAndServe(); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
