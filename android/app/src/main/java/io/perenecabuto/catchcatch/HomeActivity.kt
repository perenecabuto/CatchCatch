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
import java.util.*


class HomeActivity : ActivityWithLocationPermission() {
    internal val TAG = HomeActivity::class.java.simpleName
    internal val updateGamesInterval: Long = 30_000

    internal var player = Player("", 0.0, 0.0)
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

        val random = Random()
        RankDialog(this, GameRank("Catch catch", (0..10).map { PlayerRank("Player $it", random.nextInt()) })).show()

        TransparentDialog(this, "welcome!").showWithTimeout(2000)
    }

    fun seekForGamesAround() {
        if (isFinishing || isDestroyed) return
        Log.d(TAG, "seekForGamesAround")
        manager?.requestAroundGames()
        Handler().postDelayed(this::seekForGamesAround, updateGamesInterval)
    }

    private fun onLocationUpdate(l: Location) {
        val point = GeoPoint(l.latitude, l.longitude)
        player.updateLocation(l)
        manager!!.sendPosition(l)
        updateMarker("me", point)
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
        manager!!.callback = GameEventHandler(this, map!!)
        TransparentDialog(this, "Game $gameID started").showWithTimeout(2000)
    }

    fun onGameLoose(gameID: String) {
        manager!!.callback = HomeEventHandler(this, map!!)
        TransparentDialog(this, "You loose $gameID").showWithTimeout(2000)
    }

    fun onGameFinish(rank: GameRank) {
        manager!!.callback = HomeEventHandler(this, map!!)
        RankDialog(this, rank).showWithTimeout(2000)
    }

    fun onRegistered(p: Player) {
        player = p
        TransparentDialog(this, "Registered as ${p.id}").showWithTimeout(2000)
    }

    fun onGameTargetReached(msg: String) {
        TransparentDialog(this, "You win $msg").showWithTimeout(2000)
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
            activity.onRegistered(p)
        }
    }

    override fun onDisconnected() {
        activity.runOnUiThread {
            Toast.makeText(activity, "onDisconnected", Toast.LENGTH_LONG).show()
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

class GameEventHandler(val activity: HomeActivity, val map: MapView) : PlayerEventHandler.EventCallback {
    private val TAG = GameEventHandler::class.java.simpleName

    override fun onGameTargetNear(meters: Double) {
        Log.d(TAG, "onGameTargetNear:" + meters)
        activity.runOnUiThread {
            OSMShortcuts.drawCircleOnMap(map, "target-dist", activity.player.point(), meters, 1000.0)
        }
    }

    override fun onGameTargetReached(msg: String) {
        activity.runOnUiThread {
            activity.onGameTargetReached(msg)
        }
    }

    override fun onGameLoose(gameID: String) {
        activity.runOnUiThread {
            activity.onGameLoose(gameID)
        }
    }

    override fun onGameFinish(rank: GameRank) {
        activity.runOnUiThread {
            activity.onGameFinish(rank)
        }
    }
}