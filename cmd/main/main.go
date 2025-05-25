package main

import (
	"database/sql"
	"fmt"
	"gafroshka-main/internal/announcement"

	annfb "gafroshka-main/internal/announcment_feedback"
	"gafroshka-main/internal/app"
	handlersAnnFeedback "gafroshka-main/internal/handlers/announcement_feedback"
	handlersUser "gafroshka-main/internal/handlers/user"
	handlersUserFeedback "gafroshka-main/internal/handlers/user_feedback"
	"gafroshka-main/internal/middleware"
	"gafroshka-main/internal/session"
	"gafroshka-main/internal/user"
	userFeedback "gafroshka-main/internal/user_feedback"
	"github.com/go-redis/redis/v8"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	_ "github.com/lib/pq"
)

const (
	cfgPath   = "config/config.yaml"
	RedisAddr = "redis:6379"
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

	// init redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     RedisAddr,
		Password: "",
		DB:       0, // стандартная БД
	})

	// init repository
	userRepository := user.NewUserDBRepository(db, logger)
	sessionRepository := session.NewSessionRepository(redisClient, logger, c.Secret, c.SessionDuration)
	userFeedbackRepository := userFeedback.NewUserFeedbackRepository(db, logger)
 	annRepo := announcement.NewAnnouncementDBRepository(db, logger)
	annFeedbackRepository := annfb.NewFeedbackDBRepository(db, logger)

	// init router
	r := mux.NewRouter()

	// init handlers
	userHandlers := handlersUser.NewUserHandler(logger, userRepository, sessionRepository)
	userFeedbackHandlers := handlersUserFeedback.NewUserFeedbackHandler(logger, userFeedbackRepository)
	annFeedbackHandlers := handlersAnnFeedback.NewAnnouncementFeedbackHandler(logger, annFeedbackRepository)
  annHandlers := handlers.NewAnnouncementHandler(logger, annRepo)

	// Ручки требующие авторизации
	authRouter := r.PathPrefix("/api").Subrouter()
	authRouter.Use(middleware.Auth(sessionRepository))

	authRouter.HandleFunc("/feedback", annFeedbackHandlers.Create).Methods("POST")
	authRouter.HandleFunc("/feedback/{id}", annFeedbackHandlers.Delete).Methods("DELETE")

	authRouter.HandleFunc("/user/{id}", userHandlers.ChangeProfile).Methods("PUT")

	authRouter.HandleFunc("/feedback", userFeedbackHandlers.Create).Methods("POST")
	authRouter.HandleFunc("/feedback/{id}", userFeedbackHandlers.Update).Methods("PUT")
	authRouter.HandleFunc("/feedback/{id}", userFeedbackHandlers.Delete).Methods("DELETE")
  
  authRouter.HandleFunc("/announcement", annHandlers.Create).Methods("POST")
	authRouter.HandleFunc("/announcement/{id}/rating", annHandlers.UpdateRating).Methods("POST")

	// Ручки НЕ требующие авторизации
	noAuthRouter := r.PathPrefix("/api").Subrouter()

	noAuthRouter.HandleFunc("/user/{id}", userHandlers.Info).Methods("GET")
	noAuthRouter.HandleFunc("/user/register", userHandlers.Register).Methods("POST")
	noAuthRouter.HandleFunc("/user/login", userHandlers.Login).Methods("POST")

	noAuthRouter.HandleFunc("/feedback/user/{user_id}", userFeedbackHandlers.GetByUserID).Methods("GET")

	noAuthRouter.HandleFunc("/feedback/announcement/{id}", annFeedbackHandlers.GetByAnnouncementID).Methods("GET")
  
  noAuthRouter.HandleFunc("/announcement/{id}", annHandlers.GetByID).Methods("GET")
  noAuthRouter.HandleFunc("/announcements/top", annHandlers.GetTopN).Methods("POST")
	noAuthRouter.HandleFunc("/announcements/search", annHandlers.Search).Methods("GET")

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
