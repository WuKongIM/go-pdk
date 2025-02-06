package pdk

import (
	"net/http"
	"strings"
	"sync"
)

type Route struct {
	postLock sync.RWMutex
	postMap  map[string]Handler

	getLock sync.RWMutex
	getMap  map[string]Handler
}

func newRoute() *Route {
	return &Route{
		postMap: make(map[string]Handler),
		getMap:  make(map[string]Handler),
	}
}

func (r *Route) POST(path string, handler Handler) {
	r.postLock.Lock()
	defer r.postLock.Unlock()
	r.postMap[path] = handler
}

func (r *Route) GET(path string, handler Handler) {
	r.getLock.Lock()
	defer r.getLock.Unlock()
	r.getMap[path] = handler
}

func (r *Route) handle(c *HttpContext) {
	method := strings.ToUpper(c.Request.Method)
	path := c.Request.Path
	switch method {
	case "POST":
		r.postLock.RLock()
		handler := r.postMap[path]
		r.postLock.RUnlock()
		if handler != nil {
			handler(c)
		} else {
			c.Response.Status = http.StatusNotFound
			c.Response.Body = []byte("404 page not found")
		}
	case "GET":
		r.getLock.RLock()
		handler := r.getMap[path]
		r.getLock.RUnlock()
		if handler != nil {
			handler(c)
		} else {
			c.Response.Status = http.StatusNotFound
			c.Response.Body = []byte("404 page not found")
		}
	}

}

type Handler func(*HttpContext)
