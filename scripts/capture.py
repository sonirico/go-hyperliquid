#!/usr/bin/env python3
"""Capture a raw Hyperliquid payload into a fixture file.

Usage:
    capture.py info '<json body>' <out>
    capture.py ws '<subscription json>' <out>
"""
import json
import sys
import time
import urllib.request

INFO_URL = "https://api.hyperliquid.xyz/info"
WS_URL = "wss://api.hyperliquid.xyz/ws"
WS_TIMEOUT_SECONDS = 30


def capture_info(body):
    req = urllib.request.Request(
        INFO_URL,
        data=body.encode(),
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=WS_TIMEOUT_SECONDS) as resp:
        return resp.read().decode()


def capture_ws(subscription):
    import websocket

    conn = websocket.create_connection(WS_URL, timeout=WS_TIMEOUT_SECONDS)
    try:
        conn.send(json.dumps({"method": "subscribe", "subscription": json.loads(subscription)}))
        deadline = time.time() + WS_TIMEOUT_SECONDS
        while time.time() < deadline:
            raw = conn.recv()
            if json.loads(raw).get("channel") != "subscriptionResponse":
                return raw
    finally:
        conn.close()
    raise TimeoutError("no data message within %d s" % WS_TIMEOUT_SECONDS)


def main():
    if len(sys.argv) != 4 or sys.argv[1] not in ("info", "ws"):
        sys.exit(__doc__)
    kind, arg, out = sys.argv[1:]
    raw = capture_info(arg) if kind == "info" else capture_ws(arg)
    with open(out, "w") as f:
        f.write(raw)


if __name__ == "__main__":
    main()
