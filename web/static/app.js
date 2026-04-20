const ENDPOINTS = {
  domains: '/api/domains',
  toggleDomain: '/api/toggle/domain',
  toggleProvider: '/api/toggle/provider',
  config: '/api/config',
  traefikConfig: '/api/config/traefik',
  providers: '/api/providers'
};

function domainEndpoint(domain) {
  return `/api/domains/${encodeURIComponent(domain)}`;
}

const AUTO_REFRESH_MS = 30000;

let domainsData = null;
let configData = null;
let isLoading = false;
let editingProviderId = null;
let autoRefreshInterval = null;

document.addEventListener('DOMContentLoaded', () => {
  loadDomains();
  initPageNavigation();
  initAutoRefresh();
});

function initAutoRefresh() {
  const toggle = document.getElementById('auto-refresh-toggle');
  const refreshBtn = document.getElementById('refresh-btn');

  refreshBtn.addEventListener('click', () => loadDomains());

  toggle.addEventListener('change', () => {
    if (toggle.checked) startAutoRefresh();
    else stopAutoRefresh();
  });

  startAutoRefresh();
}

function startAutoRefresh() {
  stopAutoRefresh();
  autoRefreshInterval = setInterval(() => {
    const page = document.getElementById('domains-page');
    if (page && !page.classList.contains('hidden')) {
      loadDomainsSilent();
    }
  }, AUTO_REFRESH_MS);
}

function stopAutoRefresh() {
  if (autoRefreshInterval) {
    clearInterval(autoRefreshInterval);
    autoRefreshInterval = null;
  }
}

async function loadDomains() {
  isLoading = true;
  showLoading();

  try {
    const response = await fetch(ENDPOINTS.domains);
    if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
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

async function loadDomainsSilent() {
  try {
    const response = await fetch(ENDPOINTS.domains);
    if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
    domainsData = await response.json();
    renderGlobalToggles();
    renderDomainTable();
  } catch (error) {
    console.error('Failed to silent refresh:', error);
  }
}

function showLoading() {
  document.getElementById('globalToggles').innerHTML = `
    <div class="loading">
      <div class="loading-spinner"></div>
      <p>加载中...</p>
    </div>
  `;
  document.getElementById('domainTableContainer').innerHTML = `
    <div class="loading">
      <div class="loading-spinner"></div>
      <p>正在加载域名列表...</p>
    </div>
  `;
}

function showError(message) {
  document.getElementById('domainTableContainer').innerHTML = `
    <div class="error-message">
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
      </svg>
      <span>${escapeHtml(message)}</span>
    </div>
  `;
}

function renderGlobalToggles() {
  const container = document.getElementById('globalToggles');
  if (!domainsData || !domainsData.providers || domainsData.providers.length === 0) {
    container.innerHTML = '<p class="empty-hint">暂无配置的提供商，请在「配置」页面添加</p>';
    return;
  }

  const providers = domainsData.providers;
  container.innerHTML = providers.map(p => {
    const globalEnabled = Object.values(domainsData.domains || {}).some(d => d && d.providers && d.providers[p.id]);
    return `
      <div class="toggle-item">
        <label class="toggle-switch">
          <input type="checkbox"
                 data-provider="${escapeHtml(p.id)}"
                 ${globalEnabled ? 'checked' : ''}
                 onchange="handleProviderToggle(this)">
          <span class="toggle-slider"></span>
        </label>
        <span class="toggle-label">${escapeHtml(p.name)}</span>
      </div>
    `;
  }).join('');
}

function renderDomainTable() {
  const container = document.getElementById('domainTableContainer');

  if (!domainsData || !domainsData.domains || Object.keys(domainsData.domains).length === 0) {
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
  const providers = domainsData.providers || [];

  container.innerHTML = `
    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th class="th-domain">域名</th>
            ${providers.map(p => `<th>${escapeHtml(p.name)}</th>`).join('')}
            <th class="th-action">操作</th>
          </tr>
        </thead>
        <tbody>
          ${Object.entries(domains).map(([domainName, entry]) => {
            const entryProviders = entry && entry.providers ? entry.providers : {};
            const records = entry && entry.records ? entry.records : {};
            const inTraefik = entry && entry.inTraefik;
            return `
              <tr>
                <td class="domain-name">
                  <a href="http://${escapeHtml(domainName)}" target="_blank" class="domain-link">${escapeHtml(domainName)}</a>
                  <div class="domain-status">
                    <span class="status-dot ${inTraefik ? 'exists' : 'missing'}"></span>
                    <span class="domain-status-text">${inTraefik ? 'Traefik' : '不在 Traefik'}</span>
                  </div>
                </td>
                ${providers.map(p => {
                  const record = records[p.id] || null;
                  const isNonManaged = record && !record.managed;
                  return `
                    <td class="toggle-cell ${isNonManaged ? 'non-managed' : ''}">
                      <div class="cell-content">
                        <div class="cell-toggle">
                          <span class="status-dot ${record ? 'exists' : 'missing'}"></span>
                          ${isNonManaged ? '<span class="managed-warning" title="该记录非本工具管理，开启将覆盖">⚠</span>' : ''}
                          <label class="toggle-switch">
                            <input type="checkbox"
                                   data-domain="${escapeHtml(domainName)}"
                                   data-provider="${escapeHtml(p.id)}"
                                   data-managed="${record ? record.managed : 'true'}"
                                   ${entryProviders[p.id] ? 'checked' : ''}
                                   onchange="handleDomainToggle(this)">
                            <span class="toggle-slider"></span>
                          </label>
                        </div>
                        ${record ? `<div class="record-info">${escapeHtml(record.value)} <span class="record-type">${escapeHtml(record.type)}</span></div>` : ''}
                      </div>
                    </td>
                  `;
                }).join('')}
                <td class="action-cell">
                  ${!inTraefik ? `<button class="btn-delete-domain" data-domain="${escapeHtml(domainName)}">删除</button>` : ''}
                </td>
              </tr>
            `;
          }).join('')}
        </tbody>
      </table>
    </div>
  `;

  container.querySelectorAll('.btn-delete-domain').forEach(btn => {
    btn.addEventListener('click', () => deleteDomain(btn.dataset.domain));
  });
}

async function handleDomainToggle(checkbox) {
  const domain = checkbox.dataset.domain;
  const providerId = checkbox.dataset.provider;
  const enabled = checkbox.checked;
  const isManaged = checkbox.dataset.managed === 'true';

  if (enabled && !isManaged) {
    const confirmed = confirm('该记录非本工具管理，开启将覆盖提供商中的现有记录，是否继续？');
    if (!confirmed) {
      checkbox.checked = false;
      return;
    }
  }

  checkbox.disabled = true;

  try {
    const response = await fetch(ENDPOINTS.toggleDomain, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ domain, providerId, enabled })
    });
    if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
    showToast(enabled ? '已开启同步' : '已关闭并删除 DNS 记录');
    loadDomainsSilent();
  } catch (error) {
    console.error('Failed to toggle domain:', error);
    checkbox.checked = !enabled;
    showToast('操作失败，请重试');
  } finally {
    checkbox.disabled = false;
  }
}

