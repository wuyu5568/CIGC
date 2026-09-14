export function parseInviteCode(href = window.location.href): string {
  const tdh = href.match(/-invitetdh-([0-9a-fA-Fx]+)-invitetdh-/i)
  if (tdh && tdh[1]) return tdh[1].trim()

  const hash = window.location.hash || ''
  const search = window.location.search || ''
  const hashQuery = hash.includes('?') ? hash.slice(hash.indexOf('?') + 1) : ''
  const params = new URLSearchParams(search.replace(/^\?/, ''))
  const hashParams = new URLSearchParams(hashQuery)
  const raw = params.get('code') || hashParams.get('code') || hashParams.get('inviteCode') || ''
  const trimmed = raw.trim()
  if (!trimmed || trimmed === 'null') return ''
  const nested = trimmed.match(/-invitetdh-([0-9a-fA-Fx]+)-invitetdh-/i)
  return nested && nested[1] ? nested[1].trim() : trimmed
}

export function inviteShareURL(address: string): string {
  const host = `${window.location.origin}${window.location.pathname}`
  return `${host}#/?code=${encodeURIComponent(address)}`
}
