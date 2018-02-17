package io.perenecabuto.catchcatch.model

import org.osmdroid.util.GeoPoint


data class Detection(val featID: String, val lat: Double, val lon: Double, val nearByInMeters: Double) {
    fun point(): GeoPoint {
        return GeoPoint(lat, lon)
    }
}