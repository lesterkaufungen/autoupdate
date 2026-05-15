import { api, type Log } from './api';

type WSHandler = (topic: string, data: any) => void;

class WSClient {
    private socket: WebSocket | null = null;
    private handlers: Set<WSHandler> = new Set();
    private reconnectTimeout: any = null;

    constructor() {}

    connect() {
        if (this.socket || typeof window === 'undefined') return;

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.host;
        const adminToken = localStorage.getItem('admin_token') || '';
        
        // Passing token in query for WS
        this.socket = new WebSocket(`${protocol}//${host}/admin/ws?token=${adminToken}`);

        this.socket.onmessage = (event) => {
            try {
                const { topic, data } = JSON.parse(event.data);
                this.handlers.forEach(h => h(topic, data));
            } catch (e) {
                console.error('WS Error:', e);
            }
        };

        this.socket.onclose = () => {
            this.socket = null;
            clearTimeout(this.reconnectTimeout);
            this.reconnectTimeout = setTimeout(() => this.connect(), 3000);
        };

        this.socket.onerror = (err) => {
            console.error('WS Socket Error:', err);
            this.socket?.close();
        };
    }

    subscribe(handler: WSHandler) {
        this.handlers.add(handler);
        if (!this.socket) this.connect();
        return () => {
            this.handlers.delete(handler);
        };
    }
}

export const ws = new WSClient();
