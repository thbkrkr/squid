
all: build up

build:
	doo bgo
	doo b

up:
	doo u

dev:
	go run main.go