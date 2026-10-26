#!/usr/bin/env python3
"""Launch an already built Hackathon binary with an isolated synthetic workspace.

Loads only MODEL_API_KEY from the local ignored .env, without shell evaluation.
"""
import os
from pathlib import Path
import shlex
import sys

if len(sys.argv) != 3 or not Path(sys.argv[2]).is_absolute():
    raise SystemExit("Usage: python3 scripts/run-discovery-demo.py /absolute/binary /absolute/synthetic-workspace")
env_file = Path(__file__).resolve().parent.parent / ".env"
if not os.environ.get("MODEL_API_KEY") and env_file.exists():
    for line in env_file.read_text().splitlines():
        line = line.strip()
        if line.startswith("export "):
            line = line[7:]
        key, sep, value = line.partition("=")
        if sep and key.strip() == "MODEL_API_KEY":
            try:
                parts = shlex.split(value, comments=True)
            except ValueError:
                raise SystemExit("MODEL_API_KEY assignment could not be parsed") from None
            if len(parts) == 1:
                os.environ["MODEL_API_KEY"] = parts[0]
if not os.environ.get("MODEL_API_KEY"):
    raise SystemExit("Set MODEL_API_KEY in the environment or the ignored .env")
os.environ["YOURMOVE_DATA_DIR"] = sys.argv[2]
os.environ.setdefault("YOURMOVE_HTTP_PORT", "3019")
os.environ["YOURMOVE_COMPAT_URL"] = "https://api.meta.ai/v1/chat/completions"
os.environ["YOURMOVE_COMPAT_MODEL"] = "muse-spark-1.3"
os.environ["YOURMOVE_COMPAT_API_KEY"] = os.environ["MODEL_API_KEY"]
os.execvp(sys.argv[1], [sys.argv[1]])
