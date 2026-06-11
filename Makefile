.PHONY: build run

build:
	CGO_ENABLED=0 go build -o mariadb-lastlogin cmd/mariadb-lastlogin/main.go

run: build
	./lastlogin
