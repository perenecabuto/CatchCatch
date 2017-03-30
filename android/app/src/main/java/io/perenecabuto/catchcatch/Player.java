package io.perenecabuto.catchcatch;


import android.location.Location;

import com.google.android.gms.maps.model.LatLng;

class Player {
    private final String id;
    private double x;
    private double y;

    Player(String id, double x, double y) {
        this.id = id;
        this.x = x;
        this.y = y;
    }

    @Override
    public String toString() {
        return "Player{" +
            "id='" + id + '\'' +
            ", x=" + x +
            ", y=" + y +
            '}';
    }

    String getId() {
        return id;
    }

    double getX() {
        return x;
    }

    double getY() {
        return y;
    }

    Player updateLocation(Location l) {
        x = l.getLatitude();
        y = l.getLongitude();
        return this;
    }

    LatLng getPoint() {
        return new LatLng(x, y);
    }
}
