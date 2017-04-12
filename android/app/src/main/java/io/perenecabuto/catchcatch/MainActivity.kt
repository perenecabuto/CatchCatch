package io.perenecabuto.catchcatch

import android.Manifest.permission.ACCESS_FINE_LOCATION
import android.annotation.SuppressLint
import android.app.Activity
import android.content.Context
import android.content.SharedPreferences
import android.content.pm.PackageManager.PERMISSION_GRANTED
import android.graphics.Color
import android.location.Location
import android.location.LocationManager
import android.location.LocationManager.GPS_PROVIDER
import android.location.LocationManager.NETWORK_PROVIDER
import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import android.os.Bundle
import android.os.Handler
import android.preference.PreferenceManager
import android.support.v4.app.ActivityCompat
import android.text.TextUtils
import android.util.Log
import android.view.KeyEvent
import android.view.KeyEvent.ACTION_DOWN
import android.view.KeyEvent.KEYCODE_ENTER
import android.view.View
import android.view.View.GONE
import android.view.View.VISIBLE
import android.widget.EditText
import android.widget.TextView
import android.widget.Toast
import io.perenecabuto.catchcatch.ServerDiscoveryListener.OnDiscoverListener
import io.socket.client.IO
import org.osmdroid.config.Configuration
import org.osmdroid.tileprovider.tilesource.TileSourceFactory
import org.osmdroid.views.MapView
import org.osmdroid.views.overlay.ItemizedIconOverlay
import org.osmdroid.views.overlay.OverlayItem
import org.osmdroid.views.overlay.Polygon
import java.util.*
import kotlin.collections.ArrayList


class MainActivity : Activity(), ConnectionManager.EventCallback, OnDiscoverListener {

    companion object {
        private val TAG = MainActivity::class.java.simpleName
        private val PREFS_SERVER_ADDRESS = "server-address"
        private val LOCATION_PERMISSION_REQUEST_CODE = (Math.random() * 10000).toInt()
    }

    private var prefs: SharedPreferences? = null
    private var manager: ConnectionManager? = null
    private var markerOverlay: ItemizedIconOverlay<OverlayItem>? = null

    private var map: MapView? = null
    private var addressText: EditText? = null

    private val markers = HashMap<String, OverlayItem>()
    private var player = Player("", 0.0, 0.0)

    private val socketOpts = object : IO.Options() {
        init {
            path = "/ws"
        }
    }


    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Configuration.getInstance().load(this, PreferenceManager.getDefaultSharedPreferences(this))
        Configuration.getInstance().userAgentValue = BuildConfig.APPLICATION_ID

        setContentView(R.layout.activity_main)

        map = findViewById(R.id.activity_main_map) as MapView
        map!!.setTileSource(TileSourceFactory.MAPNIK)
        map!!.setMultiTouchControls(true)

        markerOverlay = ItemizedIconOverlay<OverlayItem>(ArrayList<OverlayItem>(), null, this)
        map!!.overlays.add(markerOverlay)

        prefs = getSharedPreferences(javaClass.name, Context.MODE_PRIVATE)
        val serverAddress = prefs!!.getString(PREFS_SERVER_ADDRESS, "")
        addressText = findViewById(R.id.activity_main_address) as EditText
        addressText!!.setText(serverAddress)
        addressText!!.setOnKeyListener { v, keyCode, event -> onChangeServerAddress(v, keyCode, event) }
        connect(serverAddress)

        val label = findViewById(R.id.activity_main_address_label)
        label.visibility = if (TextUtils.isEmpty(addressText!!.text)) VISIBLE else GONE

        val nsdManager = getSystemService(Context.NSD_SERVICE) as NsdManager
        val mdnsListener = ServerDiscoveryListener(nsdManager, this)
        nsdManager.discoverServices("_catchcatch._tcp", NsdManager.PROTOCOL_DNS_SD, mdnsListener)

