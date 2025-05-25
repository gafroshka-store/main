package main

import (
	"database/sql"
	"fmt"
	"gafroshka-main/internal/announcement"
	"gafroshka-main/internal/app"
	"gafroshka-main/internal/handlers"
	"gafroshka-main/internal/user"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	_ "github.com/lib/pq"
)

const (
	cfgPath = "config/config.yaml"
)

func main() {
	// init logger
	zapLogger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	logger := zapLogger.Sugar()
	defer func() {
		if err := zapLogger.Sync(); err != nil {
			logger.Warnf("error to sync logger: %v", err)
		}
	}()

	// парсим конфиг
	c, err := app.NewConfig(cfgPath)
	if err != nil {
		logger.Fatalf("error to parsing config: %v", err)
	}

	// init db
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable",
		c.CfgDB.Host, c.CfgDB.Port, c.CfgDB.Login, c.CfgDB.Password, c.CfgDB.Database,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Fatalf("error to database start: %v", err)
	}

	db.SetMaxOpenConns(c.MaxOpenConns)
	if err := db.Ping(); err != nil {
		logger.Infof("Failed to get response to ping: %v", err)
	}

	// init repositories
	userRepo := user.NewUserDBRepository(db, logger)
	annRepo := announcement.NewAnnouncementDBRepository(db, logger)

	// init handlers
	userHandler := handlers.NewUserHandler(logger, userRepo)
	annHandler := handlers.NewAnnouncementHandler(logger, annRepo)

	// init router
	r := mux.NewRouter()
	// user routes
	r.HandleFunc("/user/{id}", userHandler.Info).Methods("GET")
	r.HandleFunc("/user/{id}", userHandler.ChangeProfile).Methods("PUT")
	// announcement routes
	r.HandleFunc("/announcement", annHandler.Create).Methods("POST")
	r.HandleFunc("/announcement/{id}", annHandler.GetByID).Methods("GET")
	r.HandleFunc("/announcements/top", annHandler.GetTopN).Methods("POST")
	r.HandleFunc("/announcements/search", annHandler.Search).Methods("GET")
	r.HandleFunc("/announcement/{id}/rating", annHandler.UpdateRating).Methods("POST")

	logger.Infow("starting server",
		"type", "START",
		"addr", c.ServerPort,
	)

	srv := &http.Server{
		Addr:         c.ServerPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		panic("can't start server: " + err.Error())
	}
}
