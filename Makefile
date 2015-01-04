GOPATH := $(shell pwd)

build: $(shell find . -name '*.go')
	@go get github.com/rwcarlsen/goexif/exif
	@mkdir -p bin
	@go build -o bin/photo2enex *.go

lint:
	@go get github.com/golang/lint/golint
	@$(GOPATH)/bin/golint *.go
