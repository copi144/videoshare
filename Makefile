.PHONY: build run clean dev

build:
	CGO_ENABLED=0 go build -o videoserver ./cmd/server

run:
	./videoserver

clean:
	rm -f videoserver
	rm -rf data/

dev:
	go run ./cmd/server
