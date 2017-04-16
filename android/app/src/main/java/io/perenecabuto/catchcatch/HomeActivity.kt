package io.perenecabuto.catchcatch

import android.location.Location
import android.os.Bundle
import android.os.Handler
import android.preference.PreferenceManager
import android.widget.Toast
import io.nlopez.smartlocation.SmartLocation
import io.nlopez.smartlocation.location.config.LocationAccuracy
import io.nlopez.smartlocation.location.config.LocationParams
import io.socket.client.IO
import org.osmdroid.config.Configuration
import org.osmdroid.tileprovider.tilesource.TileSourceFactory
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.MapView


class HomeActivity : ActivityWithLocationPermission() {

    internal val updateGamesInterval: Long = 60_000

    private var markerOverlay: MarkerOverlay? = null
    private var manager: PlayerEventHandler? = null
    private var map: MapView? = null

    private val address = "https://beta-catchcatch.ddns.net/"

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Configuration.getInstance().load(this, PreferenceManager.getDefaultSharedPreferences(this))
        Configuration.getInstance().userAgentValue = BuildConfig.APPLICATION_ID
        setContentView(R.layout.activity_home)

        map = findViewById(R.id.home_activity_map) as MapView
        map!!.setTileSource(TileSourceFactory.MAPNIK)
        map!!.setBuiltInZoomControls(false)
        map!!.setMultiTouchControls(false)

        val tiles = map!!.overlayManager.tilesOverlay
        tiles.overshootTileCache = tiles.overshootTileCache * 3

        markerOverlay = MarkerOverlay(this)
        map!!.overlays.add(markerOverlay)

        val socketOpts = IO.Options()
        socketOpts.path = "/ws"
        val socket = IO.socket(address, socketOpts)
        manager = PlayerEventHandler(socket, HomeEventHandler(this, map!!))
        manager!!.connect()

        val conf = LocationParams.Builder().setAccuracy(LocationAccuracy.HIGH).build()
        SmartLocation.with(this).location().continuous().config(conf).start(this::onLocationUpdate)

        var seekForGamesAround: (() -> Unit)? = null
        seekForGamesAround = stop@ {
            if (isFinishing || isDestroyed) return@stop
            manager?.requestAroundGames()
            Handler().postDelayed(seekForGamesAround, updateGamesInterval)
        }
        seekForGamesAround()
    }

    private fun onLocationUpdate(l: Location) {
        val point = GeoPoint(l.latitude, l.longitude)
        manager!!.sendPosition(l)
        updateMarker("me", point)
    }

    override fun onResume() {
        super.onResume()
        Configuration.getInstance().load(this, PreferenceManager.getDefaultSharedPreferences(this))
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
}

class HomeEventHandler(private val activity: HomeActivity, private val map: MapView) : PlayerEventHandler.EventCallback {
    override fun onConnect() {
        activity.runOnUiThread {
            Toast.makeText(activity, "onConnect", Toast.LENGTH_LONG).show()
        }
    }

    override fun onRegistered(p: Player) {
        activity.runOnUiThread {
            activity.updateMarker("me", p.point())
            Toast.makeText(activity, "onRegistered" + p, Toast.LENGTH_LONG).show()
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
}


