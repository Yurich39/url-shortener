package main

import (
	"fmt"
	"net/http"
	"os"
	"url_shortener/internal/config"
	"url_shortener/internal/http-server/handlers/redirect"
	"url_shortener/internal/http-server/handlers/url/save"
	mwLogger "url_shortener/internal/http-server/middleware/logger"
	"url_shortener/internal/lib/logger/sl"
	"url_shortener/internal/storage/postgresql"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/exp/slog"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	//TODO: init config
	cfg := config.MustLoad()
	fmt.Println(cfg)

	// TODO: init logger
	log := setupLogger(cfg.Env)
	log.Info("starting url_shortener", slog.String("env", cfg.Env))
	log.Debug("debug messages are enabled")

	//TODO: init storage
	storage, err := postgresql.New(cfg.StorageConfig)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	_ = storage

	// пробуем добавить url и его alias в БД
	// err = storage.SaveURL("https://google.com", "google")
	// if err != nil {
	// 	log.Error("failed to save url in DB", sl.Err(err))
	// 	os.Exit(1)
	// }

	//TODO: init router
	router := chi.NewRouter()

	// это стек middleware, через которые будет проходить каждый запрос request
	router.Use(middleware.RequestID) // Добавляет request_id в каждый запрос, для трейсинга
	router.Use(middleware.Logger)    // Логирование всех запросов
	router.Use(mwLogger.New(log))    // Мой логгер - Логирование всех запросов
	router.Use(middleware.Recoverer) // Если где-то внутри сервера (обработчика запроса) произойдет паника, приложение не должно упасть
	router.Use(middleware.URLFormat) // Парсер URLов поступающих запросов

	// Роутер для админа. Все пути этого роутера будут начинаться с префикса `/url`
	router.Route("/url", func(r chi.Router) {
		// Подключаем авторизацию
		r.Use(middleware.BasicAuth("url_shortener", map[string]string{
			// Передаем в middleware креды
			cfg.HTTPServer.User: cfg.HTTPServer.Password,
			// Если у вас более одного пользователя,
			// то можете добавить остальные пары по аналогии.
		}))

		// Handler #1 - добавление из запроса url и alias в БД
		r.Post("/", save.New(log, storage)) // подключили наш handler - save.New(log, storage)
	})

	// Handler #2 - получение из БД url по alias в GET запросе и делаем redirect на этот url
	router.Get("/{alias}", redirect.New(log, storage)) // подключили handler

	//TODO: run server
	log.Info("starting server", slog.String("address", cfg.Address)) // логируем запись о запуске сервера

	// создаем сервер
	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	// запускаем сервер - исполнение функции main остановится на вызове функции ListenAndServe
	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server")
	}

	// сюда код дойдет только если произошла ошибка при запуске сервера выше
	log.Error("server stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	// переключатель логов для разных серверов
	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
