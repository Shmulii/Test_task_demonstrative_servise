package main

import (
    "context"
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    zerolog.SetGlobalLevel(zerolog.InfoLevel)
    log.Info().Msg("starting orders-service")

    cfg := DefaultConfig()
    // Краткое определение стандартных конфигураций 

    ctx := context.Background()
    db, err := NewDB(ctx, cfg.PostgresDSN)
    if err != nil {
        log.Fatal().Err(err).Msg("failed to connect to db")
    }
    defer db.Close()
    // Создание БД подключения, проверка ошибок

    cache := NewCache(cfg.CacheLimit)
    /* При старте загружаем последние N заказов в кеш
    инициализация кэша для заказов*/

    recent, err := db.LoadRecentOrders(ctx, cfg.StartupLoad)
    if err != nil {
        log.Error().Err(err).Msg("failed to load recent orders")
    } else {
        for i := range recent {
            cache.Set(&recent[i]) //указатель на адресацию 
        }
        log.Info().Int("loaded", len(recent)).Msg("cache warmup")
    }
    //Предзагрузка предыдущих заказов в кэше

    // Запускаем сервер для разрешения HTTP запросов
    srv := StartHTTPServer(cfg, db, cache)

    // Запускаем брокер сообщений kafka consumer
    StartConsumer(ctx, cfg, db, cache)

    // Обеспечивает корректное завершение работы приложения при получении системных сигналов
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    <-stop
    log.Info().Msg("shutting down")
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    srv.Shutdown(shutdownCtx)
}