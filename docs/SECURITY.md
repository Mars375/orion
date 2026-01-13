# ORION — Security Model

## Zero-Trust

- Tailscale mandatory
- No open inbound ports
- Explicit ACLs
- Identity per node

## Approvals

- Telegram mandatory for risky actions
- Time-limited approvals
- Full audit trail

## Secrets

- Never committed
- Stored in env files
- Backed up encrypted
