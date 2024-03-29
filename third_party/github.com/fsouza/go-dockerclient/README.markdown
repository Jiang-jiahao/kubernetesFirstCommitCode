#go-dockerclient

[![Build Status](https://drone.io/github.com/fsouza/go-dockerclient/status.png)](https://drone.io/github.com/fsouza/go-dockerclient/latest)
[![Build Status](https://travis-ci.org/fsouza/go-dockerclient.png)](https://travis-ci.org/fsouza/go-dockerclient)

[![GoDoc](http://godoc.org/github.com/fsouza/go-dockerclient?status.png)](http://godoc.org/github.com/fsouza/go-dockerclient)

This package presents a client for the Docker remote API.

For more details, check the [remote API documentation](http://docs.docker.io/en/latest/reference/api/docker_remote_api/).

##Versioning

* Version 0.1 is compatible with Docker v0.7.1
* The master is compatible with Docker's master


## Example

    package main

    import (
            "fmt"
            "kubernetes/third_party/github.com/fsouza/go-dockerclient"
    )

    func main() {
            endpoint := "unix:///var/run/docker.sock"
            client, _ := docker.NewClient(endpoint)
            imgs, _ := client.ListImages(true)
            for _, img := range imgs {
                    fmt.Println("ID: ", img.ID)
                    fmt.Println("RepoTags: ", img.RepoTags)
                    fmt.Println("Created: ", img.Created)
                    fmt.Println("Size: ", img.Size)
                    fmt.Println("VirtualSize: ", img.VirtualSize)
                    fmt.Println("ParentId: ", img.ParentId)
                    fmt.Println("Repository: ", img.Repository)
            }
    }

## Developing

You can run the tests with:

    go get -d ./...
    go test ./...
