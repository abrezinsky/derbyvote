// Admin-specific utilities (requires common.js to be loaded first)

// WebSocket connection management for admin real-time updates
const AdminWS = {
    ws: null,
    handlers: {},

    connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsURL = `${protocol}//${window.location.host}/ws`;

        this.ws = new WebSocket(wsURL);

        this.ws.onopen = () => {
            console.log('Admin WebSocket connected');
        };

        this.ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                this.handleMessage(message);
            } catch (error) {
                console.error('Error parsing WebSocket message:', error);
            }
        };

        this.ws.onclose = () => {
            console.log('WebSocket disconnected, reconnecting...');
            setTimeout(() => this.connect(), 3000);
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    },

    handleMessage(message) {
        const handler = this.handlers[message.type];
        if (handler) {
            handler(message.payload);
        }
    },

    on(type, handler) {
        this.handlers[type] = handler;
    }
};

// Alias AdminAPI to API from common.js for backward compatibility
const AdminAPI = API;

// Initialize WebSocket on page load
document.addEventListener('DOMContentLoaded', () => {
    AdminWS.connect();
});
