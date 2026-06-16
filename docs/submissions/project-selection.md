# Assignment 2: Project Selection

**Team Name:** LOKK (Group 14)

**Project Name:** Helmsman

---

## Slide Outline (7 slides, 10 minutes)

1. **Title**: Helmsman: an open-source, self-hosted control plane for OpenTelemetry Collector fleets. Team LOKK.
2. **Project Idea**: what it is, plus a one-paragraph primer on OpAMP and OTel Collectors.
3. **The Problem**: config sprawl, no live fleet visibility, risky manual edits, vendor lock-in.
4. **Competitor and Differentiation**: Bindplane (ObservIQ), and how Helmsman is unique.
5. **Who It Helps and How**: target users and how the feature set serves them.
6. **Can an LLM Build It?**: why an LLM could not build this, and an even harder problem we can grow into.
7. **Architecture, Data, and Plan**: two-plane design, data model / ERD, user roles, team split, demo plan.

---

## Project Idea

Helmsman is a self-hostable web application for managing a fleet of OpenTelemetry (OTel) Collectors from one place. It speaks OpAMP (the Open Agent Management Protocol, an open standard from the OpenTelemetry project) to every collector it manages. Through a single web UI an operator can see every collector that has checked in, watch its live health and effective configuration, push a new configuration to one or many collectors, keep a full version history of every config, and roll back to a previous version with one click.

In short: it is what you reach for when you run telemetry agents on more than a handful of hosts and you are tired of SSHing into each one to edit a YAML file by hand.

Two short definitions for the presentation:

- **OTel Collector** - the standard agent that receives, processes, and exports logs, metrics, and traces. Teams run many of them across VMs and Kubernetes.
- **OpAMP** - the open protocol that lets a central server remotely monitor and configure those agents over a persistent connection, instead of managing each one locally.

---

## What problem(s) does your software hope to solve?

Teams that adopt OpenTelemetry quickly end up running dozens or hundreds of collectors across virtual machines and Kubernetes clusters. That creates four concrete problems:

1. **Configuration sprawl** - each collector has its own YAML config edited by hand on the host. There is no single source of truth and drift is inevitable.
2. **No live visibility** - there is no easy way to answer "is every collector healthy, and what config is each one actually running right now?" without logging into each machine.
3. **Risky changes** - a bad config edit can silently break telemetry collection, and there is no built-in way to version a change or undo it.
4. **Vendor lock-in and cost** - the managed tools that solve this are commercial, gated behind paid tiers, and pull teams away from open standards.

---

## What is one competing product?

**Bindplane** (by ObservIQ). Bindplane is an agent-management platform built on OpAMP that manages OTel and Fluent agents, offering a UI for fleet inventory, remote configuration, and a visual pipeline builder. It is the closest direct competitor and the clearest reference point for what a mature product in this space looks like.

