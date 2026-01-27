# Orion SRE - Homelab sur Raspberry Pi 5

---

## What This Is

Orion SRE est un écosystème numérique complet qui vit chez toi et interagit avec ta vraie vie, pas juste un projet infra sur GitHub. C'est un homelab opérel comme une production SRE : métriques, logs, alertes, auto-remédiation, workflows de maintenance, et un agent LLM local "Orion" qui analyse métriques, logs et événements, résume les incidents et propose des actions concrètes.

Au centre, deux Raspberry Pi 5 et un HDD de 4 To qui hébergent ton homelab : stockage de tes fichiers perso, photos, backups, bibliothèques médias type Netflix maison (Jellyfin + Arr stack), apps du quotidien (recettes, gestion des courses, e‑books, audiobooks, etc.). Tout ça est monitoré et sécurisé comme une prod SRE, avec un agent LLM local branché sur ces données et sur ta domotique.

---

## Core Value

**L'assistant SRE domestique qui comprend ton système et t'aide à le gérer.**

Concrètement, tu peux parler à Orion comme à un opérateur SRE/domotique : "Qu'est‑ce qui a tourné cette nuit ?", "Pourquoi le Pi 2 swap ?", "Qu'est‑ce qui consomme le plus d'espace sur le HDD ?", "Qu'est‑ce que j'ai à regarder ce soir sur le serveur média ?", et il répond en fouillant dans les métriques, les logs, les bases et les apps de ton homelab.

Orion devient un hub central pour tes usages réels : il gère la sauvegarde de tes souvenirs (photos, docs), te sert tes séries, automatise des tâches chiantes (courses, to‑do, rappels), surveille la santé de ton infra, t'aide à tester de nouvelles stacks, et documente tout ça d'une manière intelligible, comme si ta maison et tes serveurs avaient une conscience technique et pouvaient te briefier chaque jour.

---

## Hardware

### Infrastructure Actuelle

| Matériel | Rôle | Specs | Notes |
|-----------|------|-------|-------|
| **orion-cortex** | Compute & Media & Entry point | Raspberry Pi 5, 16GB RAM |
| **orion-synapse** | SRE & Support & Résilience | Raspberry Pi 5, 8GB RAM |
| **HDD** | Stockage principal | 4 To, USB 3.0 (connecté à cortex) |

### Réseau

- **LAN** : Gigabit Ethernet
- **Tailscale** : Configuré sur les 2 Pi (accès privé)
- **Cloudflare Tunnel** : cloudflared déjà ouvert (exposition sécurisée)
- **VPN** : AirVPN (WireGuard/OpenVPN) - pour qBittorrent

### DNS

- **Domaine** : jarvis-hub.tech (Cloudflare-managé)
- **DNS wildcard** : `*.jarvis-hub.tech` disponible

---

## Constraints

### Budget & Temps

- **Budget cloud backup** : V1 = 0 €/mois (disque externe + éventuel espace gratuit chiffré)
- **Futur** : 5-10 €/mois si besoin se justifie (Backblaze B2, Wasabi)
- **Temps** : Base "set and forget" pour backups/monitoring/alerting/maintenance + zone "bidouille" pour nouveaux services/Orion LLM

### HADOPI (Anti-contrefaçon)

- **CRITIQUE** : qBittorrent DOIT passer par VPN AirVPN avec killswitch
- **Killswitch** : Bloque tout traffic si VPN tombe (iptables)
- **Configuration** : Gluetun + qBittorrent sur orion-cortex
- **Objectif** : Éviter toute fuite d'IP même en cas de panne VPN

### Performance & Optimisation

- **Architecture** : Docker Compose distribué (V1 simple, évolutif vers K3s V2/V3)
- **Monitoring** : Stack légère mais complète (Prometheus, Grafana, Loki, Alertmanager)
- **LLM** : CPU-only V1 (Phi-3 Mini 4-bit), GPU/coral possible V2
- **Media** : Direct Play uniquement (pas de hardware transcoding sur Pi 5)

### Sécurité

