# TODO

## Deploy

+ continuous deploy
    - steps
    - auto deploy
+ nodes digital ocean


## Server

+ monitor games per machine
+ write tests
    - game worker with game watcher
        - TestGameWorkerNotifiesWhenHunterIsNearToTarget
        - TestGameWorkerNotifiesWhenTargetWinsWhileInArenaAfterTimeout
        - TestGameWorkerNotifiesWhenTargetIsReached
    - tile 38 observers

+ prepare integrated tests
+ separate "game queue" (pre game) and "game arena" (when game start)
+ create gamewith events
+ refactor: use game with events on worker to simplify the logics
+ refactor: rename worker components for better
+ refactor: use errors.Wrap on all error parts
+ worker:
    - heartbeat to update locks valid time
    - retry task
    - reenqueue on error (retry)
    - tests
+ game handler:
    - tests
+ monitor:
    - game start/destroy
    - errors

fix: player isn't being remove on lost connection

## Client

+ Cli lib
+ Cli WASM
+ PWA
+ New Android APP

- player web interface
    - splash
    - games around
    - global score
    - game score
    - game play score
    - radar
- show events over players
- use WASM lib on admin map
- games around
- show game status on map (by color change)