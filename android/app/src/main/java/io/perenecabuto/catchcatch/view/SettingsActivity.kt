package io.perenecabuto.catchcatch.view

import android.app.Activity
import android.os.Bundle
import android.view.View
import android.widget.AdapterView
import android.widget.ArrayAdapter
import android.widget.ListView
import android.widget.Toast
import io.perenecabuto.catchcatch.CatchCatch
import io.perenecabuto.catchcatch.R
import io.perenecabuto.catchcatch.drivers.GameVoice
import io.perenecabuto.catchcatch.sensors.ServerDiscoveryListener


class SettingsActivity : Activity() {
    private val TAG: String = SettingsActivity::class.java.simpleName

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_settings)

        (findViewById(R.id.activity_settings_voice_list) as ListView).let {
            val items = GameVoice.voices().toList()
            it.adapter = ArrayAdapter<String>(this, android.R.layout.simple_expandable_list_item_1, items)
            it.onItemClickListener = AdapterView.OnItemClickListener { _, _, position, _ ->
                (it.adapter.getItem(position) as String).let { voice ->
                    val app = application as CatchCatch
                    app.tts?.apply { changeVoice(voice).speak("Voice changed to $voice") }
                    finish()
                }
            }
        }
    }

    fun autoDiscover(view: View) {
        ServerDiscoveryListener.listenServerAddress(this, { address ->
            Toast.makeText(this, "Discovered $address", Toast.LENGTH_LONG).show()
        })
        Toast.makeText(this, "Auto discover started", Toast.LENGTH_LONG).show()
    }
}
