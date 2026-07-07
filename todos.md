# todos for sprint 2 (ends jul 7)

## completed

- scaffold api: fastapi + jwt auth + rbac (liam)
- scaffold agent control plane: go + opamp-go skeleton (liam)
- postgres schema + seed: PR #11 (kristie), correctness fixes layered on
  feat/sprint-2-update (liam)
- docker compose for local postgres + api (liam)
- scaffold react ui + wire auth flow: login -> protected fleet dashboard
  (base: d'andre's feat/ui-scaffolding, my-app/ moved to ui/) (liam)
- assignment 5: use cases + sequence diagrams submission (liam)
- sprint report 2 (liam)

## pending -> sprint 3

- review + merge sprint 2 PRs (#11 + feat/sprint-2-update)
- opamp round trip against a live collector (supervisor mode)
- config editor + version history views in ui
- rollout engine: reconcile desired vs effective config, per-agent status
- enforce rbac + audit coverage on all remaining endpoints
- ci to run api/ui/control-plane tests on PRs

# todos for sprint 1

## completed

- personal bios
- project selection document
- team contract & meeting schedule
- functional + non-functional requirements
- data model

## pending

- system architecture diagram
- review and finalize func v nonfunc reqs
- use case diagram
- scaffold react UI app


# task delegation

## liam
- [x] personal bio
- [x] agree on team contract & meeting schedule
- [x] draft functional v non-functional reqs
- [x] sys arch diagram
- [x] data model
- [x] submit use case and requirements doc
- [x] scaffold api + control plane (sprint 2)
- [x] wire auth flow between ui and api (sprint 2)
- [x] assignment 5: use cases + sequence diagrams (sprint 2)
- [x] review PR #11 + fix schema/seed (sprint 2)

## d'andre
- [x] personal bio
- [x] submit team contract & meeting schedule
- [ ] review react + fastapi basics
- [ ] review & finalize func v nonfunc reqs
- [x] scaffold react ui app: vite base on feat/ui-scaffolding (sprint 2)

## obaid
- [x] personal bio
- [x] agree on team contract & meeting schedule
- [ ] use case diagram

## kristie
- [x] personal bio
- [x] agree on team contract & meeting schedule
- [ ] review react + fastapi basics
- [x] submit project selection doc
- [ ] cleanup/improve sys arch diagram
- [x] postgres schema + seed data: PR #11 (sprint 2, fixes pending review)
