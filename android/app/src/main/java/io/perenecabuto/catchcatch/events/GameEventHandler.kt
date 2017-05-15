package io.perenecabuto.catchcatch.events

import io.perenecabuto.catchcatch.drivers.WebSocketClient
import io.perenecabuto.catchcatch.model.GameInfo
import io.perenecabuto.catchcatch.model.GameRank
import io.perenecabuto.catchcatch.view.HomeActivity
import org.json.JSONObject

class GameEventHandler(override val sock: WebSocketClient, val info: GameInfo, val activity: HomeActivity) : EventHandler {
    override var running = false

    override fun onStart() {
        activity.showInfo("Game ${info.game} started you are ${info.role}")
        sock.on(GAME_LOOSE) finish@ { gameID:String ->
                onGameLoose(gameID)
            }
            .on(GAME_TARGET_NEAR) finish@ { msg:String ->
                val dist = msg.toDouble()
                onGameTargetNear(dist)
            }
            .on(GAME_TARGET_REACHED) { msg:String ->
                val dist = msg.toDouble()
                onGameTargetReached(dist)
            }
            .on(GAME_TARGET_WIN) { onGameTargetWin() }
            .on(GAME_FINISH) finish@ { msg:String ->
                val json = JSONObject(msg)
                onGameFinish(GameRank(json))
            }
            .onDisconnect { onDisconnected() }
    }

    override fun onStop() {
        activity.showInfo("Game ${info.game} just finished")
        activity.gameOver()
    }

    fun onGameTargetNear(meters: Double) {
        activity.showCircleAroundPlayer(meters)
    }

    fun onGameTargetReached(meters: Double) {
        activity.showMessage("Congratulations!\nYou win!\nTarget was ${meters.toInt()}m closer")
    }

    fun onGameTargetWin() {
        activity.showMessage("Congratulations!\nYou survived")
    }

    fun onGameLoose(gameID: String) {
        activity.showMessage("Holy shit!\nYou loose")
        stop()
    }

    fun onGameFinish(rank: GameRank) {
        activity.showMessage("This game is over")
        activity.showRank(rank)
        stop()
    }

    fun onDisconnected() {
        stop()
    }
}