async function handleProviderToggle(checkbox) {
  const providerId = checkbox.dataset.provider;
  const enabled = checkbox.checked;

  checkbox.disabled = true;

  try {
    const response = await fetch(ENDPOINTS.toggleProvider, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ providerId, enabled })
    });
    if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
    updateAllDomainToggles(providerId, enabled);
    showToast(enabled ? '已全部开启' : '已全部关闭并删除 DNS 记录');
  } catch (error) {
    console.error('Failed to toggle provider:', error);
    checkbox.checked = !enabled;
    showToast('操作失败，请重试');
  } finally {
    checkbox.disabled = false;
  }
}

function updateAllDomainToggles(providerId, enabled) {
  document.querySelectorAll(`input[data-provider="${providerId}"]`).forEach(checkbox => {
    if (checkbox.dataset.domain) checkbox.checked = enabled;
  });
}

function showToast(message) {
  const toast = document.getElementById('toast');
  document.getElementById('toastMessage').textContent = message;
  toast.classList.add('show');
  setTimeout(() => toast.classList.remove('show'), 3000);
}

function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

function initPageNavigation() {
  document.querySelectorAll('.page-tab').forEach(tab => {
    tab.addEventListener('click', () => {
      document.querySelectorAll('.page-tab').forEach(t => t.classList.remove('active'));
      tab.classList.add('active');

      const page = tab.dataset.page;
      document.querySelectorAll('.page').forEach(p => p.classList.add('hidden'));
      document.getElementById(page + '-page').classList.remove('hidden');

      if (page === 'config') loadConfig();
      else if (page === 'domains') loadDomains();
    });
  });
}

async function loadConfig() {
  try {
    const response = await fetch(ENDPOINTS.config);
    configData = await response.json();
    renderConfig();
  } catch (err) {
    showToast('加载配置失败: ' + err.message);
  }
}

function renderConfig() {
  if (!configData) return;

  document.getElementById('tf-host').value = configData.traefik.host || '';
  document.getElementById('tf-username').value = configData.traefik.username || '';
  document.getElementById('tf-password').value = '';

  const providersList = document.getElementById('providers-list');
  if (configData.providers.length === 0) {
    providersList.innerHTML = '<p class="empty-hint">暂无提供商，请点击下方按钮添加</p>';
  } else {
    providersList.innerHTML = configData.providers.map(p => `
      <div class="provider-card">
        <div class="provider-info">
          <h4>${escapeHtml(p.name)}</h4>
          <div class="provider-type">${escapeHtml(p.type)}</div>
          <div class="provider-meta">
            ${p.host ? 'Host: ' + escapeHtml(p.host) + '<br>' : ''}
            Record: ${escapeHtml(p.record_value)}
          </div>
        </div>
        <div style="display:flex;gap:8px;">
          <button class="btn btn-secondary edit-provider" data-id="${escapeHtml(p.provider_id)}">编辑</button>
          <button class="btn btn-danger delete-provider" data-id="${escapeHtml(p.provider_id)}" data-name="${escapeHtml(p.name)}">删除</button>
        </div>
      </div>
    `).join('');
  }

  providersList.querySelectorAll('.edit-provider').forEach(btn => {
    btn.addEventListener('click', () => editProvider(btn.dataset.id));
  });
  providersList.querySelectorAll('.delete-provider').forEach(btn => {
    btn.addEventListener('click', () => deleteProvider(btn.dataset.id, btn.dataset.name));
  });
}

