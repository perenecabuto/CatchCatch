package io.perenecabuto.catchcatch

import android.location.Location
import org.json.JSONArray
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

data class GameInfo(val game: String, val role: String) {
    constructor(json: JSONObject) : this(json.getString("game"), json.getString("role"))
}

data class PlayerRank(val player: String, val points: Int)
data class GameRank(val game: String?, val pointsPerPlayer: List<PlayerRank>) {
    constructor(json: JSONObject) : this(json.getString("game"),
        json.getJSONArray("points_per_player").let { points ->
            (0..points.length() - 1)
                .map { points.getJSONObject(it) }
                .map { PlayerRank(it.getString("player"), it.getInt("points")) }
        })
}

data class Feature(val id: String, val geojson: String) {
    constructor(json: JSONObject) : this(json.getString("id"), json.getString("coords"))
}
data class FeatureList(val list: List<Feature>) {
    constructor(items: JSONArray) : this(
        (0..items.length() - 1).map { Feature(items.getJSONObject(it)) }
    )
}

data class Detection(val featID: String, val lat: Double, val lon: Double, val nearByInMeters: Double) {
    fun point(): GeoPoint {
        return GeoPoint(lat, lon)
    }
}