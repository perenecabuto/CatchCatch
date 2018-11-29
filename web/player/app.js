const init = () => {
    const go = new Go();
    const addr = "localhost:5000";
    let mod, inst;

    WebAssembly.instantiateStreaming(fetch("catchcatch.wasm"), go.importObject).then(async (result) => {
        mod = result.module;
        inst = result.instance;
        await go.run(inst);
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
            };

            player.onRegistered(function(p) {
                console.log(p)
                navigator.geolocation.watchPosition(updatePosition, null, options);
                navigator.geolocation.getCurrentPosition(updatePosition);
                player.update(10, 10);
            });
        });
    });
}

init();