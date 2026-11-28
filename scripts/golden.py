"""Generate testdata/golden/*.json: signing vectors produced by the Python SDK.

Usage: python scripts/golden.py [out_dir]   (default: testdata/golden)
No network is used: Exchange._post_action is replaced by a capture.
"""
import copy
import json
import os
import sys

import eth_account
from hyperliquid import exchange as hl_exchange
from hyperliquid.exchange import Exchange
from hyperliquid.utils import signing as sg
from hyperliquid.utils.types import Cloid

KEY = "0x0123456789012345678901234567890123456789012345678901234567890123"
AGENT_KEY_HEX = "00" * 31 + "02"
NONCE = 1700000000000
OTHER = "0x1111111111111111111111111111111111111111"
MIXED = "0xAbCdEf1111111111111111111111111111111111"
TOKEN = "USDC:0x6d1e7cde53ba9467b783cb7c530ce054"
CLOID = "0x00000000000000000000000000000001"
LIMIT_GTC = {"limit": {"tif": "Gtc"}}
SCHEDULE_TIME = 4102444800000
BASE_URL = "https://api.hyperliquid-testnet.xyz"

out_dir = sys.argv[1] if len(sys.argv) > 1 else os.path.join("testdata", "golden")
wallet = eth_account.Account.from_key(KEY)
ex = Exchange(
    wallet,
    BASE_URL,
    meta={"universe": [{"name": "BTC", "szDecimals": 5}, {"name": "ETH", "szDecimals": 4}]},
    spot_meta={"universe": [], "tokens": []},
)

rec = {}


def capture_post(action, signature, nonce):
    rec["action"] = copy.deepcopy(action)
    rec["signature"] = signature
    rec["nonce"] = nonce
    return {}


def record_l1(wallet, action, active_pool, nonce, expires_after, is_mainnet):
    rec.update(
        kind="l1",
        vault=active_pool,
        expires=expires_after,
        mainnet=is_mainnet,
        primary="Agent",
        types=[{"name": "source", "type": "string"}, {"name": "connectionId", "type": "bytes32"}],
    )
    return real_sign_l1(wallet, action, active_pool, nonce, expires_after, is_mainnet)


def record_user(wallet, action, payload_types, primary_type, is_mainnet):
    rec.update(
        kind="user",
        vault=None,
        expires=None,
        mainnet=is_mainnet,
        primary=primary_type,
        types=[{"name": t["name"], "type": t["type"]} for t in payload_types],
    )
    return real_sign_user(wallet, action, payload_types, primary_type, is_mainnet)


real_sign_l1 = sg.sign_l1_action
real_sign_user = sg.sign_user_signed_action
sg.sign_l1_action = record_l1
hl_exchange.sign_l1_action = record_l1
sg.sign_user_signed_action = record_user
sg.get_timestamp_ms = lambda: NONCE
hl_exchange.get_timestamp_ms = lambda: NONCE
hl_exchange.secrets.token_hex = lambda n: AGENT_KEY_HEX
ex._post_action = capture_post


def l1(action):
    sig = sg.sign_l1_action(wallet, action, None, NONCE, None, False)
    return ex._post_action(action, sig, NONCE)


def user(action, payload_types, primary_type):
    sig = sg.sign_user_signed_action(wallet, action, payload_types, primary_type, False)
    return ex._post_action(action, sig, NONCE)


SEND_TO_EVM_WITH_DATA_SIGN_TYPES = [
    {"name": "hyperliquidChain", "type": "string"},
    {"name": "token", "type": "string"},
    {"name": "amount", "type": "string"},
    {"name": "sourceDex", "type": "string"},
    {"name": "destinationRecipient", "type": "string"},
    {"name": "addressEncoding", "type": "string"},
    {"name": "destinationChainId", "type": "uint32"},
    {"name": "gasLimit", "type": "uint64"},
    {"name": "data", "type": "bytes"},
    {"name": "nonce", "type": "uint64"},
]


def send_to_evm_with_data(data_hex):
    action = {
        "type": "sendToEvmWithData",
        "token": TOKEN,
        "amount": "1.5",
        "sourceDex": "spot",
        "destinationRecipient": OTHER,
        "addressEncoding": "hex",
        "destinationChainId": 1337,
        "gasLimit": 100000,
        "data": data_hex,
        "nonce": NONCE,
    }
    signed = dict(action, data=bytes.fromhex(data_hex[2:]))
    sig = sg.sign_user_signed_action(
        wallet, signed, SEND_TO_EVM_WITH_DATA_SIGN_TYPES, "HyperliquidTransaction:SendToEvmWithData", False
    )
    action["signatureChainId"] = signed["signatureChainId"]
    action["hyperliquidChain"] = signed["hyperliquidChain"]
    return ex._post_action(action, sig, NONCE)


