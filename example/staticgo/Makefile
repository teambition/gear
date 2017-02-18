GO=$(shell which go)

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -a -installsuffix cgo -o staticgo_linux staticgo.go
build-osx:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -a -installsuffix cgo -o staticgo staticgo.go
build-win:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -a -installsuffix cgo -o staticgo.exe staticgo.go
build: build-osx build-linux build-win

.PHONY: build
