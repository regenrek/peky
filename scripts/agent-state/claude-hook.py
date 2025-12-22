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
    session_id = payload.get("session_id") or payload.get("sessionId")
    if session_id:
        record["session_id"] = session_id
    transcript = payload.get("transcript_path") or payload.get("transcriptPath")
    if transcript:
        record["transcript_path"] = transcript
    target = path / f"{pane_id}.json"
    tmp = target.with_suffix(".json.tmp")
    tmp.write_text(json.dumps(record))
    tmp.replace(target)


def normalize_event(payload: dict) -> str:
    for key in ("event", "hook", "type", "name"):
        value = payload.get(key)
        if isinstance(value, str) and value.strip():
            return value.strip().lower()
    return ""


def map_event_to_state(event: str, payload: dict) -> str:
    if event in {"sessionstart", "session_start"}:
        return "idle"
    if event in {"userpromptsubmit", "user_prompt_submit"}:
        return "running"
    if event in {"pretooluse", "pre_tool_use"}:
        return "running"
    if event in {"posttooluse", "post_tool_use"}:
        return "running"
    if event in {"permissionrequest", "permission_request"}:
        return "waiting"
    if event in {"notification"}:
        return "waiting"
    if event in {"stop"}:
        return "idle"
    if event in {"sessionend", "session_end"}:
        reason = payload.get("reason") or payload.get("end_reason")
        if isinstance(reason, str) and reason.lower() in {"error", "failed", "failure"}:
            return "error"
        exit_code = payload.get("exit_code") or payload.get("exitCode")
        if isinstance(exit_code, int) and exit_code != 0:
            return "error"
        return "done"
    return ""


def main() -> int:
    pane_id = os.environ.get("TMUX_PANE", "").strip()
    if not pane_id:
        return 0
    try:
        payload = json.load(sys.stdin)
    except json.JSONDecodeError:
        return 0
    event = normalize_event(payload)
    if not event:
        return 0
    state = map_event_to_state(event, payload)
    if not state:
        return 0
    write_state(pane_id, state, "claude", payload)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
