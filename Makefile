TILE38PORT := 9851

SERVER_SRC := cd server;
BINARY := server

DOKKU_CMD = ssh dokku@$(DOKKU_HOST)
DOKKU_ROOT_CMD = ssh root@$(DOKKU_HOST) dokku

DOKKU_HOST = 50.116.10.21
LOCAL_BRANCH = master

DOCKER_MACHINE_BINARY := docker-machine-Linux-x86_64
DOCKER_MACHINE_URL := https://github.com/docker/machine/releases/download/v0.13.0/$(DOCKER_MACHINE_BINARY)


%-digitalocean: DOKKU_HOST=192.34.56.53
%-digitalocean: DOMAIN=catchcatch.ddns.net
%-beta: DOKKU_HOST=159.203.15.29
%-beta: DOMAIN=beta-catchcatch.ddns.net


test:
	go get -u github.com/rakyll/gotest
	$(SERVER_SRC) gotest -count=1 -timeout=30s -cover -race -v ./...

test-with-compose:
	docker-compose -f docker-compose.test.yml run --rm test

clean-redis:
	@-echo FLUSHALL | nc -w1 localhost 6379

test-forever:
	$(SERVER_SRC) CompileDaemon -color -command "go test -v ./..."

coverage:
	-go get github.com/lawrencewoodman/roveralls
	$(SERVER_SRC) roveralls
	$(SERVER_SRC) go tool cover -html=roveralls.coverprofile

gen-proto:
	#https://github.com/protocolbuffers/protobuf/releases/download/v3.6.1/protoc-3.6.1-linux-x86_64.zip
	-go get -v -u github.com/gogo/protobuf/protoc-gen-gogoslick
	$(SERVER_SRC) protoc --gogoslick_out=. protobuf/message.proto

gen-mocks:
	-go get github.com/vektra/mockery/...
	for x in `find $(PWD)/server/* -type d -prune | grep -v vendor`; do \
		cd $$x; \
		mockery -all; \
	done

build:
	# Ref: https://blog.filippo.io/shrink-your-go-binaries-with-this-one-weird-trick/
	$(SERVER_SRC) go build

build-wasm:
	GOOS=js GOARCH=wasm \
		go build -v -o web/player/catchcatch.wasm client/wasm/*.go

docker-compose:
	docker-compose up --build

run: run-tile38 run-nats run-redis run-influxdb
	$(SERVER_SRC) CompileDaemon -color -command "./$(BINARY) -zconf"

run-debug:
	$(SERVER_SRC) CompileDaemon -color -command "./$(BINARY) -zconf -debug"

run-redis:
	@-docker rm -f redis-local
	@docker run --restart unless-stopped -p 6379:6379 \
		--name redis-local \
		-d redis:alpine

run-nats:
	@-docker rm -f nats-local
	@docker run --restart unless-stopped -p 4222:4222 \
		--name nats-local \
		-d nats

run-influxdb:
	@-docker rm -f influxdb-local
	@docker run --restart unless-stopped -p 8086:8086 \
		--name influxdb-local \
		-v $(PWD)/:/etc/influxdb/:ro \
		-e INFLUXDB_ADMIN_ENABLED=true \
		-d influxdb:1.6-alpine -config /etc/influxdb/influxdb.conf

run-grafana:
	@-docker rm -f grafana-local
	@docker run --restart unless-stopped -p 3000:3000 -P \
		--name grafana-local \
		-d grafana/grafana

run-tile38:
	@-docker rm -f tile38-local
	@-docker run -d --rm --name tile38-local -v $(PWD):/data -p $(TILE38PORT):$(TILE38PORT) -P tile38/tile38

deploy-beta: deploy
deploy-digitalocean: deploy
deploy:
	git push -f dokku@$(DOKKU_HOST):catchcatch $(LOCAL_BRANCH):master

setup-ssl-beta: setup-ssl
setup-ssl-digitalocean: setup-ssl
setup-ssl:
	$(DOKKU_ROOT_CMD) plugin:install https://github.com/dokku/dokku-letsencrypt.git; echo
	#$(DOKKU_CMD) certs:generate  catchcatch $(DOMAIN); echo
	#$(DOKKU_CMD) proxy:ports-add catchcatch https:443:5000; echo
	$(DOKKU_CMD) domains:add     catchcatch $(DOMAIN)
	$(DOKKU_CMD) config:set --no-restart catchcatch DOKKU_LETSENCRYPT_EMAIL=perenecabuto@gmail.com
	$(DOKKU_CMD) letsencrypt catchcatch

update-deps:
	-go get github.com/golang/dep
	$(SERVER_SRC) dep ensure -v -update

install-docker-machine:
	wget -c $(DOCKER_MACHINE_URL)
	chmod +x $(DOCKER_MACHINE_BINARY)
	mv $(DOCKER_MACHINE_BINARY) $(HOME)/bin/docker-machine
