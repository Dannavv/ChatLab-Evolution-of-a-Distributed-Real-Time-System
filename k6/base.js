import ws from 'k6/ws';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const messagesSent = new Counter('chat_messages_sent');
const messagesReceived = new Counter('chat_messages_received');
const duplicateMessages = new Counter('chat_duplicate_messages');
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

function buildMessagePayload(messageId, roomId, targetBytes) {
    const payload = {
        user_id: `benchmark-user-${String(__VU).padStart(4, '0')}`,
        room_id: roomId,
        content: '',
        trace_id: messageId,
        message_id: messageId,
        client_send_ts: Date.now(),
    };

    if (!targetBytes || targetBytes <= 0) {
        payload.content = `Load check from VU ${__VU}`;
        return payload;
    }

    payload.content = 'x';
    let encoded = JSON.stringify(payload);
    if (encoded.length < targetBytes) {
        payload.content = 'x'.repeat(targetBytes - encoded.length + 1);
        encoded = JSON.stringify(payload);
    }
    while (encoded.length > targetBytes && payload.content.length > 1) {
        payload.content = payload.content.slice(0, -1);
        encoded = JSON.stringify(payload);
    }

    return payload;
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
    const targetMessageBytes = parseIntEnv('TARGET_MESSAGE_BYTES', 0);
    const roomId = __ENV.ROOM_ID || 'benchmark-room';
    const logSampleMessages = __ENV.LOG_SAMPLE_MESSAGES === 'true' && __VU === 1;
    const urlString = __ENV.WS_URLS || __ENV.WS_URL || 'ws://localhost:8080/ws';
    const urls = urlString
        .split(',')
        .map((u) => u.trim())
        .filter((u) => u.length > 0);
    const url = pickEndpoint(urls);
    
    const res = ws.connect(url, {}, function (socket) {
        const pendingMessages = {};
        const seenMessageIds = {};
        let sequence = 0;
        let loggedSends = 0;
        let loggedReceives = 0;

        socket.on('open', () => {
            socket.setInterval(() => {
                sequence += 1;
                const messageId = `vu-${__VU}-msg-${sequence}-${Date.now()}`;
                const payload = buildMessagePayload(messageId, roomId, targetMessageBytes);
                pendingMessages[messageId] = payload.client_send_ts;
                socket.send(JSON.stringify(payload));
                messagesSent.add(1);

                if (logSampleMessages && loggedSends < 5) {
                    console.log(`[send] message_id=${messageId} client_send_ts=${payload.client_send_ts}`);
                    loggedSends += 1;
                }
            }, messageIntervalMs);
        });

        socket.on('message', (data) => {
            const msg = JSON.parse(data);
            if (!msg.message_id || !(msg.message_id in pendingMessages)) {
                return;
            }

            const receiveTs = Date.now();
            if (seenMessageIds[msg.message_id]) {
                duplicateMessages.add(1);
                if (logSampleMessages) {
                    console.log(`[dup] message_id=${msg.message_id} client_receive_ts=${receiveTs}`);
                }
                return;
            }

            seenMessageIds[msg.message_id] = true;
            const latency = receiveTs - pendingMessages[msg.message_id];
            delete pendingMessages[msg.message_id];

            messageLatency.add(latency);
            messagesReceived.add(1);

            if (logSampleMessages && loggedReceives < 5) {
                console.log(
                    `[recv] message_id=${msg.message_id} client_receive_ts=${receiveTs} `
                    + `server_receive_ts=${msg.server_receive_ts || 'n/a'} `
                    + `server_broadcast_ts=${msg.server_broadcast_ts || msg.timestamp || 'n/a'} `
                    + `e2e_latency_ms=${latency}`
                );
                loggedReceives += 1;
            }
        });

        socket.setTimeout(() => {
            socket.close();
        }, 600000);
    });

    check(res, { 'status is 101': (r) => r && r.status === 101 });
}
