# Magent

Magent is build for select k3s master / cloudcore according to etcd leader

## Requirement
* ipvsadm should be installed
* running magent as root

## Build 

### Build for arm64 linux

```shell
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o magent main.go
```

### Build for amd64 linux
```shell
GOOS=linux GOARCH=adm64 CGO_ENABLED=0 go build -o magent main.go
```

## Run
Running with VirtualIP of the k3s master, for example:
```shell
sudo ./magent "200.23.34.56:6443,200.23.34.56:10352"
```