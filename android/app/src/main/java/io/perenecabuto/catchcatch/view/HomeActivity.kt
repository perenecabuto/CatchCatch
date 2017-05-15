package io.perenecabuto.catchcatch.view

import android.content.Context
import android.content.Intent
import android.hardware.SensorManager
import android.location.Location
import android.os.Bundle
import android.os.Handler
import android.os.Vibrator
import android.view.View
import android.view.View.GONE
import android.view.View.VISIBLE
import android.view.WindowManager.LayoutParams.FLAG_LAYOUT_NO_LIMITS
import android.view.animation.AnimationUtils
import android.widget.TextView
import io.nlopez.smartlocation.OnLocationUpdatedListener
import io.nlopez.smartlocation.SmartLocation
import io.nlopez.smartlocation.location.config.LocationAccuracy
import io.nlopez.smartlocation.location.config.LocationParams
import io.perenecabuto.catchcatch.*
import io.perenecabuto.catchcatch.drivers.GameVoice
import io.perenecabuto.catchcatch.drivers.GeoJsonPolygon
import io.perenecabuto.catchcatch.drivers.OSMShortcuts
import io.perenecabuto.catchcatch.drivers.PolygonAnimator
import io.perenecabuto.catchcatch.sensors.CompassEventListener
import io.perenecabuto.catchcatch.events.GameEventHandler
import io.perenecabuto.catchcatch.events.RadarEventHandler
import io.perenecabuto.catchcatch.model.*
import org.json.JSONObject
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
    private var radarView: RadarView? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        OSMShortcuts.onCreate(this)

        window.setFlags(FLAG_LAYOUT_NO_LIMITS, FLAG_LAYOUT_NO_LIMITS)
        setContentView(R.layout.activity_home)

        val map = OSMShortcuts.findMapById(this, R.id.activity_home_map)
        this.map = map

        map.setOnTouchListener({ _, _ -> true })
        radarView = findViewById(R.id.activity_home_radar) as RadarView

        val sensors = getSystemService(Context.SENSOR_SERVICE) as SensorManager
        CompassEventListener.listenCompass(sensors) { heading ->
            if (animator?.running != true) {
                map.mapOrientation = heading
            }
        }

        val conf = LocationParams.Builder().setAccuracy(LocationAccuracy.HIGH).build()
        SmartLocation.with(this).location().continuous().config(conf).start(this)

        val app = application as CatchCatch
        radar = RadarEventHandler(app.socket, this)
        tts = GameVoice(this) {
            showMessage("welcome to CatchCatch!")
            radar?.start()
        }

        showInfo("starting...")
    }

    override fun onPostCreate(savedInstanceState: Bundle?) {
        super.onPostCreate(savedInstanceState)

        Handler().postDelayed({
            showRank(GameRank("CatchCatch", (0..10).map { PlayerRank("Player $it", Random().nextInt()) }))
        }, dialogsDelay)
    }

    override fun onLocationUpdated(l: Location) {
        sendPosition(l)
        val point = player.updateLocation(l).point()
        val map = map ?: return
        OSMShortcuts.showMarkerOnMap(map, "me", point)
        if (animator?.running != true) {
            OSMShortcuts.focus(map, point, 18)
        }
    }

    override fun onResume() {
        super.onResume()
        OSMShortcuts.onResume(this)
    }

    override fun onDestroy() {
        super.onDestroy()
        val map = map ?: return
        map.overlays.clear()
    }

    fun showSettings(view: View) {
        val intent = Intent(this, SettingsActivity::class.java)
        startActivity(intent)
    }

    fun startGame(info: GameInfo) = runOnUiThread finish@ {
        hideRadar()

        val map = map ?: return@finish
        animator = OSMShortcuts.animatePolygonOverlay(map, info.game)
        animator?.overlay?.let { OSMShortcuts.focus(map, it.boundingBox) }

        val sock = (application as CatchCatch).socket
        val radar = radar ?: return@finish
        GameEventHandler(sock, info, this).let {
            game = it
            radar.switchTo(it)
        }
    }

    fun gameOver() = runOnUiThread finish@ {
        animator?.stop()
        game?.switchTo(radar ?: return@finish)
    }

    fun sendPosition(l: Location) {
        val coords = JSONObject(mapOf("lat" to l.latitude, "lon" to l.longitude))
        val sock = (application as CatchCatch).socket
        sock.emit("player:update", coords.toString())
    }

    fun showMessage(msg: String) = runOnUiThread {
        val vibrator = getSystemService(Context.VIBRATOR_SERVICE) as Vibrator
        vibrator.vibrate(100)
        tts?.speak(msg)
        TransparentDialog(this, msg).showWithTimeout(dialogsDelay)
    }

    fun showInfo(text: String) = runOnUiThread {
        val info = findViewById(R.id.activity_home_info) as TextView
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
        OSMShortcuts.drawCircleOnMap(map, "player-circle", player.point(), meters, 100.0)
    }

    fun showRadar() = runOnUiThread {
        radarView?.visibility = VISIBLE
    }

    fun hideRadar() = runOnUiThread {
        radarView?.visibility = GONE
    }
}