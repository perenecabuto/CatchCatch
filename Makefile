INSTANCE_NAME := tile38-local
TEMPLATE := tile38/tile38
PORT := 9851

.PHONY=run
run:
	cd catchcatch-server && CompileDaemon -color -command "./catchcatch-server"

run-debug:
	cd catchcatch-server && CompileDaemon -color -command "./catchcatch-server -zconf -debug"

.PHONY=run-tile38
run-tile38:
	@if test "`docker ps -a | grep $(INSTANCE_NAME)`"; then \
		docker start $(INSTANCE_NAME); \
	else \
		docker run --rm --name $(INSTANCE_NAME) -v $$PWD:/data -p $(PORT):$(PORT) -P $(TEMPLATE);\
	fi

.PHONY=deploy-digitalocean
deploy-digitalocean:
	git push dokku@192.34.56.53:catchcatch master


.PHONY=deploy-linode
deploy-linode:
	git push dokku@50.116.10.21:catchcatch master