def multi_sig_l1_payload():
    action = {"type": "noop"}
    sig = sg.sign_multi_sig_l1_action_payload(wallet, action, False, None, NONCE, None, OTHER, MIXED)
    return ex._post_action(action, sig, NONCE)


def multi_sig_user_signed_payload():
    action = {"type": "usdSend", "destination": OTHER, "amount": "1.5", "time": NONCE}
    sig = sg.sign_multi_sig_user_signed_action_payload(
        wallet, action, False, sg.USD_SEND_SIGN_TYPES, "HyperliquidTransaction:UsdSend", OTHER, MIXED
    )
    return ex._post_action(action, sig, NONCE)


def limit_modify(oid):
    return {
        "oid": oid,
        "order": {
            "coin": "BTC",
            "is_buy": True,
            "sz": 0.001,
            "limit_px": 50000.0,
            "order_type": LIMIT_GTC,
            "reduce_only": False,
            "cloid": None,
        },
    }


def trigger_order(tpsl, trigger_px):
    return {
        "coin": "BTC",
        "is_buy": False,
        "sz": 0.001,
        "limit_px": 50000.0,
        "order_type": {"trigger": {"triggerPx": trigger_px, "isMarket": True, "tpsl": tpsl}},
        "reduce_only": True,
    }


ENTRY_ORDER = {
    "coin": "BTC",
    "is_buy": True,
    "sz": 0.001,
    "limit_px": 50000.0,
    "order_type": LIMIT_GTC,
    "reduce_only": False,
}


def compact_signers(run):
    dumps = json.dumps
    json.dumps = lambda obj: dumps(obj, separators=(",", ":"))
    try:
        return run()
    finally:
        json.dumps = dumps


