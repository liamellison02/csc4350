# /// script
# requires-python = ">=3.12"
# dependencies = ["httpx"]
# ///
"""opamp e2e: push a config version through the api and assert a live
supervisor-managed collector applies it (effective hash + rollout row),
twice, to prove the reconciler reacts to change."""

import os
import sys
import time

import httpx

API = os.environ.get("API_URL", "http://localhost:8000")
TIMEOUT = 120  # first round includes collector boot
POLL = 3

YAML_V1 = """receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

exporters:
  debug:

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [debug]
"""

YAML_V2 = YAML_V1.replace("debug:", 'debug:\n    verbosity: detailed')


def wait_for(desc, fn, timeout=TIMEOUT):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        result = fn()
        if result is not None:
            return result
        time.sleep(POLL)
    print(f"TIMEOUT waiting for {desc}", file=sys.stderr)
    return None


def main() -> int:
    client = httpx.Client(base_url=API, timeout=10)

    def healthy():
        try:
            return client.get("/healthz").status_code == 200 or None
        except httpx.HTTPError:
            return None

    if wait_for("api /healthz", healthy, timeout=60) is None:
        return 1

    login = client.post(
        "/auth/login",
        json={"email": "operator@helmsman.local", "password": "operator123!"},
    )
    login.raise_for_status()
    headers = {"Authorization": f"Bearer {login.json()['access_token']}"}

    name = f"e2e-config-{int(time.time())}"
    config = client.post(
        "/configurations",
        json={"name": name, "label_selector": None},
        headers=headers,
    )
    config.raise_for_status()
    config_id = config.json()["id"]

    def live_agent():
        agents = client.get("/agents", headers=headers).json()
        live = [a for a in agents if a["status"] == "healthy" and not a["instance_uid"].startswith("agent-00")]
        return live[0] if live else None

    agent = wait_for("supervisor agent to connect", live_agent)
    if agent is None:
        dump(client, headers, config_id)
        return 1
    uid = agent["instance_uid"]
    print(f"agent connected: {uid}")

    for round_no, yaml_body in ((1, YAML_V1), (2, YAML_V2)):
        version = client.post(
            f"/configurations/{config_id}/versions",
            json={"yaml": yaml_body},
            headers=headers,
        )
        version.raise_for_status()
        vhash = version.json()["hash"]
        vid = version.json()["id"]
        print(f"round {round_no}: created version {version.json()['version_no']} hash {vhash[:12]}...")

        def applied():
            agents = {a["instance_uid"]: a for a in client.get("/agents", headers=headers).json()}
            if agents.get(uid, {}).get("effective_config_hash") != vhash:
                return None
            rollouts = client.get(
                f"/configurations/{config_id}/rollouts", headers=headers
            ).json()
            ok = [r for r in rollouts if r["config_version_id"] == vid and r["agent_instance_uid"] == uid and r["status"] == "applied"]
            return ok or None

        if wait_for(f"round {round_no} apply", applied) is None:
            dump(client, headers, config_id)
            return 1
        print(f"round {round_no}: applied and acknowledged")

    print("E2E PASS")
    return 0


def dump(client, headers, config_id):
    print("--- agents ---", file=sys.stderr)
    print(client.get("/agents", headers=headers).text, file=sys.stderr)
    print("--- rollouts ---", file=sys.stderr)
    print(
        client.get(f"/configurations/{config_id}/rollouts", headers=headers).text,
        file=sys.stderr,
    )


if __name__ == "__main__":
    sys.exit(main())
