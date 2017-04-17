package io.perenecabuto.catchcatch

import android.os.Handler
import android.util.Log
import io.socket.client.Socket
import org.json.JSONArray
import org.json.JSONObject


data class Feature(val id: String, val geojson: String)
data class GameRank(val game: String?, val pointsPerPlayer: List<PlayerRank>)
data class PlayerRank(val player: String, val points: Int)
data class GameInfo(val game: String, val role: String)

class GameEventListener(override val sock: Socket, override val handler: Handler) : ConnectableListener {
    private val TAG = javaClass.simpleName

    internal val AROUND = "game:around"
    internal val STARTED = "game:started"
    internal val LOOSE = "game:loose"
    internal val TARGET_NEAR = "game:target:near"
    internal val TARGET_REACHED = "game:target:reached"
    internal val FINISH = "game:finish"

    private val interval: Long = 30_000

    override fun bind() {
        sock.on(AROUND) { onGamesAround(it) }
            .on(STARTED) { onGameStarted(it) }
            .on(LOOSE) { onGameLoose(it) }
            .on(TARGET_NEAR) { onGameTargetNear(it) }
            .on(TARGET_REACHED) { onGameTargetReached(it) }
            .on(FINISH) { onGameFinish(it) }
            .on(Socket.EVENT_CONNECT) { handler.onDisconnected() }

    }

    private var stopSeek: Boolean = false

    internal fun startRadar() {
        if (stopSeek) {
            stopSeek = false
            return
        }
        Log.d(TAG, "startRadar")
        sock.emit("player:request-games")
        Handler().postDelayed(this::startRadar, interval)
    }

    fun stopRadar() {
        stopSeek = true
    }

    private fun onGamesAround(args: Array<Any>?) {
        val items = args?.get(0) as? JSONArray ?: return
        val games = (0..items.length() - 1).map {
            val item = items.getJSONObject(it)
            Feature(item.getString("id"), item.getString("coords"))
        }
        handler.onGamesAround(games)
    }

    private fun onGameStarted(args: Array<Any>?) {
        val json = args?.get(0) as? JSONObject ?: return
        handler.onGameStarted(GameInfo(json.getString("game"), json.getString("role")))
    }

    private fun onGameLoose(args: Array<Any>?) {
        handler.onGameLoose(args?.get(0).toString())
    }

    private fun onGameTargetNear(args: Array<Any>?) {
        handler.onGameTargetNear(args?.get(0).toString().toDouble())
    }

    private fun onGameTargetReached(args: Array<Any>?) {
        handler.onGameTargetReached(args?.get(0).toString().toDouble())
    }

    private fun onGameFinish(args: Array<Any>?) {
        val json = args?.get(0) as? JSONObject ?: return

        val game = json.getString("game")
        val points = json.getJSONArray("points_per_player")
        val pointsPerPlayer = (0..points.length() - 1)
            .map { points.getJSONObject(it) }
            .map { PlayerRank(it.getString("player"), it.getInt("points")) }

        val rank = GameRank(game, pointsPerPlayer)
        handler.onGameFinish(rank)
    }

    interface Handler : ConnectableHandler {
        fun onGamesAround(games: List<Feature>)
        fun onGameStarted(info: GameInfo)
        fun onGameTargetNear(meters: Double)
        fun onGameTargetReached(meters: Double)
        fun onGameLoose(gameID: String)
        fun onGameFinish(rank: GameRank) {}
    }
}