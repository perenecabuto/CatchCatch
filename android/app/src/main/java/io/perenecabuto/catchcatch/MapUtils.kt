package io.perenecabuto.catchcatch

import android.app.Activity
import android.content.Context
import android.graphics.Color
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

        val tiles = map!!.overlayManager.tilesOverlay
        tiles.overshootTileCache = tiles.overshootTileCache * 3

        return map
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