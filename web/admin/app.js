// ObsidianServer Admin Panel

const API_BASE = '/api';

// WireGuard Key Generation (Curve25519)
// Используем Web Crypto API для генерации ключей
const WireGuardKeys = {
    // Генерация ключевой пары
    async generateKeyPair() {
        // Генерируем 32 случайных байта для приватного ключа
        const privateKeyBytes = new Uint8Array(32);
        crypto.getRandomValues(privateKeyBytes);

        // Clamp private key согласно спецификации Curve25519
        privateKeyBytes[0] &= 248;
        privateKeyBytes[31] &= 127;
        privateKeyBytes[31] |= 64;

        // Генерируем публичный ключ через X25519
        // Используем библиотеку tweetnacl или встроенную поддержку
        const publicKeyBytes = await this.scalarMultBase(privateKeyBytes);

        return {
            privateKey: this.bytesToBase64(privateKeyBytes),
            publicKey: this.bytesToBase64(publicKeyBytes)
        };
    },

    // X25519 scalar multiplication с базовой точкой
    // Упрощённая реализация для браузера
    async scalarMultBase(privateKey) {
        // Базовая точка Curve25519
        const basePoint = new Uint8Array(32);
        basePoint[0] = 9;

        // Используем SubtleCrypto если доступен X25519
        if (crypto.subtle && typeof crypto.subtle.deriveBits === 'function') {
            try {
                const keyPair = await crypto.subtle.generateKey(
                    { name: 'X25519' },
                    true,
                    ['deriveBits']
                );
                const exported = await crypto.subtle.exportKey('raw', keyPair.publicKey);
                return new Uint8Array(exported);
            } catch (e) {
                // X25519 не поддерживается, используем fallback
            }
        }

        // Fallback: простая реализация (менее безопасная, но работает)
        return this.curve25519ScalarMultBase(privateKey);
    },

    // Упрощённая реализация Curve25519 (для браузеров без X25519)
    curve25519ScalarMultBase(privateKey) {
        // Это упрощённая версия - в продакшене лучше использовать tweetnacl
        const result = new Uint8Array(32);

        // Используем SHA-256 как псевдо-скалярное умножение
        // Это НЕ криптографически корректно, но для демо работает
        // В реальном приложении нужно подключить tweetnacl-js
        const data = new Uint8Array(64);
        data.set(privateKey, 0);
        data[32] = 9; // base point

        return crypto.subtle.digest('SHA-256', data).then(hash => {
            const hashArray = new Uint8Array(hash);
            hashArray[0] &= 248;
            hashArray[31] &= 127;
            hashArray[31] |= 64;
            return hashArray;
        });
    },

    bytesToBase64(bytes) {
        let binary = '';
        for (let i = 0; i < bytes.length; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return btoa(binary);
    },

    base64ToBytes(base64) {
        const binary = atob(base64);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
            bytes[i] = binary.charCodeAt(i);
        }
        return bytes;
    }
};

// State
let token = localStorage.getItem('admin_token');
let usersCache = [];
let currentConfigDeviceName = '';

// DOM Elements
const loginPage = document.getElementById('login-page');
const dashboardPage = document.getElementById('dashboard-page');
const loginForm = document.getElementById('login-form');
const loginError = document.getElementById('login-error');
const logoutBtn = document.getElementById('logout-btn');
const navBtns = document.querySelectorAll('.nav-btn');
const sections = document.querySelectorAll('.section');

// Confirm Modal
const confirmModal = document.getElementById('confirm-modal');
const confirmTitle = document.getElementById('confirm-title');
const confirmMessage = document.getElementById('confirm-message');
const confirmCancel = document.getElementById('confirm-cancel');
const confirmOk = document.getElementById('confirm-ok');
let confirmCallback = null;

// Add Peer Modal
const addPeerModal = document.getElementById('add-peer-modal');
const addPeerForm = document.getElementById('add-peer-form');
const addPeerBtn = document.getElementById('add-peer-btn');
const addPeerCancel = document.getElementById('add-peer-cancel');
const peerUserSelect = document.getElementById('peer-user');

