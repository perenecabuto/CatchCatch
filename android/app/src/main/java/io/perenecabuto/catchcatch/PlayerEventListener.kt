package io.perenecabuto.catchcatch

import android.location.Location
import io.socket.client.Socket
import org.json.JSONException
import org.json.JSONObject
import org.osmdroid.util.GeoPoint
import java.util.*



data class Detection(val featID: String, val lat: Double, val lon: Double, val nearByInMeters: Double) {
    fun point(): GeoPoint {
        return GeoPoint(lat, lon)
    }
}

data class Player(val id: String, var lat: Double, var lon: Double) {
    fun updateLocation(l: Location): Player {
        lat = l.latitude; lon = l.longitude
        return this
    }

    fun point(): GeoPoint {
        return GeoPoint(lat, lon)
    }
}

class PlayerEventListener(override val sock: Socket, override val handler: Handler) : ConnectableListener {
    internal val TAG = javaClass.name

    internal val PLAYER_REGISTERED = "player:registered"
    internal val REMOTE_PLAYER_LIST = "remote-player:list"
    internal val REMOTE_PLAYER_NEW = "remote-player:new"
    internal val REMOTE_PLAYER_UPDATED = "remote-player:updated"
    internal val REMOTE_PLAYER_DESTROY = "remote-player:destroy"
    internal val DETECT_CHECKPOINT = "checkpoint:detected"

    @Throws(NoConnectionException::class)
    override fun bind() {
        sock.on(PLAYER_REGISTERED) { onPlayerRegistered(it) }
            .on(REMOTE_PLAYER_LIST) { onRemotePlayerList(it) }
            .on(REMOTE_PLAYER_NEW) { onRemotePlayerNew(it) }
            .on(REMOTE_PLAYER_UPDATED) { onRemotePlayerUpdate(it) }
            .on(REMOTE_PLAYER_DESTROY) { onRemotePlayerDestroy(it) }
            .on(DETECT_CHECKPOINT) { onDetectCheckpoint(it) }
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