export const CART_KEY = 'web3_shop_cart'

export const cartLineKey = (item) => `${Number(item?.id || 0)}:${Number(item?.sku_id || 0)}`

export const enabledSKUs = (item) => {
  const rows = Array.isArray(item?.skus) ? item.skus : []
  return rows.filter((x) => x && Number(x.id) > 0 && x.enabled !== 0 && x.enabled !== '0' && x.enabled !== false)
}

export const hasSKUs = (item) => enabledSKUs(item).length > 0

export const goodsAmountRange = (item) => {
  const skus = enabledSKUs(item)
  const source = skus.length ? skus.map((x) => x.amount) : [item?.amount]
  const amts = source.map((v) => Number(v)).filter((n) => Number.isFinite(n) && n > 0)
  if (!amts.length) return { min: item?.amount || '', max: '' }
  const min = Math.min(...amts)
  const max = Math.max(...amts)
  const minRaw = source.find((v) => Number(v) === min)
  const maxRaw = source.find((v) => Number(v) === max)
  return {
    min: minRaw == null ? String(min) : String(minRaw),
    max: max === min ? '' : (maxRaw == null ? String(max) : String(maxRaw))
  }
}

export const readCart = () => {
  try {
    const raw = JSON.parse(sessionStorage.getItem(CART_KEY) || '[]')
    return Array.isArray(raw) ? raw.filter((x) => x && x.id) : []
  } catch {
    return []
  }
}

export const persistCart = (cart) => {
  try {
    sessionStorage.setItem(CART_KEY, JSON.stringify(cart))
  } catch {}
}

export const snapshotCartItem = (item, sku) => {
  const skuId = sku ? Number(sku.id) : 0
  const skuName = sku ? (sku.name || sku.name_zh || sku.name_en || '') : ''
  const name = item?.name || item?.desc || ''
  return {
    id: Number(item?.id),
    sku_id: skuId,
    name,
    sku_name: skuName,
    image: (sku && sku.image) || item?.image || '',
    amount: String((sku && sku.amount) || item?.amount || ''),
    qty: 1
  }
}

export const changeCartQty = (cart, item, delta, sku) => {
  const snap = snapshotCartItem(item, sku)
  const key = cartLineKey(snap)
  const next = (Array.isArray(cart) ? cart : []).map((x) => ({ ...x }))
  const i = next.findIndex((x) => cartLineKey(x) === key)
  if (i < 0) {
    if (delta <= 0) return next
    next.push({ ...snap, qty: delta })
  } else {
    next[i].qty = (Number(next[i].qty) || 0) + delta
    if (next[i].qty <= 0) next.splice(i, 1)
  }
  return next
}

export const refreshCartRow = (row, fresh) => {
  if (!fresh) return row
  const sku = enabledSKUs(fresh).find((x) => Number(x.id) === Number(row.sku_id))
  if (row.sku_id && !sku) {
    return { ...row, name: fresh.name || fresh.desc || row.name, image: fresh.image || row.image }
  }
  return { ...row, ...snapshotCartItem(fresh, sku), qty: row.qty }
}
