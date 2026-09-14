# BuySomething (CIGC)

BSC 主网已部署（当前分账 80 / 10 / 5 / 3 / 1.5 / 0.5）：

- **合约**: `0x162bfFAcf7a89Bb6EbA05972C1DE0E1e97617c18`
- **部署交易**: `0xbbb6ac3c8be28868ace48f22c67cd5cdff8f16cc10c61c56edc385e75deb86fb`
- **部署者**: `0x06d1A66345BA7366135E33db89961ca6C545A6be`
- **USDT**: `0x55d398326f99059fF775485246999027B3197955`

旧合约 `0xCb63733FB936c7B3f147C757D383645e55769bF3` 为 89/5/5/1，已停用。

## 用户支付

dapp 充值页：

1. `USDT.approve(0x162bfFAcf7a89Bb6EbA05972C1DE0E1e97617c18, type(uint256).max)`
2. `BuySomething.buy(num)` — `num` 为整数 USDT（如 `1000`），合约内再乘 `10**18`

源码已写死分账（无需 constructor 传参）：

| 比例 | 地址 |
|---|---|
| 80% | `0xa1E54373034aaE3C00Df1b9b89B20d2df55e2CAD` |
| 10% | `0xE7Da6c5D90f6a88fEEa228d5C9a0611c61F7500D` |
| 5% | `0x623ecc54647605C220199F4d273Cf9F43fDdD5c1` |
| 3% | `0x907D9173ab226C698C178981c4D135f8168dD6eb` |
| 1.5% | `0x279F2B0B788b50c90ceCd74C9134D152083A87B7` |
| 0.5% | `0xd3E7fE539c291010B8948Fe19372ac9223288109` |

合约按比例 `transferFrom` 到六个收款地址；`Transfer.from` 仍是用户钱包，现有 `deposit_scan` 可按份额核销入账。

换合约后改 `VITE_BUY` / `CIGC_BUY_CONTRACT`。

## 重新部署

```bash
cd contracts/buy-something
npx hardhat run scripts/deploy-buy-something.js --network bsc
```
