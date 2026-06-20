SQLC := $(shell go env GOPATH)/bin/sqlc

.PHONY: build build-frontend run clean dev sqlc

build-frontend:
	cd frontend && npm run build && cp dist/index.html ../backend/web/spa/index.html

sqlc:
	$(SQLC) generate -f backend/database/sqlc.yaml

build: build-frontend sqlc
	cd backend && CGO_ENABLED=0 go build -o ../videoserver .

run:
	./videoserver

clean:
	rm -f videoserver
	rm -rf data/

dev: build-frontend
	cd backend && go run .
