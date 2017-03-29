var groupStyles = {
    circle: new ol.style.Style({ stroke: new ol.style.Stroke({ color: 'rgba(239, 21, 9, 0.53)', width: 1 }) }),
    checkpoint: new ol.style.Style({ image: new ol.style.Icon(({ anchor: [0.4, 1], src: 'checkpoint.png' })) }),
    player: new ol.style.Style({ image: new ol.style.Icon(({ anchor: [0.5, 0.9], src: 'marker.png' })) }),
    geofences: new ol.style.Style({
        stroke: new ol.style.Stroke({ color: 'black', width: 2 }),
        fill: new ol.style.Fill({ color: 'rgba(0, 0, 255, 0.1)' })
    })
}

window.addEventListener("DOMContentLoaded", function () {
    var socket = io(location.host);
    var source = new ol.source.Vector({ wrapX: false });
    var raster = new ol.layer.Tile({ source: new ol.source.OSM() });
    var vector = new ol.layer.Vector({ source: source });
    var view = new ol.View({ center: [-51.21766, -30.034647], zoom: 15, projection: "EPSG:4326" })
    var map = new ol.Map({ layers: [raster, vector], target: 'map', view: view });

    var controller = new AdminController(socket, source);
    controller.bindDrawGroupButton("geofences", map, "Polygon");
    controller.bindDrawGroupButton("checkpoint", map, "Point");
    document.getElementById("reset").addEventListener("click", controller.reset);

    var evtHandler = new EventHandler(socket, controller);
    document.getElementById("toggle-run").addEventListener("click", evtHandler.toggleRun);

    socket.on('connect', evtHandler.onConnect);
    socket.on('player:registred', evtHandler.onPlayerRegistred)
    socket.on('player:updated', evtHandler.onPlayerUpdated)
    socket.on('disconnect', evtHandler.onDisconnected)

    socket.on('remote-player:list', evtHandler.onRemotePlayerList);
    socket.on('remote-player:updated', evtHandler.onRemotePlayerUpdated)
    socket.on('remote-player:new', evtHandler.onRemotePlayerNew);
    socket.on("remote-player:destroy", evtHandler.onRemotePlayerDestroy);

    socket.on("admin:feature:list", evtHandler.onFeatureList);
    socket.on("admin:feature:added", evtHandler.onFeatureAdded);

    socket.on("admin:feature:checkpoint", evtHandler.onFeatureCheckpoint)
});

function log(msg) {
    let logEl = document.getElementById("log");
    logEl.innerHTML = msg;
};

