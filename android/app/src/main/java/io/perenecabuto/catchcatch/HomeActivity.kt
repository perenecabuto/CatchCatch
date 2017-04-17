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
    private val TAG = HomeActivity::class.java.simpleName
    private val updateGamesInterval: Long = 30_000
    private val dialogsDelay: Long = 10000L

    internal var player = Player("", 0.0, 0.0)
    private var manager: PlayerEventHandler? = null
    private var map: MapView? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        OSMShortcuts.onCreate(this)
        setContentView(R.layout.activity_home)

        map = OSMShortcuts.findMapById(this, R.id.home_activity_map)

        val app = application as CatchCatch
        manager = PlayerEventHandler(app.socket!!, HomeEventHandler(this, map!!))
        manager!!.connect()

        val conf = LocationParams.Builder().setAccuracy(LocationAccuracy.HIGH).build()
        SmartLocation.with(this).location().continuous().config(conf).start(this::onLocationUpdate)

        seekForGamesAround()

        val random = Random()
        RankDialog(this, GameRank("Catch catch", (0..10).map { PlayerRank("Player $it", random.nextInt()) })).show()
        TransparentDialog(this, "welcome!").showWithTimeout(dialogsDelay)
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
        OSMShortcuts.showMarkerOnMap(map!!, id, point)
    }

    fun onRegistered(p: Player) {
        player = p
        TransparentDialog(this, "Registered as ${p.id}").showWithTimeout(dialogsDelay)
    }

    fun onGameStarted(info: GameInfo) {
        manager!!.callback = GameEventHandler(this, map!!)
        TransparentDialog(this, "Game ${info.game} started.\nYour role is: ${info.role}").showWithTimeout(dialogsDelay)
    }

    fun onGameLoose(gameID: String) {
        manager!!.callback = HomeEventHandler(this, map!!)
        TransparentDialog(this, "You loose $gameID").showWithTimeout(dialogsDelay)
    }

    fun onGameTargetReached(meters: Double) {
        TransparentDialog(this, "You win!\nTarget was ${meters.toInt()}m closer").showWithTimeout(dialogsDelay)
    }

    fun onGameFinish(rank: GameRank) {
        Handler().postDelayed({
            manager!!.callback = HomeEventHandler(this, map!!)
            RankDialog(this, rank).showWithTimeout(dialogsDelay)
        }, 5000)
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
            map.overlays.clear()
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

    override fun onGameStarted(info: GameInfo) {
        activity.runOnUiThread {
            activity.onGameStarted(info)
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

    override fun onGameTargetReached(meters: Double) {
        activity.runOnUiThread {
            activity.onGameTargetReached(meters)
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