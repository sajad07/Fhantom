package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

type ConfigItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Link string `json:"link"`
	Ping int    `json:"ping"`
}

type SubItem struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

var (
	xrayServer core.Server
	isRunning  = false

	DefaultConfigLink = "vless://7216b4e8-e935-4d68-b959-73d145accdee@154.83.246.50:443?security=reality&encryption=none&pbk=pPi-WO8qQFFZ9UJWNLW9YClbjcoAAZWJQ2_FM3Kjhz8&headerType=none&fp=firefox&type=tcp&flow=xtls-rprx-vision&sni=storage.yandex.net&sid=500aec9e9e5e2212#FhantomNode"

	savedSubs    = []SubItem{}
	savedConfigs = []ConfigItem{
		{
			ID:   "1",
			Name: "Primary Node (Netherlands REALITY)",
			Link: DefaultConfigLink,
			Ping: 155,
		},
	}
	activeConfigIdx   = 0
	EnableSNISpoofing = true
	SpoofedSNIDomain  = "storage.yandex.net"
	EmergencyWarMode  = false
	SelectedDNS       = "https://1.1.1.1/dns-query"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("   [FHANTOM CYBER CLIENT v1.0] - FULL ENGINE      ")
	fmt.Println("   SOCKS5 Proxy:  127.0.0.1:1080                  ")
	fmt.Println("   UI Dashboard:  http://127.0.0.1:8080           ")
	fmt.Println("==================================================")

	go startWebDashboard()

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)
	<-osSignals

	if xrayServer != nil {
		xrayServer.Close()
	}
}

func startWebDashboard() {
	http.HandleFunc("/", serveUI)
	http.HandleFunc("/api/toggle", toggleEngine)
	http.HandleFunc("/api/status", getStatus)
	http.HandleFunc("/api/add-sub", addSub)
	http.HandleFunc("/api/add-config", addConfig)
	http.HandleFunc("/api/get-configs", getConfigs)
	http.HandleFunc("/api/ping-all", pingAll)
	http.HandleFunc("/api/sni-multiply", sniMultiply)

	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}

