package io.perenecabuto.catchcatch

import android.Manifest
import android.annotation.SuppressLint
import android.app.Activity
import android.content.Context
import android.content.pm.PackageManager
import android.location.Location
import android.os.Bundle
import android.preference.PreferenceManager
import android.support.v4.app.ActivityCompat
import io.nlopez.smartlocation.SmartLocation
import io.nlopez.smartlocation.location.config.LocationAccuracy
import io.nlopez.smartlocation.location.config.LocationParams
import org.osmdroid.config.Configuration
import org.osmdroid.tileprovider.tilesource.TileSourceFactory
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.MapView
import org.osmdroid.views.overlay.ItemizedIconOverlay
import org.osmdroid.views.overlay.OverlayItem


class HomeActivity : WithLocationActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        Configuration.getInstance().load(this, PreferenceManager.getDefaultSharedPreferences(this))
        Configuration.getInstance().userAgentValue = BuildConfig.APPLICATION_ID
        setContentView(R.layout.activity_home)

        val map = findViewById(R.id.home_activity_map) as MapView
        map.setTileSource(TileSourceFactory.MAPNIK)
        map.setBuiltInZoomControls(false)
        map.setMultiTouchControls(false)

        val tiles = map.overlayManager.tilesOverlay
        tiles.overshootTileCache = tiles.overshootTileCache * 3

        val markerOverlay = MarkerOverlay(this)
        map.overlays.add(markerOverlay)

        val conf = LocationParams.Builder().setAccuracy(LocationAccuracy.HIGH).build()
        SmartLocation.with(this).location().continuous().config(conf).start(onLocationUpdate(map, markerOverlay))
    }

    private fun onLocationUpdate(map: MapView, markerOverlay: ItemizedIconOverlay<OverlayItem>): (Location) -> Unit {
        return fun(l: Location) {
            val point = GeoPoint(l.latitude, l.longitude)
            val item = OverlayItemWithID("me", point)
            markerOverlay.removeItem(item)
            markerOverlay.addItem(item)
            map.controller.setCenter(point)
            map.controller.setZoom(20)
            map.invalidate()
        }
    }

    override fun onResume() {
        super.onResume()
        Configuration.getInstance().load(this, PreferenceManager.getDefaultSharedPreferences(this))
    }
}

class MarkerOverlay(context: Context) :
    ItemizedIconOverlay<OverlayItem>(ArrayList<OverlayItem>(), context.resources.getDrawable(R.mipmap.marker, context.theme), null, context)

class OverlayItemWithID(private val id: String, point: GeoPoint) : OverlayItem(id, id, point) {
    override fun equals(other: Any?): Boolean {
        val otherItem = (other as OverlayItemWithID)
        return otherItem.id == this.id
    }

    override fun hashCode(): Int {
        return id.hashCode()
    }
}

open class WithLocationActivity : Activity() {
    companion object {
        private val LOCATION_PERMISSION_REQUEST_CODE = (Math.random() * 10000).toInt()
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        requestPermission()
    }

    private fun requestPermission() {
        ActivityCompat.requestPermissions(this,
            arrayOf(Manifest.permission.ACCESS_FINE_LOCATION), LOCATION_PERMISSION_REQUEST_CODE)
    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<String>, grants: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, grants)
        val permitted = requestCode == LOCATION_PERMISSION_REQUEST_CODE
            && grants.isNotEmpty() && grants[0] == PackageManager.PERMISSION_GRANTED

        if (!permitted) {
            requestPermission()
        }
    }
}
