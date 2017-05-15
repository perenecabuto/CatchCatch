package io.perenecabuto.catchcatch.view

import android.app.Activity
import android.os.Bundle
import android.view.View
import android.widget.Toast
import io.perenecabuto.catchcatch.R
import io.perenecabuto.catchcatch.sensors.ServerDiscoveryListener

class SettingsActivity : Activity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_settings)
    }

    fun autoDiscover(view: View) {
        ServerDiscoveryListener.listenServerAddress(this, { address ->
            Toast.makeText(this, "Discovered $address", Toast.LENGTH_LONG).show()
        })
        Toast.makeText(this, "Auto discover started", Toast.LENGTH_LONG).show()
    }
}
