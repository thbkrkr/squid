
up:
	doo dc squid.yml up -d

pull:
	doo dc squid.yml pull

build:
	doo bgo
	doo b

push: build
	doo p

run:
	docker pull krkr/squid
	docker rm -f squid || true
	docker run -d \
		--name squid \
		--hostname=squid-$$(hostname) \
		-p 4242:4242 \
		-v $$(pwd)/compose:/app/compose \
		-v /var/run/docker.sock:/var/run/docker.sock \
		--restart=always \
	  krkr/squid

dev:
	go run main.go -c http://localhost:4242

test:
	docker run -d \
		-v $$(pwd)/compose:/app/compose \
		-v /var/run/docker.sock:/var/run/docker.sock \
		--restart=always \
	  krkr/squid -c http://172.17.0.1:4242


sync:
	docker-machine scp squid.yml n1:compose/