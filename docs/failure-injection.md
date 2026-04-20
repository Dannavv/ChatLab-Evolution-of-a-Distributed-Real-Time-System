# ChatLab Failure Injection

Failure injection is now part of the learning loop, especially for Labs 05 through 11.

## Supported Experiments

The shared control script supports three basic actions:
- `kill`: stop a service to simulate a crash
- `delay`: inject latency and jitter into a running container network namespace
- `heal`: remove injected delay and restart a stopped service

## Examples

Kill a worker:

```bash
python3 scripts/chatlab.py fail lab-06-chaos-and-resilience kill chat-worker
```

Inject latency into Redis:

```bash
python3 scripts/chatlab.py fail lab-06-chaos-and-resilience delay redis --latency-ms 300 --jitter-ms 50
```

Heal the service:

```bash
python3 scripts/chatlab.py fail lab-06-chaos-and-resilience heal redis
```

## Suggested Resilience Drills

- Lab 05: delay `redis` or stop `chat-worker` and observe backlog growth
- Lab 06: kill `chat-worker` and verify circuit-breaker or dead-letter behavior
- Lab 08: delay the regional bridge service and observe cross-region convergence lag
- Lab 10: stop `history-service` and confirm the gateway still serves the real-time path
- Lab 11: inject delay into `redis` or `db` and verify the blueprint dashboards surface the degradation quickly

## Notes

- Delay injection uses a `netshoot` helper container with `tc`.
- These experiments are intended for local learning environments, not shared production hosts.
- After each experiment, rebuild the comparison report so the failure posture is captured alongside performance artifacts.
