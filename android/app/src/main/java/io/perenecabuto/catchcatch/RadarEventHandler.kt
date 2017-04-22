package io.perenecabuto.catchcatch

import android.os.Handler
import android.os.Looper
import io.socket.client.Socket
import org.json.JSONArray
import org.json.JSONObject

class RadarEventHandler(val sock: Socket, val activity: HomeActivity) : EventHandler {
    private val looper = Looper.myLooper()
    private val interval: Long = 20_000
    override var running = false

    override fun onStart() {
        sock.off()
            .on(PLAYER_REGISTERED) finish@ { args: Array<Any?>? ->
                val json = args?.get(0) as JSONObject? ?: return@finish
                onRegistered(Player(json))
            }
            .on(PLAYER_UPDATED) {
                onUpdated()
            }
            .on(GAME_AROUND) finish@ { args: Array<Any?>? ->
                val items = args?.get(0) as? JSONArray ?: return@finish
                onGamesAround(FeatureList(items))
            }
            .on(GAME_STARTED) finish@ { args: Array<Any?>? ->
                val json = args?.get(0) as JSONObject? ?: return@finish
                onGameStarted(GameInfo(json))
            }
            .on(Socket.EVENT_DISCONNECT) { onDisconnect() }
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