- **Accès distant** : Tailscale pour privé, Cloudflare Tunnel pour public
- **SSH** : Clés Ed25519 uniquement, port 2222, password désactivé
- **Exposition** : Minimum de services, derrière Caddy + Authelia
- **Container** : Rootless Docker (si possible), seccomp profiles, resource limits

---

## Requirements

### Validated

(None yet — greenfield project, ship to validate)

### Active

#### Sauvegarde & Récupération

- [ ] **BACKUP-01** : Sauvegarde quotidienne configs + DB + docs (quotidien)
- [ ] **BACKUP-02** : Sauvegarde hebdo/bi-hebdo gros volumes médias (incrémental)
- [ ] **BACKUP-03** : Snapshots locaux réguliers sur HDD + backup offsite léger
- [ ] **BACKUP-04** : Backup configs système (Pi, Ansible, .env, docker-compose)
- [ ] **BACKUP-05** : Backup volumes Docker critiques (DB, apps Orion/SRE)
- [ ] **BACKUP-06** : Test de restauration mensuel automatisé
- [ ] **BACKUP-07** : Vérification intégrité backups (hash/restore test)

#### Monitoring & Observabilité

- [ ] **MON-01** : Stack métriques (Prometheus + Node Exporter) sur cortex + synapse
- [ ] **MON-02** : Stack logs (Loki + Promtail) centralisés
- [ ] **MON-03** : Alertes temps réel (warning + critical) vers Telegram bot Orion
- [ ] **MON-04** : Dashboards Grafana (infra globale, stockage & HDD, services clés)
- [ ] **MON-05** : Monitoring conteneurs (CPU, RAM, disque, réseau)
- [ ] **MON-06** : Monitoring santé HDD (SMART check, utilisation espace)

#### Media Center

- [ ] **MEDIA-01** : Jellyfin opérationnel avec bibliothèques médias
- [ ] **MEDIA-02** : Arr stack fonctionnel (Radarr films, Sonarr séries, Prowlarr indexeurs)
- [ ] **MEDIA-03** : Intégration monitoring (dispo jellyfin, espace disque)
- [ ] **MEDIA-04** : Auto-import médias depuis downloads (Arr stack → Jellyfin)
- [ ] **MEDIA-05** : Stockage sur HDD 4 To (garder 10-20% marge libre)

#### qBittorrent + VPN

- [ ] **TORR-01** : qBittorrent derrière AirVPN (WireGuard) sur orion-cortex
- [ ] **TORR-02** : Killswitch actif (bloque traffic si VPN down)
- [ ] **TORR-03** : Port forwarded correct depuis VPN (pour inbound peers)
- [ ] **TORR-04** : Monitoring qBittorrent (speed, active torrents, erreurs)

#### Reverse Proxy & Exposition

- [ ] **PROXY-01** : Caddy comme reverse proxy local (wildcard `*.jarvis-hub.tech`)
- [ ] **PROXY-02** : Certificats TLS via Cloudflare DNS challenge
- [ ] **PROXY-03** : Cloudflare Tunnel pour services publics (Jellyfin, Grafana, Orion)
- [ ] **PROXY-04** : Authelia configuré (2FA/SSO)
- [ ] **PROXY-05** : Services profonds accessibles UNIQUEMENT via Tailscale

#### Workflows Maintenance

- [ ] **MAINT-01** : Rotation/purge logs et fichiers temporaires automatisée
- [ ] **MAINT-02** : Vérification sauvegardes (test restauration ou hash)
- [ ] **MAINT-03** : Check SMART / santé HDD
- [ ] **MAINT-04** : Restart automatique conteneurs/services down
- [ ] **MAINT-05** : Nettoyage téléchargements médias une fois importés dans Jellyfin
- [ ] **MAINT-06** : Scan périodique sécurité basique (ports ouverts, services exposés)
- [ ] **MAINT-07** : Rapport hebdo Orion ("état du système", incidents, actions, recommandations)

#### Agent LLM Orion

