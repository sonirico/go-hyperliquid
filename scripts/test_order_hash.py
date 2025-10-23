#!/usr/bin/env python3
import msgpack
import sys

# The msgpack data from Go (hex)
# The msgpack data from Go (hex) - includes msgpack + nonce + vault
go_full_hex = "83a474797065a56f72646572a66f72646572739187a161ccada162c3a170a6302e31323333a173a23435a172c2a17481a56c696d697481a3746966a3477463a163d9203036633630303030303030303030303030303030303030303030303033663561a867726f7570696e67a26e610000019a0bc4c00800"

# The last 8 bytes are nonce, then 1 byte for vault flag
# Let's extract just the msgpack part
go_full_data = bytes.fromhex(go_full_hex)
print(f"Go full data length: {len(go_full_data)}")

# Try to find where msgpack ends
# Msgpack should be much shorter than 120 bytes
# Let's decode progressively
for i in range(80, len(go_full_data)):
    try:
        decoded = msgpack.unpackb(go_full_data[:i], raw=False)
        print(f"Msgpack ends at byte {i}")
        print(f"Go msgpack decoded: {decoded}")
        go_msgpack = go_full_data[:i]
        break
    except:
        continue

# Extract nonce (next 8 bytes after msgpack)
nonce_start = len(go_msgpack)
go_nonce_bytes = go_full_data[nonce_start:nonce_start+8]
go_nonce = int.from_bytes(go_nonce_bytes, 'big')
print(f"\nGo nonce: {go_nonce}")
print(f"Go nonce bytes: {go_nonce_bytes.hex()}")

# Extract vault flag
vault_flag = go_full_data[nonce_start+8:nonce_start+9]
print(f"Go vault flag: {vault_flag.hex()}")


# Now create the same action in Python
action = {
    "type": "order",
    "orders": [{
        "a": 173,  # DOGE asset ID
        "b": True,  # is buy
        "p": "0.1233",  # price
        "s": "45",  # size
        "r": False,  # reduce only
        "t": {"limit": {"tif": "Gtc"}},
        "c": "06c60000000000000000000000003f5a"  # cloid (without 0x)
    }],
    "grouping": "na"
}

nonce = 1761133685960  # Example nonce from Go
vault_address = None

# Python way
python_msgpack = msgpack.packb(action)
print(f"\nPython msgpack length: {len(python_msgpack)}")
print(f"Python msgpack hex: {python_msgpack.hex()}")
print(f"Python msgpack decoded: {msgpack.unpackb(python_msgpack, raw=False)}")

# Add nonce (8 bytes big endian)
python_msgpack += nonce.to_bytes(8, 'big')

# Add vault address
if vault_address is None:
    python_msgpack += b'\x00'
else:
    python_msgpack += b'\x01'
    python_msgpack += bytes.fromhex(vault_address[2:] if vault_address.startswith('0x') else vault_address)

print(f"\nPython full data hex: {python_msgpack.hex()}")
print(f"Python full data length: {len(python_msgpack)}")

# Compare
print(f"\n=== COMPARISON ===")
print(f"Go data:     {go_full_hex[:100]}...")
print(f"Python data: {python_msgpack.hex()[:100]}...")
print(f"Match: {go_full_hex == python_msgpack.hex()}")
