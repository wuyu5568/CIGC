require("dotenv").config();
require("@nomicfoundation/hardhat-ethers");
require("@nomicfoundation/hardhat-verify");

const PRIVATE_KEY = process.env.PRIVATE_KEY || "";
const RPC_URL = process.env.RPC_URL || "https://bsc-dataseed.binance.org/";
const BSCSCAN_API_KEY = process.env.BSCSCAN_API_KEY || "";

module.exports = {
  solidity: {
    version: process.env.SOLC_VERSION || "0.8.26",
    settings: {
      optimizer: {
        enabled: true,
        runs: Number(process.env.OPTIMIZER_RUNS || 200),
      },
      viaIR: true,
      evmVersion: process.env.EVM_VERSION || "cancun",
    },
  },
  networks: {
    bsc: {
      url: RPC_URL,
      chainId: 56,
      accounts: PRIVATE_KEY ? [PRIVATE_KEY] : [],
    },
    eoeo: {
      url: process.env.EOEO_RPC_URL || "https://rpc1.eoeo.info",
      chainId: 86233268,
      accounts: PRIVATE_KEY ? [PRIVATE_KEY] : [],
    },
  },
  etherscan: {
    apiKey: BSCSCAN_API_KEY,
  },
};