- [ ] **LLM-01** : Ollama installé sur orion-cortex (Phi-3 Mini 4-bit, CPU-only)
- [ ] **LLM-02** : Web UI locale accessible
- [ ] **LLM-03** : Bot Telegram Orion répondant aux questions système
- [ ] **LLM-04** : Query métriques Prometheus (via HTTP API)
- [ ] **LLM-05** : Query logs Loki (via HTTP API)
- [ ] **LLM-06** : Intégration progressive avec Home Assistant (futur)

#### Sécurité

- [ ] **SEC-01** : SSH hardening (clés Ed25519, port 2222, password off)
- [ ] **SEC-02** : UFW + Fail2ban/SSHGuard actifs
- [ ] **SEC-03** : Tailscale configuré sur les 2 Pi
- [ ] **SEC-04** : Rootless Docker activé (si possible)
- [ ] **SEC-05** : Seccomp profiles + resource limits sur conteneurs
- [ ] **SEC-06** : Authelia 2FA configuré pour services exposés

#### Infrastructure

- [ ] **INFRA-01** : HDD formaté ext4, monté sur orion-cortex
- [ ] **INFRA-02** : Stockage NFS exporté vers orion-synapse
- [ ] **INFRA-03** : Docker Compose distribué (cortex.yml + synapse.yml)
- [ ] **INFRA-04** : Réseau interne isolé (docker bridge networks)
- [ ] **INFRA-05** : Git pour configs (etckeeper/Ansible)

### Out of Scope

- [**K3s/Kubernetes**] — V2/V3 seulement, V1 utilise Docker Compose pour simplicité
- [**Hardware transcoding**] — Pi 5 n'a pas d'encodeurs H.264 hardware, utilse Direct Play
- [**Transcoding 4K**] — Pas faisable sur Pi 5 CPU-only
- [**Exposition DB directe**] — Bases de données accessibles UNIQUEMENT via VPN/Tailscale
- [**Services publiques sans auth**] — Tout service exposé nécessite Authelia + 2FA
- [**Home Assistant**] — Pour V2 (futur), pas V1

---

## Architecture

### Distribution des Services

#### orion-cortex (16GB RAM) — "Compute & Media & Entry Point"

**Rôle :** Nœud principal pour charges lourdes, média, LLM, et exposition

**Services :**
- **Media Center** : Jellyfin + Arr stack (Radarr, Sonarr, Prowlarr)
- **qBittorrent** : Derrière AirVPN (Gluetun + killswitch)
- **LLM Orion** : Ollama + Phi-3 Mini + Web UI + Bot Telegram
- **Reverse Proxy** : Caddy (wildcard TLS `*.jarvis-hub.tech`)
- **Cloudflare Tunnel** : cloudflared → Caddy (services publics)
- **Stockage** : HDD 4 To (ext4) monté localement

**Pourquoi 16GB ?**
- Media center (Jellyfin) : ~3-4GB
- LLM Orion (Phi-3) : ~3-4GB
- Arr stack : ~1-2GB
- Buffer système + conteneurs : ~4-6GB
- Transcoding/mise en cache : ~2-3GB

#### orion-synapse (8GB RAM) — "SRE & Support & Résilience"

**Rôle :** Nœud support/contrôle, survit à panne de cortex

**Services :**
- **Monitoring contrôle** : Prometheus + Alertmanager (scrape cortex + synapse)
- **Logs** : Loki (stockage logs sur HDD 4 To) + Promtail local
- **Backups** : Restic/Borg clients (offsite + local)
- **Agents logs** : Promtail (synapse) + éventuellement Uptime Kuma
- **Jobs récurrents** : docker-volume-backup, maintenance scripts
- **Services légers mais critiques** : Pi-hole/DNS local, Uptime Kuma
- **Stockage** : Import NFS depuis cortex (nécessaire)

**Pourquoi 8GB ?**
- Prometheus + Alertmanager : ~500MB
- Loki : ~500MB
- Agents backup : ~500MB
- Pi-hole : ~200MB
- Uptime Kuma : ~200MB
- Buffer système : ~4-6GB

