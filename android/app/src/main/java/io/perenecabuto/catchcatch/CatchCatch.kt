package io.perenecabuto.catchcatch

import android.app.Application


class CatchCatch : Application() {
    private var address = "https://beta-catchcatch.ddns.net/ws"
    internal var socket: WebSocketClient = WebSocketClient("http://192.168.23.102:5000/ws")

    override fun onCreate() {
        socket.onDisconnect({ socket.connect() })
        socket.connect()
    }

    override fun onTerminate() {
        socket.shutdown()
        super.onTerminate()
    }
}

