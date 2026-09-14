//SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

contract BuySomething is ReentrancyGuard {
    address public usdt = 0x55d398326f99059fF775485246999027B3197955; // AIX-USDT

    address[] public users;
    uint256[] public usersAmount;

    constructor() {}

    function buy(uint256 num) external nonReentrant {
        require(5 <= num, "err num");

        uint256 amount = num * 10**18;

        IERC20(usdt).transferFrom(msg.sender, 0xa1E54373034aaE3C00Df1b9b89B20d2df55e2CAD, amount * 80 / 100);
        IERC20(usdt).transferFrom(msg.sender, 0xE7Da6c5D90f6a88fEEa228d5C9a0611c61F7500D, amount * 10 / 100);
        IERC20(usdt).transferFrom(msg.sender, 0x623ecc54647605C220199F4d273Cf9F43fDdD5c1, amount * 5 / 100);
        IERC20(usdt).transferFrom(msg.sender, 0x907D9173ab226C698C178981c4D135f8168dD6eb, amount * 3 / 100);
        IERC20(usdt).transferFrom(msg.sender, 0x279F2B0B788b50c90ceCd74C9134D152083A87B7, amount * 15 / 1000);
        IERC20(usdt).transferFrom(msg.sender, 0xd3E7fE539c291010B8948Fe19372ac9223288109, amount - (amount * 80 / 100 + amount * 10 / 100 + amount * 5 / 100 + amount * 3 / 100 + amount * 15 / 1000));

        users.push(msg.sender);
        usersAmount.push(num);
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
}