### Flux de Données

```
[Internet]
        |
        | Cloudflare Tunnel (HTTPS)
        ↓
    orion-cortex (16GB)
    ┌──────────────────────────────────┐
    │                              │
    │  ┌────────────────────────┐ │
    │  │ Caddy (Proxy Local)   │ │ NFS (export)
    │  │                      │ ├──────────→
    │  │ ┌─── Jellyfin     │ │
    │  │ │ ──── Radarr      │ │
    │  │ │ ──── Sonarr      │ │
    │  │ │ ──── Prowlarr    │ │ orion-synapse (8GB)
    │  │ │                  │ │ ┌─────────────────────────┐
    │  │ └─── Orion LLM     │ │ │                        │
    │  │                      │ │ │  ┌────────────────┐ │
    │  └─── qBittorrent      │ │ │  │ Prometheus     │ │
    │     │                   │ │ │  │ Alertmanager  │ │
    │     │ AirVPN (WireGuard)│ │ │  └────────────────┘ │
    │  ┌─── Gluetun (VPN)    │ │ │                       │
    │  └────────────────────┘ │ │  ┌────────────────┐ │
    │                          │ │  │ Loki           │ │
    │  HDD 4 To (ext4)       │ │  └────────────────┘ │
    └──────────────────────────┘ │  ┌────────────────┐ │
                                │  │ Pi-hole        │ │
                                │  └────────────────┘ │
                                │  ┌────────────────┐ │
                                │  │ Uptime Kuma   │ │
                                │  └────────────────┘ │
                                └─────────────────────────┘
```

### Orchestration

**V1 : Docker Compose Distribué**

- **cortex.yml** : Services orion-cortex (Media, LLM, qBittorrent, Caddy, cloudflared)
- **synapse.yml** : Services orion-synapse (Monitoring, Backups, Pi-hole, Uptime Kuma)
- **Déploiement** : Manuel (docker compose up -d) ou scripts bash/Ansible simples
- **Optionnel** : Quelques scripts de sync (NFS mounts, configs partagées)

**V2/V3 : K3s**

- **GitOps** : ArgoCD pour déploiements automatiques
- **Auto-healing** : Pods redémarrés automatiquement
- **Advanced networking** : Services mesh, ingress controllers
- **Évolutif** : Plus facile d'ajouter des nodes/services

---

## Stack Technique

### Compute & Orchestration

| Composant | Version | Notes |
|-----------|--------|-------|
| **Docker** | 27.x | Containerisation |
| **Docker Compose** | 2.x | Orchestration V1 |
| **Ansible** | 2.x (optionnel) | Déploiement configs |
| **etckeeper** | 1.x | Git pour /etc |

### Monitoring & Logs

| Composant | Version | ARM64 Support | Ressources |
|-----------|--------|-------------|-------------|
| **Prometheus** | 2.55+ | ✅ natif | ~300MB RAM |
| **Node Exporter** | 1.8+ | ✅ natif | ~50MB RAM |
| **Grafana** | 11.x | ✅ natif | ~300MB RAM |
| **Loki** | 3.0+ | ✅ natif | ~200MB RAM |
| **Promtail** | 3.0+ | ✅ natif | ~50MB RAM |
| **Alertmanager** | 0.27+ | ✅ natif | ~100MB RAM |

### Media Center

| Composant | Version | Image | Notes |
|-----------|--------|-------|-------|
| **Jellyfin** | 10.11+ | lscr.io/linuxserver/jellyfin | Direct Play, pas transcoding |
| **Radarr** | 5.10+ | lscr.io/linuxserver/radarr | Films |
| **Sonarr** | 4.0+ | lscr.io/linuxserver/sonarr | Séries |
| **Prowlarr** | 2.3+ | lscr.io/linuxserver/prowlarr | Indexeurs |

### qBittorrent + VPN

| Composant | Version | Image | Notes |
|-----------|--------|-------|-------|
| **qBittorrent** | 5.1+ | lscr.io/linuxserver/qbittorrent | network_mode: gluetun |
| **Gluetun** | 3.x | qmcgaw/gluetun | AirVPN support, killswitch |

