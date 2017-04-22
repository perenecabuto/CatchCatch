package io.perenecabuto.catchcatch

import io.socket.client.Socket
import org.json.JSONObject

class GameEventHandler(val sock: Socket, val info: GameInfo, val activity: HomeActivity) : EventHandler {
    override var running = false

    override fun onStart() {
        activity.showInfo("Game:${info.game} Role:${info.role}")
        sock.off()
            .on(GAME_LOOSE) finish@ { args: Array<Any?>? ->
                val gameID = args?.get(0) as String? ?: return@finish
                onGameLoose(gameID)
            }
            .on(GAME_TARGET_NEAR) { args: Array<Any?>? ->
                val dist = args?.get(0).toString().toDouble()
                onGameTargetNear(dist)
            }
            .on(GAME_TARGET_REACHED) { args: Array<Any?>? ->
                val dist = args?.get(0).toString().toDouble()
                onGameTargetReached(dist)
            }
            .on(GAME_TARGET_WIN) { onGameTargetWin() }
            .on(GAME_FINISH) finish@ { args: Array<Any?>? ->
                val json = args?.get(0) as? JSONObject ?: return@finish
                onGameFinish(GameRank(json))
            }
            .on(Socket.EVENT_DISCONNECT) { onDisconnected() }
    }

    override fun onStop() {
        activity.showInfo("Game ${info.game} just finished")
        activity.gameOver()
    }

    fun onGameTargetNear(meters: Double) {
        activity.showCircleAroundPlayer(meters)
    }

    fun onGameTargetReached(meters: Double) {
        activity.showMessage("You win!\nTarget was ${meters.toInt()}m closer")
    }

    fun onGameTargetWin() {
        activity.showMessage("Congratulations!\nYou survived")
    }

    fun onGameLoose(gameID: String) {
        activity.showMessage("You loose =/")
        stop()
    }

    fun onGameFinish(rank: GameRank) {
        activity.showRank(rank)
        stop()
    }

    fun onDisconnected() {
        stop()
    }
}