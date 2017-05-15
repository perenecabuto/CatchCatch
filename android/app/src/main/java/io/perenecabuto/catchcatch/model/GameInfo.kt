package io.perenecabuto.catchcatch.model

import org.json.JSONObject

data class GameInfo(val game: String, val role: String) {
    constructor(json: JSONObject) : this(json.getString("game"), json.getString("role"))
}