const BASE = '';

export async function api(path, opts = {}) {
  const res = await fetch(BASE + path, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  });
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || 'Request failed');
  return data;
}

export function withConnection(path, connectionId) {
  if (!connectionId) return path;
  const sep = path.includes('?') ? '&' : '?';
  return `${path}${sep}connection_id=${connectionId}`;
}

export function opSym(op) {
  return { gt: '>', gte: '>=', lt: '<', lte: '<=', eq: '==', neq: '!=' }[op] || op;
}

export function timeAgo(ts) {
  const s = Math.floor((Date.now() - new Date(ts).getTime()) / 1000);
  if (s < 60) return s + 's ago';
  if (s < 3600) return Math.floor(s / 60) + 'm ago';
  if (s < 86400) return Math.floor(s / 3600) + 'h ago';
  return Math.floor(s / 86400) + 'd ago';
}
