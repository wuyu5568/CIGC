export function shippingComplete(addr) {
  if (!addr || typeof addr !== 'object') return false
  return Boolean(
    String(addr.name || '').trim() &&
    String(addr.contact || '').trim() &&
    String(addr.address || '').trim()
  )
}
