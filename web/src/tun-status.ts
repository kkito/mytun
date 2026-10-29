import { api, type StatusData } from './api.js';

export class TunStatus extends HTMLElement {
  private timer: ReturnType<typeof setInterval> | null = null;

  connectedCallback(): void {
    void this.refresh();
    this.timer = setInterval(() => void this.refresh(), 2000);
  }

  disconnectedCallback(): void {
    if (this.timer) clearInterval(this.timer);
    this.timer = null;
  }

  async refresh(): Promise<void> {
    try {
      const j = await api<StatusData>('/api/status');
      const d = j.data!;
      const s = d.stats || { rxPackets: 0, txPackets: 0, rxBytes: 0, txBytes: 0 };
      this.innerHTML =
        `<div class="tun-status">` +
        `<div>state: <span data-state>${d.state}</span></div>` +
        `<div>ifname: <span data-ifname>${d.ifname}</span></div>` +
        `<div>addr: <span data-addr>${d.addr}</span></div>` +
        `<div>mtu: <span data-mtu>${d.mtu}</span></div>` +
        `<div>rx: <span data-rx>${s.rxPackets}</span> packets / <span data-rxbytes>${s.rxBytes}</span> bytes</div>` +
        `<div>tx: <span data-tx>${s.txPackets}</span> packets / <span data-txbytes>${s.txBytes}</span> bytes</div>` +
        `<div>version: <span data-version>${d.version}</span></div>` +
        `</div>`;
    } catch (e) {
      this.innerHTML = `<div class="tun-status error">status error: ${(e as Error).message}</div>`;
    }
  }
}

customElements.define('tun-status', TunStatus);
