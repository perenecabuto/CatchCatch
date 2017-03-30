package io.perenecabuto.catchcatch;

import android.net.nsd.NsdManager;
import android.net.nsd.NsdServiceInfo;
import android.util.Log;


class ServerDiscoveryListener implements NsdManager.DiscoveryListener {

    private static final String TAG = "----> " + ServerDiscoveryListener.class.getSimpleName();
    private final NsdManager nsdManager;
    private OnDiscoverListener listener;

    ServerDiscoveryListener(NsdManager nsdManager, OnDiscoverListener listener) {
        this.nsdManager = nsdManager;
        this.listener = listener;
    }

    @Override
    public void onDiscoveryStarted(String regType) {
        Log.d(TAG, "Service discovery started");
    }

    @Override
    public void onServiceFound(NsdServiceInfo service) {
        Log.d(TAG, "Service discovery success" + service);
        nsdManager.resolveService(service, new NsdManager.ResolveListener() {
            @Override
            public void onResolveFailed(NsdServiceInfo serviceInfo, int errorCode) {
            }

            @Override
            public void onServiceResolved(NsdServiceInfo serviceInfo) {
                Log.d(TAG, serviceInfo.toString());
                listener.onDiscovered(serviceInfo);
            }
        });
    }

    @Override
    public void onServiceLost(NsdServiceInfo service) {
        Log.e(TAG, "service lost" + service);
    }

    @Override
    public void onDiscoveryStopped(String serviceType) {
        Log.i(TAG, "Discovery stopped: " + serviceType);
    }

    @Override
    public void onStartDiscoveryFailed(String serviceType, int errorCode) {
        Log.e(TAG, "Discovery failed: Error code:" + errorCode + " - " + serviceType);
        nsdManager.stopServiceDiscovery(this);
    }

    @Override
    public void onStopDiscoveryFailed(String serviceType, int errorCode) {
        Log.e(TAG, "Discovery failed: Error code:" + errorCode);
        nsdManager.stopServiceDiscovery(this);
    }

    interface OnDiscoverListener {
        void onDiscovered(NsdServiceInfo info);
    }
}
