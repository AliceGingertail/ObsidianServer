// ObsidianServer Admin Panel

const API_BASE = '/api';

// State
let token = localStorage.getItem('admin_token');
let usersCache = [];

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
const configClose = document.getElementById('config-close');

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
    configClose.addEventListener('click', () => {
        configModal.classList.add('hidden');
        loadPeers();
        loadStats();
    });
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
async function loadPeers() {
    try {
        const peers = await apiRequest('/admin/peers');
        const tbody = document.querySelector('#peers-table tbody');
        tbody.innerHTML = '';

        if (!peers || peers.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#aaa;">Пиры не найдены</td></tr>';
            return;
        }

        peers.forEach(peer => {
            const ipAddress = peer.wg_ip_address || peer.ovpn_ip_address || '-';
            const tr = document.createElement('tr');
            tr.innerHTML = `
                <td title="${peer.id}">${peer.id.substring(0, 8)}...</td>
                <td title="${peer.user_id}">${peer.user_id.substring(0, 8)}...</td>
                <td>${escapeHtml(peer.device_name)}</td>
                <td><span class="badge badge-info">${peer.protocol}</span></td>
                <td>${escapeHtml(ipAddress)}</td>
                <td><span class="badge ${peer.is_active ? 'badge-success' : 'badge-danger'}">${peer.is_active ? 'Да' : 'Нет'}</span></td>
                <td>
                    <button class="btn btn-danger btn-sm" onclick="deletePeer('${peer.id}', '${escapeHtml(peer.device_name)}')">Удалить</button>
                </td>
            `;
            tbody.appendChild(tr);
        });
    } catch (err) {
        console.error('Ошибка загрузки пиров:', err);
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
            option.textContent = `${user.username} (${user.email})`;
            peerUserSelect.appendChild(option);
        });

        addPeerModal.classList.remove('hidden');
    } catch (err) {
        alert('Ошибка загрузки пользователей: ' + err.message);
    }
}

async function handleCreatePeer(e) {
    e.preventDefault();

    const userId = document.getElementById('peer-user').value;
    const deviceName = document.getElementById('peer-device').value;
    const protocol = document.getElementById('peer-protocol').value;

    if (!userId || !deviceName || !protocol) {
        alert('Заполните все обязательные поля');
        return;
    }

    try {
        const result = await apiRequest('/admin/peers', {
            method: 'POST',
            body: JSON.stringify({
                user_id: userId,
                device_name: deviceName,
                protocol: protocol,
            }),
        });

        addPeerModal.classList.add('hidden');
        addPeerForm.reset();

        // Show config
        if (result.config) {
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
        document.getElementById('setting-wg-endpoint').textContent = info.wireguard_endpoint || '-';
        document.getElementById('setting-wg-port').textContent = info.wireguard_port || '-';
        document.getElementById('setting-wg-subnet').textContent = info.wireguard_subnet || '-';
        document.getElementById('setting-max-peers').textContent = info.max_peers_per_user || '-';
        document.getElementById('setting-version').textContent = info.server_version || '-';
    } catch (err) {
        console.error('Ошибка загрузки настроек:', err);
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

// Make delete functions global for onclick handlers
window.deleteUser = deleteUser;
window.deletePeer = deletePeer;
