export interface ApiEnvelope<T = unknown> {
  ok: boolean;
  data?: T;
  error?: { code?: string; message?: string };
}

export async function api<T = unknown>(path: string, opts?: RequestInit): Promise<ApiEnvelope<T>> {
  const r = await fetch(path, opts);
  const j = (await r.json()) as ApiEnvelope<T>;
  if (!j.ok) throw new Error((j.error && j.error.message) || 'HTTP ' + r.status);
  return j;
}

export interface StatusData {
  state: string;
  ifname: string;
  addr: string;
  mtu: number;
  stats: { rxPackets: number; txPackets: number; rxBytes: number; txBytes: number };
  version: string;
}

export interface LogEntry {
  Time: string;
  Dir: string;
  Summary: string;
}
