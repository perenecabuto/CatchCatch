#!/bin/bash

export MANAGER=manager
export WORKERS="worker1 worker2"
export MACHINES="$MANAGER $WORKERS"

for machine in $MACHINES; do
    docker-machine rm $machine
    docker-machine create -d virtualbox $machine
    #docker-machine create \
	#	--driver digitalocean \
	#	--digitalocean-access-token $(TOKEN) \
    #    $machine

    docker-machine start $machine
done

eval "$(docker-machine env $MANAGER)"

docker-machine ssh $MANAGER "docker swarm init \
    --listen-addr $(docker-machine ip $MANAGER) \
    --advertise-addr $(docker-machine ip $MANAGER)"

export worker_token=$(docker-machine ssh $MANAGER "docker swarm \
join-token worker -q")

for worker in $WORKERS; do
    docker-machine ssh $worker "docker swarm leave"
    docker-machine ssh $worker "docker swarm join \
    --token=${worker_token} \
    --listen-addr $(docker-machine ip $worker) \
    --advertise-addr $(docker-machine ip $worker) \
    $(docker-machine ip $MANAGER)"
done

docker stack deploy -c docker-compose.yml catchcatch
