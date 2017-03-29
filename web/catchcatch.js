var groupStyles = {
    circle: new ol.style.Style({ stroke: new ol.style.Stroke({ color: 'rgba(239, 21, 9, 0.53)', width: 1 }) }),
    checkpoint: new ol.style.Style({ image: new ol.style.Icon(({ anchor: [0.4, 1], src: 'checkpoint.png' })) }),
    player: new ol.style.Style({ image: new ol.style.Icon(({ anchor: [0.5, 0.9], src: 'marker.png' })) }),
    fakePlayer: new ol.style.Style({ image: new ol.style.Icon(({ anchor: [0.5, 0.9], src: 'blue-marker.png' })) }),
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

    controller.bindDrawFeatureButton("add-player", map, "Point", groupStyles.player,
        function (feat) {
            var coords = feat.getGeometry().getCoordinates();
            var p = new Player(coords[1], coords[0]);
            p.connect(function (id) {
                var playerFeat = source.getFeatureById(id);
                playerFeat.setStyle(groupStyles.fakePlayer);
            });
            map.addInteraction(p.getInteraction());
        });

    var evtHandler = new EventHandler(controller);
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

var Player = function (x, y) {
    var socket;
    var player = { id: undefined, x: 0, y: 0 };

    function onConnect() {
    }
    function onDisconnected() {
        log("disconnected " + player.id);
    }
    function onPlayerRegistred(p) {
        player = p;
        updatePosition(x, y);
        if (registredCallback !== undefined) {
            registredCallback(p.id);
        }
    }
    function onPlayerUpdated(p) {
        player = p;
    }

    function updatePosition(x, y) {
        console.log('player:update', JSON.stringify({ x: x, y: y }));
        socket.emit('player:update', JSON.stringify({ x: x, y: y }));
    }

    function coords() {
        return { x: player.x, y: player.y };
    }

    function disconnect() {
        socket.close();
        if (disconnectedCallback !== undefined) {
            disconnectedCallback();
        }
    }

    var registredCallback = function () { }, disconnectedCallback = function () { };
    function connect(registredFn, disconnectedFn) {
        registredCallback = registredFn;
        disconnectedCallback = disconnectedFn;
        socket = io(location.host, { reconnection: false });
        socket.on('connect', onConnect);
        socket.on('player:registred', onPlayerRegistred)
        socket.on('player:updated', onPlayerUpdated)
        socket.on('disconnect', onDisconnected)
    }

    var feature;
    var interaction = new ol.interaction.Pointer({
        handleDownEvent: function (evt) {
            var map = evt.map;
            feature = map.forEachFeatureAtPixel(evt.pixel, function (feature, layer) {
                if (feature.getId() == player.id) {
                    return feature;
                }
            });
            return !!feature;
        },
        handleDragEvent: function (evt) {
            feature.getGeometry().setCoordinates(evt.coordinate);
            updatePosition(evt.coordinate[1], evt.coordinate[0])
        },
        handleUpEvent: function () {
            feature = null;
            console.log("handleUpEvent")
        }
    });

    this.getInteraction = function () {
        return interaction;
    }

    this.updatePosition = updatePosition;
    this.coords = coords;
    this.disconnect = disconnect;
    this.connect = connect;
}

var AdminController = function (socket, sourceLayer) {
    var playerHTML = document.getElementById("player-template").innerText;

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

    this.removePlayer = function (player) {
        var playerEl = document.getElementById("conn-" + player.id);
        if (playerEl !== null) playerEl.remove();

        var feat = sourceLayer.getFeatureById(player.id);
        if (feat !== null) {
            sourceLayer.removeFeature(feat);
        }
    };

    this.updatePlayer = function (player) {
        var elId = "conn-" + player.id;
        var playerEl = document.getElementById(elId);
        if (playerEl === null) {
            playerEl = document.createElement("div");
            playerEl.innerHTML = playerHTML;
            playerEl.id = elId;
            playerEl.getElementsByClassName("disconnect-btn")[0]
                .addEventListener("click", () => this.disconnectPlayer(player.id));
            document.getElementById("connections").appendChild(playerEl);
        }

        playerEl.getElementsByClassName("player-data")[0].innerHTML = player.id + '<br/>' +
            '<div class="glyphicon glyphicon-map-marker" aria-hidden="true"> (' + player.x + ', ' + player.y + ')</div>';

        this.showPlayerOnMap(player);
    };

    this.showPlayerOnMap = function (player) {
        var coords = [player.y, player.x];
        var feat = sourceLayer.getFeatureById(player.id);
        if (feat !== null) {
            feat.getGeometry().setCoordinates(coords);
            return;
        }
        feat = new ol.Feature({ name: player.id, geometry: new ol.geom.Point(coords, 'XY') });
        feat.setId(player.id);
        feat.setStyle(groupStyles.player);
        sourceLayer.addFeature(feat);
    }

    this.showCircleOnMap = function (id, center, distanceInMeters) {
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
        setTimeout(function () {
            sourceLayer.removeFeature(feat);
        }, 1000);
    };

    this.resetInterface = function () {
        document.getElementById("connections").innerHTML = "";
        sourceLayer.clear(true);
    };

    this.bindDrawFeatureButton = function (elID, map, type, style, drawendCalback) {
        var draw = new ol.interaction.Draw({ source: sourceLayer, type: type, style: style });
        draw.on("drawend", function (evt) {
            drawendCalback(evt.feature);
            setTimeout(function () {
                map.removeInteraction(draw);
                sourceLayer.removeFeature(evt.feature);
            });
        });
        document.getElementById(elID).addEventListener("click", function () {
            map.addInteraction(draw);
        });
        document.addEventListener("keydown", function (e) {
            if (e.key.toLowerCase() == "escape") {
                map.removeInteraction(draw);
            }
        });
    };

    this.bindDrawGroupButton = function (group, map, type) {
        this.bindDrawFeatureButton("draw-" + group, map, type, groupStyles[group],
            function (feat) {
                var geojson = new ol.format.GeoJSON().writeGeometry(feat.getGeometry());
                var name = prompt("What is this " + group + " name?");
                socket.emit('admin:feature:add', group, name, geojson);
            }
        );
    };

    this.addFeature = function (id, group, feat) {
        if (groupStyles[group] !== undefined) {
            feat.setStyle(groupStyles[group]);
        }
        feat.setId(id);
        sourceLayer.addFeature(feat);
    };
};

var EventHandler = function (controller) {
    var player = { x: 0, y: 0 };
    var lastPos = { coords: { latitude: 0, longitude: 0 } };

    function recoverPosition() {
        var hasCachedPosition = lastPos.coords.latitude != 0 && lastPos.coords.longitude != 0;
        if (hasCachedPosition) {
            updatePosition(lastPos);
        }
    };

    function updatePosition(pos) {
        lastPos = pos;
        var coords = { x: pos.coords.latitude, y: pos.coords.longitude };
        socket.emit('player:update', JSON.stringify(coords));
    };

    this.onConnect = function (so) {
        log("connected");
        navigator.geolocation.getCurrentPosition(updatePosition);
        navigator.geolocation.watchPosition(updatePosition);
    };
    this.onDisconnected = function () {
        log("disconnected");
        controller.resetInterface();
    };
    this.onPlayerRegistred = function (p) {
        if (p.x == 0 && p.y == 0) recoverPosition();
        player = p;
        log("connected as \n" + player.id);
        controller.requestFeatures();
    };
    this.onPlayerUpdated = function (p) {
        player = p;
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
        controller.showCircleOnMap
            (checkPointID + "" + featID, [lon, lat], distance);
    }
};