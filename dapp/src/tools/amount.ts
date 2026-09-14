/** 展示金额：无小数不带小数点，有小数则去掉尾零。 */
export function displayAmount(v: unknown, fallback = '0'): string {
  if (v == null || v === '') return fallback
  const s = String(v).trim().replace(/,/g, '')
  if (!/^-?\d+(\.\d+)?$/.test(s)) return String(v)
  return s.replace(/(\.\d*?)0+$/, '$1').replace(/\.$/, '')
}

const round8 = (n: number) => displayAmount((Math.round(n * 1e8) / 1e8).toFixed(8))

const dash = { coins: '-', dailyCoins: '-', usdt: '-', ispay: '-', dailyValue: '-' }

/** 静态释放预览：购币 = 订单额 / 档位买价；日产值再拆一半 U + 一半 ispay。 */
export function previewStatic(amount: unknown, days: unknown, buyPrice: unknown, spot: unknown) {
  const amt = Number(amount || 0)
  const d = Number(days || 0)
  const buy = Number(buyPrice || 0)
  const sp = Number(spot || 0)
  if (!amt || !d || !buy || !sp) return dash
  const coins = amt / buy
  const dailyCoins = coins / d
  const dailyValue = dailyCoins * sp
  const usdt = dailyValue / 2
  const ispay = usdt / sp
  return {
    coins: round8(coins),
    dailyCoins: round8(dailyCoins),
    dailyValue: round8(dailyValue),
    usdt: round8(usdt),
    ispay: round8(ispay)
  }
}
