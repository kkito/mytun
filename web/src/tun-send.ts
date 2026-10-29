import { api } from './api.js';

export class TunSend extends HTMLElement {
  connectedCallback(): void {
    this.innerHTML =
      `<div class="tun-send">` +
      `<label>dst <input name="dst" value="10.10.0.2"></label>` +
      `<label>size <input name="size" value="64"></label>` +
      `<button data-send>发送</button>` +
      `<div data-result></div>` +
      `</div>`;
    this.querySelector('button[data-send]')!.addEventListener('click', () => void this.send());
  }

  async send(): Promise<void> {
    const dst = (this.querySelector('input[name="dst"]') as HTMLInputElement).value.trim();
    const size = parseInt((this.querySelector('input[name="size"]') as HTMLInputElement).value.trim(), 10);
    const out = this.querySelector('[data-result]') as HTMLElement;
    try {
      const j = await api<{ len: number }>('/api/send', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ dst, size }),
      });
      out.textContent = 'sent len=' + (j.data && j.data.len);
    } catch (e) {
      out.textContent = 'send failed: ' + (e as Error).message;
    }
  }
}

customElements.define('tun-send', TunSend);
