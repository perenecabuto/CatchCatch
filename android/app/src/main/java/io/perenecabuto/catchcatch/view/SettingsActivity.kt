package io.perenecabuto.catchcatch.view

import android.app.Activity
import android.databinding.DataBindingUtil
import android.os.Bundle
import android.widget.AdapterView
import android.widget.ArrayAdapter
import android.widget.Toast
import io.perenecabuto.catchcatch.CatchCatch
import io.perenecabuto.catchcatch.R
import io.perenecabuto.catchcatch.databinding.ActivitySettingsBinding
import io.perenecabuto.catchcatch.drivers.GameVoice
import io.perenecabuto.catchcatch.sensors.ServerDiscoveryListener


class SettingsActivity : Activity(), ActivityWithApp {
    private val TAG: String = SettingsActivity::class.java.simpleName

    private var binding: ActivitySettingsBinding? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_settings)
        binding = DataBindingUtil.setContentView<ActivitySettingsBinding>(this, R.layout.activity_settings)
        binding?.address = app.address
        binding?.voice = app.tts?.voice

        binding?.activitySettingsVoiceList?.let {
            val items = GameVoice.voices().toList()
            it.adapter = ArrayAdapter<String>(this, android.R.layout.simple_expandable_list_item_1, items)
            it.onItemClickListener = AdapterView.OnItemClickListener { _, _, position, _ ->
                (it.adapter.getItem(position) as String).let { voice ->
                    app.tts?.apply { changeVoice(voice).speak("Voice changed to $voice") }
                    finish()
                }
            }
        }

        binding?.activitySettingsUrlOptions?.let {
            val autoDiscoverLabel = "auto discover"
            val items = CatchCatch.serverAddresses.toMutableList()
            items.add(autoDiscoverLabel)
            it.adapter = ArrayAdapter<String>(this, android.R.layout.simple_expandable_list_item_1, items)
            it.onItemClickListener = AdapterView.OnItemClickListener { _, _, position, _ ->
                (it.adapter.getItem(position) as String).let finish@ { address ->
                    if (address == autoDiscoverLabel) {
                        autoDiscover()
                        return@finish
                    }
                    changeAddress(address)
                }
            }
        }
    }

    fun autoDiscover() {
        ServerDiscoveryListener.listenServerAddress(this, { address ->
            Toast.makeText(this, "Discovered $address", Toast.LENGTH_LONG).show()
            changeAddress(address)
        })
        Toast.makeText(this, "Auto discover started", Toast.LENGTH_LONG).show()
    }

    private fun changeAddress(address: String) {
        binding?.address = address
        app.connectTo(address)
        finish()
    }
}

