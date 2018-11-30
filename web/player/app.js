const init = () => {
    const go = new Go();
    const addr = "localhost:5000";
    let mod, inst;

    WebAssembly.instantiateStreaming(fetch("catchcatch.wasm"), go.importObject).then(async (result) => {
        mod = result.module;
        inst = result.instance;
        await go.run(inst);
    });

    let mapSource = new ol.source.Stamen({layer:"toner"});
    let raster = new ol.layer.Tile({ source: mapSource });
    let source = new ol.source.Vector({ wrapX: false });
    let vector = new ol.layer.Vector({ source: source });
    let view = new ol.View({ center: [0, 0], zoom: 17, projection: "EPSG:4326" })
    let map = new ol.Map({
        interactions: ol.interaction.defaults({ mouseWheelZoom: false }),
        controls: [],
        layers: [raster, vector], target: 'map', view: view
    });

    document.addEventListener("catchcatch:ready", function() {
        catchcatch.NewPlayer(addr, function(player) {
            let options = {
                enableHighAccuracy: true,
                timeout: 5000,
                maximumAge: 0
            };

            const updatePosition = pos => {
                let lon = pos.coords.longitude;
                let lat = pos.coords.latitude;
                player.update(lat, lon);
                view.setCenter([lon, lat]);
            };

            player.onRegistered(state => {
                console.log("registered as:", state);
                navigator.geolocation.watchPosition(updatePosition, null, options);
                navigator.geolocation.getCurrentPosition(updatePosition);
            });
            player.onDisconnect(() => {
                console.log("disconnected!");
            });
            player.onGameStarted((game, role) => {
                console.log("game stared", game, "role", role);
                player.coords(lonlat => console.log('lonlat', lonlat));
                navigator.geolocation.watchPosition(updatePosition, null, options);
                navigator.geolocation.getCurrentPosition(updatePosition);
            });
            player.onGamePlayerNearToTarget(dist => {
                console.log("near to target:", dist);
            })
            player.onGamePlayerLose(() => {
                console.log("you lose");
            });
            player.onGamePlayerWin(dist => {
                console.log("you win - dist to target:", dist)
            });
            player.onGameFinished((game, rank) => {
                console.warn("game finished", game, "rank", rank);
            });

            map.on('moveend', function(evt) {
                let center = map.getView().getCenter();
                player.update(center[1], center[0]);
            });
        });
    });
}

init();