### Reverse Proxy & Sécurité

| Composant | Version | ARM64 Support | Notes |
|-----------|--------|-------------|-------|
| **Caddy** | 2.8+ | ✅ natif | Auto HTTPS, DNS Cloudflare |
| **Authelia** | 4.x | ✅ natif | 2FA/SSO |
| **cloudflared** | 2025.x+ | ✅ natif | Cloudflare Tunnel |
| **Tailscale** | 1.x | ✅ natif | WireGuard, SSO/MFA |

### Backups

| Composant | Version | ARM64 Support | Usage |
|-----------|--------|-------------|--------|
| **Restic** | 0.18+ | ✅ natif | Offsite/cloud |
| **Borg** | 1.4+ | ✅ natif | Local + offsite |
| **docker-volume-backup** | 2.47+ | ✅ natif | Volumes Docker |
| **etckeeper** | 1.x | ✅ natif | Configs système |

### LLM Orion

| Composant | Version | ARM64 Support | Notes |
|-----------|--------|-------------|-------|
| **Ollama** | 0.5+ | ✅ Pi 5 supporté | Inference engine |
| **Phi-3 Mini** | 3.8B 4-bit | ✅ optimal Pi 5 | ~15-20 tokens/sec |
| **Python/Node** | 3.x+ | ✅ natif | Bot Telegram, Web UI |

---

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| **Architecture V1** | Docker Compose distribué plutôt que K3s | Simple, lisible, évolutif. K3s pour V2/V3 quand GitOps nécessaire |
| **Répartition 16/8GB** | Cortex 16GB pour charges lourdes, Synapse 8GB support | Exploite la RAM disponible tout en gardant un nœud résilient |
| **qBittorrent + Gluetun** | Killswitch via network_mode plutôt que image intégrée | Flexibilité (changer provider sans toucher qBittorrent), killswitch propre |
| **Cloudflare Tunnel** | Seulement pour services publics (Jellyfin, Grafana, Orion) | Minimise exposition, reste accessible via Tailscale |
| **Caddy comme reverse proxy** | Local, derrière Cloudflare Tunnel | Simple, auto HTTPS, DNS wildcard supporté |
| **Direct Play Media** | Pi 5 n'a pas d'encodeurs hardware | Seule stratégie viable, accepter limitation |
| **LLM Phi-3 Mini V1** | CPU-only pour démarrer vite | ~15-20 tokens/sec acceptable pour V1. GPU/Coral V2 |
| **Restic pour offsite** | ARM64 natif, encryption built-in | Backblaze B2 compatible, déduplication efficace |
| **NFS pour stockage** | Partage HDD entre les 2 Pi | Cortex exporte, Synapse import, Docker mountable |
| **Tailscale pour privé** | SSO/MFA supporté, WireGuard standard | Accès privé, plus simple que VPN manuel |

---

## Success Criteria

### V1 Success

- [ ] Sauvegardes quotidiennes en place et vérifiées mensuellement
- [ ] Monitoring complet (métriques + logs + alertes Telegram) opérationnel
- [ ] Media center fonctionnel (Jellyfin + Arr stack) avec Direct Play
- [ ] qBittorrent derrière AirVPN avec killswitch actif
- [ ] Orion LLM capable de répondre aux questions basiques (métriques, logs)
- [ ] Reverse proxy Caddy avec TLS wildcard sur `*.jarvis-hub.tech`
- [ ] Services critiques accessibles via Tailscale, publics via Cloudflare Tunnel
- [ ] Workflows maintenance auto (log rotation, cleanup, health checks) actifs
- [ ] Documentation complète (docker-compose files, scripts, procédures)

---

## Future Considerations

### V2 : Domotique Intégration

- **Home Assistant** : Intégration avec Orion LLM
- **Sensors** : Orion peut analyser capteurs, proposer actions ("réduire chauffage", "déplacer service")
- **Automation** : Workflows automatisés basés sur événements domotique

