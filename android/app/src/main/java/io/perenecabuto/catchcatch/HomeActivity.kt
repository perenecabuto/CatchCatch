package io.perenecabuto.catchcatch

import android.content.Context
import android.location.Location
import android.os.Bundle
import android.os.Handler
import android.os.Vibrator
import android.view.WindowManager.LayoutParams.FLAG_LAYOUT_NO_LIMITS
import android.view.animation.AnimationUtils
import android.widget.TextView
import io.nlopez.smartlocation.OnLocationUpdatedListener
import io.nlopez.smartlocation.SmartLocation
import io.nlopez.smartlocation.location.config.LocationAccuracy
import io.nlopez.smartlocation.location.config.LocationParams
import org.json.JSONObject
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.MapView
import java.util.*


private val dialogsDelay: Long = 5000L


class HomeActivity : ActivityWithLocationPermission(), OnLocationUpdatedListener {

    internal var player = Player("", 0.0, 0.0)
    private var animator: PolygonAnimator? = null
    private var map: MapView? = null

    private var radar: RadarEventHandler? = null
    private var game: GameEventHandler? = null

    private var tts: GameVoice? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        OSMShortcuts.onCreate(this)

        window.setFlags(FLAG_LAYOUT_NO_LIMITS, FLAG_LAYOUT_NO_LIMITS)
        setContentView(R.layout.activity_home)

        map = OSMShortcuts.findMapById(this, R.id.home_activity_map)

        val conf = LocationParams.Builder().setAccuracy(LocationAccuracy.HIGH).build()
        SmartLocation.with(this).location().continuous().config(conf).start(this)

        val app = application as CatchCatch
        radar = RadarEventHandler(app.socket!!, this)

        tts = GameVoice(this) {
            radar!!.start()
            showMessage("Welcome to CatchCatch!")
        }
    }

    override fun onPostCreate(savedInstanceState: Bundle?) {
        super.onPostCreate(savedInstanceState)

        Handler().postDelayed({
            showRank(GameRank("CatchCatch", (0..10).map { PlayerRank("Player $it", Random().nextInt()) }))
        }, dialogsDelay)
    }

    override fun onLocationUpdated(l: Location) {
        player.updateLocation(l)
        sendPosition(l)

        if (animator == null || animator!!.running.not()) {
            val point = GeoPoint(l.latitude, l.longitude)
            OSMShortcuts.focus(map!!, point, 18)
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

    fun startGame(info: GameInfo) = runOnUiThread finish@ {
        val map = map ?: return@finish
        animator = OSMShortcuts.animatePolygonOverlay(map, info.game)
        animator?.overlay?.let { OSMShortcuts.focus(map, it.boundingBox) }

        val app = application as CatchCatch
        game = GameEventHandler(app.socket!!, info, this)
        radar!!.switchTo(game!!)
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

    fun showMessage(msg: String) = runOnUiThread {
        val vibrator = getSystemService(Context.VIBRATOR_SERVICE) as Vibrator
        vibrator.vibrate(100)
        tts?.speak(msg)
        TransparentDialog(this, msg).showWithTimeout(dialogsDelay)
    }

    fun showInfo(text: String) = runOnUiThread {
        val info = findViewById(R.id.home_activity_info) as TextView
        info.startAnimation(AnimationUtils.loadAnimation(this, android.R.anim.slide_out_right))
        info.text = text.capitalize()
    }

    fun showRank(rank: GameRank) = runOnUiThread {
        RankDialog(this, rank, player).showWithTimeout(dialogsDelay * 2)
    }

    fun showFeatures(games: List<Feature>) = runOnUiThread finish@ {
        val map = map ?: return@finish
        OSMShortcuts.refreshGeojsonFeaturesOnMap(map, games.map { GeoJsonPolygon(it.id, it.geojson) })
    }

    fun showCircleAroundPlayer(meters: Double) = runOnUiThread finish@ {
        val map = map ?: return@finish
        OSMShortcuts.drawCircleOnMap(map, "player-circle", player.point(), meters, 1000.0)
    }
}