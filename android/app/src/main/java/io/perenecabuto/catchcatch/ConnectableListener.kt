package io.perenecabuto.catchcatch

import io.socket.client.Socket

interface ConnectableHandler {
    fun onConnect() {}
    fun onDisconnected() {}
}


interface ConnectableListener {
    val sock: Socket
    val handler: ConnectableHandler
    fun bind()

    fun connect() {
        sock.on(Socket.EVENT_CONNECT) { handler.onConnect() }
        sock.on(Socket.EVENT_DISCONNECT) { handler.onDisconnected() }
        bind()

        sock.connect()
    }

    fun stop() {
        sock.off()
    }

}