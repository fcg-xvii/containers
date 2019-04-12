package containers

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

//
//type CacheSearchMethod func(field interface{}, items map[interface{}]interface{}) (interface{}, bool)
//type CacheLoadMethod func(field)

type SearchMethod func(key, item interface{}) bool

type cacheItem struct {
	object interface{}
	expire int64
}

func NewCache(cleanInterval, itemExpired time.Duration) *Cache {
	cache := &Cache{
		locker:   new(sync.RWMutex),
		items:    make(map[interface{}]cacheItem),
		interval: cleanInterval, expired: itemExpired,
		stopCleanerChan: make(chan bool),
	}
	runtime.SetFinalizer(cache, destroyCache)
	return cache
}

type Cache struct {
	locker            *sync.RWMutex
	items             map[interface{}]cacheItem
	interval, expired time.Duration
	stopCleanerChan   chan bool
	cleanerWork       bool
}

func (s *Cache) Set(key, value interface{}) {
	s.locker.Lock()
	s.items[key] = cacheItem{value, time.Now().Add(s.expired).UnixNano()}
	if !s.cleanerWork {
		s.cleanerWork = true
		go s.runCleaner()
	}
	s.locker.Unlock()
}

func (s *Cache) Get(key interface{}) (res interface{}, check bool) {
	s.locker.RLock()
	var item cacheItem
	if item, check = s.items[key]; check {
		res = item.object
	}
	s.locker.RUnlock()
	return
}

func (s *Cache) Search(method SearchMethod) (res interface{}, check bool) {
	s.locker.RLock()
	for i, v := range s.items {
		obj := v.object
		if check = method(i, obj); check {
			res = obj
			s.locker.RUnlock()
			return
		}
	}
	s.locker.RUnlock()
	return
}

func (s *Cache) runCleaner() {
	fmt.Println("RUN_CLENER_STARTED")
	//defer fmt.Println("EXIT........")
	ticker := time.NewTicker(s.interval)
loop:
	for {
		select {
		case <-ticker.C:
			fmt.Println("TICK")
			now := time.Now().UnixNano()
			fmt.Println("FIRST_LENGTH", len(s.items))
			s.locker.Lock()
			for key, v := range s.items {
				if now > v.expire {
					delete(s.items, key)
				}
			}
			fmt.Println("ITEMS_LENGTH", len(s.items))
			if len(s.items) == 0 {
				ticker.Stop()
				s.locker.Unlock()
				break loop
			}
			s.locker.Unlock()
		case <-s.stopCleanerChan:
			ticker.Stop()
			break loop
		}
	}
	s.cleanerWork = true
	fmt.Println("CLOSED.........")
}

func destroyCache(cache *Cache) {
	fmt.Println("DESTROY_CACHE")
	close(cache.stopCleanerChan)
}