(Honorable mention as alternatives: Grafana Fleet Management and Dynatrace's collector management. Bindplane is the most direct comparison.)

---

## How will your software be unique from its competitors?

- **Truly open source and self-host-first.** Bindplane's open-source tier has thinned over time, and its most useful capabilities live behind a commercial license. Helmsman is fully OSS with no seat limits, license keys, or feature gates. You clone it, run it, and own it.
- **Built on the open OpAMP standard, not a proprietary agent.** Any OpAMP-compliant collector can connect. No custom forked agent required.
- **Lightweight to deploy.** A small Go control-plane service, a React and FastAPI management app, and a Postgres database. It runs comfortably on a single small host or a homelab, not just a production cluster.
- **OTel-native by default.** The data model and config workflow are built around OpenTelemetry Collector configuration, not retrofitted onto a generic agent abstraction.

---

## Who are the people your app hopes to help?

- **Platform engineering, DevOps, and SRE teams** who operate fleets of OTel Collectors across mixed VM and Kubernetes environments and need central control and visibility.
- **Small and mid-sized organizations** that have adopted OpenTelemetry but cannot justify a commercial pipeline-management SaaS subscription.
- **Open-source and homelab operators** who want a self-hosted, no-cost way to manage telemetry agents without vendor lock-in.

---

## How will the feature set help them?

| Feature | How it helps |
|---|---|
| Agent registry and live health dashboard | One screen answers "which collectors are up, and what is each one running" without touching any host. |
| Remote configuration push | Change a config once in the UI and apply it across the fleet, instead of editing YAML on every machine. |
| Config versioning | Every change is a tracked version with an author and timestamp, giving a single source of truth and an audit trail. |
| One-click rollback | A bad config is undone instantly by re-pushing a known-good prior version, which removes the fear from making changes. |
| Role-based access control | Admins, operators, and viewers get scoped permissions, so not everyone who can look can also change production config. |

The through-line: it converts a manual, error-prone, host-by-host chore into a safe, observable, central workflow.

---

## Can your app be generated simply by prompting an LLM/agent? If so, what features should be added or what more difficult problem can you solve?

**No, not the core of it.** An LLM can scaffold a CRUD dashboard with a database and a login page. It cannot one-shot the part that makes Helmsman real: a correct OpAMP control plane. That requires implementing a stateful, bidirectional protocol over persistent WebSocket connections to a live fleet, reconciling a desired configuration against what each agent is actually running (an eventual-consistency problem), hashing and validating configs, tracking apply success or failure per agent, and supporting atomic rollback under RBAC. Correctness here comes from runtime behavior across many connected agents, not from boilerplate, which is exactly where one-shot generation falls down.

**The harder problem we grow into (future enhancement): AI-assisted configuration.** Once the control plane is solid, we add a feature where an operator describes intent in natural language ("collect host metrics and ship them to our Prometheus, drop debug logs") and the system generates a validated OTel Collector config, shows a safety diff and a plain-English explanation of what will change, and only then offers to roll it out. This flips the LLM question on its head: instead of "could an LLM build this app," the answer becomes "this app uses an LLM as a feature to solve a genuinely hard config-authoring problem," while the distributed-systems core remains the thing we actually engineer.

---

## Architecture and Technical Plan (Slide 7)

### Two control planes plus one database

| Component | Tech | Responsibility |
|---|---|---|
| Agent control plane | Go + `opamp-go` server | Holds live WebSocket connections to collectors, receives health and effective-config status, reconciles desired config to connected agents, writes agent state to Postgres. |
| Management plane | React + FastAPI (UI + user-facing API) | Agent dashboard, config editor, version history, rollback; authentication and RBAC; writes desired config and intent to Postgres. |
| State | PostgreSQL | Single source of truth for agents, configurations, versions, rollouts, users, and audit logs. |
| Managed agents | OTel Collector + OpAMP supervisor | Real collectors pointed at Helmsman, used for the live demo. |

**The control loop (key design idea):** The management plane's FastAPI backend writes *desired state* (a config version targeted at agents) to Postgres. The Go plane *reconciles* it: when a connected agent's effective-config hash differs from the desired one, it pushes the new config over OpAMP and writes back the apply result. This is the same declarative reconciler pattern as a Kubernetes controller, and it lets the two services coordinate through the database instead of brittle direct calls.

### Data model (ERD source, satisfies the database and user-role requirements)

- `users` (id, email, role: admin / operator / viewer) - drives RBAC
- `agents` (instance_uid, hostname, labels, agent_type, version, status, last_seen, effective_config_hash)
- `configurations` (id, name, label_selector, current_version_id)
- `config_versions` (id, configuration_id, version_no, yaml, hash, author_id, created_at) - versioning and rollback
- `rollouts` (id, config_version_id, agent_id, status, applied_at, error) - per-agent push tracking
- `audit_logs` (id, user_id, action, target_type, target_id, detail, created_at) - immutable record of every config change, push, rollback, and role change

Relationships: a `configuration` has many `config_versions`; a `config_version` produces many `rollouts`, one per `agent`; a `user` authors many `config_versions` and generates many `audit_logs`.

### User roles

- **Admin** - manage users, configurations, and rollouts.
- **Operator** - edit and push configurations, trigger rollbacks.
- **Viewer** - read-only access to fleet health and config history.

### Demo plan

Run two or three OTel Collectors locally via the OpAMP supervisor pointed at Helmsman, show them appear in the dashboard, push a config change from the UI, watch the collectors reload, then roll back and watch them revert.
