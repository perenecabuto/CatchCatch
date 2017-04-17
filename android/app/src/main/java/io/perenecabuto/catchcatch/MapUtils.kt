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
import org.osmdroid.util.BoundingBox
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.MapView
import org.osmdroid.views.overlay.ItemizedIconOverlay
import org.osmdroid.views.overlay.OverlayItem
import org.osmdroid.views.overlay.Polygon
import java.util.*


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
        map.invalidate()
    }

    fun focus(map: MapView, point: GeoPoint) {
        map.controller?.setCenter(point)
        map.controller?.setZoom(20)
        map.invalidate()
    }

    fun refreshGeojsonFeaturesOnMap(map: MapView, geojsons: List<GeoJsonPolygon>) {
        val gameOverlays = map.overlays.filter { it is GeoJsonPolygon }
        map.overlays.removeAll(gameOverlays)
        map.overlays.addAll(geojsons)
        map.invalidate()
    }

    fun animatePolygonOverlay(map: MapView, id: String): PolygonAnimator? {
        val overlay = map.overlays.firstOrNull({ it is PolygonWithID && it.id == id }) as? PolygonWithID
            ?: return null

        return PolygonAnimator(map, overlay).start()
    }
}

class PolygonAnimator(val map: MapView, val overlay: PolygonWithID) {
    private var stopSign = false
    private var ramdom = Random()

    fun start(): PolygonAnimator {
        animate()
        return this
    }

    fun stop() {
        stopSign = true
    }

    private fun animate() {
        if (stopSign) {
            stopSign = false
            return
        }
        val color = listOf(ramdom.nextInt(254), ramdom.nextInt(254), ramdom.nextInt(254))
        overlay.strokeColor = Color.argb(127, color[0], color[1], color[2])
        overlay.fillColor = Color.argb(25, color[0], color[1], color[2])
        map.invalidate()
        Handler().postDelayed(this::animate, 1_000)
    }

}

class GeoJsonPolygon(id: String, geojson: String) : PolygonWithID(id) {
    init {
        val jsonObject = JsonParser().parse(geojson).asJsonObject
        val geom = KmlGeometry.parseGeoJSON(jsonObject)
        strokeColor = Color.BLACK
        strokeWidth = 3F
        fillColor = 0x12121212
        points = geom.mCoordinates
    }
}


class DistanceCircle(id: String, center: GeoPoint, dist: Double, maxDist: Double) : PolygonWithID(id) {
    val color = when {
        dist < maxDist / 3 -> listOf(Color.RED, Color.argb(127, 169, 86, 66))
        dist < maxDist / 2 -> listOf(Color.YELLOW, Color.argb(127, 169, 165, 66))
        else -> listOf(Color.GRAY, Color.argb(96, 66, 162, 169))
    }

    init {
        points = Polygon.pointsAsCircle(center, dist)
        strokeColor = color[0]
        fillColor = color[1]
        strokeWidth = 2F
    }
}

open class PolygonWithID(val id: String) : Polygon() {
    override fun equals(other: Any?): Boolean {
        return other is PolygonWithID && id == other.id
    }

    override fun hashCode(): Int {
        return id.hashCode()
    }

    val boundingBox: BoundingBox by lazy {
        val max = Double.MAX_VALUE
        val box = mutableListOf(max, max, -max, -max)
        points.forEach { p ->
            box[0] = if (p.latitude < box[0]) p.latitude else box[0]
            box[1] = if (p.longitude < box[1]) p.longitude else box[1]
            box[2] = if (p.latitude > box[2]) p.latitude else box[2]
            box[3] = if (p.longitude > box[3]) p.longitude else box[3]
        }
        BoundingBox(box[0], box[1], box[2], box[3])
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