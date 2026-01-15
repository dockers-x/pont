const API_BASE = '/api';

const state = {
    tunnels: [],
    statuses: {},
    currentTheme: localStorage.getItem('theme') || 'light',
    logStream: null,
    isStreamConnected: false,
    editingTunnelId: null
};

const elements = {
    versionInfo: document.getElementById('version-info'),
    tunnelsList: document.getElementById('tunnels-list'),
    logsContainer: document.getElementById('logs-container'),
    clearLogsBtn: document.getElementById('clear-logs'),
    toggleStreamBtn: document.getElementById('toggle-stream'),
    themeToggle: document.getElementById('theme-toggle'),
    addTunnelBtn: document.getElementById('add-tunnel-btn'),
    tunnelModal: document.getElementById('tunnel-modal'),
    tunnelForm: document.getElementById('tunnel-form'),
    modalTitle: document.getElementById('modal-title'),
    tunnelType: document.getElementById('tunnel-type'),
    tunnelProtocol: document.getElementById('tunnel-protocol'),
    ngrokFields: document.getElementById('ngrok-fields'),
    languageSelector: document.getElementById('language-selector')
};

// Apply translations to all elements with data-i18n attribute
function applyTranslations() {
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        el.textContent = i18n.t(key);
    });

    // Update HTML lang attribute
    const htmlRoot = document.getElementById('html-root');
    if (htmlRoot) {
        const langMap = { 'zh': 'zh-CN', 'ja': 'ja-JP', 'en': 'en' };
        htmlRoot.setAttribute('lang', langMap[i18n.getLocale()] || 'en');
    }

    // Update dynamic content
    if (state.isStreamConnected) {
        elements.toggleStreamBtn.textContent = i18n.t('ui.disable_realtime_logs');
    } else {
        elements.toggleStreamBtn.textContent = i18n.t('ui.enable_realtime_logs');
    }
}

function initTheme() {
    document.documentElement.setAttribute('data-theme', state.currentTheme);
}

function toggleTheme() {
    state.currentTheme = state.currentTheme === 'light' ? 'dark' : 'light';
    document.documentElement.setAttribute('data-theme', state.currentTheme);
    localStorage.setItem('theme', state.currentTheme);
}

async function init() {
    // Load i18n first
    console.log('[i18n] Loading translations for locale:', i18n.getLocale());
    await i18n.load();
    console.log('[i18n] Translations loaded:', Object.keys(i18n.translations).length, 'keys');

    // Set language selector to current locale
    elements.languageSelector.value = i18n.getLocale();
    console.log('[i18n] Language selector set to:', i18n.getLocale());

    initTheme();
    applyTranslations();
    console.log('[i18n] Translations applied');

    elements.logsContainer.innerHTML = '';
    addLog(i18n.t('ui.system_ready'), 'system');

    await fetchVersion();
    await fetchTunnels();
    setInterval(fetchStatuses, 2000);
}

async function fetchVersion() {
    try {
        const res = await fetch(`${API_BASE}/version`);
        const data = await res.json();
        elements.versionInfo.textContent = data.version.startsWith('v') ? data.version : `v${data.version}`;
        elements.versionInfo.title = `Version: ${data.version}\nBuild: ${data.build_time}\nCommit: ${data.git_commit}`;
    } catch (err) {
        console.error('Failed to fetch version:', err);
        elements.versionInfo.textContent = 'Unknown';
    }
}

async function fetchTunnels() {
    try {
        const res = await fetch(`${API_BASE}/tunnels`);
        state.tunnels = await res.json();
        renderTunnels();
    } catch (err) {
        console.error('Failed to fetch tunnels:', err);
        addLog(`Error fetching tunnels: ${err.message}`, 'error');
    }
}

async function fetchStatuses() {
    try {
        const res = await fetch(`${API_BASE}/status`);
        state.statuses = await res.json();
        updateTunnelStatuses();
    } catch (err) {
        console.error('Failed to fetch statuses:', err);
    }
}

