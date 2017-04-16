package io.perenecabuto.catchcatch

import android.location.Location
import android.os.Bundle
import android.os.Handler
import android.util.Log
import android.widget.Toast
import io.nlopez.smartlocation.SmartLocation
import io.nlopez.smartlocation.location.config.LocationAccuracy
import io.nlopez.smartlocation.location.config.LocationParams
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.MapView


class HomeActivity : ActivityWithLocationPermission() {
    internal val TAG = HomeActivity::class.java.simpleName
    internal val updateGamesInterval: Long = 30_000

    private var manager: PlayerEventHandler? = null
    private var markerOverlay: MarkerOverlay? = null
    private var map: MapView? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        OSMShortcuts.onCreate(this)
        setContentView(R.layout.activity_home)

        map = OSMShortcuts.findMapById(this, R.id.home_activity_map)
        markerOverlay = MarkerOverlay(this)
        map!!.overlays.add(markerOverlay)

        val app = application as CatchCatch
        manager = PlayerEventHandler(app.socket!!, HomeEventHandler(this, map!!))
        manager!!.connect()

        val conf = LocationParams.Builder().setAccuracy(LocationAccuracy.HIGH).build()
        SmartLocation.with(this).location().continuous().config(conf).start(this::onLocationUpdate)

        seekForGamesAround()
    }

    fun seekForGamesAround() {
        if (isFinishing || isDestroyed) return
        Log.d(TAG, "seekForGamesAround")
        manager?.requestAroundGames()
        Handler().postDelayed(this::seekForGamesAround, updateGamesInterval)
    }

    private fun onLocationUpdate(l: Location) {
        val point = GeoPoint(l.latitude, l.longitude)
        updateMarker("me", point)
        manager!!.sendPosition(l)
    }

    override fun onResume() {
        super.onResume()
        OSMShortcuts.onResume(this)
    }

    override fun onDestroy() {
        super.onDestroy()
        map!!.overlays.clear()
        manager!!.disconnect()
    }

    fun updateMarker(id: String, point: GeoPoint) {
        if (markerOverlay == null) return

        val item = OverlayItemWithID(id, point)
        markerOverlay?.removeItem(item)
        markerOverlay?.addItem(item)
        map?.controller?.setCenter(point)
        map?.controller?.setZoom(20)
        map?.invalidate()
    }

    fun onGameStarted(gameID: String) {
        Log.d(TAG, "onGameStarted:" + gameID)
        manager!!.callback = GameEventHandler(this)
    }

    fun onGameLoose(gameID: String) {
        Log.d(TAG, "onGameLoose:" + gameID)
        manager!!.callback = HomeEventHandler(this, map!!)
    }

    fun onGameFinish(rank: GameRank) {
        Log.d(TAG, "onGameFinish:" + rank)
        manager!!.callback = HomeEventHandler(this, map!!)
    }
}

class HomeEventHandler(private val activity: HomeActivity, private val map: MapView) : PlayerEventHandler.EventCallback {
    override fun onConnect() {
        activity.runOnUiThread {
            Toast.makeText(activity, "onConnect", Toast.LENGTH_LONG).show()
        }
    }

    override fun onRegistered(p: Player) {
        activity.runOnUiThread {
            Toast.makeText(activity, "onRegistered" + p, Toast.LENGTH_LONG).show()
        }
    }

    override fun onDisconnected() {
        activity.runOnUiThread {
            Toast.makeText(activity, "onDisconnected",   Toast.LENGTH_LONG).show()
        }
    }

    override fun onGamesAround(games: List<Feature>) {
        activity.runOnUiThread {
            val gameOverlays = map.overlays.filter { it is GeoJsonPolygon }
            map.overlays.removeAll(gameOverlays)
            map.overlays.addAll(games.map { GeoJsonPolygon(it.id, it.geojson) })
            map.invalidate()
        }
    }

    override fun onGameStarted(gameID: String) {
        activity.runOnUiThread {
            activity.onGameStarted(gameID)
        }
    }
}

class GameEventHandler(val activity: HomeActivity) : PlayerEventHandler.EventCallback {
    private val TAG = GameEventHandler::class.java.simpleName

    override fun onGameTargetNear(meters: Int) {
        Log.d(TAG, "onGameTargetNear:" + meters)
    }

    override fun onGameLoose(gameID: String) {
        activity.onGameLoose(gameID)
    }

    override fun onGameFinish(rank: GameRank) {
        activity.onGameFinish(rank)
    }
}