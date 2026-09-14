// 导入模块
import detectEthereumProvider from '@metamask/detect-provider' // 用于检测以太坊提供者（例如MetaMask）
import { ethers, BigNumber } from 'ethers' // 导入 ethers 库中的 ethers 和 BigNumber 对象
import { showFailToast } from 'vant' // 导入用于显示提示信息的组件
import abi from './abi.json' // 导入智能合约 ABI
import lang from '@/i18n/index'

/* 链接钱包类 */
export class ETH {
  public static provider: any = undefined // 提供者
  public static account: string = '' // 钱包地址
  public static signer: any = undefined // 用户签名者
  // 链接钱包返回钱包地址
  public static async getAccount(): Promise<string> {
    const ethereum: any = await detectEthereumProvider()
    if (!ethereum) {
      showFailToast(lang('请安装钱包'))
      throw lang('请安装钱包')
    }

    try {
      // 先尝试切换
      await ethereum.request({
        method: 'wallet_switchEthereumChain',
        params: [{ chainId: '0x38' }]
      })
    } catch (error: any) {
      // 4902 = 链不存在，走添加流程
      if (error.code === 4902) {
        await ethereum.request({
          method: 'wallet_addEthereumChain',
          params: [
            {
              chainId: '0x38',
              chainName: 'Binance Smart Chain',
              rpcUrls: ['https://bsc-dataseed.binance.org/'],
              nativeCurrency: {
                name: 'BNB',
                symbol: 'BNB',
                decimals: 18
              },
              blockExplorerUrls: ['https://bscscan.com']
            }
          ]
        })
      } else {
        throw error
      }
    }

    ETH.provider = new ethers.providers.Web3Provider(ethereum)
    const chainId = Number(await ethereum.request({ method: 'eth_chainId' }))
    console.log('chainId', chainId)
    // alert(chainId)
    // if (chainId !== 56 && chainId !== 1) {
    //     showFailToast(lang("请连接BSC主网"));
    //     throw lang("请连接BSC主网");
    // }
    ETH.account = ethers.utils.getAddress(
      (await ethereum.request({ method: 'eth_requestAccounts' }))[0]
    )
    ETH.signer = ETH.provider.getSigner()
    return ETH.account
  }
  public static async getUsdtValueFromIsps(amount: any): Promise<string> {
    try {
      const contract = new ethers.Contract(import.meta.env.VITE_CONTRACT, abi.ERC20, ETH.signer) // 创建合约对象
      const path = [import.meta.env.VITE_BNBS, import.meta.env.VITE_USDT]
      console.log('contract', contract)
      const amountsOut = await contract.getAmountsOut(ethers.utils.parseEther(amount), path)
      return ethers.utils.formatUnits(amountsOut[1].toString(), 'ether')
    } catch (error) {
      console.log(error)
      return Promise.reject(error)
    }
  }
  public static async getBalance() {
    const contract = new ethers.Contract(import.meta.env.VITE_BNBS, abi.ERC20, ETH.signer) // 创建合约对象
    const balance = await contract.balanceOf(ETH.account)
    return ethers.utils.formatUnits(balance, '18')
  }
  public static async getUSDTBalance() {
    try {
      const ERC20_ABI = [
        'function balanceOf(address) view returns (uint256)',
        'function decimals() view returns (uint8)'
      ]

      const usdtAddress = import.meta.env.VITE_USDT

      // 1️⃣ 校验合约
      const code = await ETH.provider.getCode(usdtAddress)
      if (code === '0x') {
        throw new Error('USDT 地址不是合约，请检查链或地址')
      }

      // 2️⃣ 用 provider（只读）
      const contract = new ethers.Contract(usdtAddress, ERC20_ABI, ETH.provider)

      const balance = await contract.balanceOf(ETH.account)
      const decimals = await contract.decimals()

      const value = ethers.utils.formatUnits(balance, decimals)

      return Number(value).toFixed(4)
    } catch (error) {
      console.error('获取 USDT 余额失败:', error)
      return '0'
    }
  }
  // 签名
  public static async signMessage(): Promise<string> {
    console.log('signMessage', ETH.signer)
    return await ETH.signer.signMessage(ETH.account)
  }
  static parseUnits(n: any, dec: number): BigNumber {
    return ethers.utils.parseUnits(`${n}`, dec)
  }
  public static formatUnits(n: any, dec: number): string {
    return ethers.utils.formatUnits(n, dec)
  }
  static format_address(v: string, n: number = 8): string {
    const reg = new RegExp(`^(.{${n}})(.*)(.{${n}})$`, 'ig')
    return v.replace(reg, '$1...$3')
  }
}
/* 合约类 */
export class Contract {
  public address!: string // 合约地址
  public abiName!: string // abi名称
  constructor(address: string, abiName: string) {
    this.address = address
    this.abiName = abiName
  }
  getInsance() {
    return new ethers.Contract(this.address, (abi as any)[this.abiName], ETH.provider).connect(
      ETH.signer
    )
  }
  async call(methods: string, params: any[] = []): Promise<any> {
    return await this.getInsance()[methods](...params)
  }
  async send(methods: string, params: any[] = []): Promise<void> {
    try {
      let tx: any = {}
      try {
        tx = await this.getInsance()[methods](...params)
      } catch (error: any) {
        console.log(error.reason, /balance/gi.test(error.reason))
        // console.log(1111, /^balance/ig.test(error.toString()))
        if (/balance/gi.test(error.reason)) {
          throw error
        }
        if (!(error.code === 'INVALID_ARGUMENT' && error.reason === 'missing from address')) {
          throw error
        }
        tx.hash = error.transactionHash
      }
      let receipt = await ETH.provider.waitForTransaction(tx.hash)
      if (receipt.status === 1) {
        // alert("交易成功");
      } else {
        throw lang('交易失败')
      }
    } catch (error: any) {
      let msg: any = ''
      if (error.data) msg = error.data.message
      else if (/balance/gi.test(error.toString())) msg = lang('余额不足')
      else if (/^Error/gi.test(error.toString())) msg = lang('参数错误')
      else if (error.message) msg = error.message
      else msg = error
      showFailToast(msg)
      throw '交易失败'
    }
  }
  /* 判断是否授权 */
  async allowance(address: string): Promise<boolean> {
    let res = await this.call('allowance', [ETH.account, address])
    return Number(res) > 0
  }
  /* 授权 */
  async approve(address: string): Promise<void> {
    await this.send('approve', [
      address,
      '115792089237316195423570985008687907853269984665640564039457584007913129639935'
    ])
  }
}
