package io.perenecabuto.catchcatch.drivers

import okhttp3.*
import okio.ByteString


class WebSocketClient(val address: String) : WebSocketListener() {
    private var disconnectCallback: (() -> Unit)? = null
    private var ws: WebSocket? = null
    private var client = OkHttpClient()

    init {
        client.retryOnConnectionFailure()
    }

    fun connect() {
        val req = Request.Builder().url(address).build()
        client = OkHttpClient()
        client.retryOnConnectionFailure()
        client.newWebSocket(req, this)
    }

    fun shutdown() {
        client.dispatcher().executorService().shutdown()
    }

    fun close() {
        ws?.close(1000, null)
    }

    fun onDisconnect(callback: () -> Unit): WebSocketClient {
        disconnectCallback = callback
        return this
    }

    override fun onOpen(webSocket: WebSocket?, response: Response?) {
        ws = webSocket
    }

    override fun onClosed(webSocket: WebSocket?, code: Int, reason: String?) {
        disconnectCallback?.invoke()
    }

    override fun onFailure(webSocket: WebSocket?, t: Throwable?, response: Response?) {
        t?.printStackTrace()
        disconnectCallback?.invoke()
        connect()
    }

    override fun onClosing(webSocket: WebSocket?, code: Int, reason: String?) {
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