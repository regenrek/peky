#!/usr/bin/env python3
import json
import os
import sys
import time
from pathlib import Path


def state_dir() -> Path:
    root = os.environ.get("PEAKYPANES_AGENT_STATE_DIR")
    if root:
        return Path(root)
    runtime = os.environ.get("XDG_RUNTIME_DIR") or "/tmp"
    return Path(runtime) / "peakypanes" / "agent-state"


def write_state(pane_id: str, state: str, tool: str, payload: dict) -> None:
    path = state_dir()
    path.mkdir(parents=True, exist_ok=True)
    record = {
        "state": state,
        "tool": tool,
        "updated_at_unix_ms": int(time.time() * 1000),
        "pane_id": pane_id,
    }
    if payload.get("turn-id"):
        record["turn_id"] = payload.get("turn-id")
    if payload.get("thread-id"):
        record["thread_id"] = payload.get("thread-id")
    if payload.get("cwd"):
        record["cwd"] = payload.get("cwd")
    target = path / f"{pane_id}.json"
    tmp = target.with_suffix(".json.tmp")
    tmp.write_text(json.dumps(record))
    tmp.replace(target)


def main() -> int:
    pane_id = os.environ.get("PEAKYPANES_PANE_ID", "").strip()
    if not pane_id:
        return 0
    if len(sys.argv) < 2:
        return 0
    try:
        payload = json.loads(sys.argv[-1])
    except json.JSONDecodeError:
        return 0
    if payload.get("type") != "agent-turn-complete":
        return 0
    write_state(pane_id, "idle", "codex", payload)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
