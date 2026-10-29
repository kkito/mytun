import { api } from './api.js';

export class TunControls extends HTMLElement {
  connectedCallback(): void {
    this.innerHTML =
      `<div class="tun-controls">` +
      `<label>addr <input name="addr" value="10.10.0.1/24"></label>` +
      `<label>mtu <input name="mtu" value="1420"></label>` +
      `<button data-create>创建</button>` +
      `<button data-delete>删除</button>` +
      `<div data-result></div>` +
      `</div>`;
    this.querySelector('button[data-create]')!.addEventListener('click', () => void this.create());
    this.querySelector('button[data-delete]')!.addEventListener('click', () => void this.remove());
  }

  private result(): HTMLElement {
    return this.querySelector('[data-result]') as HTMLElement;
  }

  async create(): Promise<void> {
    const addr = (this.querySelector('input[name="addr"]') as HTMLInputElement).value.trim();
    const mtu = parseInt((this.querySelector('input[name="mtu"]') as HTMLInputElement).value.trim(), 10);
    try {
      const j = await api<{ ifname: string }>('/api/tun', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ addr, mtu }),
      });
      this.result().textContent = 'created: ' + (j.data && j.data.ifname);
    } catch (e) {
      this.result().textContent = 'create failed: ' + (e as Error).message;
    }
  }

  async remove(): Promise<void> {
    try {
      await api('/api/tun', { method: 'DELETE' });
      this.result().textContent = 'deleted';
    } catch (e) {
      this.result().textContent = 'delete failed: ' + (e as Error).message;
    }
  }
}

customElements.define('tun-controls', TunControls);
