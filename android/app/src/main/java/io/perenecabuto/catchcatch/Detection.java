package io.perenecabuto.catchcatch;

class Detection {
    final private String checkpoint;
    final private double lon;
    final private double lat;
    final private double distance;

    Detection(String checkpoint, double lon, double lat, double distance) {
        this.checkpoint = checkpoint;
        this.lon = lon;
        this.lat = lat;
        this.distance = distance;
    }

    public String getCheckpoint() {
        return checkpoint;
    }

    public double getLon() {
        return lon;
    }

    public double getLat() {
        return lat;
    }

    public double getDistance() {
        return distance;
    }

    @Override
    public String toString() {
        return "Detection{checkpoint='" + checkpoint + '\'' + ", lon=" + lon + ", lat=" + lat + ", distance=" + distance + '}';
    }
}
