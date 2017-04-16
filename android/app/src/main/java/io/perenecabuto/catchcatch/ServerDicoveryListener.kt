package io.perenecabuto.catchcatch

import android.content.Context
import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import android.util.Log


internal class ServerDiscoveryListener(private val nsdManager: NsdManager, private val discoverCallback: (NsdServiceInfo) -> Unit) : NsdManager.DiscoveryListener {
    companion object {
        private val TAG = "----> " + ServerDiscoveryListener::class.java.simpleName
        fun listen(context: Context, callback: (NsdServiceInfo) -> Unit) {
            val nsdManager = context.getSystemService(Context.NSD_SERVICE) as NsdManager
            val mdnsListener = ServerDiscoveryListener(nsdManager, callback)
            nsdManager.discoverServices("_catchcatch._tcp", NsdManager.PROTOCOL_DNS_SD, mdnsListener)
        }

        fun listen(context: HomeActivity, callback: (String) -> Unit) {
            listen(context, fun(info: NsdServiceInfo) {
                val address = "http://" + info.host.hostAddress + ":" + info.port
                callback(address)
            })
        }
    }

    override fun onDiscoveryStarted(regType: String) {
        Log.d(TAG, "Service discovery started")
    }

    override fun onServiceFound(service: NsdServiceInfo) {
        Log.d(TAG, "Service discovery success" + service)
        nsdManager.resolveService(service, object : NsdManager.ResolveListener {
            override fun onResolveFailed(serviceInfo: NsdServiceInfo, errorCode: Int) {}
            override fun onServiceResolved(serviceInfo: NsdServiceInfo) = discoverCallback(serviceInfo)
        })
    }

    override fun onServiceLost(service: NsdServiceInfo) {
        Log.e(TAG, "service lost" + service)
    }

    override fun onDiscoveryStopped(serviceType: String) {
        Log.i(TAG, "Discovery stopped: " + serviceType)
    }

    override fun onStartDiscoveryFailed(serviceType: String, errorCode: Int) {
        Log.e(TAG, "Discovery failed: Error code:$errorCode - $serviceType")
        nsdManager.stopServiceDiscovery(this)
    }

    override fun onStopDiscoveryFailed(serviceType: String, errorCode: Int) {
        Log.e(TAG, "Discovery failed: Error code:" + errorCode)
        nsdManager.stopServiceDiscovery(this)
    }
}
