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

interface ConnectableHandler {
    fun onConnect() {}
    fun onDisconnected() {}
}

class GameXEventListener(private val sock: Socket, internal val handler: Handler) {
    internal val AROUND = "game:around"
    internal val STARTED = "game:started"
    internal val LOOSE = "game:loose"
    internal val TARGET_NEAR = "game:target:near"
    internal val TARGET_REACHED = "game:target:reached"
    internal val FINISH = "game:finish"

    fun connect() {
        sock.off()
        sock.on(Socket.EVENT_CONNECT) { handler.onConnect() }
            .on(AROUND) { onGamesAround(it) }
            .on(STARTED) { onGameStarted(it) }
            .on(LOOSE) { onGameLoose(it) }
            .on(TARGET_NEAR) { onGameTargetNear(it) }
            .on(TARGET_REACHED) { onGameTargetReached(it) }
            .on(FINISH) { onGameFinish(it) }
            .on(Socket.EVENT_CONNECT) { handler.onDisconnected() }
        sock.connect()
    }

    fun requestAroundGames() {
        sock.emit("player:request-games")
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


class GameRadarListener(private val sock: Socket, private val handler: Handler) {
    internal val TAG = javaClass.name

    internal val PLAYER_REGISTERED = "player:registered"
    internal val REMOTE_PLAYER_LIST = "remote-player:list"
    internal val REMOTE_PLAYER_NEW = "remote-player:new"
    internal val REMOTE_PLAYER_UPDATED = "remote-player:updated"
    internal val REMOTE_PLAYER_DESTROY = "remote-player:destroy"
    internal val DETECT_CHECKPOINT = "checkpoint:detected"


    @Throws(URISyntaxException::class, NoConnectionException::class)
    fun connect() {
        sock.off()
        sock.on(Socket.EVENT_CONNECT) { handler.onConnect() }
            .on(PLAYER_REGISTERED) { onPlayerRegistered(it) }
            .on(REMOTE_PLAYER_LIST) { onRemotePlayerList(it) }
            .on(REMOTE_PLAYER_NEW) { onRemotePlayerNew(it) }
            .on(REMOTE_PLAYER_UPDATED) { onRemotePlayerUpdate(it) }
            .on(REMOTE_PLAYER_DESTROY) { onRemotePlayerDestroy(it) }
            .on(DETECT_CHECKPOINT) { onDetectCheckpoint(it) }
            .on(Socket.EVENT_DISCONNECT) { handler.onDisconnected() }
        sock.connect()
    }

    private fun onPlayerRegistered(args: Array<Any>) {
        try {
            val player = playerFromJSON(args[0].toString())
            handler.onRegistered(player)
        } catch (e: JSONException) {
            e.printStackTrace()
        }
    }

    private fun onDetectCheckpoint(args: Array<Any>) {
        try {
            val detection = getDetectionFromJSON(args[0].toString())
            handler.onDetectCheckpoint(detection)
        } catch (e: JSONException) {
            e.printStackTrace()
        }
    }

    private fun onRemotePlayerDestroy(args: Array<Any>) {
        try {
            val player = playerFromJSON(args[0].toString())
            handler.onRemotePlayerDestroy(player)
        } catch (e: JSONException) {
            e.printStackTrace()
        }
    }

    private fun onRemotePlayerNew(args: Array<Any>) {
        try {
            val player = playerFromJSON(args[0].toString())
            handler.onRemoteNewPlayer(player)
        } catch (e: JSONException) {
            e.printStackTrace()
        }
    }

    private fun onRemotePlayerUpdate(args: Array<Any>) {
        try {
            val player = playerFromJSON(args[0].toString())
            handler.onRemotePlayerUpdate(player)
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

        handler.onPlayerList(players)
    }

    fun sendPosition(l: Location) {
        val coords = JSONObject(mapOf("lat" to l.latitude, "lon" to l.longitude))
        sock.emit("player:update", coords.toString())
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

    interface Handler : ConnectableHandler {
        fun onPlayerList(players: List<Player>) {}
        fun onRemotePlayerUpdate(player: Player) {}
        fun onRemoteNewPlayer(player: Player) {}
        fun onRegistered(p: Player) {}
        fun onRemotePlayerDestroy(player: Player) {}
        fun onDetectCheckpoint(detection: Detection) {}
    }

    inner class NoConnectionException(msg: String) : Exception(msg)
}