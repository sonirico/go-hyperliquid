go_hex = "86a16100a162c3a170a53430303030a173a5302e303031a172c2a17481a56c696d697481a3746966a3477463"
py_hex = "86a16100a162c3a170a53430303030a172c2a173a5302e303031a17481a56c696d697481a3746966a3477463"

print(f"Go length: {len(go_hex)}")
print(f"Py length: {len(py_hex)}")
print(f"Are equal: {go_hex == py_hex}")

if go_hex != py_hex:
    print("\nDifferences:")
    for i, (g, p) in enumerate(zip(go_hex, py_hex)):
        if g != p:
            print(f"Position {i}: Go='{g}' Python='{p}'")
            print(f"  Context Go:     ...{go_hex[max(0,i-10):i+10]}...")
            print(f"  Context Python: ...{py_hex[max(0,i-10):i+10]}...")
            break
