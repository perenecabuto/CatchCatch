function makeText(feat, text) {
    return new ol.style.Text({
        text: text,
        offsetX: 0, offsetY: -35,
        fill: new ol.style.Fill({ color: '#330' }),
        stroke: new ol.style.Stroke({ color: '#fff', width: 4 })
    })
}

let groupStyles = {
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
    let socket = io(location.host, { path: "/ws" });
    let source = new ol.source.Vector({ wrapX: false });
    let raster = new ol.layer.Tile({ source: new ol.source.OSM() });
    let vector = new ol.layer.Vector({ source: source });
    let view = new ol.View({ center: [-51.21766, -30.034647], zoom: 15, projection: "EPSG:4326" })
    let map = new ol.Map({ layers: [raster, vector], target: 'map', view: view });

    let popup = new ol.Overlay({ element: document.getElementById("map-info") });
    var el = popup.getElement();
    map.addOverlay(popup);

    map.on('click', function (evt) {
        let feature = map.forEachFeatureAtPixel(evt.pixel, function (feat, layer) {
            return feat;
        });

        if (feature === undefined || feature.getId() === undefined) {
            el.style.display = "none";
            return;
        }

        el.style.display = "block";
        popup.setPosition(evt.coordinate);
        view.setCenter(evt.coordinate);
        el.getElementsByClassName("panel-body")[0].innerText = feature.getId();
    });

    let controller = new AdminController(socket, source);
    controller.bindDrawGroupButton("geofences", map, "Polygon");
    controller.bindDrawGroupButton("checkpoint", map, "Point");
    document.getElementById("reset").addEventListener("click", controller.reset);

    controller.bindPosition();

    controller.bindDrawFeatureButton("add-player", map, "Point", groupStyles.player.clone(),
        function (feat) {
            let coords = feat.getGeometry().getCoordinates();
            let p = new Player(coords[1], coords[0]);
            p.connect(function (id) {
                let playerFeat = source.getFeatureById(id);
                playerFeat.setStyle(groupStyles.fakePlayer.clone());
                playerFeat.getStyle().setText(makeText(playerFeat, id));
            }, function () {
                map.removeInteraction(p.getInteraction());
                p = null;
            });
            map.addInteraction(p.getInteraction());
        });

    let evtHandler = new EventHandler(controller);
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

let Player = function (x, y) {
    let socket;
    let player = { id: undefined, x: 0, y: 0 };

    function onDisconnected() {
        if (disconnectedCallback !== undefined) {
            disconnectedCallback();
        }
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
        socket.emit('player:update', JSON.stringify({ x: x, y: y }));
    }
    function coords() {
        return { x: player.x, y: player.y };
    }
    function disconnect() {
        socket.close();
    }

    let registredCallback = function () { }, disconnectedCallback = function () { };
    function connect(registredFn, disconnectedFn) {
        registredCallback = registredFn;
        disconnectedCallback = disconnectedFn;
        socket = io(location.host, { reconnection: false, path: "/ws" });
        socket.on('player:registred', onPlayerRegistred)
        socket.on('player:updated', onPlayerUpdated)
        socket.on('disconnect', onDisconnected)
    }

    let feature;
    let interaction = new ol.interaction.Pointer({
        handleDownEvent: function (evt) {
            let map = evt.map;
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
        }
    });

    function getInteraction() {
        return interaction;
    }

    this.updatePosition = updatePosition;
    this.coords = coords;
    this.disconnect = disconnect;
    this.connect = connect;
    this.getInteraction = getInteraction;
}

let AdminController = function (socket, sourceLayer) {
    let playerHTML = document.getElementById("player-template").innerText;

    let lastPos = { coords: { latitude: 0, longitude: 0 } };

    this.updatePosition = function (pos) {
        if (pos.coords.latitude == 0 && pos.coords.longitude == 0) {
            pos = lastPos;
        }
        let coords = { x: pos.coords.latitude, y: pos.coords.longitude };
        socket.emit('player:update', JSON.stringify(coords));
        lastPos = pos;
    };

    this.bindPosition = function () {
        navigator.geolocation.getCurrentPosition(this.updatePosition);
        navigator.geolocation.watchPosition(this.updatePosition);
    }

    this.reset = function () {
        console.log("admin:clear");
        socket.emit('admin:clear');
    };

    this.disconnectPlayer = function (playerId) {
        console.log("admin:disconnect", playerId);
        socket.emit('admin:disconnect', playerId);
    };

    this.requestFeatures = function () {
        socket.emit("admin:feature:request-list", "checkpoint", function () {
            socket.emit("admin:feature:request-list", "geofences");
        });
    };

    this.removePlayer = function (player) {
        let playerEl = document.getElementById("conn-" + player.id);
        if (playerEl !== null) playerEl.remove();

        let feat = sourceLayer.getFeatureById(player.id);
        if (feat !== null) {
            sourceLayer.removeFeature(feat);
        }
    };

    this.updatePlayer = function (player) {
        let elId = "conn-" + player.id;
        let playerEl = document.getElementById(elId);
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
        let coords = [player.y, player.x];
        let feat = sourceLayer.getFeatureById(player.id);
        if (feat !== null) {
            feat.getGeometry().setCoordinates(coords);
            return;
        }
        feat = new ol.Feature({ name: player.id, geometry: new ol.geom.Point(coords, 'XY') });
        feat.setId(player.id);
        feat.setStyle(groupStyles.player.clone());
        feat.getStyle().setText(makeText(feat, player.id));        
        sourceLayer.addFeature(feat);
    }

    this.showCircleOnMap = function (featId, center, distanceInMeters) {
        let id = "circle-" + featId;
        let feat = sourceLayer.getFeatureById(id);
        if (distanceInMeters >= 800) {
            if (feat !== null) sourceLayer.removeFeature(feat);
            return;
        }

        let distance = (distanceInMeters / 100000);
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
        let draw = new ol.interaction.Draw({ source: sourceLayer, type: type, style: style.clone() });
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
        this.bindDrawFeatureButton("draw-" + group, map, type, groupStyles[group].clone(),
            function (feat) {
                let geojson = new ol.format.GeoJSON().writeGeometry(feat.getGeometry());
                let name = prompt("What is this " + group + " name?");
                socket.emit('admin:feature:add', group, name, geojson);
            }
        );
    };

    this.addFeature = function (id, group, feat) {
        if (groupStyles[group] !== undefined) {
            feat.setStyle(groupStyles[group].clone());
        }
        feat.setId(id);
        feat.getStyle().setText(makeText(feat, id));
        sourceLayer.addFeature(feat);
    };
};

let EventHandler = function (controller) {
    let player = { x: 0, y: 0 };

    this.onConnect = function () {
        log("connected");
    };
    this.onDisconnected = function () {
        log("disconnected");
        controller.resetInterface();
    };
    this.onPlayerRegistred = function (p) {
        player = p;
        log("connected as \n" + player.id);
        controller.updatePosition({ coords: { longitude: p.x, latitude: player.y } })
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
        let feat = new ol.format.GeoJSON().readFeature(jsonF.coords);
        controller.addFeature(jsonF.id, jsonF.group, feat);
    };

    let that = this;
    this.onFeatureList = function (features) {
        for (let i in features) {
            that.onFeatureAdded(features[i]);
        }
    };

    this.onFeatureCheckpoint = function (featID, checkPointID, lon, lat, distance) {
        controller.showCircleOnMap
            (checkPointID + "" + featID, [lon, lat], distance);
    }
};
