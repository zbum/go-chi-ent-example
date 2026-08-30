APP := go-chi-ent-example
BIN_DIR := dist
.PHONY: build-linux build-windows build-darwin build-all test clean

build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/$(APP)-linux-amd64 .

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BIN_DIR)/$(APP)-windows-amd64 .

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -o $(BIN_DIR)/$(APP)-darwin-amd64 .

build-all: build-linux build-windows build-darwin

test: 
	go test ./...

clean:
	rm -rf $(BIN_DIR)



