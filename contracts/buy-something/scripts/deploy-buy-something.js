require("dotenv").config();
const hre = require("hardhat");
const { gas, deploy, printSigner } = require("./lib/utils.cjs");

// 对齐 CIGC configs/config.yaml
const CIGC_USDT = "0x55d398326f99059fF775485246999027B3197955";
const CIGC_RECEIVERS = [
  "0xdf4cbc6c9c4f084b6e75177852c6a29f206b8fe1",
  "0x1c630cC605C96B1Bbcc1aCE704a7Ba0f259E6098",
  "0xa1e54373034aae3c00df1b9b89b20d2df55e2cad",
  "0xE7Da6c5D90f6a88fEEa228d5C9a0611c61F7500D",
];
const CIGC_PERCENTS = [89, 5, 5, 1];

async function main() {
  const [signer] = await hre.ethers.getSigners();
  printSigner(signer);

  console.log("network:", hre.network.name);
  console.log("usdt (hardcoded):", CIGC_USDT);
  console.log("receivers (hardcoded):", CIGC_RECEIVERS);
  console.log("percents (hardcoded):", CIGC_PERCENTS);

  const bal = await hre.ethers.provider.getBalance(signer.address);
  console.log("deployer bnb:", hre.ethers.formatEther(bal));

  const contract = await deploy("BuySomething", [], {
    gasLimit: gas("BUY_SOMETING_DEPLOY_GAS", 2_000_000),
  });

  console.log("BuySomething deployed at:", await contract.getAddress());
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
