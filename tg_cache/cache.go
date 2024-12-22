package tg_cache

import (
	"errors"
	"github.com/mxcker/telegram/tg_client"
	"sync"
)

type ClientCache struct {
	waitMap  sync.Map
	readyMap sync.Map
	OnReady  func(id any)
}

const (
	ClientCacheNull = iota
	ClientCacheWaiting
	ClientCacheReady
)

var ErrClientWaiting = errors.New("the tg_client is logging in")
var ErrClientNotExist = errors.New("the tg_client does not exist")

func (c *ClientCache) IdStatusMap() map[any]int {
	onlineMap := make(map[any]int)
	c.waitMap.Range(func(key, value interface{}) bool {
		onlineMap[key] = ClientCacheWaiting
		return true
	})
	c.readyMap.Range(func(key, value interface{}) bool {
		onlineMap[key] = ClientCacheReady
		return true
	})
	return onlineMap
}

func (c *ClientCache) Remove(id any) {
	wait, loaded := c.waitMap.LoadAndDelete(id)
	if loaded {
		wait.(*tg_client.Client).Done(nil)
	}
	ready, loaded := c.readyMap.LoadAndDelete(id)
	if loaded {
		ready.(*tg_client.Client).Done(nil)
	}
}

func (c *ClientCache) Get(id any) (*tg_client.Client, error) {
	value, ok := c.readyMap.Load(id)
	if ok {
		return value.(*tg_client.Client), nil
	} else {
		_, has := c.waitMap.Load(id)
		if has {
			return nil, ErrClientWaiting
		} else {
			return nil, ErrClientNotExist
		}
	}
}

func (c *ClientCache) Add(id any, clientConfig *tg_client.Config) error {
	if _, ok := c.readyMap.Load(id); ok {
		return errors.New("tg_client already ready")
	}
	value, has := c.waitMap.LoadOrStore(id, tg_client.NewClient(clientConfig))
	if !has {
		go func() {
			defer c.waitMap.Delete(id)
			cl := value.(*tg_client.Client)
			err := cl.Start()
			if err == nil {
				c.readyMap.Store(id, cl)
				if c.OnReady != nil {
					c.OnReady(id)
				}
			}
		}()
		return nil
	} else {
		return errors.New("tg_client already exist")
	}
}
