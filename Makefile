TILE38PORT := 9851

SERVER_SRC := cd server;

DOKKU_CMD = ssh dokku@$(DOKKU_HOST)
DOKKU_ROOT_CMD = ssh root@$(DOKKU_HOST) dokku

DOKKU_HOST = 50.116.10.21
LOCAL_BRANCH = master

%-digitalocean: DOKKU_HOST=192.34.56.53
%-digitalocean: DOMAIN=catchcatch.ddns.net
%-beta: DOKKU_HOST=159.203.15.29
%-beta: DOMAIN=beta-catchcatch.ddns.net

test:
	$(SERVER_SRC) go test -cover -v ./...

test-forever:
	$(SERVER_SRC) CompileDaemon -color -command "go test -v ./..."

coverage:
	-go get github.com/lawrencewoodman/roveralls
	$(SERVER_SRC) roveralls
	$(SERVER_SRC) go tool cover -html=roveralls.coverprofile

gen-mocks:
	-go get github.com/vektra/mockery/...
	$(SERVER_SRC) mockery -all

build:
	# Ref: https://blog.filippo.io/shrink-your-go-binaries-with-this-one-weird-trick/
	$(SERVER_SRC) CGO_ENABLED=0 go build -ldflags="-s -w" -tags netgo -a

docker-compose: build
	docker-compose up --build

run: run-tile38
	$(SERVER_SRC) CompileDaemon -color -command "./catchcatch-server -zconf"

run-debug:
	$(SERVER_SRC) CompileDaemon -color -command "./catchcatch-server -zconf -debug"

run-influxdb:
	@-docker rm -f influxdb-local
	@docker run --restart unless-stopped -p 8086:8086 \
		--name influxdb-local \
		-v $(PWD)/:/etc/influxdb/:ro \
		-e INFLUXDB_ADMIN_ENABLED=true \
		-d influxdb -config /etc/influxdb/influxdb.conf

run-grafana:
	@-#docker rm -f grafana-local
	@-docker run -d --name=grafana-local -p 3000:3000 -P grafana/grafana
	@docker start grfana-local

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
