package io.perenecabuto.catchcatch

import android.location.Location
import io.socket.client.Socket
import org.json.JSONArray
import org.json.JSONException
import org.json.JSONObject
import org.osmdroid.util.GeoPoint
import java.net.URISyntaxException
import java.util.*


data class Feature(val id: String, val geojson: String)

data class Detection(val featID: String, val lat: Double, val lon: Double, val nearByInMeters: Double) {
    fun point(): GeoPoint {
        return GeoPoint(lat, lon)
    }
}

data class GameInfo(val game: String, val role: String)

data class Player(val id: String, var lat: Double, var lon: Double) {
    fun updateLocation(l: Location): Player {
        lat = l.latitude; lon = l.longitude
        return this
    }

    fun point(): GeoPoint {
        return GeoPoint(lat, lon)
    }
}


data class GameRank(val game: String?, val pointsPerPlayer: List<PlayerRank>)
data class PlayerRank(val player: String, val points: Int)


class PlayerEventHandler(private val sock: Socket, internal var callback: EventCallback) {
    internal val TAG = javaClass.name

    internal val PLAYER_REGISTERED = "player:registered"
    internal val REMOTE_PLAYER_LIST = "remote-player:list"
    internal val REMOTE_PLAYER_NEW = "remote-player:new"
    internal val REMOTE_PLAYER_UPDATED = "remote-player:updated"
    internal val REMOTE_PLAYER_DESTROY = "remote-player:destroy"
    internal val DETECT_CHECKPOINT = "checkpoint:detected"

    internal val GAME_AROUND = "game:around"
    internal val GAME_STARTED = "game:started"
    internal val GAME_LOOSE = "game:loose"
    internal val GAME_TARGET_NEAR = "game:target:near"
    internal val GAME_TARGET_REACHED = "game:target:reached"
    internal val GAME_FINISH = "game:finish"

    @Throws(URISyntaxException::class, NoConnectionException::class)
    fun connect() {
        disconnect()
        sock.on(Socket.EVENT_CONNECT) { onConnect() }
            .on(PLAYER_REGISTERED) { onPlayerRegistered(it) }

            .on(GAME_AROUND) { onGamesAround(it) }
            .on(GAME_STARTED) { onGameStarted(it) }
            .on(GAME_LOOSE) { onGameLoose(it) }
            .on(GAME_TARGET_NEAR) { onGameTargetNear(it) }
            .on(GAME_TARGET_REACHED) { onGameTargetReached(it) }
            .on(GAME_FINISH) { onGameFinish(it) }

            .on(REMOTE_PLAYER_LIST) { onRemotePlayerList(it) }
            .on(REMOTE_PLAYER_NEW) { onRemotePlayerNew(it) }
            .on(REMOTE_PLAYER_UPDATED) { onRemotePlayerUpdate(it) }
            .on(REMOTE_PLAYER_DESTROY) { onRemotePlayerDestroy(it) }
            .on(DETECT_CHECKPOINT) { onDetectCheckpoint(it) }
            .on(Socket.EVENT_DISCONNECT) { onDisconnect() }

        sock.connect()
    }

    private fun onDisconnect() {
        callback.onDisconnected()
    }

    private fun onConnect() {
        callback.onConnect()
    }

    private fun onPlayerRegistered(args: Array<Any>) {
        try {
            val player = playerFromJSON(args[0].toString())
            callback.onRegistered(player)
        } catch (e: JSONException) {
            e.printStackTrace()
        }
    }

    private fun onGamesAround(args: Array<Any>?) {
        val items = args?.get(0) as? JSONArray ?: return
        val games = (0..items.length() - 1).map {
            val item = items.getJSONObject(it)
            Feature(item.getString("id"), item.getString("coords"))
        }
        callback.onGamesAround(games)
    }

    private fun onGameStarted(args: Array<Any>?) {
        val json = args?.get(0) as? JSONObject ?: return
        callback.onGameStarted(GameInfo(json.getString("game"), json.getString("role")))
    }

