.PHONY: build build-frontend run clean dev

build-frontend:
	cd frontend && npm run build && cp dist/index.html ../backend/web/spa/index.html

build: build-frontend
	cd backend && CGO_ENABLED=0 go build -o ../videoserver .

run:
	./videoserver

clean:
	rm -f videoserver
	rm -rf data/

dev: build-frontend
	cd backend && go run .
