/*
Copyright 2014 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Simple proxy for tcp connections between a localhost:lport and services that provide
// the actual implementations.

package proxy

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"kubernetes/pkg/api"
)

type Proxier struct {
	loadBalancer LoadBalancer
	serviceMap   map[string]int
}

func NewProxier(loadBalancer LoadBalancer) *Proxier {
	return &Proxier{loadBalancer: loadBalancer, serviceMap: make(map[string]int)}
}

func CopyBytes(in, out *net.TCPConn) {
	log.Printf("Copying from %v <-> %v <-> %v <-> %v",
		in.RemoteAddr(), in.LocalAddr(), out.LocalAddr(), out.RemoteAddr())
	_, err := io.Copy(in, out)
	if err != nil && err != io.EOF {
		log.Printf("I/O error: %v", err)
	}

	in.CloseRead()
	out.CloseWrite()
}

// Create a bidirectional byte shuffler. Copies bytes to/from each connection.
func ProxyConnection(in, out *net.TCPConn) {
	log.Printf("Creating proxy between %v <-> %v <-> %v <-> %v",
		in.RemoteAddr(), in.LocalAddr(), out.LocalAddr(), out.RemoteAddr())
	go CopyBytes(in, out)
	go CopyBytes(out, in)
}

func (proxier Proxier) AcceptHandler(service string, listener net.Listener) {
	for {
		// 等待请求端口，并返回一个conn请求
		inConn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept failed: %v", err)
			continue
		}
		log.Printf("Accepted connection from: %v to %v", inConn.RemoteAddr(), inConn.LocalAddr())

		// Figure out where this request should go.
		// 找出这个请求应该发送到哪里
		endpoint, err := proxier.loadBalancer.LoadBalance(service, inConn.RemoteAddr())
		if err != nil {
			log.Printf("Couldn't find an endpoint for %s %v", service, err)
			inConn.Close()
			continue
		}

		log.Printf("Mapped service %s to endpoint %s", service, endpoint)
		// 在网络network上连接地址endpoint，并返回一个Conn接口，5秒超时
		outConn, err := net.DialTimeout("tcp", endpoint, time.Duration(5)*time.Second)
		// We basically need to take everything from inConn and send to outConn
		// and anything coming from outConn needs to be sent to inConn.
		if err != nil {
			log.Printf("Dial failed: %v", err)
			inConn.Close()
			continue
		}
		// 将inConn的东西发送到outConn，任何outConn的东西也要发送到inConn
		go ProxyConnection(inConn.(*net.TCPConn), outConn.(*net.TCPConn))
	}
}

// AddService starts listening for a new service on a given port.
func (proxier Proxier) AddService(service string, port int) error {
	// Make sure we can start listening on the port before saying all's well.
	// 监听端口
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	log.Printf("Listening for %s on %d", service, port)
	// If that succeeds, start the accepting loop.
	go proxier.AcceptHandler(service, ln)
	return nil
}

func (proxier Proxier) OnUpdate(services []api.Service) {
	log.Printf("Received update notice: %+v", services)
	for _, service := range services {
		// 查看该服务id和端口的映射关系
		port, exists := proxier.serviceMap[service.ID]
		// 如果不存在或者是port发生了改变
		if !exists || port != service.Port {
			log.Printf("Adding a new service %s on port %d", service.ID, service.Port)
			err := proxier.AddService(service.ID, service.Port)
			if err == nil {
				// 添加成功，更新serviceHandler的配置
				proxier.serviceMap[service.ID] = service.Port
			} else {
				log.Printf("Failed to start listening for %s on %d", service.ID, service.Port)
			}
		}
	}
}
