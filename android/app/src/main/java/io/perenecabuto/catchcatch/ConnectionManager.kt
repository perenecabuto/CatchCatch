package io.perenecabuto.catchcatch

import android.location.Location
import android.util.Log
import com.google.android.gms.maps.model.LatLng
import io.socket.client.Socket
import org.json.JSONException
import org.json.JSONObject
import java.net.URISyntaxException
import java.util.*


data class Detection(val checkpoint: String, val lon: Double, val lat: Double, val distance: Double)
data class Player(val id: String, var x: Double, var y: Double) {
    fun updateLocation(l: Location): Player {
        x = l.latitude; y = l.longitude
        return this
    }

    fun point(): LatLng {
        return LatLng(x, y)
    }
}

class ConnectionManager(private val socket: Socket, private val callback: EventCallback) {
    internal val REMOTE_PLAYER_LIST = "remote-player:list"
    internal val PLAYER_REGISTRED = "player:registred"
    internal val REMOTE_PLAYER_NEW = "remote-player:new"
    internal val REMOTE_PLAYER_UPDATED = "remote-player:updated"
    internal val CHECKPOINT_DESTROY = "checkpoint:destroy"
    internal val REMOTE_PLAYER_DESTROY = "remote-player:destroy"
    internal val DETECT_CHECKPOINT = "checkpoint:detected"
    internal val TAG = javaClass.name

    @Throws(URISyntaxException::class, NoConnectionException::class)
    fun connect() {
        socket
            .on(Socket.EVENT_CONNECT) { onConnect(it) }
            .on(REMOTE_PLAYER_LIST) { onRemotePlayerList(it) }
            .on(PLAYER_REGISTRED) { onPlayerRegistred(it) }
            .on(REMOTE_PLAYER_NEW) { onRemotePlayerNew(it) }
            .on(REMOTE_PLAYER_UPDATED) { onRemotePlayerUpdate(it) }
            .on(CHECKPOINT_DESTROY) { onRemotePlayerDestroy(it) }
            .on(REMOTE_PLAYER_DESTROY) { onRemotePlayerDestroy(it) }
            .on(DETECT_CHECKPOINT) { onDetectCheckpoint(it) }
            .on(Socket.EVENT_DISCONNECT) { callback.onDiconnected() }

        socket.connect()
    }

    private fun onDetectCheckpoint(args: Array<Any>) {
        try {
            val detection = getDetectionFronJSON(args[0].toString())
            callback.onDetectCheckpoint(detection)
        } catch (e: JSONException) {
            e.printStackTrace()
        }

    }

    private fun onConnect(args: Array<Any>) {
        callback.onConnect()
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

    private fun onPlayerRegistred(args: Array<Any>) {
        try {
            Log.d(TAG, "---->" + this)
            val player = playerFromJSON(args[0].toString())
            callback.onRegistred(player)
        } catch (e: JSONException) {
            e.printStackTrace()
        }

    }

    fun sendPosition(l: Location) {
        try {
            val coords = JSONObject(mapOf("x" to l.latitude, "y" to l.longitude))
            socket.emit("player:update", coords.toString())
        } catch (e: JSONException) {
            e.printStackTrace()
        }
    }

    fun disconnect() {
        socket.disconnect()
    }

    @Throws(JSONException::class)
    private fun playerFromJSON(json: String): Player {
        val pJson = JSONObject(json)
        return Player(pJson.getString("id"), pJson.getDouble("x"), pJson.getDouble("y"))
    }

    @Throws(JSONException::class)
    private fun getDetectionFronJSON(json: String): Detection {
        val pJson = JSONObject(json)
        return Detection(pJson.getString("checkpoint_id"),
            pJson.getDouble("lon"), pJson.getDouble("lat"), pJson.getDouble("distance"))
    }


    interface EventCallback {
        fun onPlayerList(players: List<Player>)
        fun onRemotePlayerUpdate(player: Player)
        fun onRemoteNewPlayer(player: Player)
        fun onRegistred(p: Player)
        fun onRemotePlayerDestroy(player: Player)
        fun onDiconnected()
        fun onDetectCheckpoint(detection: Detection)
        fun onConnect()
    }

    inner class NoConnectionException(msg: String) : Exception(msg)
}
