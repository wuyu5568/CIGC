// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { ERC20Burnable } from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import { ERC20Permit } from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Permit.sol";
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

interface IUniswapV2Factory {
    function getPair(address tokenA, address tokenB) external view returns (address pair);
    function createPair(address tokenA, address tokenB) external returns (address pair);
}

contract IspayToken is ERC20, ERC20Burnable, ERC20Permit, Ownable {
    // ---------------------------
    // Errors
    // ---------------------------
    error TradingClosed();
    error ZeroAddress();
    error FeeTooHigh();
    error MustKeepMinimum(uint256 minimum);

    // ---------------------------
    // Constants
    // ---------------------------
    uint256 public constant TOTAL_SUPPLY = 1_000_000_000 * 1e18; // 10亿
    uint256 public constant BPS_DENOMINATOR = 10_000;           // 10000 = 100%

    // ---------------------------
    // BSC / PancakeSwap V2 固定地址（硬编码）
    // ---------------------------
    address public constant PANCAKE_ROUTER_V2 =
        0x10ED43C718714eb63d5aA57B78B54704E256024E;

    address public constant PANCAKE_FACTORY_V2 =
        0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73;

    // BSC USDT (BEP20 / Binance-Peg)
    address public constant USDT =
        0x55d398326f99059fF775485246999027B3197955;

    // ---------------------------
    // Pair
    // ---------------------------
    address public pair; // ispay/USDT 主交易对（构造函数自动创建/获取）

    // ---------------------------
    // Trading / Tax Settings
    // ---------------------------
    bool public tradingEnabled; // 默认关闭交易

    // 买税/卖税（bps）
    uint16 public buyFeeBps = 300;   // 3%
    uint16 public sellFeeBps = 1000; // 10%

    // 手续费地址（初始化用你给的）
    address public buyFeeWallet = 0xa1e54373034aae3c00df1b9b89b20d2df55e2cad;
    address public sellFeeWallet = 0xDf4cbC6c9c4F084B6E75177852C6a29f206b8Fe1;

    // 每个地址最少保留 0.000001 个币（18 decimals 下就是 1e12）
    uint256 public minRemainAmount = 1e12;

    // AMM 交易对标记：用于识别买卖（pair 可多个）
    mapping(address => bool) public isAMMPair;

    // 白名单：交易关闭时放行（from 或 to 在白名单即可）
    mapping(address => bool) public whitelist;

    // 手续费豁免
    mapping(address => bool) public feeExempt;

    // 最小保留豁免（pair/router/owner 等建议豁免）
    mapping(address => bool) public minRemainExempt;

    // ---------------------------
    // Events
    // ---------------------------
    event TradingStatusChanged(bool enabled);
    event PairInitialized(address indexed pair, address indexed tokenB);
    event AMMPairUpdated(address indexed pair, bool indexed enabled);
    event WhitelistUpdated(address indexed account, bool indexed enabled);
    event FeeExemptUpdated(address indexed account, bool indexed enabled);
    event MinRemainExemptUpdated(address indexed account, bool indexed enabled);
    event FeeWalletsUpdated(address indexed buyWallet, address indexed sellWallet);
    event FeeRatesUpdated(uint16 buyFeeBps, uint16 sellFeeBps);
    event MinRemainAmountUpdated(uint256 minRemainAmount);

    // ---------------------------
    // Constructor
    // ---------------------------
    constructor()
        ERC20("ispay", "ispay")
        ERC20Permit("ispay")
        Ownable(0x6F311dfCBDB9bfc227f2A6bBafDef1eB6EfDF771) // 你的权限地址作为 owner
    {
        // 初始：关闭交易
        tradingEnabled = false;

        // ✅ 初始化/创建 ispay-USDT 交易对
        address _pair = IUniswapV2Factory(PANCAKE_FACTORY_V2).getPair(address(this), USDT);
        if (_pair == address(0)) {
            _pair = IUniswapV2Factory(PANCAKE_FACTORY_V2).createPair(address(this), USDT);
        }
        pair = _pair;

        // 标记为 AMM Pair（用于买/卖收税）
        isAMMPair[_pair] = true;
        emit AMMPairUpdated(_pair, true);

        // pair 豁免“保留最小数量”
        minRemainExempt[_pair] = true;
        emit MinRemainExemptUpdated(_pair, true);

        emit PairInitialized(_pair, USDT);

        // 默认白名单：owner + 合约 + 税钱包 + router
        whitelist[owner()] = true;
        whitelist[address(this)] = true;
        whitelist[buyFeeWallet] = true;
        whitelist[sellFeeWallet] = true;
        whitelist[PANCAKE_ROUTER_V2] = true; // 方便关盘时仍可移除流动性等
        emit WhitelistUpdated(PANCAKE_ROUTER_V2, true);

        // 默认手续费豁免：owner + 合约 + 税钱包 + router
        feeExempt[owner()] = true;
        feeExempt[address(this)] = true;
        feeExempt[buyFeeWallet] = true;
        feeExempt[sellFeeWallet] = true;

        // 很关键：router 必须手续费豁免，否则“移除流动性(pair->router)”会被当成 buy 收税
        feeExempt[PANCAKE_ROUTER_V2] = true;
        emit FeeExemptUpdated(PANCAKE_ROUTER_V2, true);

        // 默认最小保留豁免：owner + 合约 + 税钱包 + router
        minRemainExempt[owner()] = true;
        minRemainExempt[address(this)] = true;
        minRemainExempt[buyFeeWallet] = true;
        minRemainExempt[sellFeeWallet] = true;
        minRemainExempt[PANCAKE_ROUTER_V2] = true;
        emit MinRemainExemptUpdated(PANCAKE_ROUTER_V2, true);

        // 铸造总量给 owner
        _mint(owner(), TOTAL_SUPPLY);
    }

    function decimals() public pure override returns (uint8) {
        return 18;
    }

    // ---------------------------
    // Owner Controls
    // ---------------------------

    function openTrading() external onlyOwner {
        tradingEnabled = true;
        emit TradingStatusChanged(true);
    }

    function closeTrading() external onlyOwner {
        tradingEnabled = false;
        emit TradingStatusChanged(false);
    }

    function setWhitelist(address account, bool enabled) external onlyOwner {
        if (account == address(0)) revert ZeroAddress();
        whitelist[account] = enabled;
        emit WhitelistUpdated(account, enabled);
    }

    function batchSetWhitelist(address[] calldata accounts, bool enabled) external onlyOwner {
        uint256 len = accounts.length;
        for (uint256 i = 0; i < len; i++) {
            address a = accounts[i];
            if (a == address(0)) revert ZeroAddress();
            whitelist[a] = enabled;
            emit WhitelistUpdated(a, enabled);
        }
    }

    function setAMMPair(address pair_, bool enabled) external onlyOwner {
        if (pair_ == address(0)) revert ZeroAddress();
        isAMMPair[pair_] = enabled;
        emit AMMPairUpdated(pair_, enabled);

        // 同步：pair 默认豁免“保留最小数量”
        minRemainExempt[pair_] = enabled;
        emit MinRemainExemptUpdated(pair_, enabled);
    }

    function setFeeExempt(address account, bool enabled) external onlyOwner {
        if (account == address(0)) revert ZeroAddress();
        feeExempt[account] = enabled;
        emit FeeExemptUpdated(account, enabled);
    }

    function setMinRemainExempt(address account, bool enabled) external onlyOwner {
        if (account == address(0)) revert ZeroAddress();
        minRemainExempt[account] = enabled;
        emit MinRemainExemptUpdated(account, enabled);
    }

    function setFeeRates(uint16 newBuyFeeBps, uint16 newSellFeeBps) external onlyOwner {
        if (newBuyFeeBps > BPS_DENOMINATOR || newSellFeeBps > BPS_DENOMINATOR) revert FeeTooHigh();
        buyFeeBps = newBuyFeeBps;
        sellFeeBps = newSellFeeBps;
        emit FeeRatesUpdated(newBuyFeeBps, newSellFeeBps);
    }

    function setFeeWallets(address newBuyFeeWallet, address newSellFeeWallet) external onlyOwner {
        if (newBuyFeeWallet == address(0) || newSellFeeWallet == address(0)) revert ZeroAddress();
        buyFeeWallet = newBuyFeeWallet;
        sellFeeWallet = newSellFeeWallet;
        emit FeeWalletsUpdated(newBuyFeeWallet, newSellFeeWallet);
    }

    /// @notice 修改“每个地址保留多少币”（默认 1e12 = 0.000001 token）
    /// 设为 0 即关闭“保留”限制
    function setMinRemainAmount(uint256 newMinRemainAmount) external onlyOwner {
        minRemainAmount = newMinRemainAmount;
        emit MinRemainAmountUpdated(newMinRemainAmount);
    }

    /// @notice 增发（owner）
    function mint(address to, uint256 amount) external onlyOwner {
        if (to == address(0)) revert ZeroAddress();
        _mint(to, amount);
    }

    // ---------------------------
    // Core Transfer Logic
    // ---------------------------
    function _update(address from, address to, uint256 value) internal override(ERC20) {
        // mint / burn 不收税、不限制
        if (from == address(0) || to == address(0)) {
            super._update(from, to, value);
            return;
        }

        // 交易关闭时：必须 from 或 to 在白名单才允许转账（含买卖/普通转账）
        if (!tradingEnabled) {
            if (!whitelist[from] && !whitelist[to]) revert TradingClosed();
        }

        // 非豁免地址转出后必须至少剩 minRemainAmount（默认 0.000001）
        if (minRemainAmount > 0 && !minRemainExempt[from]) {
            uint256 bal = balanceOf(from);
            if (bal >= value) {
                uint256 afterBal = bal - value;
                if (afterBal < minRemainAmount) revert MustKeepMinimum(minRemainAmount);
            }
        }

        // 手续费豁免：直接转
        if (feeExempt[from] || feeExempt[to]) {
            super._update(from, to, value);
            return;
        }

        // 识别买卖：只对 AMM pair 收税（普通转账不收）
        uint256 feeAmount = 0;

        // Buy: pair -> user
        if (isAMMPair[from] && buyFeeBps > 0) {
            feeAmount = (value * buyFeeBps) / BPS_DENOMINATOR;
            if (feeAmount > 0) {
                super._update(from, buyFeeWallet, feeAmount);
            }
        }
        // Sell: user -> pair
        else if (isAMMPair[to] && sellFeeBps > 0) {
            feeAmount = (value * sellFeeBps) / BPS_DENOMINATOR;
            if (feeAmount > 0) {
                super._update(from, sellFeeWallet, feeAmount);
            }
        }

        uint256 netAmount = value - feeAmount;
        super._update(from, to, netAmount);
    }
}
