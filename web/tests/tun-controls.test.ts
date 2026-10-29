import { describe, it, expect, vi, afterEach } from 'vitest';
import '../src/tun-controls.js';

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.innerHTML = '';
});

function flush(ms = 20): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}

describe('tun-controls', () => {
  it('defines element', () => {
    expect(customElements.get('tun-controls')).toBeDefined();
  });

  it('renders addr/mtu inputs, create and delete buttons', () => {
    const el = document.createElement('tun-controls');
    document.body.appendChild(el);
    expect(el.innerHTML).toContain('addr');
    expect(el.innerHTML).toContain('mtu');
    expect(el.querySelector('button[data-create]')).not.toBeNull();
    expect(el.querySelector('button[data-delete]')).not.toBeNull();
    document.body.removeChild(el);
  });

  it('create button POSTs /api/tun and shows result', async () => {
    const f = vi.fn().mockResolvedValue({
      json: async () => ({ ok: true, data: { ifname: 'utun9' } }),
      status: 200,
    });
    vi.stubGlobal('fetch', f);
    const el = document.createElement('tun-controls');
    document.body.appendChild(el);
    (el.querySelector('input[name="addr"]') as HTMLInputElement).value = '10.10.0.1/24';
    (el.querySelector('input[name="mtu"]') as HTMLInputElement).value = '1420';
    (el.querySelector('button[data-create]') as HTMLButtonElement).click();
    await flush();
    expect(f).toHaveBeenCalledWith(
      '/api/tun',
      expect.objectContaining({ method: 'POST' }),
    );
    expect(el.innerHTML).toContain('utun9');
    document.body.removeChild(el);
  });

  it('delete button DELETEs /api/tun and shows result', async () => {
    const f = vi.fn().mockResolvedValue({
      json: async () => ({ ok: true, data: { deleted: true } }),
      status: 200,
    });
    vi.stubGlobal('fetch', f);
    const el = document.createElement('tun-controls');
    document.body.appendChild(el);
    (el.querySelector('button[data-delete]') as HTMLButtonElement).click();
    await flush();
    expect(f).toHaveBeenCalledWith('/api/tun', expect.objectContaining({ method: 'DELETE' }));
    expect(el.innerHTML.toLowerCase()).toContain('delet');
    document.body.removeChild(el);
  });
});
