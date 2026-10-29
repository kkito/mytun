"use strict";
var MytunWeb = (() => {
  // web/src/api.ts
  async function api(path, opts) {
    const r = await fetch(path, opts);
    const j = await r.json();
    if (!j.ok) throw new Error(j.error && j.error.message || "HTTP " + r.status);
    return j;
  }

  // web/src/tun-status.ts
  var TunStatus = class extends HTMLElement {
    constructor() {
      super(...arguments);
      this.timer = null;
    }
    connectedCallback() {
      void this.refresh();
      this.timer = setInterval(() => void this.refresh(), 2e3);
    }
    disconnectedCallback() {
      if (this.timer) clearInterval(this.timer);
      this.timer = null;
    }
    async refresh() {
      try {
        const j = await api("/api/status");
        const d = j.data;
        const s = d.stats || { rxPackets: 0, txPackets: 0, rxBytes: 0, txBytes: 0 };
        this.innerHTML = `<div class="tun-status"><div>state: <span data-state>${d.state}</span></div><div>ifname: <span data-ifname>${d.ifname}</span></div><div>addr: <span data-addr>${d.addr}</span></div><div>mtu: <span data-mtu>${d.mtu}</span></div><div>rx: <span data-rx>${s.rxPackets}</span> packets / <span data-rxbytes>${s.rxBytes}</span> bytes</div><div>tx: <span data-tx>${s.txPackets}</span> packets / <span data-txbytes>${s.txBytes}</span> bytes</div><div>version: <span data-version>${d.version}</span></div></div>`;
      } catch (e) {
        this.innerHTML = `<div class="tun-status error">status error: ${e.message}</div>`;
      }
    }
  };
  customElements.define("tun-status", TunStatus);

  // web/src/tun-controls.ts
  var TunControls = class extends HTMLElement {
    connectedCallback() {
      this.innerHTML = `<div class="tun-controls"><label>addr <input name="addr" value="10.10.0.1/24"></label><label>mtu <input name="mtu" value="1420"></label><button data-create>\u521B\u5EFA</button><button data-delete>\u5220\u9664</button><div data-result></div></div>`;
      this.querySelector("button[data-create]").addEventListener("click", () => void this.create());
      this.querySelector("button[data-delete]").addEventListener("click", () => void this.remove());
    }
    result() {
      return this.querySelector("[data-result]");
    }
    async create() {
      const addr = this.querySelector('input[name="addr"]').value.trim();
      const mtu = parseInt(this.querySelector('input[name="mtu"]').value.trim(), 10);
      try {
        const j = await api("/api/tun", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ addr, mtu })
        });
        this.result().textContent = "created: " + (j.data && j.data.ifname);
      } catch (e) {
        this.result().textContent = "create failed: " + e.message;
      }
    }
    async remove() {
      try {
        await api("/api/tun", { method: "DELETE" });
        this.result().textContent = "deleted";
      } catch (e) {
        this.result().textContent = "delete failed: " + e.message;
      }
    }
  };
  customElements.define("tun-controls", TunControls);

  // web/src/tun-send.ts
  var TunSend = class extends HTMLElement {
    connectedCallback() {
      this.innerHTML = `<div class="tun-send"><label>dst <input name="dst" value="10.10.0.2"></label><label>size <input name="size" value="64"></label><button data-send>\u53D1\u9001</button><div data-result></div></div>`;
      this.querySelector("button[data-send]").addEventListener("click", () => void this.send());
    }
    async send() {
      const dst = this.querySelector('input[name="dst"]').value.trim();
      const size = parseInt(this.querySelector('input[name="size"]').value.trim(), 10);
      const out = this.querySelector("[data-result]");
      try {
        const j = await api("/api/send", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ dst, size })
        });
        out.textContent = "sent len=" + (j.data && j.data.len);
      } catch (e) {
        out.textContent = "send failed: " + e.message;
      }
    }
  };
  customElements.define("tun-send", TunSend);

  // web/src/tun-log.ts
  function esc(s) {
    return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  }
  var TunLog = class extends HTMLElement {
    constructor() {
      super(...arguments);
      this.timer = null;
    }
    connectedCallback() {
      void this.refresh();
      this.timer = setInterval(() => void this.refresh(), 2e3);
    }
    disconnectedCallback() {
      if (this.timer) clearInterval(this.timer);
      this.timer = null;
    }
    async refresh() {
      try {
        const j = await api("/api/logs?limit=50");
        const logs = j.data && j.data.logs || [];
        this.innerHTML = `<ul class="tun-log">` + logs.map((l) => `<li><span>${esc(l.Time)}</span> <span>${esc(l.Dir)}</span> <span>${esc(l.Summary)}</span></li>`).join("") + `</ul>`;
      } catch (e) {
        this.innerHTML = `<div class="tun-log error">logs error: ${e.message}</div>`;
      }
    }
  };
  customElements.define("tun-log", TunLog);
})();
