package io.perenecabuto.catchcatch.view

import android.Manifest
import android.app.Activity
import android.content.pm.PackageManager
import android.os.Bundle
import android.support.v4.app.ActivityCompat

open class ActivityWithLocationPermission : Activity() {
    companion object {
        private val LOCATION_PERMISSION_REQUEST_CODE = (Math.random() * 10000).toInt()
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        requestPermission()
    }

    private fun requestPermission() {
        ActivityCompat.requestPermissions(this,
            arrayOf(Manifest.permission.ACCESS_FINE_LOCATION), LOCATION_PERMISSION_REQUEST_CODE)
    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<String>, grants: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, grants)
        val permitted = requestCode == LOCATION_PERMISSION_REQUEST_CODE
            && grants.isNotEmpty() && grants[0] == PackageManager.PERMISSION_GRANTED

        if (!permitted) {
            requestPermission()
        }
    }
}