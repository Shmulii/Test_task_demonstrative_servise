package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"time"
)

// создает маршрутизатор, регистрирует маршруты, настраивает сервер, запускает его в горутине
func StartHTTPServer(cfg *Config, db *DB, cache *Cache) *http.Server {
	r := mux.NewRouter()
	// Обслуживает все файлы в папке ./static по адресу /static/....
	staticDir := http.Dir("./static")
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(staticDir)))

	r.HandleFunc("/orders/{order_uid}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		uid := vars["order_uid"]
		if uid == "" {
			http.Error(w, "missing order_uid", http.StatusBadRequest)
			return
		}

		// Сначала пробуем кеш
		if o, ok := cache.Get(uid); ok {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(o)
			return
		}

		// Иначе подтянем из БД
		ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
		defer cancel()
		o, err := db.GetOrder(ctx, uid)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		// положим в кеш и вернем
		cache.Set(o)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(o)
	}).Methods("GET")

	// Корневой(Root) обработчик для главной страницы
	r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "./static/index.html")
	})
	/* Привязывает адрес из конфигурации;
	устанавливает тайм-ауты чтения/записи/простоя;
	подключает маршрутизатор*/
	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	// Регистрирует запуск
	go func() {
		log.Info().Str("addr", cfg.HTTPAddr).Msg("http server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("http server failed")
		}
	}()

	return srv
}