function renderTunnels() {
    if (!state.tunnels || state.tunnels.length === 0) {
        elements.tunnelsList.innerHTML = `<p style="color: var(--text-secondary); text-align: center;">${i18n.t('ui.no_tunnels')}</p>`;
        return;
    }

    elements.tunnelsList.innerHTML = state.tunnels.map(tunnel => {
        const status = state.statuses[tunnel.id] || { status: 'stopped' };
        const statusText = status.status === 'running' ? i18n.t('ui.tunnel.running') : i18n.t('ui.tunnel.stopped');
        return `
            <div class="tunnel-item" data-id="${tunnel.id}">
                <div class="tunnel-info">
                    <div class="tunnel-header">
                        <span class="tunnel-name">${tunnel.name}</span>
                        <span class=\"tunnel-type\" data-type="${tunnel.type}">${tunnel.type}</span>
                    </div>
                    <div class="tunnel-target">${tunnel.target}</div>
                    ${status.public_url ? `<div class="tunnel-url"><a href="${status.public_url}" target="_blank" rel="noopener noreferrer">${status.public_url}</a><button class="copy-url-btn" onclick="copyUrl('${status.public_url}', event)">Copy</button></div>` : ''}
                    ${status.error ? `<div class="log-entry error">${status.error}</div>` : ''}
                </div>
                <div class="tunnel-actions">
                    <div class="status-indicator ${status.status}" title="${statusText}"></div>
                    ${status.status === 'running' ?
                        `<button class="btn btn-danger btn-sm" onclick="stopTunnel('${tunnel.id}')">${i18n.t('ui.tunnel.stop')}</button>` :
                        `<button class="btn btn-success btn-sm" onclick="startTunnel('${tunnel.id}')">${i18n.t('ui.tunnel.start')}</button>`
                    }
                    <button class="btn btn-ghost btn-sm" onclick="editTunnel('${tunnel.id}')">${i18n.t('ui.tunnel.edit')}</button>
                    <button class="btn btn-ghost btn-sm" onclick="deleteTunnel('${tunnel.id}')">${i18n.t('ui.tunnel.delete')}</button>
                </div>
            </div>
        `;
    }).join('');
}

function updateTunnelStatuses() {
    state.tunnels.forEach(tunnel => {
        const status = state.statuses[tunnel.id] || { status: 'stopped' };
        const item = document.querySelector(`.tunnel-item[data-id="${tunnel.id}"]`);
        if (!item) return;

        const statusText = status.status === 'running' ? i18n.t('ui.tunnel.running') : i18n.t('ui.tunnel.stopped');

        const actions = item.querySelector('.tunnel-actions');
        const buttons = status.status === 'running' ?
            `<button class="btn btn-danger btn-sm" onclick="stopTunnel('${tunnel.id}')">${i18n.t('ui.tunnel.stop')}</button>` :
            `<button class="btn btn-success btn-sm" onclick="startTunnel('${tunnel.id}')">${i18n.t('ui.tunnel.start')}</button>`;

        actions.innerHTML = `
            <div class="status-indicator ${status.status}" title="${statusText}"></div>
            ${buttons}
            <button class="btn btn-ghost btn-sm" onclick="editTunnel('${tunnel.id}')">${i18n.t('ui.tunnel.edit')}</button>
            <button class="btn btn-ghost btn-sm" onclick="deleteTunnel('${tunnel.id}')">${i18n.t('ui.tunnel.delete')}</button>
        `;

        const info = item.querySelector('.tunnel-info');
        const existingUrl = info.querySelector('.tunnel-url');
        const existingError = info.querySelector('.log-entry.error');

        if (existingUrl) existingUrl.remove();
        if (existingError) existingError.remove();

        if (status.public_url) {
            const urlContainer = document.createElement('div');
            urlContainer.className = 'tunnel-url';

            const urlLink = document.createElement('a');
            urlLink.href = status.public_url;
            urlLink.target = '_blank';
            urlLink.rel = 'noopener noreferrer';
            urlLink.textContent = status.public_url;

            const copyBtn = document.createElement('button');
            copyBtn.className = 'copy-url-btn';
            copyBtn.textContent = 'Copy';
            copyBtn.onclick = (e) => copyUrl(status.public_url, e);

            urlContainer.appendChild(urlLink);
            urlContainer.appendChild(copyBtn);
            info.appendChild(urlContainer);
        }

        if (status.error) {
            const errorDiv = document.createElement('div');
            errorDiv.className = 'log-entry error';
            // Check if it's ngrok free account limit error
            if (status.error.includes('Free ngrok accounts can only run one tunnel') ||
                status.error.includes('only 1 endpoint allowed') ||
                status.error.includes('Free account limit')) {
                errorDiv.textContent = i18n.t('ui.error.ngrok_limit');
            } else {
                errorDiv.textContent = status.error;
            }
            info.appendChild(errorDiv);
        }
    });
}

async function startTunnel(id) {
    try {
        const res = await fetch(`${API_BASE}/tunnels/${id}/start`, { method: 'POST' });
        if (!res.ok) throw new Error(await res.text());
        addLog(`Starting tunnel ${id}...`, 'info');
        await fetchStatuses();
    } catch (err) {
        addLog(`Failed to start tunnel: ${err.message}`, 'error');
    }
}