    private fun onGameLoose(args: Array<Any>?) {
        callback.onGameLoose(args?.get(0).toString())
    }

    private fun onGameTargetNear(args: Array<Any>?) {
        callback.onGameTargetNear(args?.get(0).toString().toDouble())
    }

    private fun onGameTargetReached(args: Array<Any>?) {
        callback.onGameTargetReached(args?.get(0).toString())
    }

    private fun onGameFinish(args: Array<Any>?) {
        val json = args?.get(0) as? JSONObject ?: return

        val game = json.getString("game")
        val points = json.getJSONArray("points_per_player")
        val pointsPerPlayer = (0..points.length() - 1)
            .map { points.getJSONObject(it) }
            .map { PlayerRank(it.getString("player"), it.getInt("points")) }

        val rank = GameRank(game, pointsPerPlayer)
        callback.onGameFinish(rank)
    }

    private fun onDetectCheckpoint(args: Array<Any>) {
        try {
            val detection = getDetectionFromJSON(args[0].toString())
            callback.onDetectCheckpoint(detection)
        } catch (e: JSONException) {
            e.printStackTrace()
        }
    }

    private fun onRemotePlayerDestroy(args: Array<Any>) {
        try {
            val player = playerFromJSON(args[0].toString())
            callback.onRemotePlayerDestroy(player)
        } catch (e: JSONException) {
            e.printStackTrace()
        }
    }

    private fun onRemotePlayerNew(args: Array<Any>) {
        try {
            val player = playerFromJSON(args[0].toString())
            callback.onRemoteNewPlayer(player)
        } catch (e: JSONException) {
            e.printStackTrace()
        }
    }

    private fun onRemotePlayerUpdate(args: Array<Any>) {
        try {
            val player = playerFromJSON(args[0].toString())
            callback.onRemotePlayerUpdate(player)
        } catch (e: JSONException) {
            e.printStackTrace()
        }
    }

    private fun onRemotePlayerList(args: Array<Any>) {
        val players = ArrayList<Player>()
        try {
            val arg = JSONObject(args[0].toString())
            val pList = arg.getJSONArray("players")
            (0..pList.length() - 1).mapTo(players) { playerFromJSON(pList.getString(it)) }
        } catch (e: JSONException) {
            e.printStackTrace()
        }

        callback.onPlayerList(players)
    }

    fun sendPosition(l: Location) {
        try {
            val coords = JSONObject(mapOf("lat" to l.latitude, "lon" to l.longitude))
            sock.emit("player:update", coords.toString())
        } catch (e: JSONException) {
            e.printStackTrace()
        }
    }

    fun requestAroundGames() {
        sock.emit("player:request-games")
    }

    fun disconnect() {
        sock.off()
        sock.disconnect()
    }

    @Throws(JSONException::class)
    private fun playerFromJSON(json: String): Player {
        val pJson = JSONObject(json)
        return Player(pJson.getString("id"), pJson.getDouble("lat"), pJson.getDouble("lon"))
    }

    @Throws(JSONException::class)
    private fun getDetectionFromJSON(json: String): Detection {
        val pJson = JSONObject(json)
        return Detection(pJson.getString("feat_id"),
            pJson.getDouble("lat"), pJson.getDouble("lon"),
            pJson.getDouble("near_by_meters"))
    }

    interface EventCallback {
        fun onPlayerList(players: List<Player>) {}
        fun onRemotePlayerUpdate(player: Player) {}
        fun onRemoteNewPlayer(player: Player) {}
        fun onRegistered(p: Player) {}
        fun onRemotePlayerDestroy(player: Player) {}
        fun onDisconnected() {}
        fun onDetectCheckpoint(detection: Detection) {}
        fun onConnect() {}
        fun onGamesAround(games: List<Feature>) {}
        fun onGameStarted(info: GameInfo) {}
        fun onGameLoose(gameID: String) {}
        fun onGameTargetNear(meters: Double) {}
        fun onGameTargetReached(msg: String) {}
        fun onGameFinish(rank: GameRank) {}
    }

    inner class NoConnectionException(msg: String) : Exception(msg)
}