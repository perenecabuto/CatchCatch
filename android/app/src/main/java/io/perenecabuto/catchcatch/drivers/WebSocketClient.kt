package io.perenecabuto.catchcatch.drivers

import android.os.Handler
import android.os.Looper
import okhttp3.*
import okio.ByteString


class WebSocketClient : WebSocketListener() {
    private var disconnectCallback: (() -> Unit)? = null
    private var ws: WebSocket? = null
    private var client: OkHttpClient? = null
    private var connected = false


    fun connect(address: String) {
        shutdown()
        val req = Request.Builder().url(address).build()
        client = OkHttpClient.Builder().build()
        client?.newWebSocket(req, this)
    }

    fun shutdown() {
        close()
        client?.apply { dispatcher().executorService().shutdown() }
    }

    fun close() {
        ws?.close(1000, null)
    }

    fun onDisconnect(callback: () -> Unit): WebSocketClient {
        disconnectCallback = {
            if (connected) callback.invoke()
            connected = false
        }
        return this
    }

    private fun reconnect() {
        ws?.request()?.let { req -> client?.newWebSocket(req, this) }
    }

    override fun onOpen(webSocket: WebSocket?, response: Response?) {
        connected = true
        ws = webSocket
    }

    override fun onClosed(webSocket: WebSocket?, code: Int, reason: String?) {
        disconnectCallback?.invoke()
    }

    override fun onFailure(webSocket: WebSocket?, t: Throwable?, response: Response?) {
        t?.printStackTrace()
        Handler(Looper.getMainLooper()).postDelayed(this::reconnect, 2_000L)
        disconnectCallback?.invoke()
    }

    override fun onClosing(webSocket: WebSocket?, code: Int, reason: String?) {
        disconnectCallback?.invoke()
    }

    override fun onMessage(webSocket: WebSocket?, bytes: ByteString?) {
        onMessage(webSocket, bytes.toString())
    }

    override fun onMessage(webSocket: WebSocket?, text: String?) {
        text?.split(delimiters = ',', limit = 2)?.let { split ->
            events[split[0]]?.invoke(split[1])
        }
    }

    private val events: HashMap<String, (msg: String) -> Unit> = HashMap()

    fun off(): WebSocketClient {
        events.clear()
        return this
    }

    fun on(event: String, callback: (String) -> Unit): WebSocketClient {
        events[event] = callback
        return this
    }

    fun emit(event: String, msg: String = ""): WebSocketClient {
        ws?.send("$event,$msg")
        return this
    }
}