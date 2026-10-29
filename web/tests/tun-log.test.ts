import { describe, it, expect, vi, afterEach } from 'vitest';
import '../src/tun-log.js';

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.innerHTML = '';
});

function flush(ms = 20): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}

describe('tun-log', () => {
  it('defines element', () => {
    expect(customElements.get('tun-log')).toBeDefined();
  });

  it('renders Time/Dir/Summary rows from /api/logs?limit=50', async () => {
    const f = vi.fn().mockResolvedValue({
      json: async () => ({
        ok: true,
        data: {
          logs: [
            { Time: '12:00:01', Dir: 'rx', Summary: '10.0.0.2->10.10.0.1 proto=1 len=64' },
            { Time: '12:00:02', Dir: 'tx', Summary: '10.10.0.1->10.0.0.2 proto=1 len=64' },
          ],
        },
      }),
      status: 200,
    });
    vi.stubGlobal('fetch', f);
    const el = document.createElement('tun-log');
    document.body.appendChild(el);
    await flush();
    expect(f.mock.calls[0][0]).toBe('/api/logs?limit=50');
    const html = el.innerHTML;
    expect(html).toContain('12:00:01');
    expect(html).toContain('rx');
    expect(html).toContain('10.0.0.2-&gt;10.10.0.1');
    expect(html).toContain('tx');
    document.body.removeChild(el);
  });
});