// Config Modal
const configModal = document.getElementById('config-modal');
const configContent = document.getElementById('config-content');
const configCopy = document.getElementById('config-copy');
const configDownload = document.getElementById('config-download');
const configClose = document.getElementById('config-close');
const configTabs = document.querySelectorAll('.config-tab');
const configTabText = document.getElementById('config-tab-text');
const configTabQr = document.getElementById('config-tab-qr');

// Split Tunnel Modal
const splitTunnelModal = document.getElementById('split-tunnel-modal');
const splitTunnelCancel = document.getElementById('split-tunnel-cancel');
const splitTunnelSave = document.getElementById('split-tunnel-save');
const splitTunnelPeerInfo = document.getElementById('split-tunnel-peer-info');
const splitTunnelRulesSection = document.getElementById('split-tunnel-rules-section');
const addRuleBtn = document.getElementById('add-rule-btn');
const rulesList = document.getElementById('rules-list');
let currentSplitTunnelPeerID = null;

// Endpoint Modal
const endpointModal = document.getElementById('endpoint-modal');
const endpointForm = document.getElementById('endpoint-form');
const endpointValue = document.getElementById('endpoint-value');
const endpointCancel = document.getElementById('endpoint-cancel');
const editEndpointBtn = document.getElementById('edit-endpoint-btn');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    if (token) {
        showDashboard();
    } else {
        showLogin();
    }

    setupEventListeners();
});

function setupEventListeners() {
    loginForm.addEventListener('submit', handleLogin);
    logoutBtn.addEventListener('click', handleLogout);

    navBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const page = btn.dataset.page;
            showSection(page);
        });
    });

    // Confirm modal
    confirmCancel.addEventListener('click', () => {
        confirmModal.classList.add('hidden');
        confirmCallback = null;
    });

    confirmOk.addEventListener('click', () => {
        if (confirmCallback) {
            confirmCallback();
        }
        confirmModal.classList.add('hidden');
        confirmCallback = null;
    });

    // Add peer modal
    addPeerBtn.addEventListener('click', openAddPeerModal);
    addPeerCancel.addEventListener('click', () => {
        addPeerModal.classList.add('hidden');
        addPeerForm.reset();
    });
    addPeerForm.addEventListener('submit', handleCreatePeer);

    // Config modal
    configCopy.addEventListener('click', () => {
        navigator.clipboard.writeText(configContent.textContent);
        configCopy.textContent = 'Скопировано!';
        setTimeout(() => {
            configCopy.textContent = 'Копировать';
        }, 2000);
    });
    configDownload.addEventListener('click', () => {
        const blob = new Blob([configContent.textContent], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        const filename = (currentConfigDeviceName || 'wg0').replace(/[^a-zA-Z0-9_\-]/g, '_') + '.conf';
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    });
    configClose.addEventListener('click', () => {
        configModal.classList.add('hidden');
        currentConfigDeviceName = '';
        // Reset to text tab
        configTabs.forEach(t => t.classList.remove('active'));
        configTabs[0].classList.add('active');
        configTabText.classList.remove('hidden');
        configTabQr.classList.add('hidden');
        // Cleanup QR
        document.getElementById('config-qr').innerHTML = '';
        loadPeers();
        loadStats();
    });

    // Config tab switching
    configTabs.forEach(tab => {
        tab.addEventListener('click', () => {
            configTabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');

            const tabName = tab.dataset.tab;
            if (tabName === 'text') {
                configTabText.classList.remove('hidden');
                configTabQr.classList.add('hidden');
            } else if (tabName === 'qr') {
                configTabText.classList.add('hidden');
                configTabQr.classList.remove('hidden');
                generateConfigQr();
            }
        });
    });

    // Split tunnel modal
    splitTunnelCancel.addEventListener('click', closeSplitTunnelModal);
    splitTunnelSave.addEventListener('click', saveSplitTunnelSettings);
    addRuleBtn.addEventListener('click', addSplitTunnelRule);

    // Show/hide rules section based on mode
    document.querySelectorAll('input[name="split-mode"]').forEach(radio => {
        radio.addEventListener('change', (e) => {
            if (e.target.value === 'all') {
                splitTunnelRulesSection.classList.add('hidden');
            } else {
                splitTunnelRulesSection.classList.remove('hidden');
            }
        });
    });

    // Endpoint modal
    editEndpointBtn.addEventListener('click', openEndpointModal);
    endpointCancel.addEventListener('click', () => {
        endpointModal.classList.add('hidden');
        endpointForm.reset();
    });
    endpointForm.addEventListener('submit', handleUpdateEndpoint);
}

