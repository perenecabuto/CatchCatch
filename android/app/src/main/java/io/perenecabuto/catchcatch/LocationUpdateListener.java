package io.perenecabuto.catchcatch;

import android.location.Location;
import android.location.LocationListener;
import android.os.Bundle;


class LocationUpdateListener implements LocationListener {
    private final Callback callback;

    LocationUpdateListener(Callback callback) {
        this.callback = callback;
    }

    @Override
    public void onLocationChanged(Location location) {
        callback.onUpdate(location);
    }

    @Override
    public void onStatusChanged(String s, int i, Bundle bundle) {

    }

    @Override
    public void onProviderEnabled(String s) {

    }

    @Override
    public void onProviderDisabled(String s) {

    }

    interface Callback {
        void onUpdate(Location l);
    }
}
