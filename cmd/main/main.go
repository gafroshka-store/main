package main

import (
	"context"
	"database/sql"
	"fmt"
	"gafroshka-main/internal/announcement"

	annfb "gafroshka-main/internal/announcment_feedback"
	"gafroshka-main/internal/app"
	elastic "gafroshka-main/internal/elastic_search"
	"gafroshka-main/internal/etl"
	userAnnHandlers "gafroshka-main/internal/handlers/announcement"
	handlersAnnFeedback "gafroshka-main/internal/handlers/announcement_feedback"
	handlersCart "gafroshka-main/internal/handlers/shopping_cart"
	handlersUser "gafroshka-main/internal/handlers/user"
	handlersUserFeedback "gafroshka-main/internal/handlers/user_feedback"
	"gafroshka-main/internal/middleware"
	"gafroshka-main/internal/session"
	cart "gafroshka-main/internal/shopping_cart"
	"gafroshka-main/internal/user"
	userFeedback "gafroshka-main/internal/user_feedback"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/elastic/go-elasticsearch/v8"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	_ "github.com/lib/pq"
)

const (
	cfgPath   = "config/config.yaml"
	RedisAddr = "redis:6379"
	ESAddr    = "http://elasticsearch:9200"
)

func main() {
	// init logger
	zapLogger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	logger := zapLogger.Sugar()

	// тк функция откладывается буду использовать
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

	// init ES
	elasticClient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			ESAddr,
		},
	})
	if err != nil {
		logger.Errorf("failed to create elastic client: %v", err)
	}

	_, err = elasticClient.Ping()
	if err != nil {
		logger.Warnf("failed to ping Elasticsearch: %v", err)
	}

	elasicService := elastic.NewService(elasticClient, logger, c.CfgES.Index)

	// init and start ETL
	extractor := etl.NewPostgresExtractor(db, logger)
	transformer := etl.NewTransformer(logger)
	loader := etl.NewElasticLoader(elasicService, logger)

	pipeline := etl.NewPipeline(extractor, transformer, loader, logger, c.ETLTimeout)

	go pipeline.Run(context.Background())

	// init repository
	userRepository := user.NewUserDBRepository(db, logger)
	announcementRepository := announcement.NewAnnouncementDBRepository(db, logger)
	sessionRepository := session.NewSessionRepository(redisClient, logger, c.Secret, c.SessionDuration)
	userFeedbackRepository := userFeedback.NewUserFeedbackRepository(db, logger)
	annRepo := announcement.NewAnnouncementDBRepository(db, logger)
	annFeedbackRepository := annfb.NewFeedbackDBRepository(db, logger)
	shoppingCartRepository := cart.NewShoppingCartRepository(db, logger)

	// init router
	r := mux.NewRouter()

	// init handlers
	userHandlers := handlersUser.NewUserHandler(logger, userRepository, sessionRepository)
	userFeedbackHandlers := handlersUserFeedback.NewUserFeedbackHandler(logger, userFeedbackRepository)
	annFeedbackHandlers := handlersAnnFeedback.NewAnnouncementFeedbackHandler(logger, annFeedbackRepository)
	annHandlers := userAnnHandlers.NewAnnouncementHandler(logger, annRepo)
	shoppingCartHandlers := handlersCart.NewShoppingCartHandler(logger, shoppingCartRepository, announcementRepository)

	// Ручки требующие авторизации
	authRouter := r.PathPrefix("/api").Subrouter()
	authRouter.Use(middleware.Auth(sessionRepository))

	authRouter.HandleFunc("/announcement/feedback", annFeedbackHandlers.Create).Methods("POST")
	authRouter.HandleFunc("/announcement/feedback/{id}", annFeedbackHandlers.Delete).Methods("DELETE")
	authRouter.HandleFunc("/announcement/feedback/{id}", annFeedbackHandlers.Update).Methods("PATCH")

	authRouter.HandleFunc("/user/{id}", userHandlers.ChangeProfile).Methods("PUT")
	authRouter.HandleFunc("/user/{id}/balance/topup", userHandlers.TopUpBalance).Methods("POST")

	authRouter.HandleFunc("/user/feedback", userFeedbackHandlers.Create).Methods("POST")
	authRouter.HandleFunc("/user/feedback/{id}", userFeedbackHandlers.Update).Methods("PUT")
	authRouter.HandleFunc("/user/feedback/{id}", userFeedbackHandlers.Delete).Methods("DELETE")

	authRouter.HandleFunc("/announcement", annHandlers.Create).Methods("POST")
	authRouter.HandleFunc("/announcement/{id}/rating", annHandlers.UpdateRating).Methods("POST")

	authRouter.HandleFunc("/cart/{userID}/item/{annID}", shoppingCartHandlers.AddToShoppingCart).Methods("POST")
	authRouter.HandleFunc("/cart/{userID}/item/{annID}", shoppingCartHandlers.DeleteFromShoppingCart).Methods("DELETE")
	authRouter.HandleFunc("/cart/{userID}", shoppingCartHandlers.GetCart).Methods("GET")
	authRouter.HandleFunc("/cart/{userID}/purchase", shoppingCartHandlers.PurchaseFromCart).Methods("POST")

	// Ручки НЕ требующие авторизации
	noAuthRouter := r.PathPrefix("/api").Subrouter()

	noAuthRouter.HandleFunc("/user/{id}", userHandlers.Info).Methods("GET")
	noAuthRouter.HandleFunc("/user/register", userHandlers.Register).Methods("POST")
	noAuthRouter.HandleFunc("/user/login", userHandlers.Login).Methods("POST")
	noAuthRouter.HandleFunc("/user/{id}/balance", userHandlers.GetBalance).Methods("GET")

	noAuthRouter.HandleFunc("/user/feedback/user/{id}", userFeedbackHandlers.GetByUserID).Methods("GET")
	noAuthRouter.HandleFunc("/announcement/feedback/announcement/{id}", annFeedbackHandlers.GetByAnnouncementID).Methods("GET")

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

	err = srv.ListenAndServe()
	if err != nil {
		panic("can't start server: " + err.Error())
	}
}
