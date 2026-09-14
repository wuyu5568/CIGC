require("dotenv").config();
const hre = require("hardhat");
const { gas, deploy, printSigner } = require("./lib/utils.cjs");

// 对齐 CIGC configs/config.yaml
const CIGC_USDT = "0x55d398326f99059fF775485246999027B3197955";
const CIGC_RECEIVERS = [
  "0xa1e54373034aae3c00df1b9b89b20d2df55e2cad",
  "0xE7Da6c5D90f6a88fEEa228d5C9a0611c61F7500D",
  "0x623ecc54647605c220199f4d273cf9f43fddd5c1",
  "0x907D9173ab226C698C178981c4D135f8168dD6eb",
  "0x279F2B0B788b50c90ceCd74C9134D152083A87B7",
  "0xd3E7fE539c291010B8948Fe19372ac9223288109",
];
const CIGC_PERCENTS = [80, 10, 5, 3, 1.5, 0.5];

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