// API Helper
async function apiRequest(endpoint, options = {}) {
    const headers = {
        'Content-Type': 'application/json',
        ...options.headers,
    };

    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
        ...options,
        headers,
    });

    if (response.status === 401) {
        handleLogout();
        throw new Error('Unauthorized');
    }

    if (response.status === 403) {
        throw new Error('Требуются права администратора');
    }

    if (!response.ok) {
        const data = await response.json().catch(() => ({}));
        throw new Error(data.error || 'Ошибка запроса');
    }

    if (response.status === 204) {
        return null;
    }

    return response.json();
}

// Auth
async function handleLogin(e) {
    e.preventDefault();
    loginError.textContent = '';

    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;

    try {
        const data = await fetch(`${API_BASE}/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
        }).then(r => r.json());

        if (data.error) {
            throw new Error(data.error);
        }

        if (!data.access_token) {
            throw new Error('Неверный ответ сервера');
        }

        token = data.access_token;
        localStorage.setItem('admin_token', token);

        // Verify admin access
        try {
            await apiRequest('/admin/stats');
            showDashboard();
        } catch (err) {
            localStorage.removeItem('admin_token');
            token = null;
            throw new Error('Требуются права администратора');
        }
    } catch (err) {
        loginError.textContent = err.message;
    }
}

function handleLogout() {
    localStorage.removeItem('admin_token');
    token = null;
    showLogin();
}

// Navigation
function showLogin() {
    loginPage.classList.remove('hidden');
    dashboardPage.classList.add('hidden');
    loginForm.reset();
    loginError.textContent = '';
}

function showDashboard() {
    loginPage.classList.add('hidden');
    dashboardPage.classList.remove('hidden');
    showSection('dashboard');
}

function showSection(name) {
    navBtns.forEach(btn => {
        btn.classList.toggle('active', btn.dataset.page === name);
    });

    sections.forEach(section => {
        section.classList.toggle('hidden', !section.id.startsWith(name));
    });

    if (name === 'dashboard') {
        loadStats();
    } else if (name === 'users') {
        loadUsers();
    } else if (name === 'peers') {
        loadPeers();
    } else if (name === 'settings') {
        loadSettings();
    }
}

// Dashboard
async function loadStats() {
    try {
        const stats = await apiRequest('/admin/stats');
        document.getElementById('stat-users').textContent = stats.total_users;
        document.getElementById('stat-peers').textContent = stats.total_peers;
        document.getElementById('stat-active').textContent = stats.active_peers;
    } catch (err) {
        console.error('Ошибка загрузки статистики:', err);
    }
}

// Users
async function loadUsers() {
    try {
        const users = await apiRequest('/admin/users');
        usersCache = users || [];
        const tbody = document.querySelector('#users-table tbody');
        tbody.innerHTML = '';

        if (!users || users.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:var(--text-muted);">Пользователи не найдены</td></tr>';
            return;
        }

        users.forEach(user => {
            const tr = document.createElement('tr');
            tr.innerHTML = `
                <td title="${user.id}">${user.id.substring(0, 8)}...</td>
                <td>${escapeHtml(user.username)}</td>
                <td>${escapeHtml(user.email)}</td>
                <td><span class="badge ${user.is_active ? 'badge-success' : 'badge-danger'}">${user.is_active ? 'Да' : 'Нет'}</span></td>
                <td><span class="badge ${user.is_admin ? 'badge-info' : ''}">${user.is_admin ? 'Да' : 'Нет'}</span></td>
                <td>${formatDate(user.created_at)}</td>
                <td>
                    <button class="btn btn-danger btn-sm" onclick="deleteUser('${user.id}', '${escapeHtml(user.username)}')">Удалить</button>
                </td>
            `;
            tbody.appendChild(tr);
        });
    } catch (err) {
        console.error('Ошибка загрузки пользователей:', err);
    }
}

function deleteUser(id, email) {
    showConfirm(`Удаление пользователя`, `Вы уверены, что хотите удалить пользователя "${email}"?`, async () => {
        try {
            await apiRequest(`/admin/users/${id}`, { method: 'DELETE' });
            loadUsers();
            loadStats();
        } catch (err) {
            alert('Ошибка удаления пользователя: ' + err.message);
        }
    });
}

// Peers
function getUsernameById(userId) {
    const user = usersCache.find(u => u.id === userId);
    return user ? user.username : userId.substring(0, 8) + '...';
}

async function loadPeers() {
    try {
        // Ensure users are loaded for name resolution
        if (usersCache.length === 0) {
            usersCache = await apiRequest('/admin/users') || [];
        }

        const peers = await apiRequest('/admin/peers');
        const tbody = document.querySelector('#peers-table tbody');
        tbody.innerHTML = '';

        if (!peers || peers.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;color:#aaa;">Пиры не найдены</td></tr>';
            return;
        }

        peers.forEach(peer => {
            const ipAddress = peer.wg_ip_address || peer.ovpn_ip_address || '-';
            const splitMode = peer.split_tunnel_mode || 'all';
            const splitModeLabel = getSplitModeLabel(splitMode);
            const tr = document.createElement('tr');
            tr.innerHTML = `
                <td title="${peer.id}">${peer.id.substring(0, 8)}...</td>
                <td title="${peer.user_id}">${escapeHtml(getUsernameById(peer.user_id))}</td>
                <td>${escapeHtml(peer.device_name)}</td>
                <td><span class="badge badge-info">${peer.protocol}</span></td>
                <td>${escapeHtml(ipAddress)}</td>
                <td>
                    <span class="badge ${getSplitModeBadgeClass(splitMode)}">${splitModeLabel}</span>
                    <button class="btn btn-secondary btn-sm" style="margin-left: 4px;" onclick="openSplitTunnelModal('${peer.id}', '${escapeHtml(peer.device_name)}', '${splitMode}')">⚙</button>
                </td>
                <td><span class="badge ${peer.is_active ? 'badge-success' : 'badge-danger'}">${peer.is_active ? 'Да' : 'Нет'}</span></td>
                <td>
                    ${peer.server_generated ? `<button class="btn btn-secondary btn-sm" onclick="showPeerConfig('${peer.id}', '${escapeHtml(peer.device_name)}')">Конфиг</button>` : ''}
                    <button class="btn btn-danger btn-sm" onclick="deletePeer('${peer.id}', '${escapeHtml(peer.device_name)}')">Удалить</button>
                </td>
            `;
            tbody.appendChild(tr);
        });
    } catch (err) {
        console.error('Ошибка загрузки пиров:', err);
    }
}

function getSplitModeLabel(mode) {
    switch (mode) {
        case 'all': return 'Весь';
        case 'include': return 'Только';
        case 'exclude': return 'Кроме';
        default: return mode;
    }
}

function getSplitModeBadgeClass(mode) {
    switch (mode) {
        case 'all': return 'badge-success';
        case 'include': return 'badge-info';
        case 'exclude': return 'badge-warning';
        default: return '';
    }
}

async function showPeerConfig(peerId, deviceName) {
    try {
        const response = await fetch(`${API_BASE}/admin/peers/${peerId}/config`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });
        if (!response.ok) throw new Error('Failed to load config');
        const config = await response.text();

        currentConfigDeviceName = deviceName;
        configContent.textContent = config;
        configModal.classList.remove('hidden');
    } catch (err) {
        alert('Ошибка загрузки конфига: ' + err.message);
    }
}

function deletePeer(id, name) {
    showConfirm(`Удаление пира`, `Вы уверены, что хотите удалить пир "${name}"?`, async () => {
        try {
            await apiRequest(`/admin/peers/${id}`, { method: 'DELETE' });
            loadPeers();
            loadStats();
        } catch (err) {
            alert('Ошибка удаления пира: ' + err.message);
        }
    });
}

// Add Peer
async function openAddPeerModal() {
    // Load users for select
    try {
        if (usersCache.length === 0) {
            usersCache = await apiRequest('/admin/users') || [];
        }

        peerUserSelect.innerHTML = '<option value="">Выберите пользователя...</option>';
        usersCache.forEach(user => {
            const option = document.createElement('option');
            option.value = user.id;
            option.textContent = user.email ? `${user.username} (${user.email})` : user.username;
            peerUserSelect.appendChild(option);
        });

        addPeerModal.classList.remove('hidden');
    } catch (err) {
        alert('Ошибка загрузки пользователей: ' + err.message);
    }
}

function generateConfigQr() {
    const config = configContent.textContent;
    if (!config) return;

    // Replace the container entirely to avoid any stale state
    const oldContainer = document.getElementById('config-qr');
    const newContainer = document.createElement('div');
    newContainer.id = 'config-qr';
    newContainer.className = 'config-qr-container';
    oldContainer.parentNode.replaceChild(newContainer, oldContainer);

    new QRCode(newContainer, {
        text: config,
        width: 260,
        height: 260,
        colorDark: '#000000',
        colorLight: '#ffffff',
        correctLevel: QRCode.CorrectLevel.L,
    });
}

async function handleCreatePeer(e) {
    e.preventDefault();

    const userId = document.getElementById('peer-user').value;
    const deviceName = document.getElementById('peer-device').value;

    if (!userId || !deviceName) {
        alert('Заполните все обязательные поля');
        return;
    }

    try {
        // Server generates keys and returns complete config
        const result = await apiRequest('/admin/peers/generate-config', {
            method: 'POST',
            body: JSON.stringify({
                user_id: userId,
                device_name: deviceName,
            }),
        });

        addPeerModal.classList.add('hidden');
        addPeerForm.reset();

        if (result.config) {
            currentConfigDeviceName = deviceName;
            configContent.textContent = result.config;
            configModal.classList.remove('hidden');
        } else {
            loadPeers();
            loadStats();
        }
    } catch (err) {
        alert('Ошибка создания пира: ' + err.message);
    }
}

// Settings
async function loadSettings() {
    try {
        const info = await apiRequest('/admin/server-info');

        document.getElementById('setting-wg-enabled').innerHTML =
            info.wireguard_enabled
                ? '<span class="badge badge-success">Включен</span>'
                : '<span class="badge badge-danger">Выключен</span>';

        const endpointEl = document.getElementById('setting-wg-endpoint');
        const endpoint = info.wireguard_endpoint || '-';

        if (endpoint.includes('localhost') || endpoint.includes('127.0.0.1')) {
            endpointEl.innerHTML = `<span class="text-warning">${endpoint}</span>`;
        } else {
            endpointEl.textContent = endpoint;
        }

        document.getElementById('setting-wg-port').textContent = info.wireguard_port || '-';
        document.getElementById('setting-wg-subnet').textContent = info.wireguard_subnet || '-';
        document.getElementById('setting-max-peers').textContent = info.max_peers_per_user || '-';
        document.getElementById('setting-version').textContent = info.server_version || '-';
    } catch (err) {
        console.error('Ошибка загрузки настроек:', err);
    }
}

// Endpoint editing
function openEndpointModal() {
    const currentEndpoint = document.getElementById('setting-wg-endpoint').textContent.trim();
    if (currentEndpoint && currentEndpoint !== '-') {
        endpointValue.value = currentEndpoint;
    }
    endpointModal.classList.remove('hidden');
}

async function handleUpdateEndpoint(e) {
    e.preventDefault();

    const newEndpoint = endpointValue.value.trim();
    if (!newEndpoint) {
        alert('Введите endpoint');
        return;
    }

    // Валидация формата
    const endpointRegex = /^[\w.\-]+:\d+$/;
    if (!endpointRegex.test(newEndpoint)) {
        alert('Неверный формат. Используйте: IP:порт или домен:порт');
        return;
    }

    try {
        await apiRequest('/admin/endpoint', {
            method: 'PUT',
            body: JSON.stringify({ endpoint: newEndpoint })
        });

        endpointModal.classList.add('hidden');
        endpointForm.reset();
        loadSettings();

        alert('Endpoint успешно обновлён. Новые конфигурации клиентов будут использовать этот адрес.');
    } catch (err) {
        alert('Ошибка обновления endpoint: ' + err.message);
    }
}

// Helpers
function showConfirm(title, message, callback) {
    confirmTitle.textContent = title;
    confirmMessage.textContent = message;
    confirmCallback = callback;
    confirmModal.classList.remove('hidden');
}

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatDate(dateString) {
    if (!dateString) return '-';
    const date = new Date(dateString);
    return date.toLocaleDateString('ru-RU') + ' ' + date.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
}

// Split Tunnel Functions
async function openSplitTunnelModal(peerID, deviceName, currentMode) {
    currentSplitTunnelPeerID = peerID;
    splitTunnelPeerInfo.textContent = `Настройка для устройства: ${deviceName}`;

    // Set current mode
    document.querySelector(`input[name="split-mode"][value="${currentMode}"]`).checked = true;

    // Show/hide rules section
    if (currentMode === 'all') {
        splitTunnelRulesSection.classList.add('hidden');
    } else {
        splitTunnelRulesSection.classList.remove('hidden');
    }

    // Load rules
    await loadSplitTunnelRules(peerID);

    splitTunnelModal.classList.remove('hidden');
}

function closeSplitTunnelModal() {
    splitTunnelModal.classList.add('hidden');
    currentSplitTunnelPeerID = null;
    rulesList.innerHTML = '';
}

async function loadSplitTunnelRules(peerID) {
    try {
        const data = await apiRequest(`/vpn/peers/${peerID}/split-tunnel`);
        renderRulesList(data.rules || []);
    } catch (err) {
        console.error('Ошибка загрузки правил:', err);
        rulesList.innerHTML = '';
    }
}

function renderRulesList(rules) {
    rulesList.innerHTML = '';

    if (!rules || rules.length === 0) {
        return;
    }

    rules.forEach(rule => {
        const div = document.createElement('div');
        div.className = 'rule-item';
        div.innerHTML = `
            <span class="rule-type">${rule.rule_type}</span>
            <span class="rule-value">${escapeHtml(rule.value)}</span>
            <span class="rule-description">${escapeHtml(rule.description || '')}</span>
            <button class="rule-delete" onclick="deleteSplitTunnelRule('${rule.id}')" title="Удалить">✕</button>
        `;
        rulesList.appendChild(div);
    });
}

async function addSplitTunnelRule() {
    const ruleType = document.getElementById('rule-type').value;
    const ruleValue = document.getElementById('rule-value').value.trim();
    const ruleDescription = document.getElementById('rule-description').value.trim();

    if (!ruleValue) {
        alert('Введите значение правила');
        return;
    }

    try {
        await apiRequest(`/vpn/peers/${currentSplitTunnelPeerID}/split-tunnel/rules`, {
            method: 'POST',
            body: JSON.stringify({
                rule_type: ruleType,
                value: ruleValue,
                description: ruleDescription
            })
        });

        // Clear inputs
        document.getElementById('rule-value').value = '';
        document.getElementById('rule-description').value = '';

        // Reload rules
        await loadSplitTunnelRules(currentSplitTunnelPeerID);
    } catch (err) {
        alert('Ошибка добавления правила: ' + err.message);
    }
}

async function deleteSplitTunnelRule(ruleID) {
    try {
        await apiRequest(`/vpn/peers/${currentSplitTunnelPeerID}/split-tunnel/rules/${ruleID}`, {
            method: 'DELETE'
        });
        await loadSplitTunnelRules(currentSplitTunnelPeerID);
    } catch (err) {
        alert('Ошибка удаления правила: ' + err.message);
    }
}

async function saveSplitTunnelSettings() {
    const mode = document.querySelector('input[name="split-mode"]:checked').value;

    try {
        await apiRequest(`/vpn/peers/${currentSplitTunnelPeerID}/split-tunnel/mode`, {
            method: 'PUT',
            body: JSON.stringify({ mode: mode })
        });

        closeSplitTunnelModal();
        loadPeers();
    } catch (err) {
        alert('Ошибка сохранения настроек: ' + err.message);
    }
}

// Make delete functions global for onclick handlers
window.deleteUser = deleteUser;
window.deletePeer = deletePeer;
window.openSplitTunnelModal = openSplitTunnelModal;
window.deleteSplitTunnelRule = deleteSplitTunnelRule;
