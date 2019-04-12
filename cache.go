package containers

import (
	"runtime"
	"sync"
	"time"
)

// Шаблон функции для использования при необходимости поиска объекта по отличным от ключа полям
type SearchMethod func(key, item interface{}) bool

// Структура для хранения элементов
type cacheItem struct {
	object interface{} // Исходный объект
	expire int64       // Временная отметка, после наступления которой объект будет удалён клинером
}

// Конструктор объекта кэша
func NewCache(cleanInterval, itemExpired time.Duration) *Cache {
	cache := &Cache{&cache{
		locker:   new(sync.RWMutex),
		items:    make(map[interface{}]*cacheItem),
		interval: cleanInterval, expired: itemExpired,
		stopCleanerChan: make(chan bool),
	}}
	runtime.SetFinalizer(cache, destroyCache)
	return cache
}

// Обёртка для рабочей структуры (когда будет удалена ссылка объект, при сборке мусора
// будет вызвана функция финализера (деструктора), которая остановит горутину клинера, если она запущена)
type Cache struct {
	*cache
}

// Рабочая структура
type cache struct {
	locker            *sync.RWMutex              // Мьютекс для работы с картой объектов
	items             map[interface{}]*cacheItem // Карта объектов
	interval, expired time.Duration              // Интервал активации клинера и время жизни объекта
	stopCleanerChan   chan bool                  // Канал для остановки клинера (закрывается в деструкторе)
	cleanerWork       bool                       // Флаг, указывающий на активность клинера
}

// Установка объекта по ключу
func (s *cache) Set(key, value interface{}) {
	s.locker.Lock()
	s.items[key] = &cacheItem{value, time.Now().Add(s.expired).UnixNano()}
	if !s.cleanerWork {
		s.cleanerWork = true
		go s.runCleaner()
	}
	s.locker.Unlock()
}

// Поиск объекта по ключу
func (s *cache) Get(key interface{}) (res interface{}, check bool) {
	s.locker.RLock()
	var item *cacheItem
	if item, check = s.items[key]; check {
		res, item.expire = item.object, time.Now().Add(s.expired).UnixNano()
	}
	s.locker.RUnlock()
	return
}

// Поиск объекта по другим признакам, кроме ключа (каким именно, определяется методом - агрументом на вход)
func (s *cache) Search(method SearchMethod) (res interface{}, check bool) {
	s.locker.RLock()
	for i, v := range s.items {
		obj := v.object
		if check = method(i, obj); check {
			res, v.expire = obj, time.Now().Add(s.expired).UnixNano()
			s.locker.RUnlock()
			return
		}
	}
	s.locker.RUnlock()
	return
}

// Запуск клинера (запускается при непустой карте объектов)
func (s *cache) runCleaner() {
	ticker := time.NewTicker(s.interval)
	for {
		select {
		case <-ticker.C:
			now := time.Now().UnixNano()
			s.locker.Lock()
			for key, v := range s.items {
				if now > v.expire {
					delete(s.items, key)
				}
			}
			if len(s.items) == 0 {
				s.cleanerWork = false
				ticker.Stop()
				s.locker.Unlock()
				return
			}
			s.locker.Unlock()
		case <-s.stopCleanerChan: // Деструктор, запущеный сборщиком мусора, закрыл канал, поэтому завершаем работу клинера
			s.cleanerWork = false
			ticker.Stop()
			return
		}
	}
}

// Деструктор, вызываемый сборщиком мусора
func destroyCache(cache *Cache) {
	close(cache.stopCleanerChan) // Канал передаст сигнал о своём закрытии клинеру, который закроется, если он запущен
}
