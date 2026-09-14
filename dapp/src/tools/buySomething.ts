import { ethers } from 'ethers'
import { ETH, Contract } from '@/tools/contract'
import userPerson from '@/pinia/person'
import lang from '@/i18n/index'

const USDT = new Contract(import.meta.env.VITE_USDT, 'ERC20')

export function resolveBuyAddress(fromApi?: string): string {
  return String(fromApi || import.meta.env.VITE_BUY || '').trim()
}

export async function payBuySomething(amount: string | number, buyAddr?: string): Promise<void> {
  const addr = resolveBuyAddress(buyAddr)
  if (!addr) {
    throw lang('未配置支付合约')
  }
  await ETH.getAccount()
  const person = userPerson()
  const bound = String(
    person.userinfo.address || person.address || localStorage.getItem('account') || ''
  ).toLowerCase()
  if (bound && ETH.account.toLowerCase() !== bound) {
    throw lang('请使用登录钱包支付')
  }
  const raw = ethers.utils.parseUnits(String(amount).replace(/,/g, '').trim(), 18)
  if (raw.lte(0) || !raw.mod(ethers.BigNumber.from('10000000000')).isZero()) {
    throw lang('金额错误')
  }
  const allowance = await USDT.call('allowance', [ETH.account, addr])
  if (ethers.BigNumber.from(allowance.toString()).lt(raw)) {
    await USDT.approve(addr)
  }
  const BUY = new Contract(addr, 'BUY')
  await BUY.send('buy', [raw])
}
