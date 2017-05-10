package io.perenecabuto.catchcatch

import android.os.Handler
import android.os.Looper
import org.json.JSONArray
import org.json.JSONObject

class RadarEventHandler(val sock: WebSocketClient, val activity: HomeActivity) : EventHandler {
    private val looper = Looper.myLooper()
    private val interval: Long = 20_000
    override var running = false

    override fun onStart() {
        sock.off()
            .on(PLAYER_REGISTERED) finish@ { msg: String ->
                val json = JSONObject(msg)
                onRegistered(Player(json))
            }
            .on(PLAYER_UPDATED) {
                onUpdated()
            }
            .on(GAME_AROUND) finish@ { msg: String ->
                val items = JSONArray(msg)
                onGamesAround(FeatureList(items))
            }
            .on(GAME_STARTED) finish@ { msg: String ->
                val json = JSONObject(msg)
                onGameStarted(GameInfo(json))
            }
            .onDisconnect { onDisconnect() }
    }

    override fun onStop() {
        radarStarted = false
    }

    private var radarStarted: Boolean = false
    private fun onUpdated() {
        if (!radarStarted) {
            radarStarted = true
            radar()
        }
    }

    private fun radar() {
        if (!running) return
        activity.showInfo("searching for games around...")
        activity.showRadar()
        sock.emit("player:request-games")
        Handler(looper).postDelayed(this::radar, interval)
    }

    private fun onGamesAround(games: FeatureList) {
        activity.showInfo("found ${games.list.size} games near you")
        activity.showFeatures(games.list)
    }

    private fun onGameStarted(info: GameInfo) {
        activity.showMessage("Game ${info.game} started.\nYour role is: ${info.role}")
        activity.startGame(info)
    }

    private fun onRegistered(p: Player) {
        activity.player = p
        activity.showInfo("Connected as ${p.id}")
    }

    private fun onDisconnect() {
        activity.showMessage("Disconnected")
    }
}