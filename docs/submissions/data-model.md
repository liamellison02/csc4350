# Assignment 4: Thinking About Your Data Model

**Team Name:** LOKK (Group 14)

**Project Name:** Helmsman

---

## 1. Overview

Helmsman stores all of its state in a single PostgreSQL database, which is the
one source of truth for the whole system. The data model below is built from six
core entities that, together, satisfy every functional requirement in the prior
submission and back each use case: fleet visibility, configuration management,
immutable versioning, per-agent rollouts, one-click rollback, role-based access
control, and an append-only audit trail.

The model is deliberately small. Each entity earns its place by storing data a
requirement needs, and the relationships between them carry the cardinality
rules that keep the data consistent (one author per version, one current version
per configuration, many rollout records per version, and so on).

Full requirements and use cases that this model serves live in the prior
submission: [requirements-and-use-cases.md](requirements-and-use-cases.md).

---

## 2. Entity Relationship Diagram

<img src="diagrams/data-model.svg" alt="Helmsman data model ERD" width="960">

> Rendered crow's-foot ERD above. The editable source is committed alongside it
> at [diagrams/data-model.drawio](diagrams/data-model.drawio) (open in draw.io /
> diagrams.net). A text source-of-truth version is also kept as a Mermaid
> `erDiagram` in section 7.3 of
> [requirements-and-use-cases.md](requirements-and-use-cases.md).

---

## 3. How to Read the Diagram (Crow's-Foot Cardinality)

Each relationship line carries a symbol at both ends. The symbol nearest an
entity states how many rows of that entity can take part:

- A bar `|` means **exactly one** (mandatory).
- A circle `o` means **zero** (optional participation).
- A crow's foot `<` means **many**.

Reading the two ends together gives the cardinality:

- `||` to `o{` is **one-to-many**: one mandatory parent, zero-or-many children.
- `||` to `o|` is **one-to-one**: one mandatory parent, zero-or-one child.
- `|o` to `o{` is **one-to-many with an optional parent**: the child may have
  zero or one parent.

Primary keys are underlined; foreign keys are shown in italics and marked `FK`.

---

## 4. Cardinalities at a Glance

The model demonstrates all three cardinality classes the course covers.

| Class | Where it appears | Reads as |
|---|---|---|
| One-to-one | CONFIGURATIONS "current is" CONFIG_VERSIONS | a configuration points at zero-or-one current version (null until the first save) |
| One-to-many | CONFIGURATIONS "has history" CONFIG_VERSIONS; USERS "authors" CONFIG_VERSIONS; CONFIG_VERSIONS / AGENTS to ROLLOUTS | one parent row, zero-or-many child rows |
| Many-to-many (resolved) | CONFIG_VERSIONS to AGENTS, resolved through ROLLOUTS | one version is deployed to many agents and one agent receives many versions over time |

---

## 5. Relationships

Source-to-target is read parent-to-child. Optionality is stated on each side.

| Relationship | Cardinality | Reads as |
|---|---|---|
| USERS authors CONFIG_VERSIONS | one-to-many (mandatory one) | every version has exactly one author; a user authors zero-or-many versions |
| USERS generates AUDIT_LOGS | one-to-many (optional one) | a user generates zero-or-many audit entries; an entry has zero-or-one user, because system and failed-login events have no acting user |
| CONFIGURATIONS has history CONFIG_VERSIONS | one-to-many (mandatory one) | a configuration has zero-or-many versions; each version belongs to exactly one configuration |
| CONFIGURATIONS current is CONFIG_VERSIONS | one-to-one (optional) | a configuration points at zero-or-one current version; this is the model's one-to-one |
| CONFIG_VERSIONS deployed via ROLLOUTS | one-to-many (mandatory one) | a version produces zero-or-many rollout records; a rollout record cannot exist without a version |
| AGENTS targeted by ROLLOUTS | one-to-many (mandatory one) | an agent receives zero-or-many rollout records; a rollout record cannot exist without an agent |

---

## 6. Entities and Attributes

### USERS

The people who use Helmsman; drives role-based access control.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| id | int | PK | surrogate key |
| email | varchar | | unique login identity |
| password_hash | varchar | | salted hash; the credential authentication checks against |
| role | varchar | | one of admin, operator, viewer |
| is_active | boolean | | false deactivates the account without deleting it |
| created_at | timestamp | | |

### AGENTS

