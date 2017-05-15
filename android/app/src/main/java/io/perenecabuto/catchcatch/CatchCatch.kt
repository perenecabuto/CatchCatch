package io.perenecabuto.catchcatch

import android.app.Application
import io.perenecabuto.catchcatch.drivers.WebSocketClient


class CatchCatch : Application() {
    private var address = "https://beta-catchcatch.ddns.net/ws"
    internal var socket: WebSocketClient = WebSocketClient(address)

    override fun onCreate() {
        socket.onDisconnect({ socket.connect() })
        socket.connect()
    }

    override fun onTerminate() {
        socket.shutdown()
        super.onTerminate()
    }
}