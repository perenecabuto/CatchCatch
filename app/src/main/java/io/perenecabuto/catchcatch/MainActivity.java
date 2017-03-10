package io.perenecabuto.catchcatch;

import android.app.ActionBar;
import android.app.Activity;
import android.content.Context;
import android.graphics.Color;
import android.graphics.drawable.ColorDrawable;
import android.location.Location;
import android.location.LocationManager;
import android.os.Bundle;
import android.support.annotation.NonNull;
import android.support.v4.app.ActivityCompat;
import android.util.Log;
import android.view.Window;

import com.google.android.gms.maps.CameraUpdateFactory;
import com.google.android.gms.maps.GoogleMap;
import com.google.android.gms.maps.MapFragment;
import com.google.android.gms.maps.model.LatLng;
import com.google.android.gms.maps.model.MarkerOptions;

import static android.Manifest.permission.ACCESS_FINE_LOCATION;
import static android.location.LocationManager.NETWORK_PROVIDER;
import static android.support.v4.content.PermissionChecker.PERMISSION_GRANTED;


public class MainActivity extends Activity {

    private static final String TAG = MainActivity.class.getSimpleName();
    private static final int LOCATION_PERMISSION_REQUEST_CODE = (int) (Math.random() * 10000);
    private String provider = NETWORK_PROVIDER;
    private Location currentLocation = new Location(provider);

    private MapFragment mapFragment;
    private GoogleMap map;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        mapFragment = (MapFragment) getFragmentManager().findFragmentById(R.id.activity_main_map);
        mapFragment.getMapAsync((GoogleMap m) -> {
            map = m;
            focusOn(currentLocation);
            // TODO OnMapCreate get features around and plot them
            m.setOnCameraMoveListener(() -> {
                // TODO OnCameraChange get features around and plot them
                Log.d(TAG, "position: " + m.getCameraPosition().target + "zoom: " + m.getCameraPosition().zoom);
            });
        });

        if (checkCallingOrSelfPermission(ACCESS_FINE_LOCATION) == PERMISSION_GRANTED) {
            setupLocation();
        } else {
            requestPermission();
        }
    }

    @Override
    public void onRequestPermissionsResult(int requestCode, @NonNull String[] permissions, @NonNull int[] grants) {
        boolean permitted = requestCode == LOCATION_PERMISSION_REQUEST_CODE
            && grants.length > 0 && grants[0] == PERMISSION_GRANTED;

        if (permitted) {
            setupLocation();
        } else {
            requestPermission();
        }
    }

    private void requestPermission() {
        ActivityCompat.requestPermissions(this,
            new String[]{ACCESS_FINE_LOCATION}, LOCATION_PERMISSION_REQUEST_CODE);
    }

    @SuppressWarnings("MissingPermission")
    private void setupLocation() {
        LocationManager locationManager = (LocationManager) this.getSystemService(Context.LOCATION_SERVICE);
        locationManager.requestLocationUpdates(provider, 0, 0, new LocationUpdateListener((Location l) -> {
            // TODO send my location to server
            Log.d(TAG, "location updated");
            focusOn(l);
        }));

        focusOn(locationManager.getLastKnownLocation(provider));
    }

    private void focusOn(Location location) {
        if (location == null || map == null)
            return;
        currentLocation = location;
        LatLng point = new LatLng(location.getLatitude(), location.getLongitude());
        map.addMarker(new MarkerOptions().position(point).title(location.getTime() + ""));
        map.moveCamera(CameraUpdateFactory.newLatLngZoom(point, 15));
    }
}
