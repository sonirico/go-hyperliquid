# AGENTS.md

Notes for coding agents working on this repository. Everything here was measured against the live API, not inferred. Read it before touching signing, an action, or a response struct.

## Commands

```
make install-tools   # easyjson, goimports, golines, golangci-lint, gosec at the versions pinned in the Makefile
make generate        # regenerate *_easyjson.go; never edit those files by hand
make fmt             # gofmt + goimports + golines
make lint            # golangci-lint v2, must report 0 issues
make test            # unit tests, examples excluded
make verify-live     # signs every action and sends it to testnet (network, no secrets)
```

CI runs exactly these targets and fails on any diff after `make fmt` or `make generate`. `go test ./...` fails in `./examples`: that package needs a real private key. Use `make test`.

If `go generate` fails because a stale `*_easyjson.go` no longer compiles, delete that generated file and run it again. A struct that holds a generic type (`Tuple2`) needs `//easyjson:skip`.

## The two signing schemes

Every exchange action belongs to exactly one.

**L1 action.** The action is msgpack-encoded, hashed with nonce, vault and `expiresAfter`, and signed through a phantom `Agent` EIP-712 struct. The server rebuilds the msgpack from its own struct, so three things must match byte for byte: field order, field type (int vs float vs string), and whether an optional field is absent or present as nil.

**User-signed action.** Direct EIP-712 with primary type `HyperliquidTransaction:<Name>`. Field order comes from the type list, so the JSON order does not matter.

Rules that follow:

- An L1 action is a struct in `actions.go` with `msgpack` and `json` tags in the API's field order. Never a `map[string]any`: Go randomises map iteration, so a map with two or more keys produces a different hash on each run. Measured: 1245 correct hashes out of 2000 for a 4-key map; 0 of 50 for two nested 3-key maps. `TestNoMapL1Actions` fails the build if a map reaches `signL1Action`.
- Structs encode in declared order and are deterministic. The historical "non-deterministic signature" bugs in this repo were all maps.
- A wrong field type gives HTTP 422 `Failed to deserialize the JSON body`. A wrong order or a nil-vs-absent mismatch gives a response naming an address that is not yours.
- Amounts in user-signed actions go through `floatToWire`, never `fmt.Sprintf("%.6f")`. USD integers go through `floatToUsdInt` (rounds, errors if lossy).
- `usdClassTransfer`, `sendAsset` and `agentSendAsset` are posted with `vaultAddress: null` even when a vault is configured.
- Always sign through `e.signL1Action` / `e.signUserSignedAction` so an injected signer is honoured. Do not call `SignL1Action(e.privateKey, ...)` from a method.
- Tagged-union actions (`spotDeploy`, `perpDeploy`, `CSignerAction`, `CValidatorAction`, `userOutcome`) carry `type` plus exactly one variant key. Some variants have a nil value (`"unjailSelf": null`) that must be encoded, not omitted.

## How to prove an action is right

Three layers, all required for a new or changed action.

1. **Vector from the Python SDK.** Add a case to `scripts/golden.py` (the SDK is pinned in `scripts/requirements-golden.txt`), run it to write `testdata/golden/<name>.json`, and add a row to `golden_test.go`. The row calls the Go method against a local server and compares the posted action and the signature with the vector. When the SDK has no method for the action, build the dict in the order the docs give and sign it with the SDK's `sign_l1_action`.
2. **Testnet.** `TestLive` (build tag `live`) runs every golden row against testnet with a key generated in the test. No funds are needed. The server answers `User or API Wallet 0x... does not exist` or `Must deposit before performing actions. User: 0x...`; when that address is the test key's address the server deserialised the action and recovered the right signer. A different address means the hash is wrong. HTTP 422 means the shape is wrong.
3. **`make verify-live` before review.**

What the testnet check does not prove: that the action succeeds for a funded account. `multiSig` is rejected with `Invalid multi-sig outer signer` before the signature is checked, so only its vector proves the signature.

## Responses

- A failed action is `{"status":"err","response":"<message>"}`. `actionError` turns that into a Go error; every method in `exchange_others.go` calls it (or `actionResponse`) before unmarshalling. There is no `error` or `txHash` field in the API.
- New exchange methods return the raw `response` JSON via `actionResponse`. Success payloads cannot be observed without a funded account, so do not invent typed results.
- Every Info and websocket response struct has a captured payload in `testdata/live/` and a test that calls `assertKeysCovered`, which fails listing each JSON key the struct drops. Capture with `scripts/capture.py info '<body>' <out>` or `scripts/capture.py ws '<subscription>' <out> [seconds]`. Trim fixtures but keep every key path.
- A field with `omitempty` that the API sends as null or empty is lost on a round trip and fails that test. Drop the `omitempty`.
- An empty list from the API proves nothing about the struct. Find a public address that has data (users in `recentTrades`, child vaults of HLP `0xdfc24b077bc1425ad1dea75bcb6f8158e10df303`).

## Sources of truth, in order

1. The live server (testnet for signing, mainnet for read-only Info and websocket).
2. The Python SDK at the pinned commit.
3. The docs at https://hyperliquid.gitbook.io/hyperliquid-docs. They lag: some documented endpoints are gone and some live fields are undocumented.

Removed server-side and deleted from this repo, do not re-add: `perpDexClassTransfer`, `userDexAbstraction` (exchange and info), `agentEnableDexAbstraction`, websocket `webData2`.

## Known traps

- Spot `activeAssetCtx` messages arrive on channel `activeSpotAssetCtx`.
- `userToMultiSigSigners` returns an object `{authorizedUsers, threshold}` or null, not a list.
- `perpAnnotation` returns null for coins of the default dex; use a builder-dex coin to see the fields.
- `vaultDetails` rejects `"user": ""` with 422; omit the key.
- `authorizeAqav2Role` rejects unknown roles at deserialisation; `technical` is accepted.
- `createVault` and `agentSendAsset` carry their nonce inside the action; it must equal the signing nonce.
- `sendToEvmWithData` posts `data` as a `0x` hex string and signs the raw bytes.
- Python `json.dumps` adds spaces, Go does not. For string fields that embed JSON (`convertToMultiSigUser.signers`) the server accepts both, but vectors must be generated with compact separators.
- `outcomeMetaUpdates` and `notification` could not be captured live; their fixtures end in `_docs.json` and come from the docs.

## Conventions

- One method per action on `Exchange`, one file per websocket subscription (`ws_sub_<name>.go`), response types in `types.go`.
- Write the failing test first and watch it fail for the right reason.
- One logical change per commit, conventional prefix (`fix:`, `feat:`, `test:`, `refactor:`, `chore:`, `docs:`, `ci:`), `!` for breaking changes.
