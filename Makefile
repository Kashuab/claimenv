BINARY := claimenv
MODULE := github.com/Kashuab/claimenv

.PHONY: build test lint clean

build:
	go build -o $(BINARY) .

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)
