(() => {
    let worker = null;

    const disconnectWorker = () => {
        if (worker) {
            worker.port.postMessage('disconnect');
        }
    };

    // Try to use SharedWorker for shared SSE connection across all windows
    if (window.SharedWorker) {
        try {
            worker = new SharedWorker('/__air_internal/worker.js', { name: 'air-sse-worker' });
            worker.port.onmessage = (event) => {
                const message = event.data;

                switch (message.type) {
                    case 'reload':
                        location.reload();
                        break;
                    case 'build-failed':
                        const data = parseBuildFailed(message.data);
                        showErrorInModal(data);
                        break;
                }
            };
            worker.port.start();

            // Gracefully disconnect from SharedWorker when the window is closed
            window.addEventListener('beforeunload', disconnectWorker);
            window.addEventListener('pagehide', disconnectWorker);
        } catch (e) {
            // Setting up SharedWorker failed, so fall back to per-window EventSource
            console.warn('air: SharedWorker setup failed, falling back to EventSource', e);
            worker = null;
        }
    }

    // SharedWorker is not available or failed somehow. Use per-window EventSource as fallback
    if (!worker) {
        const eventSource = new EventSource("/__air_internal/sse");

        window.addEventListener('beforeunload', function () {
            eventSource.close();
        });
        window.addEventListener('pagehide', function () {
            eventSource.close();
        });

        eventSource.addEventListener('reload', () => {
            location.reload();
        });

        eventSource.addEventListener('build-failed', (event) => {
            const data = parseBuildFailed(event.data);
            showErrorInModal(data);
        });
    }

    function parseBuildFailed(raw) {
        try {
            const parsed = JSON.parse(raw);
            return {
                error: parsed.error ?? "Build failed",
                command: parsed.command ?? "",
                output: parsed.output ?? "",
            };
        } catch (e) {
            console.warn("air: failed to parse build-failed payload", e);
            return {
                error: "Build failed",
                command: "",
                output: String(raw),
            };
        }
    }

    function showErrorInModal(data) {
        document.body.insertAdjacentHTML(`beforeend`, `
            <style>
                .air__modal {
                    display: none;
                    position: fixed;
                    z-index: 1000;
                    left: 0;
                    top: 0;
                    width: 100%;
                    height: 100%;
                    background-color: rgba(0, 0, 0, 0.5);
                    justify-content: center;
                    align-items: center;
                }
                .air__modal-content {
                    background-color: white;
                    color: black;
                    padding: 20px;
                    border-radius: 5px;
                    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
                    width: 80%;
                }
                .air__modal-header {
                    font-size: 1.5em;
                    margin-bottom: 10px;
                }
                .air__modal-body {
                    margin-bottom: 20px;
                    overflow-x: auto;
                }
                .air__modal-close {
                    background-color: #007bff;
                    color: white;
                    border: none;
                    padding: 10px 15px;
                    border-radius: 5px;
                    cursor: pointer;
                }
                .air__modal pre {
                    background-color: #1e1e1e;
                    color: #f8f8f2;
                    padding: 10px;
                    border-radius: 5px;
                    overflow-x: auto;
                    white-space: pre;
                }
                .air__modal code {
                    font-family: 'Courier New', Courier, monospace;
                }
            </style>
            <div class="air__modal" id="air__modal">
                <div class="air__modal-content">
                    <div class="air__modal-header">Build Error</div>
                    <div class="air__modal-body" id="air__modal-body"></div>
                    <button class="air__modal-close" id="air__modal-close">Close</button>
                </div>
            </div>
        `);
        const modal = document.getElementById('air__modal');
        const modalBody = document.getElementById('air__modal-body');
        const modalClose = document.getElementById('air__modal-close');
        modalBody.innerHTML = `
            <strong>Build Cmd:</strong> <pre><code>${data.command}</code></pre><br>
            <strong>Output:</strong> <pre><code>${data.output}</code></pre><br>
            <strong>Error:</strong> <pre><code>${data.error}</code></pre>
        `;
        modal.style.display = 'flex';

        modalClose.addEventListener('click', () => {
            modal.style.display = 'none';
        });
    }
})();
