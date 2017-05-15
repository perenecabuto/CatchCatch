package io.perenecabuto.catchcatch.events

import io.perenecabuto.catchcatch.drivers.WebSocketClient

internal val GAME_AROUND = "game:around"
internal val GAME_STARTED = "game:started"
internal val GAME_LOOSE = "game:loose"
internal val GAME_TARGET_NEAR = "game:target:near"
internal val GAME_TARGET_REACHED = "game:target:reached"
internal val GAME_TARGET_WIN = "game:target:win"
internal val GAME_FINISH = "game:finish"

internal val PLAYER_REGISTERED = "player:registered"
internal val PLAYER_UPDATED = "player:updated"
internal val REMOTE_PLAYER_LIST = "remote-player:list"
internal val REMOTE_PLAYER_NEW = "remote-player:new"
internal val REMOTE_PLAYER_UPDATED = "remote-player:updated"
internal val REMOTE_PLAYER_DESTROY = "remote-player:destroy"
internal val CHECKPOINT_DETECTED = "checkpoint:detected"

interface EventHandler {
    val sock: WebSocketClient
    var running: Boolean
    fun onStart()
    fun onStop() {}

    fun start() {
        if (!running) {
            running = true
            sock.off()
            onStart()
        }
    }

    fun stop() {
        if (running) {
            running = false
            onStop()
        }
    }

    fun switchTo(handler: EventHandler) {
        stop()
        handler.start()
    }
}