package io.perenecabuto.catchcatch

import android.app.Activity
import android.content.Context
import android.graphics.Color
import android.os.Handler
import android.preference.PreferenceManager
import com.google.gson.JsonParser
import org.osmdroid.bonuspack.kml.KmlGeometry
import org.osmdroid.config.Configuration
import org.osmdroid.tileprovider.tilesource.TileSourceFactory
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.MapView
import org.osmdroid.views.overlay.ItemizedIconOverlay
import org.osmdroid.views.overlay.OverlayItem
import org.osmdroid.views.overlay.Polygon


object OSMShortcuts {
    fun onCreate(context: Context) {
        Configuration.getInstance().load(context, PreferenceManager.getDefaultSharedPreferences(context))
        Configuration.getInstance().userAgentValue = BuildConfig.APPLICATION_ID
    }

    fun onResume(context: Context) {
        Configuration.getInstance().load(context, PreferenceManager.getDefaultSharedPreferences(context))
    }

    fun findMapById(context: Activity, viewId: Int): MapView {
        val map = context.findViewById(viewId) as MapView
        map.setTileSource(TileSourceFactory.MAPNIK)
        map.setBuiltInZoomControls(false)
        map.setMultiTouchControls(false)

        val tiles = map.overlayManager.tilesOverlay
        tiles.overshootTileCache = tiles.overshootTileCache * 3

        return map
    }

    fun drawCircleOnMap(map: MapView, id: String, center: GeoPoint, meters: Double, maxDist: Double) {
        val oldCircle = map.overlays.filter { it is DistanceCircle && it.id == id }
        map.overlays.removeAll(oldCircle)
        val circle = DistanceCircle(id, center, meters, maxDist)
        map.overlays.add(0, circle)
        map.invalidate()
        Handler().postDelayed({ map.overlays.remove(circle) }, 2000)
    }

    fun showMarkerOnMap(map: MapView, id: String, point: GeoPoint) {
        val markerOverlay: MarkerOverlay = map.overlays.firstOrNull({ it is MarkerOverlay && it.id == id }) as? MarkerOverlay ?:
            MarkerOverlay(id, map.context).let { map.overlays.add(it); it }

        val item = OverlayItemWithID(id, point)
        markerOverlay.removeItem(item)
        markerOverlay.addItem(item)
        map.controller?.setCenter(point)
        map.controller?.setZoom(20)
        map.invalidate()
    }
}

class GeoJsonPolygon(val id: String, geojson: String) : Polygon() {
    init {
        val jsonObject = JsonParser().parse(geojson).asJsonObject
        val geom = KmlGeometry.parseGeoJSON(jsonObject)
        strokeColor = Color.BLACK
        strokeWidth = 2F
        fillColor = 0x12121212
        points = geom.mCoordinates
    }

    override fun equals(other: Any?): Boolean {
        return other is GeoJsonPolygon && id == other.id
    }

    override fun hashCode(): Int {
        return id.hashCode()
    }
}

class DistanceCircle(val id: String, center: GeoPoint, dist: Double, maxDist: Double) : Polygon() {
    val color = when {
        dist < maxDist / 3 -> Color.RED
        dist < maxDist / 2 -> Color.YELLOW
        else -> Color.GRAY
    }

    init {
        points = Polygon.pointsAsCircle(center, dist)
        strokeColor = color
        strokeWidth = 3F
    }

    override fun equals(other: Any?): Boolean {
        return other is DistanceCircle && id == other.id
    }

    override fun hashCode(): Int {
        return id.hashCode()
    }
}


class MarkerOverlay(val id: String, context: Context) :
    ItemizedIconOverlay<OverlayItem>(ArrayList<OverlayItem>(), context.resources.getDrawable(R.mipmap.marker, context.theme), null, context) {

    override fun equals(other: Any?): Boolean {
        return other is MarkerOverlay && other.id == this.id
    }

    override fun hashCode(): Int {
        return id.hashCode()
    }
}


class OverlayItemWithID(private val id: String, point: GeoPoint) : OverlayItem(id, id, point) {
    override fun equals(other: Any?): Boolean {
        return other is OverlayItemWithID && other.id == this.id
    }

    override fun hashCode(): Int {
        return id.hashCode()
    }
}