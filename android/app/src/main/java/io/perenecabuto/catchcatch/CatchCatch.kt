package io.perenecabuto.catchcatch

import android.app.Application
import io.socket.client.IO
import io.socket.client.Socket


class CatchCatch : Application() {
    private val TAG = HomeActivity::class.java.simpleName
    private val address = "https://beta-catchcatch.ddns.net/"

    internal var socket: Socket? = null

    override fun onCreate() {
        val socketOpts = object: IO.Options() {
            init {
                path = "/ws"
                rememberUpgrade = true
                timestampRequests = false
                reconnection = true
                reconnectionAttempts = -1
            }
        }
        socket = IO.socket(address, socketOpts)
    }

    override fun onTerminate() {
        super.onTerminate()
        socket?.close()
    }
}
