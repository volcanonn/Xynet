const fs = require('fs');
let proxiesContent = fs.readFileSync('frontend/src/components/views/Proxies.vue', 'utf8');

proxiesContent = proxiesContent.replace(
  "import { ImportWireguardConfig } from '../../../wailsjs/go/main/App';",
  "import { ImportWireguardConfig } from '../../../wailsjs/go/main/App';\nimport WireguardTabs from '../WireguardTabs.vue';"
);

const headerSectionHTML = `<div class="header-section">
      <Globe :size="48" class="icon" />
      <h2>Proxies</h2>
      <p>Proxy group and node management will be displayed here.</p>
      <button @click="importConfig" class="import-btn">Import WireGuard Config</button>
    </div>`;

proxiesContent = proxiesContent.replace(
  headerSectionHTML,
  headerSectionHTML + '\n\n    <WireguardTabs />'
);

fs.writeFileSync('frontend/src/components/views/Proxies.vue', proxiesContent);
