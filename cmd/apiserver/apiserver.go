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
// apiserver is the main api server and master for the cluster.
// it is responsible for serving the cluster management API.
package main

import (
	"flag"
	"fmt"
	"kubernetes/pkg/apiserver"
	kube_client "kubernetes/pkg/client"
	"kubernetes/pkg/registry"
	"kubernetes/pkg/util"
	"kubernetes/third_party/github.com/coreos/go-etcd/etcd"
	"log"
	"net/http"
	"time"
)

var (
	port                        = flag.Uint("port", 8080, "The port to listen on.  Default 8080.")
	address                     = flag.String("address", "127.0.0.1", "The address on the local server to listen to. Default 127.0.0.1")
	apiPrefix                   = flag.String("api_prefix", "/api/v1beta1", "The prefix for API requests on the server. Default '/api/v1beta1'")
	etcdServerList, machineList util.StringList
)

func init() {
	flag.Var(&etcdServerList, "etcd_servers", "Servers for the etcd (http://ip:port), comma separated")
	flag.Var(&machineList, "machines", "List of machines to schedule onto, comma separated.")
}

func main() {
	flag.Parse()

	if len(machineList) == 0 {
		log.Fatal("No machines specified!")
	}

	// 定义registry，不同的registry主要是对不同的registry做增删改查操作
	var (
		taskRegistry       registry.TaskRegistry
		controllerRegistry registry.ControllerRegistry
		serviceRegistry    registry.ServiceRegistry
	)
	// 如果etcd的服务不存在，则使用内存的registry
	if len(etcdServerList) > 0 {
		log.Printf("Creating etcd client pointing to %v", etcdServerList)
		// 创建etcd客户端
		etcdClient := etcd.NewClient(etcdServerList)
		taskRegistry = registry.MakeEtcdRegistry(etcdClient, machineList)
		controllerRegistry = registry.MakeEtcdRegistry(etcdClient, machineList)
		serviceRegistry = registry.MakeEtcdRegistry(etcdClient, machineList)
	} else {
		taskRegistry = registry.MakeMemoryRegistry()
		controllerRegistry = registry.MakeMemoryRegistry()
		serviceRegistry = registry.MakeMemoryRegistry()
	}

	containerInfo := &kube_client.HTTPContainerInfo{
		Client: http.DefaultClient,
		Port:   10250,
	}
	// 创建一个map，将registry包装成registryStorage（里面有调度算法等等）并设置进map，task（将来的pod），Scheduler默认使用首次适应算法
	storage := map[string]apiserver.RESTStorage{
		"tasks":                  registry.MakeTaskRegistryStorage(taskRegistry, containerInfo, registry.MakeFirstFitScheduler(machineList, taskRegistry)),
		"replicationControllers": registry.MakeControllerRegistryStorage(controllerRegistry),
		"services":               registry.MakeServiceRegistryStorage(serviceRegistry),
	}
	// 创建端点控制器
	endpoints := registry.MakeEndpointController(serviceRegistry, taskRegistry)
	// 在协程中每隔10秒更新service的端点
	go util.Forever(func() { endpoints.SyncServiceEndpoints() }, time.Second*10)

	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", *address, *port),
		Handler:        apiserver.New(storage, *apiPrefix), //这个是处理服务器请求的结构体
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