        setupLocation()
    }

    override fun onResume() {
        super.onResume()
        Configuration.getInstance().load(this, PreferenceManager.getDefaultSharedPreferences(this));
    }

    override fun onDiscovered(info: NsdServiceInfo) {
        val disoveredAddress = "http://" + info.host.hostAddress + ":" + info.port
        runOnUiThread { addressText!!.setText(disoveredAddress) }
        connect(disoveredAddress)
    }

    private fun onChangeServerAddress(v: View, keyCode: Int, event: KeyEvent): Boolean {
        val address = (v as TextView).text.toString()
        val label = findViewById(R.id.activity_main_address_label)
        label.visibility = if (TextUtils.isEmpty(address)) VISIBLE else GONE

        val addressChanged = event.action == ACTION_DOWN && keyCode == KEYCODE_ENTER
        if (addressChanged) {
            Toast.makeText(this, "Address updated to " + address, Toast.LENGTH_SHORT).show()
            connect(address)
        }
        return true
    }

    override fun onDestroy() {
        manager!!.disconnect()
        super.onDestroy()
    }

    private fun connect(address: String) {
        if (TextUtils.isEmpty(address)) {
            Toast.makeText(this, "Can't connect. Address is empty", Toast.LENGTH_SHORT).show()
            return
        }
        prefs!!.edit().putString(PREFS_SERVER_ADDRESS, address).apply()

        try {
            manager?.disconnect()
            val socket = IO.socket(address, socketOpts)
            manager = ConnectionManager(socket, this)
            manager!!.connect()
        } catch (e: Throwable) {
            Log.e(TAG, e.message)
            Toast.makeText(this, "Error to connect to " + address, Toast.LENGTH_SHORT).show()
        }

    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<String>, grants: IntArray) {
        val permitted = requestCode == LOCATION_PERMISSION_REQUEST_CODE
            && grants.isNotEmpty() && grants[0] == PERMISSION_GRANTED

        if (permitted) {
            setupLocation()
        } else {
            requestPermission()
        }
    }

    private fun requestPermission() {
        ActivityCompat.requestPermissions(this, arrayOf(ACCESS_FINE_LOCATION), LOCATION_PERMISSION_REQUEST_CODE)
    }

    private fun setupLocation() {
        if (checkCallingOrSelfPermission(ACCESS_FINE_LOCATION) != PERMISSION_GRANTED) {
            requestPermission()
            return
        }
        val locationManager = this.getSystemService(Context.LOCATION_SERVICE) as LocationManager
        val listener = LocationUpdateListener { l: Location ->
            updateLocalPlayer(l)
            Log.d(TAG, "p:updated:" + player)
        }

        locationManager.requestLocationUpdates(NETWORK_PROVIDER, 0, 0f, listener)
        locationManager.requestLocationUpdates(GPS_PROVIDER, 0, 0f, listener)
    }

    override fun onPlayerList(players: List<Player>) {
        runOnUiThread {
            Log.d(TAG, "remote-player:list " + players)
            clearMarkers()
            showPlayerOnMap(player)
            players.filter { it.id != player.id }.forEach { showPlayerOnMap(it) }
        }
    }


    @SuppressLint("NewApi")
    private fun showPlayerOnMap(p: Player) {
        Log.d(TAG, "showPlayerOnMap:" + p + "-" + player.id + "- " + (p.id != player.id).toString())
        val item = OverlayItem(p.id, p.id, p.point())
        val icon = resources.getDrawable(R.drawable.marker_default, theme)
        item.setMarker(icon)
        if (markers.contains(p.id)) {
            markerOverlay!!.removeItem(markers[p.id])
        }
        markers[p.id] = item
        markerOverlay!!.addItem(item)

        map!!.invalidate()
    }

    override fun onRemotePlayerUpdate(player: Player) {
        runOnUiThread {
            Log.d(TAG, "remote-player:updated " + player)
            showPlayerOnMap(player)
        }
    }

    override fun onRemoteNewPlayer(player: Player) {
        runOnUiThread {
            Log.d(TAG, "remote-player:new " + player)
            showPlayerOnMap(player)
        }
    }

    override fun onConnect() {
        runOnUiThread {
            Toast.makeText(this, "connected", Toast.LENGTH_SHORT).show()
        }
    }

    override fun onRegistred(p: Player) {
        runOnUiThread finish@ {
            this.player = p
            val locationManager = this.getSystemService(Context.LOCATION_SERVICE) as LocationManager
            val l = locationManager.getLastKnownLocation(GPS_PROVIDER) ?:
                locationManager.getLastKnownLocation(NETWORK_PROVIDER) ?:
                return@finish

            updateLocalPlayer(l)
            map!!.controller.setZoom(20)
            map!!.controller.setCenter(player.point())
            Log.d(TAG, "p:register:" + player)
            Toast.makeText(this, "registred as " + player.id, Toast.LENGTH_SHORT).show()
        }
    }

    override fun onRemotePlayerDestroy(player: Player) {
        runOnUiThread {
            markerOverlay?.removeItem(markers[player.id])
            markers.remove(player.id)
        }
    }

    override fun onDiconnected() {
        Log.d(TAG, "diconnected " + player + " " + markers[player.id])
        clearMarkers()
    }

    override fun onDetectCheckpoint(detection: Detection) {
        runOnUiThread {
            Log.d(TAG, "onDetectCheckpoint: " + detection)
            var color = Color.GRAY
            if (detection.nearByInMeters < 100) {
                color = Color.RED
            } else if (detection.nearByInMeters < 500) {
                color = Color.YELLOW
            }

            val circle = Polygon()
            circle.points = Polygon.pointsAsCircle(detection.point(), detection.nearByInMeters);
            circle.strokeColor = color
            circle.strokeWidth = 3F

            map!!.overlays.add(circle)
            map!!.overlays.reverse()
            map!!.invalidate()

            Handler().postDelayed({
                map!!.overlays.remove(circle)
                map!!.invalidate()
            }, 2000)
        }
    }

    private fun updateLocalPlayer(l: Location) {
        this.player = player.updateLocation(l)
        manager?.sendPosition(l)
        showPlayerOnMap(player)
    }

    private fun clearMarkers() {
        runOnUiThread {
            markers.forEach { _, value -> markerOverlay?.removeItem(value) }
            markers.clear()
            map?.invalidate()
            map?.refreshDrawableState()
        }
    }
}
