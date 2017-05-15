package io.perenecabuto.catchcatch.model

import org.json.JSONObject

data class GameRank(val game: String?, val pointsPerPlayer: List<PlayerRank>) {
    constructor(json: JSONObject) : this(json.getString("game"),
        json.getJSONArray("points_per_player").let { points ->
            (0..points.length() - 1)
                .map { points.getJSONObject(it) }
                .map { PlayerRank(it.getString("player"), it.getInt("points")) }
        })
}