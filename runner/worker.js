(() => {
    const ports = new Set();
    let sse = null;
    let terminationTimer = null;
    let reconnectTimer = null;
    let reconnectAttempts = 0;

    self.onconnect = (event) => {
        const port = event.ports[0];
        ports.add(port);

        if (terminationTimer) { // We're still alive
            cancelTermination();
        }

        // Initialize the EventSource once
        if (!sse) {
            initSSE();
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


    function initSSE() {
        if (sse) {
            return;
        }

        sse = new EventSource("/__air_internal/sse");

        sse.addEventListener('reload', () => {
            broadcast({ type: 'reload' });
        });

        sse.addEventListener('build-failed', (e) => {
            broadcast({ type: 'build-failed', data: e.data });
        });

        sse.onopen = () => {
            reconnectAttempts = 0;
        };

        sse.onerror = () => {
            if (sse) {
                sse.close();
                sse = null;
            }
            scheduleReconnect();
        };
    }

    function scheduleReconnect() {
        if (reconnectTimer || ports.size === 0) {
            return;
        }

        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 10000);
        reconnectAttempts += 1;
        reconnectTimer = setTimeout(() => {
            reconnectTimer = null;
            if (ports.size === 0) {
                return;
            }
            initSSE();
        }, delay);
    }

    function clearReconnect() {
        if (reconnectTimer) {
            clearTimeout(reconnectTimer);
            reconnectTimer = null;
        }
        reconnectAttempts = 0;
    }

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

        clearReconnect();
        terminationTimer = setTimeout(() => {
            if (sse) {
                sse.close();
                sse = null;
            }
            clearReconnect();
            self.close();
        }, 3000);
    }
})();
