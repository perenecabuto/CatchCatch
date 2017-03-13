package io.perenecabuto.catchcatch;

import android.app.Activity;
import android.content.Context;
import android.location.Location;
import android.location.LocationManager;
import android.os.Bundle;
import android.support.annotation.NonNull;
import android.support.v4.app.ActivityCompat;
import android.util.Log;
import android.widget.Toast;

import com.google.android.gms.maps.CameraUpdateFactory;
import com.google.android.gms.maps.GoogleMap;
import com.google.android.gms.maps.MapFragment;
import com.google.android.gms.maps.model.LatLng;
import com.google.android.gms.maps.model.MarkerOptions;

import org.json.JSONException;

import java.net.URISyntaxException;

import io.socket.client.IO;
import io.socket.client.Socket;

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
    private ConnectionManager manager;

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
                // Log.d(TAG, "position: " + m.getCameraPosition().target + "zoom: " + m.getCameraPosition().zoom);
            });
        });

        setupConnection();

        if (checkCallingOrSelfPermission(ACCESS_FINE_LOCATION) == PERMISSION_GRANTED) {
            setupLocation();
        } else {
            requestPermission();
        }
    }

    @Override
    protected void onDestroy() {
        super.onDestroy();
        manager.disconnect();
    }

    private void setupConnection() {
        try {
            Socket socket = IO.socket("http://192.168.23.102:5000");
            manager = new ConnectionManager(socket);
            manager.connect();
        } catch (URISyntaxException | ConnectionManager.NoConnectionException e) {
            e.printStackTrace();
            Log.e(TAG, e.getMessage(), e);
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
            Log.d(TAG, "location updated");
            try {
                manager.sendPosition(l);
            } catch (JSONException e) {
                e.printStackTrace();
                Toast.makeText(this, "Error to send position", Toast.LENGTH_SHORT).show();
            }
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
