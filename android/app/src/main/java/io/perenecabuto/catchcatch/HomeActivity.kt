package io.perenecabuto.catchcatch

import android.location.Location
import android.os.Bundle
import android.os.Handler
import io.nlopez.smartlocation.SmartLocation
import io.nlopez.smartlocation.location.config.LocationAccuracy
import io.nlopez.smartlocation.location.config.LocationParams
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.MapView
import java.util.*

private val dialogsDelay: Long = 5000L


class HomeActivity : ActivityWithLocationPermission(), PlayerEventListener.Handler {
    private val TAG = HomeActivity::class.java.simpleName

    internal var player = Player("", 0.0, 0.0)
    private var gameListener: GameEventListener? = null
    private var playerListener: PlayerEventListener? = null
    private var map: MapView? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        OSMShortcuts.onCreate(this)
        setContentView(R.layout.activity_home)

        map = OSMShortcuts.findMapById(this, R.id.home_activity_map)

        val app = application as CatchCatch
        gameListener = GameEventListener(app.socket!!, GameEventHandler(this, map!!))
        gameListener!!.connect()
        playerListener = PlayerEventListener(app.socket!!, this)
        playerListener!!.connect()

        val conf = LocationParams.Builder().setAccuracy(LocationAccuracy.HIGH).build()
        SmartLocation.with(this).location().continuous().config(conf).start(this::onLocationUpdate)

        val random = Random()
        RankDialog(this, GameRank("CatchCatch", (0..10).map { PlayerRank("Player $it", random.nextInt()) })).show()
        TransparentDialog(this, "welcome!").showWithTimeout(dialogsDelay)
    }

    private fun onLocationUpdate(l: Location) {
        val point = GeoPoint(l.latitude, l.longitude)
        OSMShortcuts.focus(map!!, point)
        player.updateLocation(l)
        playerListener!!.sendPosition(l)
        OSMShortcuts.showMarkerOnMap(map!!, "me", point)
    }

    override fun onResume() {
        super.onResume()
        OSMShortcuts.onResume(this)
    }

    override fun onDestroy() {
        super.onDestroy()
        map!!.overlays.clear()
    }

    override fun onRegistered(p: Player) = runOnUiThread {
        player = p
        TransparentDialog(this, "Connected as\n${p.id}").showWithTimeout(dialogsDelay)
    }

    override fun onDisconnected() = runOnUiThread {
        TransparentDialog(this, "Disconnected").showWithTimeout(5000)
    }
}

class GameEventHandler(val activity: HomeActivity, val map: MapView) : GameEventListener.Handler {
    private val TAG = GameEventHandler::class.java.simpleName
    private var info: GameInfo? = null
    private var animator: PolygonAnimator? = null

    override fun onGamesAround(games: List<Feature>) = activity.runOnUiThread {
        OSMShortcuts.refreshGeojsonFeaturesOnMap(map, games.map { GeoJsonPolygon(it.id, it.geojson) })
    }

    override fun onGameStarted(info: GameInfo) = activity.runOnUiThread {
        this.info = info
        this.animator = OSMShortcuts.animatePolygonOverlay(map, info.game)
        animator?.overlay?.let { OSMShortcuts.focus(map, it.boundingBox.center) }
        TransparentDialog(activity, "Game ${info.game} started.\nYour role is: ${info.role}").showWithTimeout(dialogsDelay)
    }

    override fun onGameTargetNear(meters: Double) = activity.runOnUiThread {
        OSMShortcuts.drawCircleOnMap(map, "target-dist", activity.player.point(), meters, 1000.0)
    }

    override fun onGameTargetReached(meters: Double) = activity.runOnUiThread {
        TransparentDialog(activity, "You win!\nTarget was ${meters.toInt()}m closer").showWithTimeout(dialogsDelay)
    }

    override fun onGameTargetWin() = activity.runOnUiThread {
        TransparentDialog(activity, "Congratulations!\nYou survived").showWithTimeout(dialogsDelay)
    }

    override fun onGameLoose(gameID: String) = activity.runOnUiThread {
        finish()
        TransparentDialog(activity, "You loose =/").showWithTimeout(dialogsDelay)
    }

    override fun onGameFinish(rank: GameRank) = activity.runOnUiThread {
        finish()
        Handler().postDelayed({
            RankDialog(activity, rank).showWithTimeout(dialogsDelay)
        }, 3000)
    }

    override fun onDisconnected() {
        animator?.stop()
    }

    private fun finish() {
        info = null
        animator?.stop()
    }
}