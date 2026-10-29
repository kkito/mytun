import { describe, it, expect, vi, afterEach } from 'vitest';
import '../src/tun-send.js';

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.innerHTML = '';
});

function flush(ms = 20): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}

describe('tun-send', () => {
  it('defines element', () => {
    expect(customElements.get('tun-send')).toBeDefined();
  });

  it('renders dst/size inputs and send button', () => {
    const el = document.createElement('tun-send');
    document.body.appendChild(el);
    expect(el.querySelector('input[name="dst"]')).not.toBeNull();
    expect(el.querySelector('input[name="size"]')).not.toBeNull();
    expect(el.querySelector('button[data-send]')).not.toBeNull();
    document.body.removeChild(el);
  });

  it('send button POSTs /api/send and shows len', async () => {
    const f = vi.fn().mockResolvedValue({
      json: async () => ({ ok: true, data: { len: 64 } }),
      status: 200,
    });
    vi.stubGlobal('fetch', f);
    const el = document.createElement('tun-send');
    document.body.appendChild(el);
    (el.querySelector('input[name="dst"]') as HTMLInputElement).value = '10.10.0.2';
    (el.querySelector('input[name="size"]') as HTMLInputElement).value = '64';
    (el.querySelector('button[data-send]') as HTMLButtonElement).click();
    await flush();
    expect(f).toHaveBeenCalledWith(
      '/api/send',
      expect.objectContaining({ method: 'POST' }),
    );
    expect(el.innerHTML).toContain('64');
    document.body.removeChild(el);
  });
});
