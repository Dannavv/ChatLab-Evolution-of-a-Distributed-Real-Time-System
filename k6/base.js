import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const messagesSent = new Counter('chat_messages_sent');
const messageLatency = new Trend('chat_message_latency_ms');

export const options = {
    thresholds: {
        chat_message_latency_ms: ['p(95)<2000'],
    },
};

// Robust Mode Configuration
if (__ENV.ROBUST_MODE === 'true') {
    options.stages = [
        { duration: '30s', target: 500 },
        { duration: '1m30s', target: 1500 },
        { duration: '1m30s', target: 2500 },
        { duration: '30s', target: 0 },
    ];
} else {
    options.vus = __ENV.VUS ? parseInt(__ENV.VUS) : 10;
    options.duration = __ENV.DURATION || '30s';
}

export default function () {
    let urlString = __ENV.WS_URLS || __ENV.WS_URL || 'ws://localhost:8080/ws';
    let urls = urlString.split(',');
    let url = urls[Math.floor(Math.random() * urls.length)];
    
    const res = ws.connect(url, {}, function (socket) {
        socket.on('open', () => {
            socket.setInterval(() => {
                const msg = JSON.stringify({
                    user_id: `user-${__VU}`,
                    room_id: 'robust-test',
                    content: `Load check from VU ${__VU}`
                });
                socket.send(msg);
                messagesSent.add(1);
            }, 5000);
        });

        socket.on('message', (data) => {
            const msg = JSON.parse(data);
            if (msg.timestamp) {
                const latency = Date.now() - msg.timestamp;
                messageLatency.add(latency);
            }
        });

        socket.setTimeout(() => {
            socket.close();
        }, 600000);
    });

    check(res, { 'status is 101': (r) => r && r.status === 101 });
}
