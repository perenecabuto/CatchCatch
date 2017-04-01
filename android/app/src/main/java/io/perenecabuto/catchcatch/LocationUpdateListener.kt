package io.perenecabuto.catchcatch

import android.location.Location
import android.location.LocationListener
import android.os.Bundle


internal class LocationUpdateListener(private val callback: Callback) : LocationListener {

    override fun onLocationChanged(location: Location) {
        callback.onUpdate(location)
    }

    override fun onStatusChanged(s: String, i: Int, bundle: Bundle) {
    }

    override fun onProviderEnabled(s: String) {
    }

    override fun onProviderDisabled(s: String) {
    }

    interface Callback {
        fun onUpdate(l: Location)
    }
}
