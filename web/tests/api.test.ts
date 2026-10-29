import { describe, it, expect, vi, afterEach } from 'vitest';
import { api } from '../src/api.js';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('api client', () => {
  it('returns data on ok=true', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        json: async () => ({ ok: true, data: { state: 'up' } }),
        status: 200,
      }),
    );
    const j = await api<{ state: string }>('/api/status');
    expect(j.data!.state).toBe('up');
  });

  it('throws message on ok=false', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        json: async () => ({ ok: false, error: { code: 'NO_TUN', message: 'no tun' } }),
        status: 404,
      }),
    );
    await expect(api('/api/status')).rejects.toThrow('no tun');
  });

  it('throws HTTP status when body has no message', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ json: async () => ({ ok: false }), status: 500 }),
    );
    await expect(api('/api/status')).rejects.toThrow('500');
  });
});
