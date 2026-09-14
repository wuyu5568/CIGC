import lang from '@/i18n/index'

export function formatShortAddress(value: string): string {
  const v = String(value || '')
  if (v.length < 12) return v || '-'
  return `${v.slice(0, 6)}...${v.slice(-4)}`
}

export function sideLabel(side: string): string {
  if (side === 'L' || side === 'left') return lang('左区')
  if (side === 'R' || side === 'right') return lang('右区')
  return ''
}

export function mapRecommendNodes(nodes: any[] = [], keyPrefix = ''): any[] {
  return (nodes || []).map((item, index) => {
    const side = sideLabel(item.side)
    const title = `${formatShortAddress(item.address)}${side ? ` · ${side}` : ''} (${lang('数量')}:${item.amount})`
    return {
      title,
      key: keyPrefix === '' ? String(index) : `${keyPrefix}-${index}`,
      amount: item.amount,
      address: item.address,
      side: item.side,
      isLeaf: Number(item.countLow || 0) === 0
    }
  })
}
