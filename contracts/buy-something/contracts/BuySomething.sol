//SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

/// @title BuySomething — CIGC 预售收款
/// @notice 收款地址和比例写死在合约里，对齐 configs/config.yaml：89 / 5 / 5 / 1。
contract BuySomething is ReentrancyGuard {
    /// @dev 业务金额 8 位小数 → USDT 18 位时的粒度 1e10。
    uint256 private constant ROUND8_UNIT = 1e10;

    address public usdt = 0x55d398326f99059fF775485246999027B3197955; // bsc-USDT

    address public constant RECEIVER_89 = 0xdf4cbc6c9c4f084b6e75177852c6a29f206b8fe1;
    address public constant RECEIVER_5A = 0x1c630cC605C96B1Bbcc1aCE704a7Ba0f259E6098;
    address public constant RECEIVER_5B = 0xa1e54373034aae3c00df1b9b89b20d2df55e2cad;
    address public constant RECEIVER_1 = 0xE7Da6c5D90f6a88fEEa228d5C9a0611c61F7500D;

    address[] public users;
    uint256[] public usersAmount;

    constructor() {}

    /// @param amount USDT 总量（18 位小数，如 1000e18），必须能整除到 8 位业务精度。
    function buy(uint256 amount) external nonReentrant {
        require(amount > 0, "err num");
        require(amount % ROUND8_UNIT == 0, "precision");

        uint256 share89 = _round8MulDiv100(amount, 89);
        uint256 share5a = _round8MulDiv100(amount, 5);
        uint256 share5b = _round8MulDiv100(amount, 5);
        uint256 share1 = amount - share89 - share5a - share5b;
        require(share89 > 0 && share5a > 0 && share5b > 0 && share1 > 0, "share zero");

        IERC20 token = IERC20(usdt);
        require(token.transferFrom(msg.sender, RECEIVER_89, share89), "pay 89");
        require(token.transferFrom(msg.sender, RECEIVER_5A, share5a), "pay 5a");
        require(token.transferFrom(msg.sender, RECEIVER_5B, share5b), "pay 5b");
        require(token.transferFrom(msg.sender, RECEIVER_1, share1), "pay 1");

        users.push(msg.sender);
        usersAmount.push(amount);
    }

    function getUserLength() public view returns (uint256) {
        return users.length;
    }

    function getUsers() public view returns (address[] memory) {
        return users;
    }

    function getUsersByIndex(uint256 startIndex, uint256 endIndex) public view returns (address[] memory) {
        require(endIndex >= startIndex, "bad range");
        require(endIndex < users.length, "out of range");

        address[] memory data = new address[](endIndex + 1 - startIndex);
        for (uint256 i = startIndex; i <= endIndex; i++) {
            data[i - startIndex] = users[i];
        }
        return data;
    }

    function getUsersAmountByIndex(uint256 startIndex, uint256 endIndex) public view returns (uint256[] memory) {
        require(endIndex >= startIndex, "bad range");
        require(endIndex < usersAmount.length, "out of range");

        uint256[] memory data = new uint256[](endIndex + 1 - startIndex);
        for (uint256 i = startIndex; i <= endIndex; i++) {
            data[i - startIndex] = usersAmount[i];
        }
        return data;
    }

    /// @dev RoundHalfEven：Round8(amount * percent / 100)，最后一笔吃余数。
    function _round8MulDiv100(uint256 amount, uint256 percent) private pure returns (uint256) {
        uint256 denom = 100 * ROUND8_UNIT;
        uint256 num = amount * percent;
        uint256 quot = num / denom;
        uint256 rem = num % denom;
        uint256 half = denom / 2;
        if (rem > half || (rem == half && (quot & 1) == 1)) {
            unchecked {
                quot += 1;
            }
        }
        return quot * ROUND8_UNIT;
    }
}
