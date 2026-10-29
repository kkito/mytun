import { describe, it, expect, vi, afterEach } from 'vitest';
import '../src/tun-status.js';

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.innerHTML = '';
});

function flush(ms = 20): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}

describe('tun-status', () => {
  it('defines element', () => {
    expect(customElements.get('tun-status')).toBeDefined();
  });

  it('renders state/ifname/addr/mtu/counters from /api/status', async () => {
    const f = vi.fn().mockResolvedValue({
      json: async () => ({
        ok: true,
        data: {
          state: 'up',
          ifname: 'utun9',
          addr: '10.10.0.1/24',
          mtu: 1420,
          stats: { rxPackets: 3, txPackets: 5, rxBytes: 300, txBytes: 500 },
          version: 'dev',
        },
      }),
      status: 200,
    });
    vi.stubGlobal('fetch', f);
    const el = document.createElement('tun-status');
    document.body.appendChild(el);
    await flush();
    const html = el.innerHTML;
    expect(html).toContain('up');
    expect(html).toContain('utun9');
    expect(html).toContain('10.10.0.1/24');
    expect(html).toContain('1420');
    expect(html).toContain('3');
    expect(html).toContain('5');
    expect(f.mock.calls[0][0]).toBe('/api/status');
    document.body.removeChild(el);
  });
});