async function stopTunnel(id) {
    try {
        const res = await fetch(`${API_BASE}/tunnels/${id}/stop`, { method: 'POST' });
        if (!res.ok) throw new Error(await res.text());
        addLog(`Stopping tunnel ${id}...`, 'info');
        await fetchStatuses();
    } catch (err) {
        addLog(`Failed to stop tunnel: ${err.message}`, 'error');
    }
}

async function copyUrl(url, event) {
    event.preventDefault();
    event.stopPropagation();
    try {
        await navigator.clipboard.writeText(url);
        const btn = event.target;
        const originalText = btn.textContent;
        btn.textContent = 'Copied!';
        setTimeout(() => {
            btn.textContent = originalText;
        }, 2000);
    } catch (err) {
        console.error('Failed to copy:', err);
    }
}

function editTunnel(id) {
    const tunnel = state.tunnels.find(t => t.id === id);
    if (!tunnel) return;

    state.editingTunnelId = id;
    elements.modalTitle.textContent = i18n.t('ui.modal.edit_tunnel');
    document.getElementById('tunnel-name').value = tunnel.name;
    document.getElementById('tunnel-type').value = tunnel.type;
    document.getElementById('tunnel-enabled').checked = tunnel.enabled;

    // Parse protocol and target
    let protocol = 'http://';
    let target = tunnel.target;
    if (target.startsWith('https://')) {
        protocol = 'https://';
        target = target.substring(8);
    } else if (target.startsWith('http://')) {
        protocol = 'http://';
        target = target.substring(7);
    } else if (target.startsWith('tcp://')) {
        protocol = 'tcp://';
        target = target.substring(6);
    } else if (target.startsWith('tls://')) {
        protocol = 'tls://';
        target = target.substring(6);
    }
    elements.tunnelProtocol.value = protocol;
    document.getElementById('tunnel-target').value = target;

    const authtokenInput = document.getElementById('ngrok-authtoken');
    const authtokenRequired = document.getElementById('ngrok-authtoken-required');

    if (tunnel.type === 'ngrok') {
        elements.ngrokFields.style.display = 'block';
        authtokenInput.value = tunnel.ngrok_authtoken || '';
        authtokenInput.required = true;
        if (authtokenRequired) authtokenRequired.style.display = 'inline';
        document.getElementById('ngrok-domain').value = tunnel.ngrok_domain || '';
    } else {
        authtokenInput.required = false;
        if (authtokenRequired) authtokenRequired.style.display = 'none';
    }

    elements.tunnelModal.classList.add('active');
}

async function deleteTunnel(id) {
    if (!confirm('Are you sure you want to delete this tunnel?')) return;

    try {
        const res = await fetch(`${API_BASE}/tunnels/${id}`, { method: 'DELETE' });
        if (!res.ok) throw new Error(await res.text());
        addLog(`Deleted tunnel ${id}`, 'info');
        await fetchTunnels();
    } catch (err) {
        addLog(`Failed to delete tunnel: ${err.message}`, 'error');
    }
}

function openAddTunnelModal() {
    state.editingTunnelId = null;
    elements.modalTitle.textContent = i18n.t('ui.modal.add_tunnel');
    elements.tunnelForm.reset();
    elements.tunnelProtocol.value = 'http://';
    elements.ngrokFields.style.display = 'none';
    const authtokenInput = document.getElementById('ngrok-authtoken');
    const authtokenRequired = document.getElementById('ngrok-authtoken-required');
    authtokenInput.required = false;
    if (authtokenRequired) authtokenRequired.style.display = 'none';
    elements.tunnelModal.classList.add('active');
}

function closeModal() {
    elements.tunnelModal.classList.remove('active');
    elements.tunnelForm.reset();
    state.editingTunnelId = null;
}

