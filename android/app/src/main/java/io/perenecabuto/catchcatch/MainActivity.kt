package io.perenecabuto.catchcatch

import android.Manifest.permission.ACCESS_FINE_LOCATION
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
import com.google.android.gms.maps.CameraUpdateFactory
import com.google.android.gms.maps.GoogleMap
import com.google.android.gms.maps.MapFragment
import com.google.android.gms.maps.model.CircleOptions
import com.google.android.gms.maps.model.LatLng
import com.google.android.gms.maps.model.Marker
import com.google.android.gms.maps.model.MarkerOptions
import io.perenecabuto.catchcatch.ServerDiscoveryListener.OnDiscoverListener
import io.socket.client.IO
import java.util.*


class MainActivity : Activity(), ConnectionManager.EventCallback, OnDiscoverListener {

    companion object {
        private val TAG = MainActivity::class.java.simpleName
        private val PREFS_SERVER_ADDRESS = "server-address"
        private val LOCATION_PERMISSION_REQUEST_CODE = (Math.random() * 10000).toInt()
    }

    private var map: GoogleMap? = null
    private var prefs: SharedPreferences? = null
    private var manager: ConnectionManager? = null

    private val markers = HashMap<String, Marker>()
    private var player = Player("", 0.0, 0.0)

    private val socketOpts = object : IO.Options() {
        init {
            path = "/ws"
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        val mapFragment = fragmentManager.findFragmentById(R.id.activity_main_map) as MapFragment
        mapFragment.getMapAsync { map -> onMapSync(map) }

        prefs = getSharedPreferences(javaClass.name, Context.MODE_PRIVATE)
        val serverAddress = prefs!!.getString(PREFS_SERVER_ADDRESS, null)

        val addressText = findViewById(R.id.activity_main_address) as EditText
        addressText.setText(serverAddress)
        addressText.setOnKeyListener { v, keyCode, event -> onChangeServerAddress(v, keyCode, event) }
        connect(serverAddress)

        val label = findViewById(R.id.activity_main_address_label)
        label.visibility = if (TextUtils.isEmpty(addressText.text)) VISIBLE else GONE

        val nsdManager = getSystemService(Context.NSD_SERVICE) as NsdManager
        val mdnsListener = ServerDiscoveryListener(nsdManager, this)
        nsdManager.discoverServices("_catchcatch._tcp", NsdManager.PROTOCOL_DNS_SD, mdnsListener)

        setupLocation()
    }

    override fun onDiscovered(info: NsdServiceInfo) {
        val disoveredAddress = "http://" + info.host.hostAddress + ":" + info.port
        connect(disoveredAddress)
    }

    private fun onMapSync(m: GoogleMap) {
        map = m
        // TODO OnMapCreate get features around and plot them
        // m.setOnCameraMoveListener(() -> {
        // TODO OnCameraChange get features around and plot them
        // Log.d(TAG, "position: " + m.getCameraPosition().target + "zoom: " + m.getCameraPosition().zoom);
        // });
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
        super.onDestroy()
        manager!!.disconnect()
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

    private fun showPlayerOnMap(p: Player) {
        Log.d(TAG, "showPlayerOnMap:" + p + "-" + player.id + "- " + (p.id != player.id).toString())

        val m: Marker = markers[p.id] ?: map!!.addMarker(MarkerOptions().position(p.point()).title(p.id))
        m.isVisible = true
        m.position = p.point()

        markers.put(p.id, m)
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
            map!!.moveCamera(CameraUpdateFactory.newLatLngZoom(player.point(), 15f))
            Log.d(TAG, "p:register:" + player)
            Toast.makeText(this, "registred as " + player.id, Toast.LENGTH_SHORT).show()
        }
    }

    private fun updateLocalPlayer(l: Location) {
        this.player = player.updateLocation(l)
        manager?.sendPosition(l)
        showPlayerOnMap(player)
    }

    override fun onRemotePlayerDestroy(player: Player) {
        runOnUiThread {
            markers[player.id]?.remove()
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
            var zindex = 1f
            if (detection.distance < 100) {
                color = Color.RED
                zindex = 2f
            } else if (detection.distance < 500) {
                color = Color.YELLOW
            }

            val circle = map!!.addCircle(CircleOptions()
                .center(LatLng(detection.lat, detection.lon)).radius(detection.distance)
                .strokeWidth(1.0f).strokeColor(color)
                .fillColor(color)
                .zIndex(zindex)
            )

            Handler().postDelayed({ circle.remove() }, 2000)
        }
    }

    private fun clearMarkers() {
        runOnUiThread {
            for ((_, value) in markers) {
                value.remove()
            }
            markers.clear()
        }
    }
}
