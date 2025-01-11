(() => {
    const eventSource = new EventSource("/__air_internal/sse")

    eventSource.addEventListener('reload', () => {
        location.reload();
    });

    eventSource.addEventListener('build-failed', (event) => {
        const data = JSON.parse(event.data);
        console.error('Error en la construcción:', data.command);
        console.error('Salida estándar:', data.output);
        console.error('Error estándar:', data.error);
    });
})()