async function saveTunnel(e) {
    e.preventDefault();

    const protocol = elements.tunnelProtocol.value;
    const targetInput = document.getElementById('tunnel-target').value;
    const target = protocol + targetInput;

    const tunnel = {
        name: document.getElementById('tunnel-name').value,
        type: document.getElementById('tunnel-type').value,
        target: target,
        enabled: document.getElementById('tunnel-enabled').checked
    };

    if (tunnel.type === 'ngrok') {
        tunnel.ngrok_authtoken = document.getElementById('ngrok-authtoken').value;
        tunnel.ngrok_domain = document.getElementById('ngrok-domain').value;
    }

    try {
        let res;
        if (state.editingTunnelId) {
            res = await fetch(`${API_BASE}/tunnels/${state.editingTunnelId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(tunnel)
            });
        } else {
            res = await fetch(`${API_BASE}/tunnels`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(tunnel)
            });
        }

        if (!res.ok) throw new Error(await res.text());

        addLog(`${state.editingTunnelId ? 'Updated' : 'Created'} tunnel: ${tunnel.name}`, 'info');
        closeModal();
        await fetchTunnels();
    } catch (err) {
        addLog(`Failed to save tunnel: ${err.message}`, 'error');
    }
}

function connectLogStream() {
    if (state.isStreamConnected) return;

    try {
        state.logStream = new EventSource(`${API_BASE}/logs/stream`);

        state.logStream.onmessage = (event) => {
            const entry = JSON.parse(event.data);
            addLog(entry.message, entry.level);
        };

        state.logStream.onerror = () => {
            addLog('Log stream disconnected', 'error');
            state.isStreamConnected = false;
            elements.toggleStreamBtn.textContent = i18n.t('ui.enable_realtime_logs');
            if (state.logStream) {
                state.logStream.close();
                state.logStream = null;
            }
        };

        state.isStreamConnected = true;
        elements.toggleStreamBtn.textContent = i18n.t('ui.disable_realtime_logs');
        addLog('Log stream connected', 'system');
    } catch (err) {
        addLog(`Failed to connect log stream: ${err.message}`, 'error');
    }
}

function disconnectLogStream() {
    if (state.logStream) {
        state.logStream.close();
        state.logStream = null;
    }
    state.isStreamConnected = false;
    elements.toggleStreamBtn.textContent = i18n.t('ui.enable_realtime_logs');
    addLog('Log stream disconnected', 'system');
}

function toggleLogStream() {
    if (state.isStreamConnected) {
        disconnectLogStream();
    } else {
        connectLogStream();
    }
}

function addLog(message, level = 'info') {
    const entry = document.createElement('div');
    entry.className = `log-entry ${level}`;
    entry.textContent = message;
    elements.logsContainer.appendChild(entry);
    elements.logsContainer.scrollTop = elements.logsContainer.scrollHeight;
}

function clearLogs() {
    elements.logsContainer.innerHTML = '';
    addLog('Logs cleared', 'system');
}

elements.themeToggle.addEventListener('click', toggleTheme);
elements.addTunnelBtn.addEventListener('click', openAddTunnelModal);
elements.clearLogsBtn.addEventListener('click', clearLogs);
elements.toggleStreamBtn.addEventListener('click', toggleLogStream);
elements.tunnelForm.addEventListener('submit', saveTunnel);

elements.languageSelector.addEventListener('change', async (e) => {
    console.log('[i18n] Language changed to:', e.target.value);
    await i18n.setLocale(e.target.value);
    console.log('[i18n] Translations reloaded:', Object.keys(i18n.translations).length, 'keys');
    applyTranslations();
    renderTunnels(); // Re-render to update button texts
    console.log('[i18n] UI updated');
});

elements.tunnelType.addEventListener('change', (e) => {
    const isNgrok = e.target.value === 'ngrok';
    elements.ngrokFields.style.display = isNgrok ? 'block' : 'none';
    const authtokenInput = document.getElementById('ngrok-authtoken');
    const authtokenRequired = document.getElementById('ngrok-authtoken-required');
    authtokenInput.required = isNgrok;
    if (authtokenRequired) {
        authtokenRequired.style.display = isNgrok ? 'inline' : 'none';
    }
});

document.querySelector('.modal-close').addEventListener('click', closeModal);
document.querySelector('.modal-cancel').addEventListener('click', closeModal);

document.getElementById('toggle-authtoken').addEventListener('click', () => {
    const input = document.getElementById('ngrok-authtoken');
    const eyeIcon = document.querySelector('#toggle-authtoken .eye-icon');
    const eyeOffIcon = document.querySelector('#toggle-authtoken .eye-off-icon');

    if (input.type === 'password') {
        input.type = 'text';
        eyeIcon.style.display = 'none';
        eyeOffIcon.style.display = 'block';
    } else {
        input.type = 'password';
        eyeIcon.style.display = 'block';
        eyeOffIcon.style.display = 'none';
    }
});

elements.tunnelModal.addEventListener('click', (e) => {
    if (e.target === elements.tunnelModal) closeModal();
});

window.startTunnel = startTunnel;
window.stopTunnel = stopTunnel;
window.editTunnel = editTunnel;
window.deleteTunnel = deleteTunnel;

init();
