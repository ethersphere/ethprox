package callback

import (
	"sync"

	"github.com/ethersphere/ethproxy/pkg/rpc"
)

type handler func(resp *rpc.JsonrpcMessage)

type Callback struct {
	mtx      sync.Mutex
	handlers map[string][]handler
}

func New() *Callback {
	return &Callback{
		handlers: make(map[string][]handler),
	}
}

func (c *Callback) On(method string, f handler) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.handlers[method] = append(c.handlers[method], f)
}

func (c *Callback) Remove(method string, f handler) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	delete(c.handlers, method)
}

func (c *Callback) Run(resp *rpc.JsonrpcMessage) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	for _, h := range c.handlers[resp.Method] {
		h(resp)
	}
}