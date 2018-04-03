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


var messages = {

    load: function(onLoaded) {
        protobuf.load("/protobuf/message.proto",  function(err, root) {
            if (err) throw err;

            let entities = Object.keys(root.toJSON().nested.protobuf.nested);
            for (var i in entities) {
                let name = entities[i];
                let namespace = "protobuf." + name;
                console.log("creating " + namespace);
                messages[name] = root.lookupType(namespace);
            }

            onLoaded();
        });
    }
}

function init() {
    let socket = new WSS(location.href.replace("http", "ws") + "admin", false);
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
    document.getElementById("reset").addEventListener("click", controller.reset);

    controller.bindDrawGroupButton("geofences", map, "Polygon");
    controller.bindDrawGroupButton("checkpoint", map, "Point");
    controller.bindDrawFeatureButton("add-player", map, "Point", groupStyles.player.clone(),
    function (feat) {
        console.log("add feat", feat);
        let coords = feat.getGeometry().getCoordinates();
        let fakePlayer = new Player(coords[1], coords[0], controller);
        map.addInteraction(fakePlayer.getInteraction());
        fakePlayer.connect(function (p) {
            controller.updatePlayer(p);
            let playerFeat = source.getFeatureById(p.id);
            playerFeat.setStyle(groupStyles.fakePlayer.clone());
            playerFeat.getStyle().setText(makeText(playerFeat, p.id));
        }, function () {
            map.removeInteraction(fakePlayer.getInteraction());
        });
    });

    let evtHandler = new AdminEventHandler(controller);
    socket.on('connect', evtHandler.onConnect);
    socket.on('disconnect', evtHandler.onDisconnected)
    socket.on('remote-player:updated', evtHandler.onRemotePlayerUpdated)
    socket.on('remote-player:new', evtHandler.onRemotePlayerNew);
    socket.on("admin:players:disconnected", evtHandler.onRemotePlayerDestroy);
    socket.on("admin:feature:added", evtHandler.onFeatureAdded);
    socket.on("admin:feature:checkpoint", evtHandler.onFeatureCheckpoint);
}


window.addEventListener("DOMContentLoaded", messages.load(init));


