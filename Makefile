# gosearx build automation.

.PHONY: all build web run test fmt vet clean

# Build the frontend then the single self-contained binary.
all: web build

web:
	cd web && npm install && npm run build

build:
	go build -o gosearx ./cmd/gosearx

run: build
	./gosearx serve

test:
	go test -race ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -f gosearx
	rm -rf web/dist/assets web/dist/index.html
