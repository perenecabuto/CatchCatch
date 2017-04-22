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

function makeText(feat, text) {
    return new ol.style.Text({
        text: text,
        offsetX: 0, offsetY: -35,
        fill: new ol.style.Fill({ color: '#330' }),
        stroke: new ol.style.Stroke({ color: '#fff', width: 4 })
    })
}

function log(msg) {
    let logEl = document.getElementById("log");
    logEl.innerHTML = msg;
};


window.addEventListener("DOMContentLoaded", function () {
    let socket = io(location.host, { path: "/ws" });
    let source = new ol.source.Vector({ wrapX: false });
    let raster = new ol.layer.Tile({ source: new ol.source.OSM() });
    let vector = new ol.layer.Vector({ source: source });
    let view = new ol.View({ center: [0, 0], zoom: 15, projection: "EPSG:4326" })
    let map = new ol.Map({ layers: [raster, vector], target: 'map', view: view });

    let popup = new ol.Overlay({ element: document.getElementById("map-info") });
    map.addOverlay(popup);

    map.on('click', function (evt) {
        let feature = map.forEachFeatureAtPixel(evt.pixel, function (feat, layer) {
            return feat;
        });

        var el = popup.getElement();
        if (feature === undefined || feature.getId() === undefined) {
            el.style.display = "none";
            return;
        }

        el.style.display = "block";
        popup.setPosition(evt.coordinate);
        view.setCenter(evt.coordinate);
        el.getElementsByClassName("panel-body")[0].innerText = feature.getId();
    });

    let controller = new AdminController(socket, source, view);
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
    socket.on('player:registered', evtHandler.onPlayerRegistred)
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
        if (registeredCallback !== undefined) {
            registeredCallback(p.id);
        }
    }
    function onPlayerUpdated(p) {
        player = p;
    }
    function updatePosition(lat, lon) {
        socket.emit('player:update', JSON.stringify({ lat: lat, lon: lon }));
    }
    function coords() {
        return { lat: player.lat, lon: player.lon };
    }
    function disconnect() {
        socket.close();
    }

    let registeredCallback = function () { }, disconnectedCallback = function () { };
    function connect(registeredFn, disconnectedFn) {
        registeredCallback = registeredFn;
        disconnectedCallback = disconnectedFn;
        socket = io(location.host, { reconnection: false, path: "/ws" });
        socket.on('player:registered', onPlayerRegistred)
        socket.on('player:updated', onPlayerUpdated)

        socket.on('game:started', function (info) {
            log(player.id + ':game:started:' + info.game + ":role:" + info.role);
        })
        socket.on('game:loose', function (game) {
            log(player.id + ':game:loose:' + game)
        })
        socket.on('game:target:near', function (distToTarget) {
            log(player.id + ':target:near:' + distToTarget);
        })
        socket.on('game:target:reached', function (distToTarget) {
            log(player.id + ':target:reached:' + distToTarget);
        })
        socket.on('game:target:win', function () {
            log(player.id + ':target:win');
        })
        socket.on('game:finish', function (rank) {
            log(player.id + ':game:finish:' + rank.game + "\n" + JSON.stringify(rank.points_per_player));
        })

        socket.on('checkpoint:detected', function (detected) {
            console.log(player.id + ':checkpoint:detected', detected);
        })
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

let AdminController = function (socket, sourceLayer, view) {
    let connectedPlayer = { id: "", x: 0, y: 0 };

    this.setPlayer = function (p) {
        connectedPlayer = p;
    }

    let playerHTML = document.getElementById("player-template").innerText;

    let lastPos = { coords: { latitude: 0, longitude: 0 } };

    this.updatePosition = function (pos) {
        if (pos.coords.latitude == 0 && pos.coords.longitude == 0) {
            pos = lastPos;
        }
        let coords = { lat: pos.coords.latitude, lon: pos.coords.longitude };
        view.setCenter([coords.lon, coords.lat]);
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
            playerEl.getElementsByClassName("player-id")[0].innerHTML = player.id;

            playerEl.addEventListener("click", this.focusOnPlayer(player));

            let connectionsEl = document.getElementById("connections");
            if (player.id === connectedPlayer.id) {
                playerEl.getElementsByClassName("btn")[0].className += " btn-primary"
                playerEl.getElementsByClassName("disconnect-btn")[0].style.display = "none";

                connectionsEl.insertBefore(playerEl, connectionsEl.children[0]);
            } else {
                playerEl.getElementsByClassName("disconnect-btn")[0]
                    .addEventListener("click", () => this.disconnectPlayer(player.id));
                connectionsEl.appendChild(playerEl);
            }
        }

        let lon = Number((player.lon).toFixed(5));
        let lat = Number((player.lat).toFixed(5));
        playerEl.getElementsByClassName("player-coords")[0].innerHTML = lat + ',' + lon

        this.showPlayerOnMap(player);
    };

    this.focusOnPlayer = function (player) {
        return () => {
            let feat = sourceLayer.getFeatureById(player.id);
            if (feat === null) {
                console.log("no feature found for player", player, feat);
                return;
            }
            let coords = feat.getGeometry().getCoordinates();
            view.setCenter(coords);
            this.showCircleOnMap("focus-" + player.id, coords, 20);
        }
    };

    this.showPlayerOnMap = function (player) {
        let coords = [player.lon, player.lat];
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
            let feat = sourceLayer.getFeatureById(id);
            if (feat !== null) sourceLayer.removeFeature(feat);
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

    this.onConnect = function () {
        log("connected");
    };
    this.onDisconnected = function () {
        log("disconnected");
        controller.resetInterface();
    };
    this.onPlayerRegistred = function (p) {
        controller.setPlayer(p);
        log("connected as " + p.id);
        controller.updatePosition({ coords: { latitude: p.lat, longitude: p.lon } })
        controller.requestFeatures();
    };
    this.onPlayerUpdated = function (p) {
        controller.setPlayer(p);
        controller.updatePlayer(p);
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

    this.onFeatureList = (features) => {
        for (let i in features) {
            this.onFeatureAdded(features[i]);
        }
    };

    this.onFeatureCheckpoint = function (detection) {
        var circleID = detection.near_by_feat_id + "-" + detection.feat_id;
        controller.showCircleOnMap(circleID, [detection.lon, detection.lat], detection.near_by_meters);
    }
};
