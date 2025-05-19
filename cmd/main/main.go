package main

import (
	"database/sql"
	"fmt"
	annfb "gafroshka-main/internal/announcmentFeedback"
	"gafroshka-main/internal/app"
	"gafroshka-main/internal/handlers"
	"gafroshka-main/internal/middleware"
	"gafroshka-main/internal/session"
	"gafroshka-main/internal/user"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"

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
	//	тк функция откладывается буду использовать
	// обертку в анонимную функцию
	defer func() {
		err = zapLogger.Sync()
		if err != nil {
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

	err = db.Ping()
	if err != nil {
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

	// init router
	r := mux.NewRouter()

	// init handlers
	userHandlers := handlers.NewUserHandler(logger, userRepository, sessionRepository)

	// Ручки требующие авторизации
	authRouter := r.PathPrefix("/api").Subrouter()
	authRouter.Use(middleware.Auth(sessionRepository))

	authRouter.HandleFunc("/user/{id}", userHandlers.ChangeProfile).Methods("PUT")
	// Ручки НЕ требующие авторизации
	noAuthRouter := r.PathPrefix("/api").Subrouter()

	noAuthRouter.HandleFunc("/user/{id}", userHandlers.Info).Methods("GET")
	noAuthRouter.HandleFunc("/user/register", userHandlers.Register).Methods("POST")
	noAuthRouter.HandleFunc("/user/login", userHandlers.Login).Methods("POST")

	feedbackRepository := annfb.NewFeedbackDBRepository(db, logger)

	feedbackHandlers := handlers.NewAnnouncementFeedbackHandler(logger, feedbackRepository)

	// Ручки, требующие авторизации
	authRouter.HandleFunc("/feedback", feedbackHandlers.Create).Methods("POST")
	authRouter.HandleFunc("/feedback/{id}", feedbackHandlers.Delete).Methods("DELETE")

	// Ручки, НЕ требующие авторизации
	noAuthRouter.HandleFunc("/feedback/announcement/{id}", feedbackHandlers.GetByAnnouncementID).Methods("GET")

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

	err = srv.ListenAndServe()
	if err != nil {
		panic("can't start server: " + err.Error())
	}
}
