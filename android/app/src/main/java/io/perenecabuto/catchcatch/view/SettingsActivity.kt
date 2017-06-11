package io.perenecabuto.catchcatch.view

import android.app.Activity
import android.databinding.DataBindingUtil
import android.os.Bundle
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
        binding = DataBindingUtil.setContentView<ActivitySettingsBinding>(this, R.layout.activity_settings)
        binding?.address = app.address
        binding?.voice = app.tts?.voice

        binding?.activitySettingsVoiceList?.let {
            GameVoice.voices().toList().map {
                SettingsOption(this, it) { voice ->
                    app.tts?.apply { changeVoice(it).speak("Voice changed to $voice ") }
                    if (!isDestroyed || !isFinishing) finish()
                }
            }.forEach { textView -> it.addView(textView) }
        }

        binding?.activitySettingsUrlOptions?.let {
            val autoDiscoverLabel = "auto discover"
            val addressList = CatchCatch.serverAddresses.toMutableList().apply {
                add(autoDiscoverLabel)
            }
            addressList.map {
                SettingsOption(this, it) finish@ { addr ->
                    if (addr == autoDiscoverLabel) autoDiscover() else changeAddress(addr)
                }
            }.forEach { view -> it.addView(view) }
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

