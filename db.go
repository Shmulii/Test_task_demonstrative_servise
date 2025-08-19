package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool" // Для объединения{pooling} Postgres
)

type DB struct {
	pool *pgxpool.Pool
} // Оберточная струкутра вокруг пула

func NewDB(ctx context.Context, dsn string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil { // Анализирует DSN в конфигурацию пула
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil { // Создаёт pgxpool.Pool с этой конфигурацией
		return nil, err
	}
	return &DB{pool: pool}, nil // Возвращает БД с готовым пулом
}

func (db *DB) Close() {
	db.pool.Close() // Закрывает базовый пул (при завершении работы)
}

func (db *DB) SaveOrder(ctx context.Context, o *Order) error {
	tx, err := db.pool.Begin(ctx) // Начинает транзакцию
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback(ctx) // Защита от отложенного отката
		}
	}()

	// Выполняет вставку в заказы (вставляет или обновляет все поля верхнего уровня при конфликте)
	_, err = tx.Exec(ctx, `
        INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
        ON CONFLICT (order_uid) DO UPDATE SET
          track_number=EXCLUDED.track_number,
          entry=EXCLUDED.entry,
          locale=EXCLUDED.locale,
          internal_signature=EXCLUDED.internal_signature,
          customer_id=EXCLUDED.customer_id,
          delivery_service=EXCLUDED.delivery_service,
          shardkey=EXCLUDED.shardkey,
          sm_id=EXCLUDED.sm_id,
          date_created=EXCLUDED.date_created,
          oof_shard=EXCLUDED.oof_shard
    `, o.OrderUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature, o.CustomerID, o.DeliveryService, o.Shardkey, o.SmID, o.DateCreated, o.OofShard)
	if err != nil {
		return err
	}

	// Очищает и повторно вставляет доставки для этого order_uid
	_, err = tx.Exec(ctx, `DELETE FROM deliveries WHERE order_uid=$1`, o.OrderUID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
        INSERT INTO deliveries (order_uid, name, phone, zip, city, address, region, email)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
    `, o.OrderUID, o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City, o.Delivery.Address, o.Delivery.Region, o.Delivery.Email)
	if err != nil {
		return err
	}

	// Очищает и повторно вставляет платежи для этого order_uid
	_, err = tx.Exec(ctx, `DELETE FROM payments WHERE order_uid=$1`, o.OrderUID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
        INSERT INTO payments (order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
    `, o.OrderUID, o.Payment.Transaction, o.Payment.RequestID, o.Payment.Currency, o.Payment.Provider, o.Payment.Amount, o.Payment.PaymentDt, o.Payment.Bank, o.Payment.DeliveryCost, o.Payment.GoodsTotal, o.Payment.CustomFee)
	if err != nil {
		return err
	}

	// Очищает и повторно вставляет все товары для этого order_uid
	_, err = tx.Exec(ctx, `DELETE FROM items WHERE order_uid=$1`, o.OrderUID)
	if err != nil {
		return err
	}
	for _, it := range o.Items {
		_, err = tx.Exec(ctx, `
            INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
        `, o.OrderUID, it.ChrtID, it.TrackNumber, it.Price, it.Rid, it.Name, it.Sale, it.Size, it.TotalPrice, it.NmID, it.Brand, it.Status)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	// Фиксирует транзакцию; предотвращает отложенный откат, устанавливая tx = nil.
	tx = nil
	return nil
}

func (db *DB) LoadRecentOrders(ctx context.Context, limit int) ([]Order, error) {
	query := `
		SELECT o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, 
			   o.customer_id, o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
			   d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
			   p.transaction, p.request_id, p.currency, p.provider, p.amount, p.payment_dt, 
			   p.bank, p.delivery_cost, p.goods_total, p.custom_fee
		FROM orders o
		LEFT JOIN deliveries d ON o.order_uid = d.order_uid
		LEFT JOIN payments p ON o.order_uid = p.order_uid
		ORDER BY o.date_created DESC
		LIMIT $1
	` /* Запрашивает последние заказы, объединенные с доставками и платежами,
	упорядоченными по дате создания (desc), с ограничением по лимиту.*/

	rows, err := db.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var d Delivery
		var p Payment
		// Сканирует в заказ и отдельные структуры доставки и оплаты
		err := rows.Scan(
			&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature,
			&o.CustomerID, &o.DeliveryService, &o.Shardkey, &o.SmID, &o.DateCreated, &o.OofShard,
			&d.Name, &d.Phone, &d.Zip, &d.City, &d.Address, &d.Region, &d.Email,
			&p.Transaction, &p.RequestID, &p.Currency, &p.Provider, &p.Amount, &p.PaymentDt,
			&p.Bank, &p.DeliveryCost, &p.GoodsTotal, &p.CustomFee,
		)
		if err != nil {
			return nil, err
		}
		// Назначает доставку и оплату обратно заказу.
		o.Delivery = d
		o.Payment = p

		// Загружает товары с помощью loadOrderItems для каждого заказа
		items, err := db.loadOrderItems(ctx, o.OrderUID)
		if err != nil {
			return nil, err
		}
		o.Items = items

		orders = append(orders, o)
	}

	return orders, rows.Err()
}

func (db *DB) GetOrder(ctx context.Context, orderUID string) (*Order, error) {
	query := `
		SELECT o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, 
			   o.customer_id, o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
			   d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
			   p.transaction, p.request_id, p.currency, p.provider, p.amount, p.payment_dt, 
			   p.bank, p.delivery_cost, p.goods_total, p.custom_fee
		FROM orders o
		LEFT JOIN deliveries d ON o.order_uid = d.order_uid
		LEFT JOIN payments p ON o.order_uid = p.order_uid
		WHERE o.order_uid = $1
	`
	//Получает отдельный заказ (вместе с доставками и платежами) по order_uid.
	var o Order
	var d Delivery
	var p Payment
	// Сканирует в Order, Delivery, Payment; назначает их
	err := db.pool.QueryRow(ctx, query, orderUID).Scan(
		&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature,
		&o.CustomerID, &o.DeliveryService, &o.Shardkey, &o.SmID, &o.DateCreated, &o.OofShard,
		&d.Name, &d.Phone, &d.Zip, &d.City, &d.Address, &d.Region, &d.Email,
		&p.Transaction, &p.RequestID, &p.Currency, &p.Provider, &p.Amount, &p.PaymentDt,
		&p.Bank, &p.DeliveryCost, &p.GoodsTotal, &p.CustomFee,
	)
	if err != nil {
		return nil, err
	}

	o.Delivery = d
	o.Payment = p

	// Загружает товары
	items, err := db.loadOrderItems(ctx, o.OrderUID)
	if err != nil {
		return nil, err
	}
	o.Items = items

	return &o, nil
}

func (db *DB) loadOrderItems(ctx context.Context, orderUID string) ([]Item, error) {
	query := `
		SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
		FROM items
		WHERE order_uid = $1
	`
	// Выбирает все столбцы элементов из элементов, отфильтрованных по order_uid.
	rows, err := db.pool.Query(ctx, query, orderUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// Сканирует каждую строку в элемент и добавляет к результирующему срезу
	var items []Item
	for rows.Next() {
		var item Item
		err := rows.Scan(
			&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name,
			&item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}
