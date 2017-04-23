package io.perenecabuto.catchcatch

import android.hardware.Sensor
import android.hardware.SensorEvent
import android.hardware.SensorEventListener
import android.hardware.SensorManager

class CompassEventListener(val callback: (Float) -> Unit) : SensorEventListener {
    companion object {
        fun listenCompass(sensors: SensorManager, callback: (Float) -> Unit) {
            val orientation = sensors.getDefaultSensor(Sensor.TYPE_ORIENTATION)
            val listener = CompassEventListener(callback)
            sensors.registerListener(listener, orientation, SensorManager.SENSOR_DELAY_NORMAL)
        }
    }

    override fun onAccuracyChanged(sensor: Sensor?, accuracy: Int) {
    }

    override fun onSensorChanged(event: SensorEvent?) {
        val heading = event?.values?.get(0) ?: return
        callback(-heading)
    }
}