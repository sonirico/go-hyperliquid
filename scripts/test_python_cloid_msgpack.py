#!/usr/bin/env python3
"""Test how Python serializes Cloid in msgpack"""

import msgpack

# Test 1: Using Cloid object from Python SDK
import sys
sys.path.insert(0, "hyperliquid-python-sdk")

from hyperliquid.utils.types import Cloid
from hyperliquid.utils.signing import order_request_to_order_wire

# Create cloid from string (includes 0x)
cloid = Cloid.from_str("0x06c60000000000000000000000003f5a")
print(f"Cloid object: {cloid}")
print(f"Cloid.to_raw(): {cloid.to_raw()}")

# Create an order request with cloid
order_request = {
    "coin": "DOGE",
    "is_buy": True,
    "sz": 45,
    "limit_px": 0.1233,
    "reduce_only": False,
    "order_type": {"limit": {"tif": "Gtc"}},
    "cloid": cloid,
}

# Convert to wire format
order_wire = order_request_to_order_wire(order_request, 173)
print(f"\nOrder wire: {order_wire}")

# Pack it
packed = msgpack.packb(order_wire)
print(f"\nPacked msgpack ({len(packed)} bytes): {packed.hex()}")

# Unpack it to see what was actually stored
unpacked = msgpack.unpackb(packed)
print(f"\nUnpacked: {unpacked}")
print(f"Cloid field ('c') value: {unpacked.get('c')}")
print(f"Cloid field ('c') type: {type(unpacked.get('c'))}")
