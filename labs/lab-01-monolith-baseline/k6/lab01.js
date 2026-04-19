import ws from 'k6/ws';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const messagesSent = new Counter('lab01_messages_sent');
const messageLatency = new Trend('lab01_message_latency_ms');

function parseIntEnv(name, fallback) {
    const raw = __ENV[name];
    if (!raw) {
        return fallback;
    }
    const parsed = parseInt(raw, 10);
    return Number.isNaN(parsed) ? fallback : parsed;
}

function parseStages() {
    if (!__ENV.STAGES_JSON) {
        return null;
    }

    try {
        const parsed = JSON.parse(__ENV.STAGES_JSON);
        if (Array.isArray(parsed) && parsed.length > 0) {
            return parsed;
        }
    } catch (error) {
        // Fall back to defaults if the manifest is malformed.
    }

    return null;
}

export const options = {
    thresholds: {
        lab01_message_latency_ms: ['p(95)<2000'],
    },
};

const stages = parseStages();
if (stages) {
    options.stages = stages;
} else {
    options.vus = parseIntEnv('VUS', 10);
    options.duration = __ENV.DURATION || '30s';
}

export default function () {
    const wsUrl = __ENV.WS_URL || 'ws://localhost:8080/ws';
    const messageIntervalMs = parseIntEnv('MESSAGE_INTERVAL_MS', 5000);

    const res = ws.connect(wsUrl, {}, function (socket) {
        socket.on('open', () => {
            socket.setInterval(() => {
                const now = Date.now();
                const payload = JSON.stringify({
                    user_id: `lab01-user-${__VU}`,
                    room_id: 'lab01-baseline',
                    content: `lab01 benchmark message from vu ${__VU}`,
                    trace_id: `lab01-${__VU}-${now}`,
                });
                socket.send(payload);
                messagesSent.add(1);
            }, messageIntervalMs);
        });

        socket.on('message', (data) => {
            try {
                const message = JSON.parse(data);
                if (message.timestamp) {
                    messageLatency.add(Date.now() - message.timestamp);
                }
            } catch (error) {
                // Ignore non-JSON messages.
            }
        });

        socket.setTimeout(() => socket.close(), 600000);
    });

    check(res, { 'status is 101': (r) => r && r.status === 101 });
}
