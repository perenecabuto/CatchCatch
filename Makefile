INSTANCE_NAME := tile38-local
TEMPLATE := tile38/tile38
PORT := 9851

DOKKU_CMD = ssh dokku@$(DOKKU_HOST)
DOKKU_ROOT_CMD = ssh root@$(DOKKU_HOST) dokku

DOKKU_HOST = 50.116.10.21
DOMAIN = catchcatch.pointto.us

%-digitalocean: DOKKU_HOST=192.34.56.53
%-digitalocean: DOMAIN=catchcatch.ddns.net
%-beta: DOKKU_HOST=159.203.15.29
%-beta: DOMAIN=beta-catchcatch.ddns.net
%-linode: DOKKU_HOST=50.116.10.21
%-linode: DOMAIN=catchcatch.pointto.us

run:
	cd catchcatch-server && CompileDaemon -color -command "./catchcatch-server -zconf"

run-debug:
	cd catchcatch-server && CompileDaemon -color -command "./catchcatch-server -zconf -debug"

run-tile38:
	@if test "`docker ps -a | grep $(INSTANCE_NAME)`"; then \
		docker start $(INSTANCE_NAME); \
	else \
		docker run --rm --name $(INSTANCE_NAME) -v $$PWD:/data -p $(PORT):$(PORT) -P $(TEMPLATE);\
	fi

deploy-beta: deploy
deploy-linode: deploy
deploy-digitalocean: deploy
deploy:
	git push dokku@$(DOKKU_HOST):catchcatch master

setup-ssl-beta: setup-ssl
setup-ssl-linode: setup-ssl
setup-ssl-digitalocean: setup-ssl
setup-ssl:
	$(DOKKU_ROOT_CMD) plugin:install https://github.com/dokku/dokku-letsencrypt.git; echo
	#$(DOKKU_CMD) certs:generate  catchcatch $(DOMAIN); echo
	#$(DOKKU_CMD) proxy:ports-add catchcatch https:443:5000; echo
	$(DOKKU_CMD) domains:add     catchcatch $(DOMAIN)
	$(DOKKU_CMD) config:set --no-restart catchcatch DOKKU_LETSENCRYPT_EMAIL=perenecabuto@gmail.com
	$(DOKKU_CMD) letsencrypt catchcatch
