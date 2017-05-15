package io.perenecabuto.catchcatch.model

import android.location.Location
import org.json.JSONObject
import org.osmdroid.util.GeoPoint

data class Player(val id: String, var lat: Double, var lon: Double) {
    constructor(json: JSONObject) : this(json.getString("id"), json.getDouble("lat"), json.getDouble("lon"))

    fun updateLocation(l: Location): Player {
        lat = l.latitude; lon = l.longitude
        return this
    }

    fun point(): GeoPoint {
        return GeoPoint(lat, lon)
    }
}