### V3 : Advanced SRE

- **K3s** : Migration vers Kubernetes pour GitOps + auto-healing avancé
- **Service Mesh** : Cilium/Linkerd pour observabilité deep
- **Operators** : PostgreSQL Operator, Backup Operator, etc.
- **Chaos Engineering** : Tests de résilience automatiques

### Scaling

- **3ème Pi** : Pour charges additionnelles (Home Assistant, autres apps)
- **NVMe via PCIe** : Pour performances stockage (cache LLM, base de données)
- **Coral USB / PCIe** : Pour modèles LLM plus gros (V2+)

---

## Context & References

### Inspiration

- [What is a Homelab](https://fr.simeononsecurity.com/articles/what-is-a-homelab-and-should-you-have-one/) — Vision générale
- [Practical Homelab Uses](https://www.reddit.com/r/homelab/comments/1oj3md5/new_to_homelab_looking_for_practical_everyday_use/) — Idées d'apps
- [SRE Principles](https://www.dotcom-monitor.com/blog/sre-principles-the-7-fundamental-rules/) — Principles SRE
- [Home Assistant + LLM](https://actualite-domotique.fr/home-assistant-llm-local/) — Intégration domotique
- [LLM Assistant Productivity](https://korben.info/khoj-assistant-ia-prive-productivite.html) — Bot IA utile

### Stack & Performance

- [Raspberry Pi 5 Ollama Guide](https://fleetstack.io/blog/ollama-raspberry-pi-guide) — Benchmarks LLM
- [LLM Performance Pi 5](https://www.stratosphereips.org/blog/2025/6/5/how-well-do-llms-perform-on-a-raspberry-pi-5) — Comparatif modèles
- [Jellyfin Hardware Pi 5](https://pidiylab.com/jellyfin-on-raspberry-pi-5-hardware-decode-network-shares-setup/) — Hardware decode
- [NFS Performance Pi](https://tasnimzotder.com/posts/setting-up-samba-on-raspberry-pi) — 100 MB/s benchmark

### Architecture & Best Practices

- [Docker to K3s Evolution](https://terminalbytes.com/kubernetes-at-home-from-docker-compose-to-k3s/) — Migration path
- [Over-engineered Home Lab](https://fernandocejas.com/blog/engineering/2023-01-06-over-engineered-home-lab-docker-kubernetes/) — Pourquoi Docker Compose V1
- [K3s Homelab Awesome](https://www.reddit.com/r/selfhosted/comments/1gqkq8y/k3s_is_awesome_for_your_home_servers_i_used_to_use/) — K3s avantages
- [Arr Stack Homelab](https://yanpaingoo.dev/posts/arrstack-homelab/) — Arr setup complet

### Sécurité & VPN

- [Gluetun + qBittorrent](https://drfrankenstein.co.uk/qbittorrent-with-gluetun-vpn-in-container-manager-on-a-synology-nas/) — VPN + killswitch
- [AirVPN Gluetun Config](https://mystupidnotes.com/how-to-configure-gluetun-with-airvpn-qbittorrent-and-sabnzbd-in-docker-compose/) — Configuration spécifique
- [qBittorrent WireGuard Docker](https://tongkl.com/wireguard-qbittorrent-ipv6/) — IPv6 support

### Cloudflare & DNS

- [Cloudflare Tunnel Docker](https://joelparkinson.me/self-hosting-with-cloudflare-tunnels-docker-compose/) — Setup guide
- [Caddy Cloudflare DNS](https://www.virtualizationhowto.com/2025/09/caddy-reverse-proxy-in-2025-the-simplest-docker-setup-for-your-home-lab/) — Wildcard TLS

### Monitoring

- [GRC Logging Prometheus](https://www.brianhaman.com/grc-blog/raspberry-pi-monitoring-grc-prometheus-grafana/) — Monitoring complet
- [Monitoring Homelab](https://stormagic.com/company/blog/building-monitoring-homelab-with-stormagic/) — Stack monitoring

---

*Last updated: 2026-01-27 after initialization*
