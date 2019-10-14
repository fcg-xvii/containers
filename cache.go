package containers

import (
	_ "log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Шаблон метода инициализации объекта на стороне вызывающего объекта. Используется при необходимости иницилизировать объект, если он отсутствует в хрвнилище.
type CreateMethod func(key interface{}) (interface{}, interface{}, bool)

type CheckMethod func(val interface{}) bool

// Структура для хранения элементов
type cacheItem struct {
	object interface{} // Исходный объект
	expire int64       // Временная отметка, после наступления которой объект будет удалён клинером
}

// Конструктор объекта кэша
func NewCache(cleanInterval, itemExpired time.Duration, clearPrepare func([]interface{})) *Cache {
	cache := &Cache{&cache{
		locker:   new(sync.RWMutex),
		items:    make(map[interface{}]*cacheItem),
		interval: cleanInterval, expired: itemExpired,
		stopCleanerChan: make(chan bool),
		clearPrepare:    clearPrepare,
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
	clearPrepare      func([]interface{})        // Пользовательский метод, в который передаются объекты перед удалением
}

func (s *cache) Keys() []interface{} {
	s.locker.RLock()
	res := make([]interface{}, 0, len(s.items))
	for key, _ := range s.items {
		res = append(res, key)
	}
	s.locker.RUnlock()
	return res
}

func (s *cache) set(key, value interface{}) {
	s.items[key] = &cacheItem{value, time.Now().Add(s.expired).UnixNano()}
	if !s.cleanerWork && s.expired > 0 {
		s.cleanerWork = true
		go s.runCleaner()
	}
}

func (s *cache) get(key interface{}, cCall CheckMethod) (res interface{}, check bool) {
	var item *cacheItem
	if item, check = s.items[key]; check {
		if cCall != nil && !cCall(item.object) {
			check = false
			return
		}
		res = item.object
		if time.Now().Add(s.expired).UnixNano() > atomic.LoadInt64(&item.expire) {
			atomic.AddInt64(&item.expire, int64(s.expired))
		}
	}
	return
}

// Установка объекта по ключу
func (s *cache) Set(key, value interface{}) {
	s.locker.Lock()
	s.set(key, value)
	s.locker.Unlock()
}

// Поиск объекта по ключу. Если объект найден, увелививается его "время жизни"
func (s *cache) Get(key interface{}, cCall CheckMethod) (res interface{}, check bool) {
	s.locker.RLock()
	res, check = s.get(key, cCall)
	s.locker.RUnlock()
	return
}

// Метод, реализующий инициализацию нового объекта при отсутствии его в хранилище.
// Если найти объект в хранилище не удалось, будет вызвана callback-функция createCall, в которой
// необходимо создать объект для хранения и вернуть его (или false вторым аргументом, если инициализация объекта невозможна)
// Внимание! В момент вызова createCall хранилище заблокировано для других горутин, поэтому
// рекомендуется выполнять в createCall минимум операций, чтобы как можно скорее вернуть управление объекту хранилища!
func (s *cache) GetOrCreate(key interface{}, cCall CheckMethod, createCall CreateMethod) (res interface{}, check bool) {
	if res, check = s.Get(key, cCall); !check {
		s.locker.Lock()
		if res, check = s.get(key, cCall); check {
			s.locker.Unlock()
			return
		}

		if key, res, check = createCall(key); check {

			s.set(key, res)
		}
		s.locker.Unlock()
	}
	return
}

func (s *cache) Each(key interface{}, checkCall CheckMethod, createCall CreateMethod) (res interface{}, check bool) {
	s.locker.Lock()
	for _, v := range s.items {
		if checkCall(v.object) {
			s.locker.Unlock()
			return v, true
		}
	}
	s.locker.Unlock()
	return
}

// Запуск клинера (запускается при непустой карте объектов и останавливается при пустой)
func (s *cache) runCleaner() {
	ticker := time.NewTicker(s.interval)
	for {
		select {
		case <-ticker.C: // По сигналу тикера начинаем удаление устаревших объектов
			now := time.Now().UnixNano()
			var removedItems []interface{}
			s.locker.Lock()
			for key, v := range s.items {
				if now > v.expire {
					removedItems = append(removedItems, v.object)
					delete(s.items, key)
				}
			}
			// Если карта объектов пуста, завершаем работу клинерв
			if len(s.items) == 0 {
				s.cleanerWork = false
				ticker.Stop()
				s.locker.Unlock()
				if s.clearPrepare != nil && len(removedItems) > 0 {
					s.clearPrepare(removedItems)
				}
				return
			}
			s.locker.Unlock()
			if s.clearPrepare != nil && len(removedItems) > 0 {
				s.clearPrepare(removedItems)
			}
		case <-s.stopCleanerChan: // Деструктор, запущеный сборщиком мусора, закрывает канал, завершаем работу клинера
			s.cleanerWork = false
			ticker.Stop()
			return
		}
	}
}

// Возвращает количество элементов в хранилище хэша
func (s *cache) Len() int {
	s.locker.RLock()
	res := len(s.items)
	s.locker.RUnlock()
	return res
}

func (s *cache) delete(key interface{}) {
	delete(s.items, key)
}

func (s *cache) Delete(key interface{}) {
	s.locker.Lock()
	s.delete(key)
	s.locker.Unlock()
}

// Деструктор, вызываемый сборщиком мусора
func destroyCache(cache *Cache) {
	close(cache.stopCleanerChan) // Канал передаст сигнал о своём закрытии клинеру, который закроется, если он запущен
}