cases = {
    "order_limit": lambda: ex.order("BTC", True, 0.001, 50000.0, LIMIT_GTC, False),
    "order_cloid_builder": lambda: ex.order(
        "BTC", True, 0.001, 50000.0, LIMIT_GTC, False, cloid=Cloid.from_str(CLOID), builder={"b": OTHER, "f": 10}
    ),
    "order_trigger": lambda: ex.order(
        "BTC", False, 0.001, 50000.0, {"trigger": {"triggerPx": 49000.0, "isMarket": True, "tpsl": "sl"}}, True
    ),
    "order_normalTpsl": lambda: ex.bulk_orders(
        [ENTRY_ORDER, trigger_order("tp", 55000.0), trigger_order("sl", 49000.0)], grouping="normalTpsl"
    ),
    "order_positionTpsl": lambda: ex.bulk_orders(
        [trigger_order("tp", 55000.0), trigger_order("sl", 49000.0)], grouping="positionTpsl"
    ),
    "order_priority": lambda: ex.bulk_orders([ENTRY_ORDER], grouping={"p": 1}),
    "modify": lambda: l1(
        {
            "type": "modify",
            "oid": 123,
            "order": sg.order_request_to_order_wire(limit_modify(123)["order"], ex.info.name_to_asset("BTC")),
        }
    ),
    "batch_modify": lambda: ex.bulk_modify_orders_new([limit_modify(123)]),
    "cancel": lambda: ex.cancel("BTC", 123),
    "cancel_by_cloid": lambda: ex.cancel_by_cloid("BTC", Cloid.from_str(CLOID)),
    "schedule_cancel": lambda: ex.schedule_cancel(SCHEDULE_TIME),
    "update_leverage": lambda: ex.update_leverage(5, "BTC", True),
    "set_referrer": lambda: ex.set_referrer("ABC"),
    "create_sub_account": lambda: ex.create_sub_account("sub1"),
    "usd_class_transfer": lambda: ex.usd_class_transfer(1.5, True),
    "sub_account_transfer": lambda: ex.sub_account_transfer(OTHER, True, 1000000),
    "sub_account_spot_transfer": lambda: ex.sub_account_spot_transfer(OTHER, True, TOKEN, 1.5),
    "vault_transfer": lambda: ex.vault_usd_transfer(OTHER, True, 1000000),
    "vault_modify": lambda: l1(
        {"type": "vaultModify", "vaultAddress": OTHER, "allowDeposits": True, "alwaysCloseOnWithdraw": False}
    ),
    "vault_distribute": lambda: l1({"type": "vaultDistribute", "vaultAddress": OTHER, "usd": 1000000}),
    "usd_send": lambda: ex.usd_transfer(1.5, OTHER),
    "spot_send": lambda: ex.spot_transfer(1.5, OTHER, TOKEN),
    "evm_user_modify": lambda: ex.use_big_blocks(True),
    "token_delegate": lambda: ex.token_delegate(OTHER, 100000000, False),
    "withdraw3": lambda: ex.withdraw_from_bridge(1.5, OTHER),
    "approve_agent_named": lambda: ex.approve_agent("bot"),
    "approve_agent_unnamed": lambda: ex.approve_agent(),
    "approve_builder_fee": lambda: ex.approve_builder_fee(OTHER, "0.001%"),
    "convert_to_multi_sig_user": lambda: compact_signers(lambda: ex.convert_to_multi_sig_user([OTHER], 1)),
    "perp_deploy_register_asset": lambda: ex.perp_deploy_register_asset(
        "test", 1000000, "test:X", 2, "10.0", 10, False, None
    ),
    "perp_deploy_register_asset2": lambda: l1(
        {
            "type": "perpDeploy",
            "registerAsset2": {
                "maxGas": 1000000,
                "assetRequest": {
                    "coin": "test:X",
                    "szDecimals": 2,
                    "oraclePx": "10.0",
                    "marginTableId": 10,
                    "marginMode": "strictIsolated",
                },
                "dex": "test",
                "schema": None,
            },
        }
    ),
    "perp_deploy_halt_trading": lambda: l1(
        {"type": "perpDeploy", "haltTrading": {"coin": "test:X", "isHalted": True}}
    ),
    "perp_deploy_set_oracle": lambda: ex.perp_deploy_set_oracle(
        "test", {"test:X": "10.0"}, [{"test:X": "10.0"}], {"test:X": "10.0"}
    ),
    "user_set_abstraction": lambda: ex.user_set_abstraction(wallet.address, "unifiedAccount"),
    "send_asset": lambda: ex.send_asset(OTHER, "", "spot", TOKEN, 1.5),
    "reserve_request_weight": lambda: l1({"type": "reserveRequestWeight", "weight": 1}),
    "twap_order": lambda: l1(
        {"type": "twapOrder", "twap": {"a": 0, "b": True, "s": "0.1", "r": False, "m": 30, "t": False}}
    ),
    "twap_cancel": lambda: l1({"type": "twapCancel", "a": 0, "t": 77738308}),
    "spotDeploy_registerToken2": lambda: ex.spot_deploy_register_token("TKN", 2, 8, 1000000, "Token"),
    "spotDeploy_userGenesis": lambda: ex.spot_deploy_user_genesis(1, [(OTHER, "100")], [(0, "50")]),
    "spotDeploy_userGenesis_mixedCase": lambda: ex.spot_deploy_user_genesis(1, [(MIXED, "100")], []),
    "spotDeploy_enableFreezePrivilege": lambda: ex.spot_deploy_enable_freeze_privilege(1),
    "spotDeploy_freezeUser": lambda: ex.spot_deploy_freeze_user(1, OTHER, True),
    "spotDeploy_revokeFreezePrivilege": lambda: ex.spot_deploy_revoke_freeze_privilege(1),
    "spotDeploy_genesis": lambda: ex.spot_deploy_genesis(1, "1000000", False),
    "spotDeploy_genesis_noHyperliquidity": lambda: ex.spot_deploy_genesis(1, "1000000", True),
    "spotDeploy_registerSpot": lambda: ex.spot_deploy_register_spot(1, 0),
    "spotDeploy_registerHyperliquidity": lambda: ex.spot_deploy_register_hyperliquidity(0, 1.5, 0.5, 10, None),
    "spotDeploy_registerHyperliquidity_nSeededLevels": lambda: ex.spot_deploy_register_hyperliquidity(0, 1.5, 0.5, 10, 3),
    "spotDeploy_setDeployerTradingFeeShare": lambda: ex.spot_deploy_set_deployer_trading_fee_share(1, "50%"),
    "spotDeploy_enableQuoteToken": lambda: ex.spot_deploy_enable_quote_token(1),
    "cSigner_unjailSelf": lambda: ex.c_signer_unjail_self(),
    "cSigner_jailSelf": lambda: ex.c_signer_jail_self(),
    "cValidator_register": lambda: ex.c_validator_register("1.2.3.4", "val", "desc", False, 500, OTHER, True, 100000000),
    "cValidator_changeProfile_nil": lambda: ex.c_validator_change_profile(None, None, None, False, None, None, None),
    "cValidator_changeProfile_set": lambda: ex.c_validator_change_profile("1.2.3.4", "val", "desc", True, True, 500, OTHER),
    "cValidator_unregister": lambda: ex.c_validator_unregister(),
    "updateIsolatedMargin_add": lambda: ex.update_isolated_margin(1.5, "BTC"),
    "updateIsolatedMargin_remove": lambda: ex.update_isolated_margin(-1.5, "BTC"),
    "createVault": lambda: l1(
        {"type": "createVault", "name": "vault", "description": "description", "initialUsd": 1000000, "nonce": NONCE}
    ),
    "sendToEvmWithData": lambda: send_to_evm_with_data("0xdeadbeef"),
    "multiSig": lambda: ex.multi_sig(OTHER, {"type": "noop"}, [{"r": "0x01", "s": "0x02", "v": 27}], NONCE),
    "multiSig_l1ActionPayload": multi_sig_l1_payload,
    "multiSig_userSignedActionPayload": multi_sig_user_signed_payload,
    "borrowLend": lambda: l1({"type": "borrowLend", "operation": "supply", "token": 0, "amount": "1.5"}),
    "trailingStop_pct": lambda: l1(
        {
            "type": "trailingStop",
            "asset": 0,
            "isBuy": True,
            "sz": "0.001",
            "reduceOnly": False,
            "retracement": {"pct": "1.5"},
            "activationPx": None,
        }
    ),
    "trailingStop_px": lambda: l1(
        {
            "type": "trailingStop",
            "asset": 0,
            "isBuy": False,
            "sz": "0.001",
            "reduceOnly": True,
            "retracement": {"px": "500"},
            "activationPx": "50000",
        }
    ),
    "topUpIsolatedOnlyMargin": lambda: l1({"type": "topUpIsolatedOnlyMargin", "asset": 0, "leverage": "5"}),
    "noop": lambda: ex.noop(NONCE),
    "agentSetAbstraction": lambda: ex.agent_set_abstraction("u"),
    "gossipPriorityBid": lambda: ex.gossip_priority_bid(1, "1.2.3.4", 1000000),
    "agentSendAsset": lambda: l1(
        {
            "type": "agentSendAsset",
            "destination": OTHER,
            "sourceDex": "",
            "destinationDex": "spot",
            "token": TOKEN,
            "amount": "1.5",
            "fromSubAccount": "",
            "nonce": NONCE,
        }
    ),
    "cDeposit": lambda: user(
        {"type": "cDeposit", "wei": 100000000, "nonce": NONCE},
        [
            {"name": "hyperliquidChain", "type": "string"},
            {"name": "wei", "type": "uint64"},
            {"name": "nonce", "type": "uint64"},
        ],
        "HyperliquidTransaction:CDeposit",
    ),
    "cWithdraw": lambda: user(
        {"type": "cWithdraw", "wei": 100000000, "nonce": NONCE},
        [
            {"name": "hyperliquidChain", "type": "string"},
            {"name": "wei", "type": "uint64"},
            {"name": "nonce", "type": "uint64"},
        ],
        "HyperliquidTransaction:CWithdraw",
    ),
    "cDeposit2": lambda: l1({"type": "cDeposit2", "wei": 100000000}),
    "claimRewards": lambda: l1({"type": "claimRewards"}),
    "hip3LiquidatorTransfer": lambda: l1({"type": "hip3LiquidatorTransfer", "dex": "xyz", "ntl": 1000000, "isDeposit": True}),
    "userOutcome_splitOutcome": lambda: l1({"type": "userOutcome", "splitOutcome": {"outcome": 1, "amount": "1.5"}}),
    "userOutcome_mergeOutcome": lambda: l1({"type": "userOutcome", "mergeOutcome": {"outcome": 1, "amount": "1.5"}}),
    "userOutcome_mergeOutcome_nilAmount": lambda: l1({"type": "userOutcome", "mergeOutcome": {"outcome": 1, "amount": None}}),
    "userOutcome_mergeQuestion": lambda: l1({"type": "userOutcome", "mergeQuestion": {"question": 1, "amount": "1.5"}}),
    "userOutcome_mergeQuestion_nilAmount": lambda: l1({"type": "userOutcome", "mergeQuestion": {"question": 1, "amount": None}}),
    "userOutcome_negateOutcome": lambda: l1({"type": "userOutcome", "negateOutcome": {"question": 1, "outcome": 2, "amount": "1.5"}}),
}

os.makedirs(out_dir, exist_ok=True)
for name, run in cases.items():
    rec.clear()
    run()
    vector = {
        "kind": rec["kind"],
        "action": rec["action"],
        "nonce": rec["nonce"],
        "vaultAddress": rec["vault"],
        "expiresAfter": rec["expires"],
        "isMainnet": rec["mainnet"],
        "primaryType": rec["primary"],
        "types": rec["types"],
        "signature": {"r": rec["signature"]["r"], "s": rec["signature"]["s"], "v": rec["signature"]["v"]},
    }
    with open(os.path.join(out_dir, name + ".json"), "w") as f:
        json.dump(vector, f, indent=2)
        f.write("\n")
print(len(cases), "vectors written to", out_dir)
