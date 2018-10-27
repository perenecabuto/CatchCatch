# TODO

## Deploy

+ continuous deploy
    - steps
    - auto deploy
+ nodes digital ocean

## Server

+ validate step size on server when player is in game
+ bug disconnect admin

+ monitor games per machine
+ write tests
    - game worker with game watcher
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
+ redis worker:
    - remove ID() from interface, just add task with name as parameter

fix: player isn't being removed by admin
(needs to disconnect it on players connections/but by message broker event)

## Client

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

+ get/request notifications about games around
+ request game ranking
+ request global ranking
+ request features info (game arena, checkpoint)
+ request how many players are around
+ admin client
+ bin to load admin shape file
+ add event to listen for players closer/inside a shape
+ admin event for player connected
+ admin event for player entered into a game