var AdminController = function (socket, sourceLayer) {
    var lastPos = { coords: { latitude: 0, longitude: 0 } };

    this.reset = function () {
        console.log("admin:clear");
        socket.emit('admin:clear');
    };

    this.disconnectPlayer = function (playerId) {
        console.log("admin:disconnect", playerId);
        socket.emit('admin:disconnect', playerId);
    };

    this.requestFeatures = function () {
        socket.emit("admin:feature:request-list", "checkpoint");
        socket.emit("admin:feature:request-list", "geofences");
    };

    this.recoverPosition = function () {
        var hasCachedPosition = lastPos.coords.latitude != 0 && lastPos.coords.longitude != 0;
        if (hasCachedPosition) {
            this.updatePosition(lastPos);
        }
    };

    this.updatePosition = function (pos) {
        var coords = { x: pos.coords.latitude, y: pos.coords.longitude };
        lastPos = pos;
        socket.emit('player:update', JSON.stringify(coords));
    };

    this.removePlayer = function (player) {
        var playerEl = document.getElementById("d-" + player.id);
        if (playerEl !== null) playerEl.remove();

        var feat = sourceLayer.getFeatureById(player.id);
        if (feat !== null) {
            sourceLayer.removeFeature(feat);
        }
    };

    this.updatePlayer = function (player) {
        var playerEl = document.getElementById("d-" + player.id);
        if (playerEl === null) {
            var playerHTML = '<div>' +
                '<span class="player-data"></span>' +
                '<a href="javascript:void(0)"class="pull-right" onclick="disconnectPlayer(\'' + player.id + '\')">X</a>' +
                '</div>';

            playerHTML = '<div class="panel panel-default">' +
                '<div class="panel-body">' + playerHTML + '</div>' +
                '</div>';

            document.getElementById("connections").innerHTML +=
                '<div id="d-' + player.id + '">' + playerHTML + '</div>';
        }

        var playerEl = document.getElementById("d-" + player.id);
        var playerData = player.id + '<br/>' +
            '<div class="glyphicon glyphicon-map-marker" aria-hidden="true"> (' + player.x + ', ' + player.y + ')</div>';
        playerEl.getElementsByClassName("player-data")[0].innerHTML = playerData;

        var coords = [player.y, player.x];
        var feat = sourceLayer.getFeatureById(player.id);
        if (feat === null) {
            feat = new ol.Feature({ name: player.id, geometry: new ol.geom.Point(coords, 'XY') });
            feat.setId(player.id);
            feat.setStyle(groupStyles.player);
            sourceLayer.addFeature(feat);
        }

        feat.getGeometry().setCoordinates(coords);
    };

    this.showCircle = function (id, center, distanceInMeters) {
        var id = "circle-" + id;
        var feat = sourceLayer.getFeatureById(id);
        if (distanceInMeters >= 800) {
            if (feat !== null) sourceLayer.removeFeature(feat);
            return;
        }

        var distance = (distanceInMeters / 100000);
        if (feat !== null) {
            feat.getGeometry().setCenterAndRadius(center, distance);
            return;
        }

        feat = new ol.Feature({ name: id, geometry: new ol.geom.Circle(center, distance, 'XY') })
        feat.setId(id);
        feat.setStyle(groupStyles.circle);
        sourceLayer.addFeature(feat);
    };

    this.resetInterface = function () {
        document.getElementById("connections").innerHTML = "";
        sourceLayer.clear(true);
    };

    this.bindDrawGroupButton = function (group, map, type) {
        var draw = new ol.interaction.Draw({ source: sourceLayer, type: type, style: groupStyles[group] });
        draw.on("drawend", function (evt) {
            var g = evt.feature.getGeometry();
            var geojson = new ol.format.GeoJSON().writeGeometry(g);
            var name = prompt("What is this " + group + " name?");
            socket.emit('admin:feature:add', group, name, geojson);
            setTimeout(function () {
                map.removeInteraction(draw);
                sourceLayer.removeFeature(evt.feature);
            });
        });
        document.getElementById("draw-" + group).addEventListener("click", function () {
            map.addInteraction(draw);
        });
        document.addEventListener("keydown", function (e) {
            if (e.key.toLowerCase() == "escape") {
                map.removeInteraction(draw);
            }
        });
    };

    this.bindGeolocation = function () {
        navigator.geolocation.watchPosition(this.updatePosition);
        navigator.geolocation.getCurrentPosition(this.updatePosition);
    };

    this.addFeature = function (id, group, feat) {
        sourceLayer.addFeature(feat.clone());

        if (groupStyles[group] !== undefined) {
            feat.setStyle(groupStyles[group]);
        }
        feat.setId(id);
        sourceLayer.addFeature(feat);
    };
};

var EventHandler = function (socket, controller) {
    var player = { x: 0, y: 0 };

    this.toggleRun = function (e) {
        var btn = e.target;
        if (this.interval !== undefined) {
            clearInterval(this.interval);
            this.interval = undefined;
            btn.innerText = btn.originalText;
            return;
        }
        btn.originalText = btn.innerText;
        btn.innerText = "stop";
        this.interval = setInterval(function () {
            player.y += 0.00005;
            player.x += 0.00005;
            socket.emit('player:update', JSON.stringify(player));
        }, 100);
    };

    this.onConnect = function (so) {
        log("connected");
        controller.bindGeolocation();
    };
    this.onDisconnected = function () {
        log("disconnected");
        controller.resetInterface();
        controller.removePlayer(player);
    };
    this.onPlayerRegistred = function (p) {
        if (p.x == 0 && p.y == 0) controller.recoverPosition();
        player = p;
        log("connected as \n" + player.id);
        controller.requestFeatures();
    };
    this.onPlayerUpdated = function (p) {
        player = p;
        controller.updatePlayer(player);
    };
    this.onRemotePlayerList = function (list) {
        controller.resetInterface();
        for (let i in list.players) {
            controller.updatePlayer(list.players[i]);
        }
    };
    this.onRemotePlayerUpdated = function (player) {
        controller.updatePlayer(player);
    };
    this.onRemotePlayerNew = function (player) {
        controller.updatePlayer(player)
    };
    this.onRemotePlayerDestroy = function (player) {
        controller.removePlayer(player);
    };

    this.onFeatureAdded = function (jsonF) {
        var feat = new ol.format.GeoJSON().readFeature(jsonF.coords);
        controller.addFeature(jsonF.id, jsonF.group, feat);
    };
    var that = this;
    this.onFeatureList = function (features) {
        for (var i in features) {
            that.onFeatureAdded(features[i]);
        }
    };

    this.onFeatureCheckpoint = function (featID, checkPointID, lon, lat, distance) {
        controller.showCircle(checkPointID + "" + featID, [lon, lat], distance);
    }
};