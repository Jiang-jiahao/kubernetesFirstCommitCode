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
// The kubelet binary is responsible for maintaining a set of containers on a particular host VM.
// It sync's data from both configuration file as well as from a quorum of etcd servers.
// It then queries Docker to see what is currently running.  It synchronizes the configuration data,
// with the running set of containers by starting or stopping Docker containers.
package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	"kubernetes/pkg/kubelet"
	"kubernetes/third_party/github.com/coreos/go-etcd/etcd"
	"kubernetes/third_party/github.com/fsouza/go-dockerclient"
)

var (
	file               = flag.String("config", "", "Path to the config file")
	etcd_servers       = flag.String("etcd_servers", "", "Url of etcd servers in the cluster")
	syncFrequency      = flag.Duration("sync_frequency", 10*time.Second, "Max seconds between synchronizing running containers and config")
	fileCheckFrequency = flag.Duration("file_check_frequency", 20*time.Second, "Seconds between checking file for new data")
	httpCheckFrequency = flag.Duration("http_check_frequency", 20*time.Second, "Seconds between checking http for new data")
	manifest_url       = flag.String("manifest_url", "", "URL for accessing the container manifest")
	address            = flag.String("address", "127.0.0.1", "The address for the info server to serve on")
	port               = flag.Uint("port", 10250, "The port for the info server to serve on")
)

const dockerBinary = "/usr/bin/docker"

func main() {
	//解析命令行参数，在解析之前是使用的默认值
	flag.Parse()
	rand.Seed(time.Now().UTC().UnixNano())

	// Set up logger for etcd client
	etcd.SetLogger(log.New(os.Stderr, "etcd ", log.LstdFlags))
	// 该docker.sock与docker的守护进程通讯用
	endpoint := "unix:///var/run/docker.sock"
	dockerClient, err := docker.NewClient(endpoint)
	if err != nil {
		log.Fatal("Couldn't connnect to docker.")
	}

	my_kubelet := kubelet.Kubelet{
		DockerClient:       dockerClient,
		FileCheckFrequency: *fileCheckFrequency,
		SyncFrequency:      *syncFrequency,
		HTTPCheckFrequency: *httpCheckFrequency,
	}
	my_kubelet.RunKubelet(*file, *manifest_url, *etcd_servers, *address, *port)
}
