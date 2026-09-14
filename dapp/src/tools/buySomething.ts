import { ethers } from 'ethers'
import { ETH, Contract } from '@/tools/contract'
import userPerson from '@/pinia/person'
import lang from '@/i18n/index'

const USDT = new Contract(import.meta.env.VITE_USDT, 'ERC20')

export function resolveBuyAddress(fromApi?: string): string {
  const person = userPerson()
  return String(
    fromApi ||
      person.userinfo?.buy_contract ||
      import.meta.env.VITE_BUY ||
      ''
  ).trim()
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
  const whole = String(amount).replace(/,/g, '').trim()
  if (!/^\d+(\.0+)?$/.test(whole)) {
    throw lang('金额错误')
  }
  const num = ethers.BigNumber.from(whole.split('.')[0])
  if (num.lt(5)) {
    throw lang('充值金额不能小于5')
  }
  const raw = ethers.utils.parseUnits(num.toString(), 18)
  const allowance = await USDT.call('allowance', [ETH.account, addr])
  if (ethers.BigNumber.from(allowance.toString()).lt(raw)) {
    await USDT.approve(addr)
  }
  const BUY = new Contract(addr, 'BUY')
  await BUY.send('buy', [num])
}
