package io.perenecabuto.catchcatch

import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import android.util.Log


internal class ServerDiscoveryListener(private val nsdManager: NsdManager, private val listener: ServerDiscoveryListener.OnDiscoverListener) : NsdManager.DiscoveryListener {
    companion object {
        private val TAG = "----> " + ServerDiscoveryListener::class.java.simpleName
    }

    override fun onDiscoveryStarted(regType: String) {
        Log.d(TAG, "Service discovery started")
    }

    override fun onServiceFound(service: NsdServiceInfo) {
        Log.d(TAG, "Service discovery success" + service)
        nsdManager.resolveService(service, object : NsdManager.ResolveListener {
            override fun onResolveFailed(serviceInfo: NsdServiceInfo, errorCode: Int) {}

            override fun onServiceResolved(serviceInfo: NsdServiceInfo) {
                Log.d(TAG, serviceInfo.toString())
                listener.onDiscovered(serviceInfo)
            }
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

    internal interface OnDiscoverListener {
        fun onDiscovered(info: NsdServiceInfo)
    }
}
