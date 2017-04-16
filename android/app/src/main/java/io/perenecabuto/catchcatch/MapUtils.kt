package io.perenecabuto.catchcatch

import android.content.Context
import android.graphics.Color
import com.google.gson.JsonParser
import org.osmdroid.bonuspack.kml.KmlGeometry
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.overlay.ItemizedIconOverlay
import org.osmdroid.views.overlay.OverlayItem
import org.osmdroid.views.overlay.Polygon

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