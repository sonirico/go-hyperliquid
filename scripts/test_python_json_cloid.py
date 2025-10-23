#!/usr/bin/env python3
import json
import sys
sys.path.insert(0, "hyperliquid-python-sdk")

from hyperliquid.utils.types import Cloid

cloid = Cloid.from_str("0x06c60000000000000000000000003f5a")
order_wire = {
    "a": 173,
    "b": True,
    "c": cloid.to_raw(),
    "p": "0.1233",
    "r": False,
    "s": "45",
    "t": {"limit": {"tif": "Gtc"}}
}

action = {
    "grouping": "na",
    "orders": [order_wire],
    "type": "order"
}

# Serialize with json.dumps like _post_action does
json_payload = json.dumps(action)
print("JSON payload:")
print(json_payload)
print()

# Check the 'c' field value in JSON
import json as json_module
parsed = json_module.loads(json_payload)
print(f"Cloid in orders[0]: {parsed['orders'][0]['c']}")
print(f"Type: {type(parsed['orders'][0]['c'])}")
