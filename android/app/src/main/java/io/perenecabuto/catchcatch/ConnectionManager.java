package io.perenecabuto.catchcatch;


import android.location.Location;
import android.util.Log;

import org.json.JSONException;
import org.json.JSONObject;

import java.net.URISyntaxException;
import java.util.Arrays;

import io.socket.client.Socket;

class ConnectionManager {

    private static final String TAG = ConnectionManager.class.getName();
    private Socket socket;

    ConnectionManager(Socket socket) {
        this.socket = socket;
    }

    void connect() throws URISyntaxException, NoConnectionException {
        socket
            .on(Socket.EVENT_CONNECT, args -> Log.d(TAG, "connect: " + Arrays.toString(args)))
            .on("player:list", args -> Log.d(TAG, "player:list" + Arrays.toString(args)))
            .on("player:updated", args -> Log.d(TAG, "player:updated" + Arrays.toString(args)))
            .on("player:new", args -> Log.d(TAG, "player:new" + Arrays.toString(args)))
            .on(Socket.EVENT_DISCONNECT, args -> Log.d(TAG, "disconnect: " + Arrays.toString(args)));

        socket.connect();
    }

    void sendPosition(Location l) throws JSONException {
        JSONObject coords = new JSONObject();
        coords.put("x", l.getLatitude());
        coords.put("y", l.getLongitude());
        socket.emit("player:update", coords.toString());
    }

    void disconnect() {
        socket.disconnect();
    }

    class NoConnectionException extends Exception {
        NoConnectionException(String msg) {
            super(msg);
        }
    }
}
