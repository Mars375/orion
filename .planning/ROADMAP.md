# Orion SRE - Roadmap

---

## Overview

**[8 phases]** | **[43 requirements]** | All V1 requirements mapped ✓

This roadmap organizes the implementation of Orion SRE V1 into 8 phases, progressing from basic infrastructure to advanced SRE features with the Orion LLM assistant.

---

## Phases

### Phase 1: Foundation - Infrastructure de Base

**Goal** : Établir l'infrastructure de base (stockage, réseau, Docker)

**Requirements** : INFRA-01, INFRA-02, INFRA-03, INFRA-04, INFRA-05

**Success Criteria** :
1. HDD 4 To formaté ext4 et monté sur orion-cortex
2. Export NFS fonctionnel entre cortex et synapse
3. Docker + Docker Compose installés sur les 2 Pi
4. Réseau interne isolé (docker bridge networks)
5. Git initialisé pour configs (etckeeper/Ansible)

**Services Deployed** :
- None (infrastructure only)

---

### Phase 2: Sécurité & Accès

**Goal** : Sécuriser l'accès (SSH, VPN, Tailscale, Firewall)

**Requirements** : SEC-01, SEC-02, SEC-03

**Success Criteria** :
1. SSH accessible uniquement via clés Ed25519 (port 2222)
2. UFW + Fail2ban/SSHGuard actifs
3. Tailscale connecté sur les 2 Pi avec SSO
4. Accès distant fonctionnel via Tailscale

**Services Deployed** :
- UFW (firewall)
- Fail2ban/SSHGuard
- Tailscale

---

### Phase 3: Reverse Proxy & Exposition

**Goal** : Mettre en place reverse proxy Caddy et Cloudflare Tunnel

**Requirements** : PROXY-01, PROXY-02, PROXY-03

**Success Criteria** :
1. Caddy opérationnel comme reverse proxy local
2. Certificats TLS wildcard `*.jarvis-hub.tech` via Cloudflare DNS
3. Cloudflare Tunnel configuré vers Caddy
4. Services derrière proxy testés (HTTP/HTTPS)

**Services Deployed** :
- Caddy
- cloudflared
- Cloudflare DNS

---

### Phase 4: Monitoring & Observabilité - Base

**Goal** : Mettre en place la stack de monitoring de base

**Requirements** : MON-01, MON-02, MON-04, MON-05, MON-06

**Success Criteria** :
1. Prometheus scrape cortex et synapse
2. Loki collecte logs depuis les 2 Pi (Promtail)
3. Dashboards Grafana opérationnels
4. Alertes configurées pour CPU/RAM/disque
5. Monitoring santé HDD (SMART check)

**Services Deployed** :
- Prometheus
- Node Exporter
- Loki
- Promtail
- Grafana

---

### Phase 5: Alerting & Notification

**Goal** : Configurer les alertes et notification Telegram

**Requirements** : MON-03, SEC-06

**Success Criteria** :
1. Alertmanager configuré avec routing Telegram
2. Alertes temps réel (warning + critical) actives
3. Authelia configuré avec 2FA pour services publics
4. Test d'alertes fonctionnel (trigger alert → Telegram)

**Services Deployed** :
- Alertmanager
- Authelia
- Telegram bot Orion Alerts

---

### Phase 6: Sauvegardes & Récupération

**Goal** : Implémenter la stratégie de sauvegarde complète

**Requirements** : BACKUP-01, BACKUP-02, BACKUP-03, BACKUP-04, BACKUP-05, BACKUP-06, BACKUP-07

**Success Criteria** :
1. Backup quotidien configs + DB + docs automatisé
2. Backup hebdo/bi-hebdo volumes médias (incrémental)
3. Snapshots locaux réguliers sur HDD
4. Backup offsite vers Backblaze B2 fonctionnel
5. Test de restauration mensuel automatisé
6. Vérification intégrité backups (hash) en place

**Services Deployed** :
- Restic/Borg
- docker-volume-backup
- etckeeper
- Scripts de backup/restore

---

### Phase 7: Media Center

**Goal** : Déployer et configurer Jellyfin + Arr stack

**Requirements** : MEDIA-01, MEDIA-02, MEDIA-03, MEDIA-04, MEDIA-05, TORR-01, TORR-02, TORR-03, TORR-04

**Success Criteria** :
1. Jellyfin opérationnel avec bibliothèques médias
2. Arr stack fonctionnel (Radarr, Sonarr, Prowlarr)
3. qBittorrent derrière AirVPN avec killswitch
4. Intégration monitoring (dispo Jellyfin, espace disque)
5. Auto-import médias depuis downloads (Arr → Jellyfin)
6. Stockage sur HDD avec 10-20% marge libre

**Services Deployed** :
- Jellyfin
- Radarr
- Sonarr
- Prowlarr
- qBittorrent
- Gluetun (AirVPN)

