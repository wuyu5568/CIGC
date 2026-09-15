const DEFAULT_CAP_TIERS = [
  { max_amount: '3000', daily_cap: '600' },
  { max_amount: '6000', daily_cap: '1800' },
  { max_amount: '12000', daily_cap: '4000' },
  { max_amount: '24000', daily_cap: '16000' },
  { max_amount: '36000', daily_cap: '24000' },
  { max_amount: '50000', daily_cap: '30000' },
  { max_amount: '70000', daily_cap: '42000' },
  { max_amount: '100000', daily_cap: '60000' },
  { max_amount: '', daily_cap: '100000' }
]

let cachedTiers = null
let inflight = null

export function defaultCapTiers() {
  return DEFAULT_CAP_TIERS.map((x) => ({ ...x }))
}

export function capForAmount(amount, tiers) {
  const a = Number(amount)
  if (!Number.isFinite(a) || a <= 0) return '0'
  const list = Array.isArray(tiers) && tiers.length ? tiers : (cachedTiers || DEFAULT_CAP_TIERS)
  for (let i = 0; i < list.length; i++) {
    const max = String(list[i].max_amount || '').trim()
    const cap = String(list[i].daily_cap ?? '0')
    if (!max) return cap
    if (a < Number(max)) return cap
  }
  return String(list[list.length - 1]?.daily_cap ?? '0')
}

export function rangeForAmount(amount, tiers) {
  const a = Number(amount)
  if (!Number.isFinite(a) || a <= 0) return ''
  const list = Array.isArray(tiers) && tiers.length ? tiers : (cachedTiers || DEFAULT_CAP_TIERS)
  let from = '0'
  for (let i = 0; i < list.length; i++) {
    const max = String(list[i].max_amount || '').trim()
    if (!max) return `${from} 及以上`
    if (a < Number(max)) return `${from} ≤ 金额 < ${max}`
    from = max
  }
  return `${from} 及以上`
}

export function currentCapTiers() {
  return cachedTiers || DEFAULT_CAP_TIERS
}

export function loadCapTiers(request) {
  if (inflight) return inflight
  inflight = request.get('admin/daily_cap_tiers').then((res) => {
    const rows = res && Array.isArray(res.tiers) ? res.tiers : null
    cachedTiers = rows && rows.length ? rows : defaultCapTiers()
    return cachedTiers
  }).catch(() => {
    if (!cachedTiers) cachedTiers = defaultCapTiers()
    return cachedTiers
  }).finally(() => {
    inflight = null
  })
  return inflight
}
