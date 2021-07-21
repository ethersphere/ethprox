// Copyright 2021 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proxy

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/ethersphere/ethproxy/pkg/callback"
	"github.com/ethersphere/ethproxy/pkg/ethrpc"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:    1024,
	WriteBufferSize:   1024,
	EnableCompression: false,
}

type proxy struct {
	call            *callback.Callback
	backendEndpoint string
}

func NewProxy(call *callback.Callback, port, backendEndpoing string) *http.Server {

	m := http.NewServeMux()

	proxy := &proxy{
		call:            call,
		backendEndpoint: backendEndpoing,
	}

	m.HandleFunc("/", proxy.wsRoute)

	return &http.Server{
		Addr:    ":" + port,
		Handler: m,
	}
}

func (p *proxy) wsRoute(w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("proxy: %v\n", err)
		return
	}
	defer conn.Close()

	backend, err := p.backendClient()
	if err != nil {
		fmt.Printf("proxy: %v\n", err)
		return
	}
	defer backend.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			t, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			fmt.Printf("CLIENT %v\n", string(msg))

			err = backend.WriteMessage(t, msg)
			if err != nil {
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for {
			t, msg, err := backend.ReadMessage()
			if err != nil {
				return
			}

			fmt.Printf("BACKEND %v\n", string(msg))

			msg, err = p.process(r, msg)
			if err != nil {
				log.Print(err)
			}

			err = conn.WriteMessage(t, msg)
			if err != nil {
				return
			}
		}
	}()

	wg.Wait()
}

func (p *proxy) process(r *http.Request, msg []byte) ([]byte, error) {

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return msg, err
	}

	jmsg, err := ethrpc.Unmarshall(msg)
	if err != nil {
		return msg, err
	}

	p.call.Run(&callback.Response{
		Body: jmsg,
		IP:   ip,
	})

	bjmsg, err := jmsg.Marshall()
	if err != nil {
		return msg, err
	}

	return bjmsg, nil
}

func (p *proxy) backendClient() (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(p.backendEndpoint, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
