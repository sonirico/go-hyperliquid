#!/usr/bin/env python3
import msgpack

# Order wire without full action wrapper
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

data = msgpack.packb(order_wire)
print("Python order_wire hex:")
print(data.hex())
print(f"Length: {len(data)}")
