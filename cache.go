package containers

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Шаблон функции для использования при необходимости поиска объекта по отличным от ключа полям
type SearchMethod func(key, item interface{}) bool

// Шаблон функции для случаев, когда необходимо установить объект кэша, при этом быть уверенным, что его в карте нет
type LockedLoadMethod func(key interface{}, calls CacheCallBacks) (item, check interface{})

// Специфический метод выборки из базы, когда необходимо выбрать элеметн не по его идентификатору, а по другим признакам
type SearchLoadMethod func() (key interface{}, value interface{})

// Шаблон метода инициализации объекта на стороне вызывающего объекта. Используется при необходимости иницилизировать объект, если он отсутствует в хрвнилище.
type CreateMethod func(key interface{}, calls CacheCallBacks) (interface{}, bool)

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

type CacheCallBacks struct {
	Set    func(interface{}, interface{})
	Delete func(interface{})
	Get    func(interface{}) (interface{}, bool)
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

func (s *cache) set(key, value interface{}) {
	s.items[key] = &cacheItem{value, time.Now().Add(s.expired).UnixNano()}
	if !s.cleanerWork {
		s.cleanerWork = true
		go s.runCleaner()
	}
}

func (s *cache) get(key interface{}) (res interface{}, check bool) {
	var item *cacheItem
	if item, check = s.items[key]; check {
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
func (s *cache) Get(key interface{}) (res interface{}, check bool) {
	s.locker.RLock()
	res, check = s.get(key)
	s.locker.RUnlock()
	return
}

// Метод, реализующий инициализацию нового объекта при отсутствии его в хранилище.
// Если найти объект в хранилище не удалось, будет вызвана callback-функция createCall, в которой
// необходимо создать объект для хранения и вернуть его (или false вторым аргументом, если инициализация объекта невозможна)
// Внимание! В момент вызова createCall хранилище заблокировано для других горутин, поэтому
// рекомендуется выполнять в createCall минимум операций, чтобы как можно скорее вернуть управление объекту хранилища!
func (s *cache) GetOrCreate(key interface{}, createCall CreateMethod) (res interface{}, check bool) {
	if res, check = s.Get(key); !check {
		s.locker.Lock()
		if res, check = s.get(key); check {
			s.locker.Unlock()
			return
		}
		if res, check = createCall(key, CacheCallBacks{s.set, s.delete, s.get}); check {
			s.set(key, res)
		}
		s.locker.Unlock()
	}
	return
}

// Поиск объекта по другим признакам, кроме ключа (каким именно, определяется методом - агрументом на вход)
// "Время жизни" найденного объекта увеличивается.
func (s *cache) Search(method SearchMethod) (res interface{}, check bool) {
	s.locker.RLock()
	for i, v := range s.items {
		obj := v.object
		if check = method(i, obj); check {
			res = obj
			if time.Now().Add(s.expired).UnixNano() > v.expire {
				atomic.AddInt64(&v.expire, int64(s.expired))
			}
			s.locker.RUnlock()
			return
		}
	}
	s.locker.RUnlock()
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
					removedItems = append(removedItems, v)
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

func (s *cache) LockedLoad(callback func(CacheCallBacks) (interface{}, bool)) (item interface{}, check bool) {
	s.locker.Lock()
	if item, check = callback(CacheCallBacks{s.set, s.delete, s.get}); check {
		s.items[key] = &cacheItem{item, time.Now().Add(s.expired).UnixNano()}
	}
	s.locker.Unlock()
	return
}

func (s *cache) LockedLoadSearch(callSearch SearchMethod, callLoad SearchLoadMethod) (item interface{}, check bool) {
	s.locker.Lock()
	for key, val := range s.items {
		if callSearch(key, val.object) {
			s.locker.Unlock()
			return val.object, true
		}
	}
	var key interface{}
	if key, item = callLoad(); key != nil {
		s.items[key], check = &cacheItem{item, time.Now().Add(s.expired).UnixNano()}, true
	}
	s.locker.Unlock()
	return
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
