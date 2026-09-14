# BuySomething (CIGC)

BSC 主网已部署：

- **合约**: `0xCb63733FB936c7B3f147C757D383645e55769bF3`
- **部署交易**: `0xc69908d99641f48d7b1bce9a26c35ae74ae42747c0faac76d04b5005c737dda5`
- **部署者**: `0x06d1A66345BA7366135E33db89961ca6C545A6be`
- **USDT**: `0x55d398326f99059fF775485246999027B3197955`
- **分账**: 89% / 5% / 5% / 1%（对齐 `configs/config.yaml` 四个收款地址）

## 用户支付

dapp 商城下单后自动走钱包：

1. `USDT.approve(0xCb63733FB936c7B3f147C757D383645e55769bF3, amount)`
2. `BuySomething.buy(amount)` — `amount` 为 18 位小数原始量（如 `1000e18`）

源码已写死分账（无需 constructor 传参）：

| 比例 | 地址 |
|---|---|
| 89% | `0xdf4cbc6c9c4f084b6e75177852c6a29f206b8fe1` |
| 5% | `0x1c630cC605C96B1Bbcc1aCE704a7Ba0f259E6098` |
| 5% | `0xa1e54373034aae3c00df1b9b89b20d2df55e2cad` |
| 1% | `0xE7Da6c5D90f6a88fEEa228d5C9a0611c61F7500D` |

待支付订单可在「订单」页再点「去支付」。

合约会按比例 `transferFrom` 到四个收款地址；`Transfer.from` 仍是用户钱包，现有 `deposit_scan` 可继续核销。

## 重新部署

```bash
cd contracts/buy-something
npx hardhat run scripts/deploy-buy-something.js --network bsc
```

说明：原路径 `D:\智能合约\...\BuySomething.sol` 在 WSL 下只读，改写后的源码在本目录 `contracts/BuySomething.sol`。

## 源码加固（未上链，需重新部署才生效）

| 问题 | 要不要改 | 做法 |
|---|---|---|
| `users` / `usersAmount` 无限增长 | **要** | 已删数组；历史只靠 `Bought` / `SplitPaid`，计数用 `buyCount` |
| 无暂停 | **建议** | owner 可 `pause` / `unpause`，暂停后 `buy` 拒绝 |
| 收款地址无上限 | **建议** | constructor 最多 8 个；现网只有 4 个 |
| 未校验 8 位精度 | **建议** | `amount % 1e10 == 0`，对齐后端 `Round(8)` |

当前主网 `0xCb63…9bF3` 仍是旧实现。换新合约后改 `VITE_BUY` / `CIGC_BUY_CONTRACT`，并让 owner 保管部署钱包。
