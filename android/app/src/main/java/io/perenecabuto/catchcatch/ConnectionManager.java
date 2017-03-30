package io.perenecabuto.catchcatch;


import android.location.Location;
import android.support.annotation.NonNull;
import android.util.Log;

import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

import java.net.URISyntaxException;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

import io.socket.client.Socket;

class ConnectionManager {

    private static final String TAG = ConnectionManager.class.getName();
    private Socket socket;
    private EventCallback callback;

    ConnectionManager(Socket socket, EventCallback callback) {
        this.socket = socket;
        this.callback = callback;
    }

    void connect() throws URISyntaxException, NoConnectionException {
        socket
            .on(Socket.EVENT_CONNECT, this::onConnect)
            .on("remote-player:list", this::onRemotePlayerList)
            .on("player:registred", this::onPlayerRegistred)
            .on("remote-player:new", this::onRemotePlayerNew)
            .on("remote-player:updated", this::onRemotePlayerUpdate)
            .on("checkpoint:destroy", this::onRemotePlayerDestroy)
            .on("remote-player:destroy", this::onRemotePlayerDestroy)
            .on("remote-player:destroy", this::onRemotePlayerDestroy)
            .on(Socket.EVENT_DISCONNECT, args -> callback.onDiconnected());

        socket.connect();
    }

    private void onConnect(Object[] args) {
        Log.d(TAG, "connect: " + Arrays.toString(args));
    }

    private void onRemotePlayerDestroy(Object[] args) {
        try {
            Player player = getPlayerFromJson(args[0].toString());
            callback.onRemotePlayerDestroy(player);
        } catch (JSONException e) {
            e.printStackTrace();
        }
    }

    private void onRemotePlayerNew(Object[] args) {
        try {
            Player player = getPlayerFromJson(args[0].toString());
            callback.onRemoteNewPlayer(player);
        } catch (JSONException e) {
            e.printStackTrace();
        }
    }

    private void onRemotePlayerUpdate(Object[] args) {
        try {
            Player player = getPlayerFromJson(args[0].toString());
            callback.onRemotePlayerUpdate(player);
        } catch (JSONException e) {
            e.printStackTrace();
        }
    }

    private void onRemotePlayerList(Object[] args) {
        List<Player> players = new ArrayList<>();
        try {
            JSONObject arg = new JSONObject(args[0].toString());
            JSONArray pList = arg.getJSONArray("players");
            for (int i = 0; i < pList.length(); i++) {
                Player player = getPlayerFromJson(pList.getString(i));
                players.add(player);
            }
        } catch (JSONException e) {
            e.printStackTrace();
        }
        callback.onPlayerList(players);
    }

    private void onPlayerRegistred(Object[] args) {
        try {
            Player player = getPlayerFromJson(args[0].toString());
            callback.onRegistred(player);
        } catch (JSONException e) {
            e.printStackTrace();
        }
    }

    @NonNull
    private Player getPlayerFromJson(String arg) throws JSONException {
        JSONObject pJson = new JSONObject(arg);
        return new Player(pJson.getString("id"), pJson.getDouble("x"), pJson.getDouble("y"));
    }

    void sendPosition(Location l) {
        JSONObject coords = new JSONObject();
        try {
            coords.put("x", l.getLatitude());
            coords.put("y", l.getLongitude());
        } catch (JSONException e) {
            e.printStackTrace();
        }
        socket.emit("player:update", coords.toString());
    }

    void disconnect() {
        socket.disconnect();
    }

    interface EventCallback {
        void onPlayerList(List<Player> players);

        void onRemotePlayerUpdate(Player player);

        void onRemoteNewPlayer(Player player);

        void onRegistred(Player player);

        void onRemotePlayerDestroy(Player player);

        void onDiconnected();
    }

    class NoConnectionException extends Exception {
        public static final long servialVersionUID = -1;

        NoConnectionException(String msg) {
            super(msg);
        }
    }

}
