export function capForAmount(amount) {
  const a = Number(amount)
  if (!Number.isFinite(a) || a <= 0) return '0'
  if (a < 3000) return '600'
  if (a < 6000) return '1800'
  if (a < 12000) return '4000'
  if (a < 24000) return '16000'
  if (a < 36000) return '24000'
  if (a < 50000) return '30000'
  if (a < 70000) return '42000'
  if (a < 100000) return '60000'
  return '100000'
}
