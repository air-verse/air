(() => {
    const ports = new Set();
    let sse = null;
    let terminationTimer = null;

    self.onconnect = (event) => {
        const port = event.ports[0];
        ports.add(port);

        if (terminationTimer) { // We're still alive
            cancelTermination();
        }

        // Initialize the EventSource once
        if (!sse) {
            sse = new EventSource("/__air_internal/sse");

            sse.addEventListener('reload', (e) => {
                broadcast({ type: 'reload' });
            });

            sse.addEventListener('build-failed', (e) => {
                broadcast({ type: 'build-failed', data: e.data })
            });
        }

        // Handle graceful disconnect message from port
        port.onmessage = (e) => {
            if (e.data === 'disconnect') {
                ports.delete(port);
                if (ports.size === 0) {
                    scheduleTermination();
                }
            }
        };

        port.start();
    };


    function broadcast(data) {
        ports.forEach(port => {
            try {
                port.postMessage(data)
            } catch (e) {
                // This port is dead so we remove it. If this was the last port, schedule termination.
                ports.delete(port);

                if (ports.size === 0) {
                    scheduleTermination();
                }
            }
        });
    }
    
    function cancelTermination() {
        clearTimeout(terminationTimer);
        terminationTimer = null;
    }

    function scheduleTermination() {
        if (terminationTimer) { // Already scheduled
            return
        }

        terminationTimer = setTimeout(() => {
            if (sse) {
                sse.close();
            }
            self.close();
        }, 3000);
    }
})();
