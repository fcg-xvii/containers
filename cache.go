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
type LockedLoadMethod func(key interface{}, callback func() (interface{}, bool)) (item, check interface{})

// Специфический метод выборки из базы, когда необходимо выбрать элеметн не по его идентификатору, а по другим признакам
type SearchLoadMethod func() (key interface{}, value interface{})

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

// Поиск объекта по ключу. Если объект найден, увелививается его "время жизни"
func (s *cache) Get(key interface{}) (res interface{}, check bool) {
	s.locker.RLock()
	var item *cacheItem
	if item, check = s.items[key]; check {
		res = item.object
		atomic.AddInt64(&item.expire, int64(s.expired))
	}
	s.locker.RUnlock()
	return
}

// Поиск объекта по другим признакам, кроме ключа (каким именно, определяется методом - агрументом на вход)
// "Время жизни" найденного объекта увеличивается.
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

// Запуск клинера (запускается при непустой карте объектов и останавливается при пустой)
func (s *cache) runCleaner() {
	ticker := time.NewTicker(s.interval)
	for {
		select {
		case <-ticker.C: // По сигналу тикера начинаем удаление устаревших объектов
			now := time.Now().UnixNano()
			s.locker.Lock()
			for key, v := range s.items {
				if now > v.expire {
					delete(s.items, key)
				}
			}
			// Если карта объектов пуста, завершаем работу клинерв
			if len(s.items) == 0 {
				s.cleanerWork = false
				ticker.Stop()
				s.locker.Unlock()
				return
			}
			s.locker.Unlock()
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

func (s *cache) LockedLoad(key interface{}, callback func() (interface{}, bool)) (item interface{}, check bool) {
	var cItem *cacheItem
	s.locker.Lock()
	if cItem, check = s.items[key]; check {
		item = cItem.object
		s.locker.Unlock()
		return
	}
	if item, check = callback(); check {
		s.items[key] = &cacheItem{item, time.Now().Add(s.expired).UnixNano()}
	}
	s.locker.Unlock()
	return
}

func (s *cache) LockedLoadSearch(callSearch SearchMethod, callLoad SearchLoadMethod) (item interface{}, check bool) {
	s.locker.Lock()
	for key, val := range s.items {
		if callSearch(key, val) {
			s.locker.Unlock()
			return val.object, true
		}
	}
	var key interface{}
	if key, item = callLoad(); key != nil {
		s.items[key] = &cacheItem{item, time.Now().Add(s.expired).UnixNano()}
	}
	s.locker.Unlock()
	return
}

func (s *cache) Delete(key interface{}) {
	s.locker.Lock()
	delete(s.items, key)
	s.locker.Unlock()
}

// Деструктор, вызываемый сборщиком мусора
func destroyCache(cache *Cache) {
	close(cache.stopCleanerChan) // Канал передаст сигнал о своём закрытии клинеру, который закроется, если он запущен
}