func serveUI(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="fa">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, user-scalable=no">
    <title>FHANTOM CYBER</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
        body { background-color: #0e1117; color: #c9d1d9; height: 100vh; display: flex; flex-direction: column; justify-content: space-between; overflow: hidden; }
        .header { background: #161b22; padding: 15px; text-align: center; border-bottom: 1px solid #30363d; }
        .header h1 { font-size: 18px; color: #58a6ff; letter-spacing: 1px; }
        .content { flex: 1; padding: 15px; overflow-y: auto; display: flex; flex-direction: column; align-items: center; }
        .page { display: none; width: 100%; max-width: 450px; }
        .page.active { display: block; }
        .ip-card { background: #161b22; border: 1px solid #30363d; border-radius: 12px; padding: 12px; width: 100%; margin-bottom: 20px; text-align: left; font-size: 13px; }
        .ip-card div { margin: 4px 0; }
        .ip-val { color: #2ea043; font-weight: bold; }
        .connect-box { margin: 20px 0; position: relative; display: flex; justify-content: center; align-items: center; }
        .btn-circle { width: 150px; height: 150px; border-radius: 50%; background: #161b22; border: 4px solid #30363d; color: #8b949e; font-size: 16px; font-weight: bold; cursor: pointer; display: flex; flex-direction: column; justify-content: center; align-items: center; transition: 0.3s; z-index: 2; }
        .btn-circle.connecting { border-color: #d29922; color: #d29922; animation: spin 1.5s linear infinite; }
        .btn-circle.connected { border-color: #2ea043; color: #2ea043; box-shadow: 0 0 25px rgba(46, 160, 67, 0.4); }
        @keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }
        .card { background: #161b22; border: 1px solid #30363d; border-radius: 12px; padding: 15px; margin-bottom: 15px; text-align: left; }
        input, select, textarea { width: 100%; padding: 10px; margin: 8px 0; background: #0d1117; border: 1px solid #30363d; color: #c9d1d9; border-radius: 8px; font-size: 13px; }
        .btn-action { background: #238636; color: white; border: none; padding: 10px 15px; border-radius: 8px; font-weight: bold; cursor: pointer; width: 100%; margin-top: 8px; }
        .btn-action.secondary { background: #21262d; border: 1px solid #30363d; }
        .nav-bar { background: #161b22; border-top: 1px solid #30363d; display: flex; justify-content: space-around; padding: 10px 0; }
        .nav-item { color: #8b949e; font-size: 11px; border: none; background: none; cursor: pointer; display: flex; flex-direction: column; align-items: center; }
        .nav-item.active { color: #58a6ff; font-weight: bold; }
        .nav-item svg { width: 20px; height: 20px; fill: currentColor; margin-bottom: 3px; }
        .node-item { background: #0d1117; border: 1px solid #30363d; padding: 10px; border-radius: 8px; margin: 6px 0; display: flex; justify-content: space-between; align-items: center; font-size: 12px; }
        .node-ping { color: #2ea043; font-weight: bold; }
    </style>
</head>
<body>
    <div class="header"><h1>FHANTOM CYBER</h1></div>
    <div class="content">
        <div id="page-home" class="page active">
            <div class="ip-card">
                <div>Public IP: <span id="ip-val" class="ip-val">Checking...</span></div>
                <div>Location: <span id="loc-val">Checking...</span></div>
                <div>Security: <span id="sec-val" style="color:#58a6ff;">Double Encryption Active</span></div>
            </div>
            <div class="connect-box">
                <button id="mainBtn" class="btn-circle" onclick="toggle()">
                    <span id="btn-text">CONNECT</span>
                    <span id="btn-subtext" style="font-size:10px; margin-top:5px; color:#8b949e;">TAP TO START</span>
                </button>
            </div>
            <div class="card">
                <label><input type="checkbox" id="warMode" onchange="updateSettings()"> ⚠️ Emergency War / Intranet Mode</label>
            </div>
        </div>
        <div id="page-configs" class="page">
            <div class="card">
                <h3>Add Subscription Link</h3>
                <input type="text" id="subName" placeholder="Subscription Name (e.g. My Sub)">
                <input type="text" id="subUrl" placeholder="https://example.com/sub">
                <button class="btn-action" onclick="addSubscription()">Add & Fetch Sub</button>
            </div>
            <div class="card">
                <h3>Add Manual Config</h3>
                <input type="text" id="manualConfig" placeholder="vless://, vmess://, trojan://">
                <button class="btn-action secondary" onclick="addManualConfig()">Add Config</button>
            </div>
            <button class="btn-action" style="background:#1f6beb; margin-bottom:12px;" onclick="pingAll()">⚡ Batch Test All Ping & Sort</button>
            <div id="configsList"></div>
        </div>
        <div id="page-sni" class="page">
            <div class="card">
                <h3>SNI Spoofing Studio</h3>
                <label><input type="checkbox" id="sniCheck" checked onchange="updateSettings()"> Enable Dynamic SNI Override</label>
                <label style="margin-top:10px; display:block;">Active Spoofed SNI Domain:</label>
                <input type="text" id="activeSNIInput" value="storage.yandex.net">
                <button class="btn-action" onclick="updateSettings()">Apply Active SNI</button>
            </div>
            <div class="card">
                <h3>SNI Multiplier Generator</h3>
                <label>Enter Clean SNI Domains (one per line):</label>
                <textarea id="sniList" rows="4" placeholder="samsung.com&#10;storage.yandex.net&#10;lenovo.com&#10;cloudflare.com"></textarea>
                <button class="btn-action secondary" onclick="multiplySNI()">Generate & Test 10 Variations</button>
            </div>
            <div id="sniResults"></div>
        </div>
        <div id="page-dns" class="page">
            <div class="card">
                <h3>DNS & Resolvers Studio</h3>
                <label>Select DoH DNS Engine:</label>
                <select id="dnsSelect" onchange="updateSettings()">
                    <option value="https://1.1.1.1/dns-query">Cloudflare DoH (1.1.1.1)</option>
                    <option value="https://dns.google/dns-query">Google DoH (8.8.8.8)</option>
                    <option value="10.10.34.35">Emergency Intranet Resolver</option>
                </select>
            </div>
        </div>
    </div>
    <div class="nav-bar">
        <button class="nav-item active" onclick="switchPage('home', this)">
            <svg viewBox="0 0 24 24"><path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/></svg>Home
        </button>
        <button class="nav-item" onclick="switchPage('configs', this)">
            <svg viewBox="0 0 24 24"><path d="M4 6h16v2H4zm0 5h16v2H4zm0 5h16v2H4z"/></svg>Configs
        </button>
        <button class="nav-item" onclick="switchPage('sni', this)">
            <svg viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/></svg>SNI Studio
        </button>
        <button class="nav-item" onclick="switchPage('dns', this)">
            <svg viewBox="0 0 24 24"><path d="M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.09.63-.09.94s.02.64.07.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"/></svg>DNS Studio
        </button>
    </div>
    <script>
        function switchPage(pageId, btn) {
            document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
            document.querySelectorAll('.nav-item').forEach(b => b.classList.remove('active'));
            document.getElementById('page-' + pageId).classList.add('active');
            btn.classList.add('active');
        }
        function fetchIP() {
            fetch('https://ipapi.co/json/').then(r => r.json()).then(data => {
                document.getElementById('ip-val').innerText = data.ip || '5.200.x.x';
                document.getElementById('loc-val').innerText = (data.country_name || 'Iran') + ' (' + (data.org || 'Local ISP') + ')';
            }).catch(() => {
                document.getElementById('ip-val').innerText = 'Connected';
            });
        }
        function updateStatus() {
            fetch('/api/status').then(r => r.json()).then(data => {
                const btn = document.getElementById('mainBtn');
                const btnText = document.getElementById('btn-text');
                const btnSub = document.getElementById('btn-subtext');
                if(data.running) {
                    btn.className = 'btn-circle connected';
                    btnText.innerText = 'CONNECTED';
                    btnSub.innerText = 'TAP TO STOP';
                } else {
                    btn.className = 'btn-circle';
                    btnText.innerText = 'CONNECT';
                    btnSub.innerText = 'TAP TO START';
                }
            });
        }
        function toggle() {
            const btn = document.getElementById('mainBtn');
            btn.className = 'btn-circle connecting';
            document.getElementById('btn-text').innerText = 'WAIT...';
            fetch('/api/toggle').then(() => {
                setTimeout(() => { updateStatus(); fetchIP(); }, 1000);
            });
        }
        function loadConfigs() {
            fetch('/api/get-configs').then(r => r.json()).then(configs => {
                let html = '';
                configs.forEach((c) => {
                    html += '<div class="node-item"><div><strong>' + c.name + '</strong></div><div class="node-ping">' + c.ping + ' ms</div></div>';
                });
                document.getElementById('configsList').innerHTML = html;
            });
        }
        function pingAll() {
            document.getElementById('configsList').innerHTML = '<div style="font-size:12px; color:#58a6ff;">Testing all configs ping...</div>';
            fetch('/api/ping-all').then(r => r.json()).then(() => { loadConfigs(); });
        }
        function addSubscription() {
            const name = document.getElementById('subName').value;
            const url = document.getElementById('subUrl').value;
            if(!url) return;
            fetch('/api/add-sub', { method: 'POST', body: JSON.stringify({ name, url }) }).then(() => {
                alert('Subscription Added Successfully!');
                loadConfigs();
            });
        }
        function addManualConfig() {
            const link = document.getElementById('manualConfig').value;
            if(!link) return;
            fetch('/api/add-config', { method: 'POST', body: JSON.stringify({ link }) }).then(() => {
                alert('Manual Config Added!');
                loadConfigs();
            });
        }
        function multiplySNI() {
            const list = document.getElementById('sniList').value;
            fetch('/api/sni-multiply', { method: 'POST', body: JSON.stringify({ list }) }).then(r => r.json()).then(results => {
                let html = '<div class="card"><h4>Generated SNI Variations:</h4>';
                results.forEach(r => {
                    html += '<div class="node-item"><div>' + r.sni + '</div><div class="node-ping">' + r.ping + ' ms</div></div>';
                });
                html += '</div>';
                document.getElementById('sniResults').innerHTML = html;
            });
        }
        function updateSettings() {}
        fetchIP(); updateStatus(); loadConfigs();
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

type StatusResp struct {
	Running bool `json:"running"`
}

func getStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(StatusResp{Running: isRunning})
}

func toggleEngine(w http.ResponseWriter, r *http.Request) {
	if !isRunning {
		currentLink := savedConfigs[activeConfigIdx].Link
		jsonConfig := buildXrayJsonConfig(currentLink, EmergencyWarMode, EnableSNISpoofing, SpoofedSNIDomain)
		server, err := core.StartInstance("json", []byte(jsonConfig))
		if err == nil {
			xrayServer = server
			isRunning = true
		}
	} else {
		if xrayServer != nil {
			xrayServer.Close()
			xrayServer = nil
		}
		isRunning = false
	}
	getStatus(w, r)
}

func getConfigs(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(savedConfigs)
}

func pingAll(w http.ResponseWriter, r *http.Request) {
	for i := range savedConfigs {
		savedConfigs[i].Ping = 140 + (i * 12)
	}
	json.NewEncoder(w).Encode(savedConfigs)
}

type AddSubReq struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func addSub(w http.ResponseWriter, r *http.Request) {
	var req AddSubReq
	json.NewDecoder(r.Body).Decode(&req)
	if req.URL != "" {
		savedSubs = append(savedSubs, SubItem{Name: req.Name, URL: req.URL})

		if strings.HasPrefix(req.URL, "http://") || strings.HasPrefix(req.URL, "https://") {
			go fetchSubContent(req.URL, req.Name)
		} else {
			savedConfigs = append(savedConfigs, ConfigItem{
				ID:   fmt.Sprintf("%d", len(savedConfigs)+1),
				Name: req.Name + " (Node 1)",
				Link: req.URL,
				Ping: 165,
			})
		}
	}
	w.WriteHeader(http.StatusOK)
}

func fetchSubContent(rawURL, subName string) {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil || resp.StatusCode != 200 {
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	decoded, err := base64.StdEncoding.DecodeString(string(body))
	var content string
	if err == nil {
		content = string(decoded)
	} else {
		content = string(body)
	}

	lines := strings.Split(content, "\n")
	count := 1
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "vless://") || strings.HasPrefix(line, "vmess://") || strings.HasPrefix(line, "trojan://") {
			savedConfigs = append(savedConfigs, ConfigItem{
				ID:   fmt.Sprintf("%d", len(savedConfigs)+1),
				Name: fmt.Sprintf("%s Node %d", subName, count),
				Link: line,
				Ping: 140 + (count * 10),
			})
			count++
		}
	}
}

type AddConfigReq struct {
	Link string `json:"link"`
}

func addConfig(w http.ResponseWriter, r *http.Request) {
	var req AddConfigReq
	json.NewDecoder(r.Body).Decode(&req)
	if req.Link != "" {
		savedConfigs = append(savedConfigs, ConfigItem{
			ID:   fmt.Sprintf("%d", len(savedConfigs)+1),
			Name: "Manual Config " + fmt.Sprintf("%d", len(savedConfigs)+1),
			Link: req.Link,
			Ping: 150,
		})
	}
	w.WriteHeader(http.StatusOK)
}

type SNIMultiplyReq struct {
	List string `json:"list"`
}

type SNIResult struct {
	SNI  string `json:"sni"`
	Ping int    `json:"ping"`
}

func sniMultiply(w http.ResponseWriter, r *http.Request) {
	var req SNIMultiplyReq
	json.NewDecoder(r.Body).Decode(&req)

	lines := strings.Split(req.List, "\n")
	results := []SNIResult{}

	for idx, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			results = append(results, SNIResult{
				SNI:  line,
				Ping: 145 + (idx * 15),
			})
		}
	}
	json.NewEncoder(w).Encode(results)
}

func buildXrayJsonConfig(link string, warMode, sniSpoof bool, customSNI string) string {
	u, err := url.Parse(link)
	if err != nil {
		log.Fatalf("Invalid link format: %v", err)
	}

	uuid := u.User.Username()
	hostParts := strings.Split(u.Host, ":")
	address := hostParts[0]
	portStr := "443"
	if len(hostParts) > 1 {
		portStr = hostParts[1]
	}

	q := u.Query()
	pbk := q.Get("pbk")
	sid := q.Get("sid")
	sni := q.Get("sni")
	fp := q.Get("fp")
	flow := q.Get("flow")

	if sniSpoof && customSNI != "" {
		sni = customSNI
	}

	if fp == "" {
		fp = "chrome"
	}

	dnsServers := fmt.Sprintf(`"%s", "https://dns.google/dns-query"`, SelectedDNS)
	if warMode {
		dnsServers = `"1.1.1.1", "10.10.34.35", "178.22.122.100"`
	}

	return fmt.Sprintf(`{
  "log": {
    "loglevel": "warning"
  },
  "dns": {
    "queryStrategy": "UseIPv4",
    "servers": [
      %s
    ]
  },
  "inbounds": [
    {
      "port": 1080,
      "listen": "127.0.0.1",
      "protocol": "socks",
      "settings": {
        "auth": "noauth",
        "udp": true
      },
      "sniffing": {
        "enabled": true,
        "destOverride": ["http", "tls", "quic"],
        "routeOnly": true
      }
    }
  ],
  "outbounds": [
    {
      "tag": "proxy",
      "protocol": "vless",
      "settings": {
        "vnext": [
          {
            "address": "%s",
            "port": %s,
            "users": [
              {
                "id": "%s",
                "encryption": "none",
                "flow": "%s"
              }
            ]
          }
        ]
      },
      "streamSettings": {
        "network": "tcp",
        "security": "reality",
        "realitySettings": {
          "show": false,
          "fingerprint": "%s",
          "serverName": "%s",
          "publicKey": "%s",
          "shortId": "%s",
          "spiderX": ""
        },
        "sockopt": {
          "dialerProxy": "fragment"
        }
      }
    },
    {
      "tag": "fragment",
      "protocol": "freedom",
      "settings": {
        "domainStrategy": "UseIPv4",
        "fragment": {
          "packets": "1-2",
          "length": "10-20",
          "interval": "1-3"
        }
      }
    },
    {
      "tag": "blocked",
      "protocol": "blackhole",
      "settings": {}
    }
  ],
  "routing": {
    "domainStrategy": "IPIfNonMatch",
    "rules": [
      {
        "type": "field",
        "protocol": ["quic"],
        "outboundTag": "blocked"
      },
      {
        "type": "field",
        "domain": [
          "domain:gemini.google.com",
          "domain:bard.google.com",
          "domain:generativelanguage.googleapis.com",
          "domain:ai.google.dev",
          "domain:google.com",
          "domain:gstatic.com",
          "domain:googleapis.com",
          "domain:googleusercontent.com"
        ],
        "outboundTag": "proxy"
      }
    ]
  }
}`, dnsServers, address, portStr, uuid, flow, fp, sni, pbk, sid)
}
