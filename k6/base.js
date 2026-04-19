import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const messagesSent = new Counter('chat_messages_sent');
const messageLatency = new Trend('chat_message_latency_ms');

function parseIntEnv(name, fallback) {
    const value = __ENV[name];
    if (!value) {
        return fallback;
    }
    const parsed = parseInt(value, 10);
    return Number.isNaN(parsed) ? fallback : parsed;
}

function parseStagesEnv() {
    if (!__ENV.STAGES_JSON) {
        return null;
    }
    try {
        const parsed = JSON.parse(__ENV.STAGES_JSON);
        if (Array.isArray(parsed) && parsed.length > 0) {
            return parsed;
        }
    } catch (e) {
        // Keep default behavior if stages are invalid.
    }
    return null;
}

function pickEndpoint(urls) {
    if (urls.length === 0) {
        return 'ws://localhost:8080/ws';
    }
    const seed = parseIntEnv('DETERMINISTIC_SEED', 42);
    const index = Math.abs((__VU + seed) % urls.length);
    return urls[index];
}

export const options = {
    thresholds: {
        chat_message_latency_ms: ['p(95)<2000'],
    },
};

const envStages = parseStagesEnv();
if (envStages) {
    options.stages = envStages;
} else if (__ENV.ROBUST_MODE === 'true') {
    options.stages = [
        { duration: '30s', target: 500 },
        { duration: '1m30s', target: 1500 },
        { duration: '1m30s', target: 2500 },
        { duration: '30s', target: 0 },
    ];
} else {
    options.vus = parseIntEnv('VUS', 10);
    options.duration = __ENV.DURATION || '30s';
}

export default function () {
    const messageIntervalMs = parseIntEnv('MESSAGE_INTERVAL_MS', 5000);
    const urlString = __ENV.WS_URLS || __ENV.WS_URL || 'ws://localhost:8080/ws';
    const urls = urlString
        .split(',')
        .map((u) => u.trim())
        .filter((u) => u.length > 0);
    const url = pickEndpoint(urls);
    
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
            }, messageIntervalMs);
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
