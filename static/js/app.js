// ═══════════════════════════════════════════════════════
// WHOIS Service — Shared JavaScript Utilities
// ═══════════════════════════════════════════════════════

// Проверяем авторизацию и обновляем навигацию
function checkAuth() {
    const user = JSON.parse(localStorage.getItem('user') || '{}');

    const loginLink = document.getElementById('loginLink');
    const adminLink = document.getElementById('adminLink');
    const logoutBtn = document.getElementById('logoutBtn');

    if (user.username) {
        if (loginLink) loginLink.style.display = 'none';
        if (logoutBtn) logoutBtn.style.display = 'inline-flex';
        if (adminLink && user.role === 'admin') {
            adminLink.style.display = 'inline-flex';
        }
    } else {
        if (loginLink) loginLink.style.display = 'inline-flex';
        if (logoutBtn) logoutBtn.style.display = 'none';
        if (adminLink) adminLink.style.display = 'none';
    }
}

// Выход из аккаунта (удаляем cookie + localStorage)
function logout() {
    // Удаляем cookie на стороне клиента (установка просроченной даты)
    document.cookie = 'jwt_token=; Path=/; Expires=Thu, 01 Jan 1970 00:00:01 GMT;';
    localStorage.removeItem('user');
    window.location.href = '/';
}

// Toast-уведомления
function showToast(message, type = 'success') {
    const toast = document.getElementById('toast');
    if (!toast) return;
    toast.textContent = (type === 'success' ? '✅ ' : '❌ ') + message;
    toast.className = 'toast ' + type;
    // Trigger reflow to restart animation
    toast.offsetHeight;
    toast.classList.add('show');

    setTimeout(() => {
        toast.classList.remove('show');
    }, 3000);
}

// Форматирование даты
function formatDate(dateStr) {
    if (!dateStr) return '—';
    return new Date(dateStr).toLocaleDateString('ru-RU', {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
    });
}

// Форматирование времени
function formatDateTime(dateStr) {
    if (!dateStr) return '—';
    return new Date(dateStr).toLocaleString('ru-RU', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
}

// ═══════════════════════════════════════════════════════
// Cookie Consent Banner
// ═══════════════════════════════════════════════════════
(function() {
    if (localStorage.getItem('cookieConsent')) return;

    const banner = document.createElement('div');
    banner.id = 'cookieConsent';
    banner.innerHTML = `
        <div style="
            position:fixed; bottom:0; left:0; right:0; z-index:9999;
            background:rgba(17,19,39,0.95); backdrop-filter:blur(20px);
            border-top:1px solid rgba(255,255,255,0.08);
            padding:16px 24px; display:flex; align-items:center;
            justify-content:space-between; flex-wrap:wrap; gap:12px;
            font-family:'Inter',sans-serif; font-size:0.9rem;
            color:#94A3B8; animation:fadeIn 0.5s ease;
        ">
            <span>🍪 Мы используем файлы cookie для улучшения работы сервиса. Продолжая использовать сайт, вы соглашаетесь с нашей
                <a href="#" style="color:#7C3AED;text-decoration:underline;">Политикой конфиденциальности</a>.
            </span>
            <button onclick="acceptCookies()" style="
                background:linear-gradient(135deg,#7C3AED,#2563EB);
                color:white; border:none; padding:10px 24px;
                border-radius:8px; font-weight:600; font-size:0.85rem;
                cursor:pointer; white-space:nowrap;
                font-family:'Inter',sans-serif;
                transition:all 0.3s ease;
            ">Понятно</button>
        </div>
    `;
    document.body.appendChild(banner);
})();

function acceptCookies() {
    localStorage.setItem('cookieConsent', 'true');
    const el = document.getElementById('cookieConsent');
    if (el) el.remove();
}