Each OpenTelemetry Collector that has registered over OpAMP.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| instance_uid | varchar | PK | the OpAMP instance UID; a natural key, which is why foreign keys to it are strings |
| hostname | varchar | | |
| labels | json | | key-value labels used to target rollouts |
| agent_type | varchar | | |
| version | varchar | | |
| status | varchar | | one of healthy, degraded, disconnected |
| last_seen | timestamp | | updated on each OpAMP message |
| effective_config_hash | varchar | | hash of the config the agent is actually running; used to detect drift |

### CONFIGURATIONS

A named collector configuration and the set of agents it targets.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| id | int | PK | surrogate key |
| name | varchar | | |
| label_selector | varchar | | selects which agents this configuration applies to |
| current_version_id | int | FK | nullable pointer to the active CONFIG_VERSIONS row; null until the first save |

### CONFIG_VERSIONS

An immutable snapshot of a configuration's body. New row on every save.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| id | int | PK | surrogate key |
| configuration_id | int | FK | the configuration this version belongs to |
| version_no | int | | sequential within a configuration; (configuration_id, version_no) is unique |
| yaml | text | | the configuration body |
| hash | varchar | | content hash |
| author_id | int | FK | the USERS row that saved this version |
| created_at | timestamp | | |

### ROLLOUTS

The associative entity that records pushing one config version to one agent.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| id | int | PK | surrogate key (see design note) |
| config_version_id | int | FK | the version being pushed |
| agent_instance_uid | varchar | FK | the agent receiving it; mirrors AGENTS.instance_uid |
| status | varchar | | one of pending, applied, failed |
| applied_at | timestamp | | nullable until a result returns |
| error | varchar | | nullable; set when status is failed |

### AUDIT_LOGS

An append-only, tamper-evident record of every notable action.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| id | int | PK | surrogate key |
| user_id | int | FK | nullable; system and failed-login events have no acting user |
| action | varchar | | login, role change, config change, rollout, rollback, denied attempt |
| target_type | varchar | | polymorphic target kind (configuration, rollout, user) |
| target_id | varchar | | id of the target row |
| detail | text | | free-form context |
| created_at | timestamp | | |

---

## 7. Design Notes

- **ROLLOUTS is an associative (junction) entity.** A config version is deployed
  to many agents, and an agent receives many config versions over time. That
  many-to-many is resolved by ROLLOUTS, which also carries its own attributes
  (status, applied_at, error) for each version-to-agent push.
- **Surrogate key on ROLLOUTS, on purpose.** The pair
  (config_version_id, agent_instance_uid) is not unique, because a rollback
  re-pushes the same version to the same agent at a later time. A surrogate
  `id` is therefore the correct primary key rather than a composite of the two
  foreign keys.
- **AGENTS uses a natural key.** The OpAMP instance UID is the agent's stable
  identity, so it is the primary key directly. This is why ROLLOUTS references
  agents with a string foreign key while the other entities use integer
  surrogates.
- **Optional participation is modeled explicitly.** AUDIT_LOGS.user_id and
  CONFIGURATIONS.current_version_id are nullable, which is what makes the
  USERS-to-AUDIT_LOGS link optional on the user side and the configuration's
  current-version link a zero-or-one relationship.
- **Agent targeting is a dynamic many-to-many.** Which agents a configuration
  applies to is decided at rollout time by matching label_selector against agent
  labels. That relationship is computed, not stored, so it needs no join table;
  ROLLOUTS records the concrete pushes that result.
- **The audit log is append-only.** Rows are never updated or deleted, which
  satisfies the tamper-evident requirement.

---

## 8. Requirements Coverage

Every entity traces back to functional requirements in the prior submission:

- USERS backs authentication and role-based access control (FR-1, FR-10).
- AGENTS backs the registry and fleet visibility (FR-2, FR-8, FR-9).
- CONFIGURATIONS and CONFIG_VERSIONS back configuration management and
  immutable versioning (FR-3, FR-4).
- ROLLOUTS backs rollout, remote push, and rollback (FR-5, FR-6).
- AUDIT_LOGS backs audit logging (FR-7).

---

## Sources

- [requirements-and-use-cases.md](requirements-and-use-cases.md) - functional and non-functional requirements, use cases, and the Mermaid source of this model
- [project-selection.md](project-selection.md) - project background and initial data model sketch
- Assignment brief: [thinking-about-your-data-model.md](../instructions/thinking-about-your-data-model.md)
