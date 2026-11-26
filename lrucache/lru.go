//go:build !solution

package lrucache

import "container/list"

type entry struct {
	key   int
	value int
}

type LRUCache struct {
	cap  int
	mapa map[int]*list.Element
	list *list.List
}

func (c *LRUCache) Get(key int) (int, bool) {
	element, exists := c.mapa[key]
	if !exists {
		return 0, false
	}
	c.list.MoveToFront(element)
	item := element.Value.(entry)
	return item.value, true
}

func (c *LRUCache) Set(key, value int) {
	if element, exists := c.mapa[key]; exists {
		element.Value = entry{key: key, value: value}
		c.list.MoveToFront(element)
	} else {
		element := c.list.PushFront(entry{key: key, value: value})
		c.mapa[key] = element
		if c.list.Len() > c.cap {
			last := c.list.Back()
			item := last.Value.(entry)
			delete(c.mapa, item.key)
			c.list.Remove(last)
		}
	}

}

func (c *LRUCache) Range(f func(key, value int) bool) {
	for element := c.list.Back(); element != nil; element = element.Prev() {
		item := element.Value.(entry)
		if !f(item.key, item.value) {
			break
		}
	}
}

func (c *LRUCache) Clear() {
	c.mapa = make(map[int]*list.Element)
	c.list.Init()
}

func New(cap int) Cache {
	return &LRUCache{
		cap:  cap,
		mapa: make(map[int]*list.Element),
		list: list.New(),
	}
}
