package main

import (
	"database/sql"
	"fmt"
	"gafroshka-main/internal/announcement"
	annfb "gafroshka-main/internal/announcment_feedback"
	"gafroshka-main/internal/app"
	handlersAnnFeedback "gafroshka-main/internal/handlers/announcement_feedback"
	handlersCart "gafroshka-main/internal/handlers/shopping_cart"
	handlersUser "gafroshka-main/internal/handlers/user"
	handlersUserFeedback "gafroshka-main/internal/handlers/user_feedback"
	"gafroshka-main/internal/middleware"
	"gafroshka-main/internal/session"
	cart "gafroshka-main/internal/shopping_cart"
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
	announcementRepository := announcement.NewAnnouncementDBRepository(db, logger)
	sessionRepository := session.NewSessionRepository(redisClient, logger, c.Secret, c.SessionDuration)
	userFeedbackRepository := userFeedback.NewUserFeedbackRepository(db, logger)
	annFeedbackRepository := annfb.NewFeedbackDBRepository(db, logger)
	shoppingCartRepository := cart.NewShoppingCartRepository(db, logger)

	// init router
	r := mux.NewRouter()

	// init handlers
	userHandlers := handlersUser.NewUserHandler(logger, userRepository, sessionRepository)
	userFeedbackHandlers := handlersUserFeedback.NewUserFeedbackHandler(logger, userFeedbackRepository)
	annFeedbackHandlers := handlersAnnFeedback.NewAnnouncementFeedbackHandler(logger, annFeedbackRepository)
	shoppingCartHandlers := handlersCart.NewShoppingCartHandler(logger, shoppingCartRepository, announcementRepository)

	// Ручки требующие авторизации
	authRouter := r.PathPrefix("/api").Subrouter()
	authRouter.Use(middleware.Auth(sessionRepository))

	authRouter.HandleFunc("/feedback", annFeedbackHandlers.Create).Methods("POST")
	authRouter.HandleFunc("/feedback/{id}", annFeedbackHandlers.Delete).Methods("DELETE")

	authRouter.HandleFunc("/user/{id}", userHandlers.ChangeProfile).Methods("PUT")
	authRouter.HandleFunc("/user/{id}/balance/topup", userHandlers.TopUpBalance).Methods("POST")

	authRouter.HandleFunc("/feedback", userFeedbackHandlers.Create).Methods("POST")
	authRouter.HandleFunc("/feedback/{id}", userFeedbackHandlers.Update).Methods("PUT")
	authRouter.HandleFunc("/feedback/{id}", userFeedbackHandlers.Delete).Methods("DELETE")

	authRouter.HandleFunc("/cart/{userID}/item/{annID}", shoppingCartHandlers.AddToShoppingCart).Methods("POST")
	authRouter.HandleFunc("/cart/{userID}/item/{annID}", shoppingCartHandlers.DeleteFromShoppingCart).Methods("DELETE")
	authRouter.HandleFunc("/cart/{userID}", shoppingCartHandlers.GetCart).Methods("GET")

	// Ручки НЕ требующие авторизации
	noAuthRouter := r.PathPrefix("/api").Subrouter()

	noAuthRouter.HandleFunc("/user/{id}", userHandlers.Info).Methods("GET")
	noAuthRouter.HandleFunc("/user/register", userHandlers.Register).Methods("POST")
	noAuthRouter.HandleFunc("/user/login", userHandlers.Login).Methods("POST")
	noAuthRouter.HandleFunc("/user/{id}/balance", userHandlers.GetBalance).Methods("GET")

	noAuthRouter.HandleFunc("/feedback/user/{user_id}", userFeedbackHandlers.GetByUserID).Methods("GET")

	noAuthRouter.HandleFunc("/feedback/announcement/{id}", annFeedbackHandlers.GetByAnnouncementID).Methods("GET")

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
