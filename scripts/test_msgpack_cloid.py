#!/usr/bin/env python3
"""
Compare msgpack serialization of orders with and without cloid
"""

import msgpack

# Order WITHOUT cloid (from Above10 cassette)
order_without_cloid = {
    "grouping": "na",
    "orders": [
        {
            "a": 173,
            "b": True,
            "p": "0.1233",
            "r": False,
            "s": "100",
            "t": {"limit": {"tif": "Gtc"}}
        }
    ],
    "type": "order"
}

# Order WITH cloid (from Cloid cassette)
order_with_cloid = {
    "grouping": "na",
    "orders": [
        {
            "a": 173,
            "b": True,
            "c": "06c60000000000000000000000003f5a",  # cloid field
            "p": "0.1233",
            "r": False,
            "s": "45",
            "t": {"limit": {"tif": "Gtc"}}
        }
    ],
    "type": "order"
}

print("=" * 80)
print("WITHOUT CLOID")
print("=" * 80)
packed_without = msgpack.packb(order_without_cloid)
print(f"Packed bytes ({len(packed_without)}): {packed_without.hex()}")
print(f"Hex dump:")
for i in range(0, len(packed_without), 16):
    chunk = packed_without[i:i+16]
    hex_str = ' '.join(f"{b:02x}" for b in chunk)
    ascii_str = ''.join(chr(b) if 32 <= b < 127 else '.' for b in chunk)
    print(f"  {i:04x}: {hex_str:<48} {ascii_str}")

print("\n" + "=" * 80)
print("WITH CLOID")
print("=" * 80)
packed_with = msgpack.packb(order_with_cloid)
print(f"Packed bytes ({len(packed_with)}): {packed_with.hex()}")
print(f"Hex dump:")
for i in range(0, len(packed_with), 16):
    chunk = packed_with[i:i+16]
    hex_str = ' '.join(f"{b:02x}" for b in chunk)
    ascii_str = ''.join(chr(b) if 32 <= b < 127 else '.' for b in chunk)
    print(f"  {i:04x}: {hex_str:<48} {ascii_str}")

print("\n" + "=" * 80)
print("COMPARISON")
print("=" * 80)
print(f"Difference in size: {len(packed_with) - len(packed_without)} bytes")

# Find first difference
for i, (b1, b2) in enumerate(zip(packed_without, packed_with)):
    if b1 != b2:
        print(f"First difference at byte {i}: {b1:02x} vs {b2:02x}")
        break
else:
    if len(packed_without) != len(packed_with):
        print(f"Same prefix, but different lengths")
