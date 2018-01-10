package io.perenecabuto.catchcatch.model

import org.json.JSONObject

data class Feature(val id: String, val geojson: String) {
    constructor(json: JSONObject) : this(json.getString("id"), json.getString("coords"))
}