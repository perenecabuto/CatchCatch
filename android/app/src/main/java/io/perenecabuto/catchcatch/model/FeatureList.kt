package io.perenecabuto.catchcatch.model

import org.json.JSONArray

data class FeatureList(val list: List<Feature>) {
    constructor(items: JSONArray) : this(
        (0..items.length() - 1).map { Feature(items.getJSONObject(it)) }
    )
}