import { base_options } from './base.js';

export const scenarios = {
    short: {
        executor: 'constant-vus',
        vus: 10,
        duration: '30s',
    },
    standard: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '1m', target: 1000 },
            { duration: '3m', target: 1000 },
            { duration: '1m', target: 0 },
        ],
    },
    spike: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '10s', target: 100 },
            { duration: '30s', target: 10000 },
            { duration: '1m', target: 10000 },
            { duration: '30s', target: 0 },
        ],
    },
};
