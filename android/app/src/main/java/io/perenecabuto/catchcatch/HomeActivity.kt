package io.perenecabuto.catchcatch

import android.location.Location
import android.os.Bundle
import io.nlopez.smartlocation.SmartLocation
import io.nlopez.smartlocation.location.config.LocationAccuracy
import io.nlopez.smartlocation.location.config.LocationParams
import org.json.JSONObject
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.MapView
import java.util.*

private val dialogsDelay: Long = 5000L


class HomeActivity : ActivityWithLocationPermission() {

    internal var player = Player("", 0.0, 0.0)
    private var animator: PolygonAnimator? = null
    private var map: MapView? = null

    private var radar: RadarEventHandler? = null
    private var game: GameEventHandler? = null


    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        OSMShortcuts.onCreate(this)
        setContentView(R.layout.activity_home)

        map = OSMShortcuts.findMapById(this, R.id.home_activity_map)

        val conf = LocationParams.Builder().setAccuracy(LocationAccuracy.HIGH).build()
        SmartLocation.with(this).location().continuous().config(conf).start(this::onLocationUpdate)

        val app = application as CatchCatch
        radar = RadarEventHandler(app.socket!!, this)
        radar!!.start()
    }

    override fun onPostCreate(savedInstanceState: Bundle?) {
        super.onPostCreate(savedInstanceState)

        showMessage("welcome!")
        showRank(GameRank("CatchCatch", (0..10).map { PlayerRank("Player $it", Random().nextInt()) }))
    }

    private fun onLocationUpdate(l: Location) {
        val point = GeoPoint(l.latitude, l.longitude)
        OSMShortcuts.focus(map!!, point)
        player.updateLocation(l)
        sendPosition(l)

        if (animator == null || animator!!.running.not()) {
            OSMShortcuts.showMarkerOnMap(map!!, "me", point)
        }
    }

    override fun onResume() {
        super.onResume()
        OSMShortcuts.onResume(this)
    }

    override fun onDestroy() {
        super.onDestroy()
        map!!.overlays.clear()
    }

    fun showMessage(msg: String) = runOnUiThread {
        TransparentDialog(this, msg).showWithTimeout(dialogsDelay)
    }

    fun showRank(rank: GameRank) = runOnUiThread {
        RankDialog(this, rank).showWithTimeout(dialogsDelay)
    }

    fun showFeatures(games: List<Feature>) = runOnUiThread finish@ {
        val map = map ?: return@finish
        OSMShortcuts.refreshGeojsonFeaturesOnMap(map, games.map { GeoJsonPolygon(it.id, it.geojson) })
    }

    fun startGame(info: GameInfo) = runOnUiThread finish@ {
        val map = map ?: return@finish
        animator = OSMShortcuts.animatePolygonOverlay(map, info.game)
        animator?.overlay?.let { OSMShortcuts.focus(map, it.boundingBox.center) }

        val app = application as CatchCatch
        game = GameEventHandler(app.socket!!, info, this)
        radar!!.switchTo(game!!)
    }

    fun showCircleAroundPlayer(meters: Double) = runOnUiThread finish@ {
        val map = map ?: return@finish
        OSMShortcuts.drawCircleOnMap(map, "player-circle", player.point(), meters, 1000.0)
    }

    fun gameOver() = runOnUiThread {
        animator?.stop()
        game!!.switchTo(radar!!)
    }

    fun sendPosition(l: Location) {
        val sock = (application as CatchCatch).socket!!
        val coords = JSONObject(mapOf("lat" to l.latitude, "lon" to l.longitude))
        sock.emit("player:update", coords.toString())
    }
}
