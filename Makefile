.PHONY: build test lint cover clean

build:
	go build -o rhino .

test:
	go test -race ./...

lint:
	go vet ./...

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	rm -f rhino coverage.out
