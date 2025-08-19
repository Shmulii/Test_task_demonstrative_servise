package main

import "sync"

type Cache struct {
    mu sync.RWMutex
    m  map[string]*Order
    limit int
}
// Read-write lock, Карта хранения и максимум входящих допущеных запросов в кэш

func NewCache(limit int) *Cache {
    return &Cache{
        m: make(map[string]*Order),
        limit: limit,
    }
}
// Конструктор. Инициализирует внутренюю карту и сохраняет ограничение по емкости 

func (c *Cache) Get(uid string) (*Order, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    o, ok := c.m[uid]
    return o, ok
}
/* Операция чтения. Устанавливает блокировку чтения, чтобы разрешить параллельное чтение.
Ищет uid в карте и возвращает порядок и флаг ok. Разблокируется через defer.*/

func (c *Cache) Set(o *Order) {
    c.mu.Lock()
    defer c.mu.Unlock()
    if len(c.m) >= c.limit {
        // простая стратегия: не добавляем если превышен лимит (LRU)
        return
    }
    c.m[o.OrderUID] = o
}
// Операция записи. Устанавливает блокировку записи (Lock) для обеспечения исключительного изменения.