---

### Phase 8: Agent LLM Orion

**Goal** : Déployer l'assistant LLM local Orion

**Requirements** : LLM-01, LLM-02, LLM-03, LLM-04, LLM-05, LLM-06

**Success Criteria** :
1. Ollama installé avec Phi-3 Mini 4-bit sur orion-cortex
2. Web UI Orion accessible
3. Bot Telegram Orion répondant aux questions
4. Query métriques Prometheus fonctionnelle
5. Query logs Loki fonctionnelle
6. Rapport hebdo généré ("état système", incidents, actions)

**Services Deployed** :
- Ollama
- Phi-3 Mini 4-bit model
- Orion Web UI
- Orion Telegram Bot
- Intégrations Prometheus/Loki

---

## Requirements Traceability

### Phase 1: Foundation
- INFRA-01 → Phase 1
- INFRA-02 → Phase 1
- INFRA-03 → Phase 1
- INFRA-04 → Phase 1
- INFRA-05 → Phase 1

### Phase 2: Sécurité
- SEC-01 → Phase 2
- SEC-02 → Phase 2
- SEC-03 → Phase 2

### Phase 3: Reverse Proxy
- PROXY-01 → Phase 3
- PROXY-02 → Phase 3
- PROXY-03 → Phase 3

### Phase 4: Monitoring Base
- MON-01 → Phase 4
- MON-02 → Phase 4
- MON-04 → Phase 4
- MON-05 → Phase 4
- MON-06 → Phase 4

### Phase 5: Alerting
- MON-03 → Phase 5
- SEC-06 → Phase 5

### Phase 6: Sauvegardes
- BACKUP-01 → Phase 6
- BACKUP-02 → Phase 6
- BACKUP-03 → Phase 6
- BACKUP-04 → Phase 6
- BACKUP-05 → Phase 6
- BACKUP-06 → Phase 6
- BACKUP-07 → Phase 6

### Phase 7: Media Center
- MEDIA-01 → Phase 7
- MEDIA-02 → Phase 7
- MEDIA-03 → Phase 7
- MEDIA-04 → Phase 7
- MEDIA-05 → Phase 7
- TORR-01 → Phase 7
- TORR-02 → Phase 7
- TORR-03 → Phase 7
- TORR-04 → Phase 7

### Phase 8: Agent LLM
- LLM-01 → Phase 8
- LLM-02 → Phase 8
- LLM-03 → Phase 8
- LLM-04 → Phase 8
- LLM-05 → Phase 8
- LLM-06 → Phase 8

**Total Requirements Mapped**: 43/43 (100%)

---

## Dependencies

```
Phase 2 (Sécurité)
    ↓ dépend de
Phase 1 (Infrastructure)

Phase 3 (Reverse Proxy)
    ↓ dépend de
Phase 2 (Sécurité)

Phase 4 (Monitoring Base)
    ↓ dépend de
Phase 3 (Reverse Proxy)

Phase 5 (Alerting)
    ↓ dépend de
Phase 4 (Monitoring Base)

Phase 6 (Sauvegardes)
    ↓ dépend de
Phase 4 (Monitoring Base)

Phase 7 (Media Center)
    ↓ dépend de
Phase 6 (Sauvegardes)

Phase 8 (Agent LLM)
    ↓ dépend de
Phase 4 (Monitoring Base)
    ↓ dépend de
Phase 5 (Alerting)
```

---

## Estimated Timeline

| Phase | Duration | Dependencies |
|--------|-----------|--------------|
| Phase 1: Foundation | 1-2 jours | None |
| Phase 2: Sécurité | 1 jour | Phase 1 |
| Phase 3: Reverse Proxy | 1 jour | Phase 2 |
| Phase 4: Monitoring Base | 1-2 jours | Phase 3 |
| Phase 5: Alerting | 1 jour | Phase 4 |
| Phase 6: Sauvegardes | 2-3 jours | Phase 4 |
| Phase 7: Media Center | 2-3 jours | Phase 6 |
| Phase 8: Agent LLM | 2-3 jours | Phase 4, 5 |

**Total Estimated**: 11-16 jours

---

## Next Steps

### Ready to start ?

**Phase 1: Foundation - Infrastructure de Base**

Run: `/gsd-discuss-phase 1` — Gather context and clarify approach

Or skip discussion and plan directly: `/gsd-plan-phase 1`

---

## Notes

- **Architecture** : Docker Compose distribué (cortex.yml + synapse.yml)
- **Orchestration** : V1 simple, évolutif vers K3s V2/V3
- **Mode** : YOLO (auto-approve) + standard depth
- **Agents** : Research + plan check + verifier activés
- **Git** : .planning/ tracké dans version control

---

*Last updated: 2026-01-27*