let Player = function (x, y, admin) {
    let socket;
    let player = { id: undefined, x: 0, y: 0 };

    function onDisconnected() {
        if (disconnectedCallback !== undefined) {
            disconnectedCallback();
        }
    }
    function onPlayerRegistered(msg) {
        let p = messages.Player.decode(msg);
        player = p;
        updatePosition(x, y);
        if (registeredCallback !== undefined) {
            registeredCallback(p);
        }
    }
    function onPlayerUpdated(msg) {
        let p = messages.Player.decode(msg);
        player = p;
    }
    function updatePosition(lat, lon) {
        let msg = messages.Player.encode({eventName: "player:update", id: player.id, lat: lat, lon: lon}).finish();
        socket.emit(msg);
    }
    function coords() {
        return {lat: player.lat, lon: player.lon};
    }
    function disconnect() {
        socket.close();
    }

    let registeredCallback = function () { }, disconnectedCallback = function () { };
    function connect(registeredFn, disconnectedFn) {
        registeredCallback = registeredFn;
        disconnectedCallback = disconnectedFn;
        socket = new WSS(location.href.replace("http", "ws") + "player");
        socket.on('player:registered', onPlayerRegistered)
        socket.on('player:updated', onPlayerUpdated)

        socket.on('game:started', function (msg) {
            let info = messages.GameInfo.decode(msg);
            log(player.id + ':game:started:' + info.game + ":role:" + info.role);
        })
        socket.on('game:loose', function (msg) {
            let game = messages.Simple.decode(msg);
            // log(player.id + ':game:loose:' + game.id)
        })
        socket.on('game:target:near', function (msg) {
            let near = messages.Distance.decode(msg);
            log(player.id + ':target:near:' + near.dist);
        })
        socket.on('game:target:reached', function (msg) {
            let near = messages.Distance.decode(msg);
            log(player.id + ':target:reached:' + near.dist);
        })
        socket.on('game:target:win', function () {
            log(player.id + ':target:win');
        })
        socket.on('game:finish', function (msg) {
            let rank = messages.GameRank.decode(msg);
            log(player.id + ':game:finish:' + rank.game + "\n" + JSON.stringify(rank.playersRank));
        })
        socket.on('checkpoint:detected', function (msg) {
            let detection = messages.Detection.decode(msg);
            admin.showCircleOnMap(player.id, [detection.lon, detection.lat], detection.nearByMeters);
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
    let playerHTML = document.getElementById("player-template").innerText;

    this.centerByLocation = function () {
        let updatePosition = function(pos) {
            let lon = pos.coords.longitude;
            let lat = pos.coords.latitude;
            view.setCenter([lon, lat]);
            socket.emit(
                messages.Player.encode({eventName: 'admin:position:update', lat: lat, lon: lon, id: 0}).finish()
            );
        };
        navigator.geolocation.getCurrentPosition(updatePosition);
        navigator.geolocation.watchPosition(updatePosition);
    };

    this.reset = function () {
        console.log("admin:clear");
        socket.emit(messages.Simple.encode({eventName: 'admin:clear'}).finish());
    };

    this.disconnectPlayer = function (playerId) {
        console.log("admin:players:disconnect", playerId);
        socket.emit(messages.Simple.encode({eventName: 'admin:players:disconnect', id: playerId}).finish());
    };

    this.requestFeatures = function () {
        socket.emit(messages.Feature.encode({eventName: "admin:players:request", group: "player"}).finish());
        socket.emit(messages.Feature.encode({eventName: "admin:feature:request", group: "checkpoint"}).finish());
        socket.emit(messages.Feature.encode({eventName: "admin:feature:request", group: "geofences"}).finish());
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
            playerEl.getElementsByClassName("disconnect-btn")[0]
                .addEventListener("click", () => this.disconnectPlayer(player.id));
            connectionsEl.appendChild(playerEl);
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
            let data = messages.Feature.encode({
                eventName: "admin:feature:add", group: group, id: name, coords: geojson}).finish();
                socket.emit(data);
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

let AdminEventHandler = function (controller) {
    this.onConnect = function (msg) {
        let data = messages.Simple.decode(msg);
        log("connected as " + data.id);
        controller.centerByLocation();
        controller.requestFeatures();
    };
    this.onDisconnected = function () {
        log("disconnected");
        controller.resetInterface();
    };
    this.onRemotePlayerUpdated = function (msg) {
        let p = messages.Player.decode(msg);
        controller.updatePlayer(p);
    };
    this.onRemotePlayerNew = function (msg) {
        let p = messages.Player.decode(msg);
        controller.updatePlayer(p)
    };
    this.onRemotePlayerDestroy = function (msg) {
        let p = messages.Player.decode(msg);
        controller.removePlayer(p);
    };
    this.onFeatureAdded = function (msg) {
        let feat = messages.Feature.decode(msg);
        let geojson = new ol.format.GeoJSON().readFeature(feat.coords);
        controller.addFeature(feat.id, feat.group, geojson);
    };

    this.onFeatureCheckpoint = function (msg) {
        var detection = messages.Detection.decode(msg);
        var circleID = detection.nearByFeatId + "-" + detection.featId;
        controller.showCircleOnMap(circleID, [detection.lon, detection.lat], detection.nearByMeters);
    }
};

function WSS(address, reconnect) {
    let eventCallbacks = {}
    let ws;

    this.on = function (event, callback) {
        eventCallbacks[event] = callback;
    }
    this.emit = function (message) {
        ws.send(message);
    }

    this.close = function () {
        ws.close();
    }

    function onMessage(event) {
        let payload = new Uint8Array(event.data);
        try {
            let evt = messages.Simple.decode(payload);
            triggerEvent(evt.eventName, payload);
        } catch(e) {
            console.error(e)
        }
    }

    function onClose() {
        triggerEvent('disconnect');
        if (!reconnect) return;
        ws.onclose();
    }

    function onError(event) {
        console.log('onError', event);
        triggerEvent('onerror');
    }

    function init() {
        ws = new WebSocket(address);
        ws.binaryType = 'arraybuffer';
        ws.onmessage = onMessage;
        ws.onclose = onClose;
        ws.onerror = onError;
    }

    function triggerEvent(eventName, message) {
        if (eventCallbacks[eventName]) {
            eventCallbacks[eventName].call(null, message);
        }
    }

    init();
    console.log("wss start", ws)
}