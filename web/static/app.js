/**
 * Domain Sync Manager - Frontend Logic
 * Handles fetching data, rendering UI, and managing toggle states
 */

// API Endpoints
const API_BASE = '';
const ENDPOINTS = {
  domains: '/api/domains',
  toggleDomain: '/api/toggle/domain',
  toggleProvider: '/api/toggle/provider'
};

// Provider display names
const PROVIDER_NAMES = {
  dnspod: 'DNSPod',
  adguard: 'AdGuard',
  cloudflare: 'Cloudflare',
  openwrt: 'OpenWRT'
};

// State
let domainsData = null;
let isLoading = false;

/**
 * Initialize the application
 */
document.addEventListener('DOMContentLoaded', () => {
  loadData();
});

/**
 * Fetch all data from the API
 */
async function loadData() {
  isLoading = true;
  showLoading();

  try {
    const response = await fetch(ENDPOINTS.domains);
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    domainsData = await response.json();
    renderGlobalToggles();
    renderDomainTable();
  } catch (error) {
    console.error('Failed to load data:', error);
    showError('加载数据失败，请刷新页面重试');
  } finally {
    isLoading = false;
  }
}

/**
 * Show loading state
 */
function showLoading() {
  const globalTogglesContainer = document.getElementById('globalToggles');
  const domainTableContainer = document.getElementById('domainTableContainer');

  globalTogglesContainer.innerHTML = `
    <div class="loading">
      <div class="loading-spinner"></div>
      <p>加载中...</p>
    </div>
  `;

  domainTableContainer.innerHTML = `
    <div class="loading">
      <div class="loading-spinner"></div>
      <p>正在加载域名列表...</p>
    </div>
  `;
}

/**
 * Show error message
 */
function showError(message) {
  const domainTableContainer = document.getElementById('domainTableContainer');
  domainTableContainer.innerHTML = `
    <div class="error-message">
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
      </svg>
      <span>${escapeHtml(message)}</span>
    </div>
  `;
}

/**
 * Render global provider toggles
 */
function renderGlobalToggles() {
  if (!domainsData || !domainsData.providers) {
    return;
  }

  const container = document.getElementById('globalToggles');
  const providers = domainsData.providers;

  const html = Object.entries(providers).map(([key, enabled]) => `
    <div class="toggle-item">
      <label class="toggle-switch">
        <input type="checkbox" 
               data-provider="${escapeHtml(key)}" 
               ${enabled ? 'checked' : ''}
               onchange="handleProviderToggle(this)">
        <span class="toggle-slider"></span>
      </label>
      <span class="toggle-label">${escapeHtml(PROVIDER_NAMES[key] || key)}</span>
    </div>
  `).join('');

  container.innerHTML = html;
}

/**
 * Render domain table
 */
function renderDomainTable() {
  const container = document.getElementById('domainTableContainer');

  if (!domainsData || !domainsData.domains) {
    container.innerHTML = `
      <div class="empty-state">
        <div class="empty-state-icon">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
          </svg>
        </div>
        <h3>暂无域名</h3>
        <p>从 Traefik 发现的域名将显示在这里</p>
      </div>
    `;
    return;
  }

  const domains = domainsData.domains;
  const providerKeys = Object.keys(PROVIDER_NAMES);

  const html = `
    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th>域名</th>
            ${providerKeys.map(key => `<th>${escapeHtml(PROVIDER_NAMES[key])}</th>`).join('')}
          </tr>
        </thead>
        <tbody>
          ${domains.map(domain => `
            <tr>
              <td class="domain-name">${escapeHtml(domain.name)}</td>
              ${providerKeys.map(key => `
                <td class="toggle-cell">
                  <label class="toggle-switch">
                    <input type="checkbox" 
                           data-domain="${escapeHtml(domain.name)}" 
                           data-provider="${escapeHtml(key)}"
                           ${domain.providers[key] ? 'checked' : ''}
                           onchange="handleDomainToggle(this)">
                    <span class="toggle-slider"></span>
                  </label>
                </td>
              `).join('')}
            </tr>
          `).join('')}
        </tbody>
      </table>
    </div>
  `;

  container.innerHTML = html;
}

/**
 * Handle domain toggle change
 * @param {HTMLInputElement} checkbox
 */
async function handleDomainToggle(checkbox) {
  const domain = checkbox.dataset.domain;
  const provider = checkbox.dataset.provider;
  const enabled = checkbox.checked;

  // Disable the checkbox while processing
  checkbox.disabled = true;

  try {
    const response = await fetch(ENDPOINTS.toggleDomain, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        domain,
        provider,
        enabled
      })
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    showToast('已保存');
  } catch (error) {
    console.error('Failed to toggle domain:', error);
    // Revert the checkbox state on error
    checkbox.checked = !enabled;
    showToast('保存失败，请重试');
  } finally {
    checkbox.disabled = false;
  }
}

/**
 * Handle global provider toggle change
 * @param {HTMLInputElement} checkbox
 */
async function handleProviderToggle(checkbox) {
  const provider = checkbox.dataset.provider;
  const enabled = checkbox.checked;

  // Disable the checkbox while processing
  checkbox.disabled = true;

  try {
    const response = await fetch(ENDPOINTS.toggleProvider, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        provider,
        enabled
      })
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    // Update all domain toggles for this provider
    updateAllDomainToggles(provider, enabled);
    showToast('已保存');
  } catch (error) {
    console.error('Failed to toggle provider:', error);
    // Revert the checkbox state on error
    checkbox.checked = !enabled;
    showToast('保存失败，请重试');
  } finally {
    checkbox.disabled = false;
  }
}

/**
 * Update all domain toggles for a provider
 * @param {string} provider
 * @param {boolean} enabled
 */
function updateAllDomainToggles(provider, enabled) {
  const checkboxes = document.querySelectorAll(`input[data-provider="${provider}"]`);
  checkboxes.forEach(checkbox => {
    if (checkbox.dataset.domain) {
      checkbox.checked = enabled;
    }
  });
}

/**
 * Show toast notification
 * @param {string} message
 */
function showToast(message) {
  const toast = document.getElementById('toast');
  const toastMessage = document.getElementById('toastMessage');

  toastMessage.textContent = message;
  toast.classList.add('show');

  setTimeout(() => {
    toast.classList.remove('show');
  }, 3000);
}

/**
 * Escape HTML to prevent XSS
 * @param {string} text
 * @returns {string}
 */
function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}
