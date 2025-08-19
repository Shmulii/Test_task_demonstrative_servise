package main

import (
    "context" // отмена/тайм-ауты для Kafka
    "encoding/json" // перенос сообщений в Order
    "github.com/segmentio/kafka-go" // клиент Kafka
    "github.com/rs/zerolog/log" // структуриованное ведение журнала
    "time" // тайм-ауты
)

 // Точка входа для запуска потребителя Kafka
func StartConsumer(ctx context.Context, cfg *Config, db *DB, cache *Cache) {
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers:   []string{cfg.KafkaBroker}, // адреса брокеров Kafka
        Topic:     cfg.KafkaTopic, // топик/тема Kafka
        GroupID:   "orders-service-group", // идентификатор группы потребителей
        MinBytes:  1, 
        MaxBytes:  10e6,
        CommitInterval: 0, // отключить автоматическую фиксацию; фиксации выполняются вручную.
    })

    //Запускает цикл потребления в горутине, чтобы вызывающий объект не блокировался.
    go func() {
        defer r.Close()
        for {
            m, err := r.FetchMessage(ctx)
            if err != nil { // Бесконечный цикл выборки сообщений 
                if ctx.Err() != nil {
                    log.Info().Msg("consumer context done")
                    return
                }
                log.Error().Err(err).Msg("fetch message error")
                time.Sleep(time.Second) //Регистрируется ошибка, происходит пауза в 1 секунду и повторяется попытка
                continue
            }

            var o Order
            if err := json.Unmarshal(m.Value, &o); err != nil {
                // json - ошибка: записываем, фиксируем, чтобы избежать его бесконечной повторной обработки и переходим к следующему сообщению.
                log.Error().Err(err).Str("topic", cfg.KafkaTopic).Msg("invalid JSON, skipping message")
                if err2 := r.CommitMessages(ctx, m); err2 != nil {
                    log.Error().Err(err2).Msg("failed to commit invalid message")
                }
                continue
            }

            // минимальная валидация: наличие order_uid
            if o.OrderUID == "" {
                log.Error().Msg("message missing order_uid; skipping")
                if err2 := r.CommitMessages(ctx, m); err2 != nil {
                    log.Error().Err(err2).Msg("failed to commit bad message")
                } // Если отсутствует: зарегистрировать, зафиксировать (пропустить), продолжить.
                continue
            }

            // Сохраняем в БД (в транзакции внутри)
            if err := db.SaveOrder(ctx, &o); err != nil {
                // при ошибке БД — НЕ фиксируем сообщение, чтобы попытаться позже (at-least-once)
                log.Error().Err(err).Str("order_uid", o.OrderUID).Msg("failed to save order")
                // Запись в журнал и кратковременное тушение(сон);
                time.Sleep(time.Second)
                continue
            }

            // Успешно обработано — добавляем в кеш и фиксируем смещение сообщения. Записать в журнал успешное выполнение.
            cache.Set(&o)
            if err := r.CommitMessages(ctx, m); err != nil {
                log.Error().Err(err).Msg("commit message failed")
            } else {
                log.Info().Str("order_uid", o.OrderUID).Msg("message processed and committed")
            }
        }
    }()
}