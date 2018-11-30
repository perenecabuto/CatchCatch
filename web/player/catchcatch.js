const go = new Go();

WebAssembly.instantiateStreaming(fetch("catchcatch.wasm"), go.importObject).then(async (result) => {
    const inst = result.instance;
    await go.run(inst);
});

const onCatchCatchReady = cb => {
    document.addEventListener("catchcatch:ready", cb);
}