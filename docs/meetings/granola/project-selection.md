# Project Selection - Meeting #2

**Date & Time:** 16 Jun 26 @ 2:30p
**Attendees:** Liam, Obaid, D'Andre, Kristie

---

### Project Selected: Helmsman (OTel Collector Control Plane)

- Open-source alternative to BindPlane for managing OpenTelemetry collector fleets
  - Core value: single web UI to manage all collector configs, push rollouts, view live health
  - BindPlane claims 40% observability cost reduction but still charges high service fees
  - No comparable open-source product exists, strong resume/portfolio angle
- Backend (Go, OpAMP control plane) already built specialized version at Liam’s internship; needs stripping down/generalization for open-source
  - Reusable open-source components: OTel Collector Contrib, opamp-go (fork)
  - New build scope: web UI + user-facing API only
- Data model: users, agents, collector configs, config versions, rollouts, audit logs
- User roles: admin, operator (deploys/rolls back configs), viewer

### Team and Tech Stack

- Group: Liam, Kristie, D'Andre, and one other
  - Kristie proposed a tutoring scheduling app (payments, attendance, student progress tracking) but deferred to Liam’s project
  - Team comfortable with Java and Python; limited React/JavaScript experience; no regular use of coding agents
- Stack decided: React (frontend) + FastAPI (backend) + SQL
  - FastAPI chosen over Flask: more performant, auto Swagger docs, more commonly used in production
- Liam will scaffold the React app; team members build out individual components

### Assignments and Deadlines

- Next deliverable: use case diagram + functional/non-functional requirements
  - Three roles to diagram: admin, operator, viewer
  - Kristie and D'Andre to tag-team the use case diagram using the project selection markdown + AI tools
  - Liam to handle functional/non-functional requirements
- Sprint report due June 26; will be worked on together next meeting
- Final presentation due July 28: video, max 15 minutes, no code segments shown (demo instead)
- Presentation format: async video submission (online course, no live lecture slot)

### Next Steps

- Open PR on feature branch with project selection submission for team review before submitting (Liam)
- Watch the 44-minute professor video posted today for any clarification on deadlines
- Review React basics and FastAPI docs before next meeting (Kristie, D'Andre)
- Work on use case diagram using project markdown as input (Kristie, D'Andre)
- Meet next Tuesday to finalize sprint report

---

Chat with meeting transcript: https://notes.granola.ai/t/18d54ec7-1281-4300-9fee-8f1486b79ca7
