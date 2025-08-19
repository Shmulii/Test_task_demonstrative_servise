package main

import "os"

type Config struct {
	PostgresDSN string // Подключение к Postgres 
	KafkaBroker string // брокер Kafka
	KafkaTopic  string // топик/тема Kafka
	HTTPAddr    string // адрес HTTP сервера (8080)
	CacheLimit  int // максимальное количество заказов в кеше оперативной памяти
	StartupLoad int // сколько последних заказов подгружаем в кеш
}

// Создаёт конфигурацию с переопределениями окружения и  значениями по умолчанию
func DefaultConfig() *Config {
	postgresDSN := os.Getenv("POSTGRES_DSN")
	if postgresDSN == "" {
		postgresDSN = "postgres://orders_user:orders_pass@localhost:5432/orders_db?sslmode=disable"
	}
//Считывает POSTGRES_DSN; если пусто, использует значение по умолчанию
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" {
		kafkaBroker = "localhost:9092"
	}
// Считывает KAFKA_BROKER; если пусто, использует localhost:9092.

	return &Config{
		PostgresDSN: postgresDSN,
		KafkaBroker: kafkaBroker,
		KafkaTopic:  "orders",
		HTTPAddr:    ":8080",
		CacheLimit:  1000,
		StartupLoad: 100,
	}
}
