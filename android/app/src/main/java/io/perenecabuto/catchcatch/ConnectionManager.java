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

@SuppressWarnings("WeakerAccess")
class ConnectionManager {
    static final String REMOTE_PLAYER_LIST = "remote-player:list";
    static final String PLAYER_REGISTRED = "player:registred";
    static final String REMOTE_PLAYER_NEW = "remote-player:new";
    static final String REMOTE_PLAYER_UPDATED = "remote-player:updated";
    static final String CHECKPOINT_DESTROY = "checkpoint:destroy";
    static final String REMOTE_PLAYER_DESTROY = "remote-player:destroy";
    static final String DETECT_CHECKPOINT = "checkpoint:detect";
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
            .on(REMOTE_PLAYER_LIST, this::onRemotePlayerList)
            .on(PLAYER_REGISTRED, this::onPlayerRegistred)
            .on(REMOTE_PLAYER_NEW, this::onRemotePlayerNew)
            .on(REMOTE_PLAYER_UPDATED, this::onRemotePlayerUpdate)
            .on(CHECKPOINT_DESTROY, this::onRemotePlayerDestroy)
            .on(REMOTE_PLAYER_DESTROY, this::onRemotePlayerDestroy)
            .on(DETECT_CHECKPOINT, this::onDetectCheckpoint)
            .on(Socket.EVENT_DISCONNECT, args -> callback.onDiconnected());

        socket.connect();
    }

    private void onDetectCheckpoint(Object[] args) {
        try {
            Detection detection = getDetectionFronJSON(args[0].toString());
            callback.onDetectCheckpoint(detection);
        } catch (JSONException e) {
            e.printStackTrace();
        }
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

    @NonNull
    private Detection getDetectionFronJSON(String json) throws JSONException {
        JSONObject pJson = new JSONObject(json);
        return new Detection(pJson.getString("checkpoint_id"),
            pJson.getDouble("lon"), pJson.getDouble("lat"), pJson.getDouble("distance"));
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

        void onDetectCheckpoint(Detection detection);
    }

    class NoConnectionException extends Exception {
        public static final long servialVersionUID = -1;

        NoConnectionException(String msg) {
            super(msg);
        }
    }

}
