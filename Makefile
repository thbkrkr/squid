
all: build run

build:
	doo bgo
	doo b

run:
	docker pull krkr/squid
	docker rm -f squid || true
	docker run -d \
		--name squid \
		--hostname=squid-$$(hostname) \
		-p 8442:4242 \
		-v $$(pwd)/compose:/app/compose \
		-v /var/run/docker.sock:/var/run/docker.sock \
		--restart=always \
	  krkr/squid

dev:
	go run main.go
