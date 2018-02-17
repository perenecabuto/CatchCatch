package io.perenecabuto.catchcatch

import android.app.Application
import android.content.Context
import android.content.SharedPreferences
import io.perenecabuto.catchcatch.drivers.GameVoice
import io.perenecabuto.catchcatch.drivers.WebSocketClient


class CatchCatch : Application() {
    companion object {
        val serverAddresses = listOf(
            "https://beta-catchcatch.ddns.net",
            "wss://catchcatch.ddns.net",
            "wss://catchcatch.pointto.us"
        )
    }

    private val prefs: SharedPreferences by lazy { getSharedPreferences(javaClass.simpleName, Context.MODE_PRIVATE) }
    internal var address
        get() = prefs.getString("address", serverAddresses[0])
        set(value) = prefs.edit().putString("address", value).apply()

    internal var socket: WebSocketClient = WebSocketClient()
    internal var tts: GameVoice? = null

    override fun onCreate() {
        tts = GameVoice(this, { tts?.speak("Welcome to CatchCatch.") })
        connectTo(address)
    }

    override fun onTerminate() {
        socket.shutdown()
        super.onTerminate()
    }

    internal fun connectTo(addr: String) {
        address = addr
        socket.connect(addr + "/ws")
    }
}
