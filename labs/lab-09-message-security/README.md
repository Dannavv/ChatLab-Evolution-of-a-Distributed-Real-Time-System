[Home](../../README.md) | [Next (Lab 10)](../lab-10-microservices-migration/README.md)

# Lab 09: Message Security and Trust
## Encrypted Envelopes, Key Rotation, and Replay Defense

Lab 09 secures the chat mesh itself. Messages are encrypted before they leave the sending node, signed with an HMAC tied to the room key version, verified on every node, and rejected when the signature, replay history, or rate controls fail.

## Architecture

```text
WebSocket clients
  -> chat-server-1 / chat-server-2
  -> AES-GCM encrypted message envelope
  -> HMAC signature verification
  -> Redis pub/sub mesh
  -> local decrypt and broadcast
```

## Security model
- AES-GCM encryption for message payloads
- HMAC signatures for integrity and trust
- Shared master secret to derive room keys by version
- Key rotation broadcast across the mesh
- Replay protection using message ID tracking
- Per-user token bucket rate limiting
- Tamper detection for invalid signatures

## Services
- chat-server-1: secure ingress and verification node
- chat-server-2: second secure node to validate cross-node trust
- redis: message mesh and key rotation bus
- prometheus: metrics scrape endpoint

## Metrics to watch
- chat_secure_messages_total
- chat_secure_rejected_total
- chat_secure_signature_rejected_total
- chat_secure_rate_limited_total
- chat_secure_replay_rejected_total
- chat_secure_key_rotations_total
- chat_messages_total
- chat_message_latency_ms

## Run Lab 09

```bash
cd labs/lab-09-message-security
docker-compose up --build -d
```

## Endpoints
- Secure UI 1: http://localhost:8088
- Secure UI 2: http://localhost:8089
- Prometheus: http://localhost:9095

## Try this sequence
1. Open both secure nodes.
2. Send a normal secure message from one node and confirm it appears on the other.
3. Send a tampered message and confirm it is rejected.
4. Rotate the key and confirm both nodes update the displayed key version.
5. Burst messages from one user and watch the rate limiter and rejection counters.

## Next Lab
[Lab 10: Microservices Migration](../lab-10-microservices-migration/README.md)
