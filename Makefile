.PHONY: build build-frontend run clean dev

build-frontend:
	cd frontend && npm run build && cp dist/index.html ../internal/web/spa/index.html

build: build-frontend
	CGO_ENABLED=0 go build -o videoserver ./cmd/server

run:
	./videoserver

clean:
	rm -f videoserver
	rm -rf data/

dev: build-frontend
	go run ./cmd/server