document.getElementById('add-provider-btn').addEventListener('click', () => {
  editingProviderId = null;
  document.getElementById('modal-title').textContent = '添加 DNS 提供商';
  document.getElementById('p-name').value = '';
  document.getElementById('p-name').disabled = false;
  document.getElementById('p-type').value = 'adguard';
  document.getElementById('p-host').value = '';
  document.getElementById('p-id').value = '';
  document.getElementById('p-secret').value = '';
  document.getElementById('p-record').value = '';
  document.getElementById('provider-modal').classList.remove('hidden');
});

document.getElementById('save-tf-btn').addEventListener('click', async () => {
  const data = {
    host: document.getElementById('tf-host').value,
    username: document.getElementById('tf-username').value,
    password: document.getElementById('tf-password').value
  };

  try {
    const response = await fetch(ENDPOINTS.traefikConfig, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    });
    if (response.ok) {
      showToast('Traefik 配置已保存');
      loadConfig();
    } else {
      showToast('保存失败: ' + await response.text());
    }
  } catch (err) {
    showToast('保存失败: ' + err.message);
  }
});

function editProvider(providerId) {
  const provider = configData.providers.find(p => p.provider_id === providerId);
  if (!provider) return;

  editingProviderId = providerId;
  document.getElementById('modal-title').textContent = '编辑 DNS 提供商';
  document.getElementById('p-name').value = provider.name;
  document.getElementById('p-name').disabled = false;
  document.getElementById('p-type').value = provider.type;
  document.getElementById('p-host').value = provider.host || '';
  document.getElementById('p-id').value = provider.id || '';
  document.getElementById('p-secret').value = '';
  document.getElementById('p-record').value = provider.record_value || '';
  document.getElementById('provider-modal').classList.remove('hidden');
}

document.getElementById('cancel-provider-btn').addEventListener('click', () => {
  document.getElementById('provider-modal').classList.add('hidden');
  editingProviderId = null;
});

document.getElementById('save-provider-btn').addEventListener('click', async () => {
  const isEdit = editingProviderId !== null;
  const data = {
    name: document.getElementById('p-name').value,
    type: document.getElementById('p-type').value,
    host: document.getElementById('p-host').value,
    id: document.getElementById('p-id').value,
    secret: document.getElementById('p-secret').value,
    record_value: document.getElementById('p-record').value
  };

  if (!data.name || !data.type || (!isEdit && !data.secret)) {
    showToast('请填写必填项（名称、类型、Secret）');
    return;
  }

  try {
    let response;
    if (isEdit) {
      response = await fetch(ENDPOINTS.providers + '/' + encodeURIComponent(editingProviderId), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
      });
    } else {
      response = await fetch(ENDPOINTS.providers, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
      });
    }

    if (response.ok || response.status === 201) {
      showToast(isEdit ? 'Provider 已更新' : 'Provider 已添加');
      document.getElementById('provider-modal').classList.add('hidden');
      editingProviderId = null;
      loadConfig();
      loadDomains();
    } else {
      showToast('操作失败: ' + await response.text());
    }
  } catch (err) {
    showToast('操作失败: ' + err.message);
  }
});

async function deleteProvider(providerId, name) {
  if (!confirm('确定要删除 Provider "' + name + '" 吗？')) return;

  try {
    const response = await fetch(ENDPOINTS.providers + '/' + encodeURIComponent(providerId), { method: 'DELETE' });
    if (response.ok) {
      showToast('Provider 已删除');
      loadConfig();
      loadDomains();
    } else {
      showToast('删除失败: ' + await response.text());
    }
  } catch (err) {
    showToast('删除失败: ' + err.message);
  }
}

async function deleteDomain(domain) {
  if (!confirm('确定要删除域名 "' + domain + '" 吗？将同时从所有已开启的提供商中删除 DNS 记录。')) return;

  try {
    const response = await fetch(domainEndpoint(domain), { method: 'DELETE' });
    if (response.ok) {
      showToast('域名已删除');
      loadDomains();
    } else {
      const data = await response.json();
      showToast(data.message || '删除失败');
    }
  } catch (err) {
    showToast('删除失败: ' + err.message);
  }
}
