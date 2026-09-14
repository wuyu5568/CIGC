const { ethers } = require("hardhat");

function isZero(addr) {
  return !addr || /^0x0{40}$/i.test(addr);
}

function req(name) {
  const v = process.env[name];
  if (!v || !String(v).trim()) {
    throw new Error(`Missing env: ${name}`);
  }
  return String(v).trim();
}

function reqAddr(name) {
  const v = req(name);
  if (!ethers.isAddress(v)) {
    throw new Error(`Invalid address in ${name}: ${v}`);
  }
  return ethers.getAddress(v);
}

function optAddr(name) {
  const v = process.env[name];
  if (!v || !String(v).trim()) return null;
  if (!ethers.isAddress(v)) {
    throw new Error(`Invalid address in ${name}: ${v}`);
  }
  return ethers.getAddress(v);
}

function parseSupply18(name) {
  return ethers.parseUnits(req(name), 18);
}

function gas(name, fallback) {
  const v = process.env[name];
  return BigInt(v && String(v).trim() ? v : fallback);
}

async function waitTx(tx, label) {
  console.log(`tx sent [${label}]:`, tx.hash);
  const receipt = await tx.wait();
  console.log(`tx confirmed [${label}] block=${receipt.blockNumber}`);
  return receipt;
}

async function deploy(factoryName, args, opts = {}) {
  const factory = await ethers.getContractFactory(factoryName);
  const contract = await factory.deploy(...args, opts);
  const tx = contract.deploymentTransaction();
  console.log(`${factoryName} deploy tx:`, tx.hash);
  await contract.waitForDeployment();
  const address = await contract.getAddress();
  console.log(`${factoryName} deployed at:`, address);
  return contract;
}

async function maybeTransferOwnership(contract, name, newOwner) {
  if (!newOwner || isZero(newOwner)) return;
  const currentOwner = await contract.owner();
  if (ethers.getAddress(currentOwner) === ethers.getAddress(newOwner)) return;
  await waitTx(await contract.transferOwnership(newOwner), `${name}.transferOwnership(${newOwner})`);
}

async function maybeSetInitAccount(contract, name, newInitAccount) {
  if (!newInitAccount || isZero(newInitAccount)) return;
  const current = await contract.initAccount();
  if (ethers.getAddress(current) === ethers.getAddress(newInitAccount)) return;
  await waitTx(await contract.setInitAccount(newInitAccount), `${name}.setInitAccount(${newInitAccount})`);
}

function printSigner(signer) {
  console.log("deployer:", signer.address);
}

module.exports = {
  req,
  reqAddr,
  optAddr,
  parseSupply18,
  gas,
  waitTx,
  deploy,
  maybeTransferOwnership,
  maybeSetInitAccount,
  printSigner,
  isZero,
};
