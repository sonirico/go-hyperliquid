#!/usr/bin/env python3
import msgpack

# Exact same structure as Go test
order_wire = {
    "a": 0,
    "b": True,
    "p": "40000",
    "s": "0.001",
    "r": False,
    "t": {
        "limit": {
            "tif": "Gtc"
        }
    }
}

action = {
    "type": "order",
    "orders": [order_wire],
    "grouping": "na",
}

data = msgpack.packb(action)
print("=== PYTHON MSGPACK HEX ===")
print(data.hex())
print(f"=== PYTHON MSGPACK BYTES LENGTH: {len(data)} ===")
