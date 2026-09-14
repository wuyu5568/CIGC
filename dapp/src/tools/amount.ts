/** 展示金额：无小数不带小数点，有小数则去掉尾零。 */
export function displayAmount(v: unknown, fallback = '0'): string {
  if (v == null || v === '') return fallback
  const s = String(v).trim().replace(/,/g, '')
  if (!/^-?\d+(\.\d+)?$/.test(s)) return String(v)
  return s.replace(/(\.\d*?)0+$/, '$1').replace(/\.$/, '')
}
