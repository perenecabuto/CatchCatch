package io.perenecabuto.catchcatch

import android.app.Application
import io.perenecabuto.catchcatch.drivers.GameVoice
import io.perenecabuto.catchcatch.drivers.WebSocketClient


class CatchCatch : Application() {
    private var address = "https://beta-catchcatch.ddns.net/ws"
    internal var socket: WebSocketClient = WebSocketClient(address)
    internal var tts: GameVoice? = null

    override fun onCreate() {
        tts = GameVoice(this, { tts?.speak("Welcome to CatchCatch.") })
        socket.onDisconnect({ socket.connect() })
        socket.connect()
    }

    override fun onTerminate() {
        socket.shutdown()
        super.onTerminate()
    }
}