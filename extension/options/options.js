let currentDeviceId = null;
let currentDeviceToken = null;
let currentServerUrl = null;

async function init() {
  await loadServerUrl();
  await loadAccountState();
}

async function loadServerUrl() {
  const data = await browser.storage.local.get(STORAGE_KEYS.SERVER_URL);
  const url = data[STORAGE_KEYS.SERVER_URL] || DEFAULT_SERVER_URL;
  document.getElementById('server-url').value = url;
  currentServerUrl = url;
}

async function loadAccountState() {
  const creds = await getCredentials();
  currentDeviceId = creds.deviceId;
  currentDeviceToken = creds.deviceToken;
  currentServerUrl = creds.serverUrl;

  if (creds.deviceToken && creds.user) {
    showLoggedIn(creds.user);
    await loadDevices(creds.serverUrl, creds.deviceToken, creds.deviceId);
  } else {
    showLoggedOut();
  }
}

function showLoggedIn(user) {
  document.getElementById('auth-form').classList.add('hidden');
  document.getElementById('account-info').classList.remove('hidden');
  document.getElementById('account-name').textContent = user.name || '';
  document.getElementById('account-email').textContent = user.email || '';
  document.getElementById('account-avatar').textContent = (user.name || user.email || '?')[0].toUpperCase();
}

function showLoggedOut() {
  document.getElementById('auth-form').classList.remove('hidden');
  document.getElementById('account-info').classList.add('hidden');
  document.getElementById('devices-list').innerHTML = '<p class="muted">Log in to see your devices.</p>';
}

async function loadDevices(serverUrl, token, thisDeviceId) {
  const container = document.getElementById('devices-list');
  try {
    const devices = await listDevices(serverUrl, token);
    if (!devices || devices.length === 0) {
      container.innerHTML = '<p class="muted">No devices registered.</p>';
      return;
    }
    renderDevicesTable(devices, thisDeviceId);
  } catch (err) {
    container.innerHTML = `<p class="muted">Failed to load devices: ${escHtml(err.message)}</p>`;
  }
}

function renderDevicesTable(devices, thisDeviceId) {
  const container = document.getElementById('devices-list');
  container.innerHTML = '';

  for (const device of devices) {
    const isThis = device.id === thisDeviceId;
    const lastSeen = device.last_seen ? timeAgo(device.last_seen) : 'Never';

    const item = document.createElement('div');
    item.className = 'device-item';
    item.innerHTML = `
      <div class="device-icon">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
          <line x1="8" y1="21" x2="16" y2="21"></line>
          <line x1="12" y1="17" x2="12" y2="21"></line>
        </svg>
      </div>
      <div class="device-info">
        <div class="device-name">
          ${escHtml(device.name)}
          ${isThis ? '<span class="this-device-badge">This device</span>' : ''}
        </div>
        <div class="device-meta">Last seen ${lastSeen}</div>
      </div>
      ${!isThis ? `<button class="btn-icon-sm remove-device-btn" data-id="${escHtml(device.id)}">Remove</button>` : ''}
    `;
    container.appendChild(item);
  }
}

function deviceNameFromUserAgent() {
  const ua = navigator.userAgent;
  let browser = 'Browser';
  let os = 'Unknown OS';

  if (ua.includes('Firefox')) browser = 'Firefox';
  else if (ua.includes('Edg/')) browser = 'Edge';
  else if (ua.includes('Chrome')) browser = 'Chrome';
  else if (ua.includes('Safari')) browser = 'Safari';

  if (ua.includes('Windows')) os = 'Windows';
  else if (ua.includes('Mac')) os = 'macOS';
  else if (ua.includes('Linux')) os = 'Linux';
  else if (ua.includes('Android')) os = 'Android';
  else if (ua.includes('iPhone') || ua.includes('iPad')) os = 'iOS';

  return `${browser} on ${os}`;
}

async function doAuth(isRegister) {
  const serverUrl = document.getElementById('server-url').value.trim() || DEFAULT_SERVER_URL;
  const email = document.getElementById('email').value.trim();
  const password = document.getElementById('password').value;
  const errorEl = document.getElementById('auth-error');

  errorEl.classList.add('hidden');

  if (!email || !password) {
    errorEl.textContent = 'Email and password are required.';
    errorEl.classList.remove('hidden');
    return;
  }

  const loginBtn = document.getElementById('login-btn');
  const registerBtn = document.getElementById('register-btn');
  loginBtn.disabled = true;
  registerBtn.disabled = true;

  try {
    let authResult;
    if (isRegister) {
      const name = email.split('@')[0]; // default name from email prefix
      authResult = await register(serverUrl, email, password, name);
    } else {
      authResult = await login(serverUrl, email, password);
    }

    const userToken = authResult.token;
    const user = authResult.user;

    // Register this browser as a device
    const deviceName = deviceNameFromUserAgent();
    const deviceResult = await registerDevice(serverUrl, userToken, deviceName, DEVICE_TYPE);

    await setCredentials({
      serverUrl,
      userToken,
      deviceToken: deviceResult.token,
      deviceId: deviceResult.device.id,
      deviceName,
      user,
    });

    currentDeviceId = deviceResult.device.id;
    currentDeviceToken = deviceResult.token;
    currentServerUrl = serverUrl;

    // Notify service worker
    browser.runtime.sendMessage({ type: MSG_TYPES.CREDENTIALS_UPDATED }).catch(() => {});

    showLoggedIn(user);
    await loadDevices(serverUrl, deviceResult.token, deviceResult.device.id);

  } catch (err) {
    errorEl.textContent = err.message || 'Authentication failed.';
    errorEl.classList.remove('hidden');
  } finally {
    loginBtn.disabled = false;
    registerBtn.disabled = false;
  }
}

async function doLogout() {
  if (currentDeviceToken && currentServerUrl && currentDeviceId) {
    try {
      await deleteDevice(currentServerUrl, currentDeviceToken, currentDeviceId);
    } catch { /* ignore errors on logout */ }
  }

  await browser.storage.local.remove([
    STORAGE_KEYS.USER_TOKEN,
    STORAGE_KEYS.DEVICE_TOKEN,
    STORAGE_KEYS.DEVICE_ID,
    STORAGE_KEYS.DEVICE_NAME,
    STORAGE_KEYS.USER,
    STORAGE_KEYS.RECENT_PUSHES,
  ]);

  currentDeviceId = null;
  currentDeviceToken = null;

  browser.runtime.sendMessage({ type: MSG_TYPES.CREDENTIALS_UPDATED }).catch(() => {});
  showLoggedOut();
}

function timeAgo(isoString) {
  const diff = Date.now() - new Date(isoString).getTime();
  const s = Math.floor(diff / 1000);
  if (s < 60) return `${s}s ago`;
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  return `${Math.floor(s / 86400)}d ago`;
}

function escHtml(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

// ── Event listeners ───────────────────────────────────────────────────────────

document.getElementById('server-url').addEventListener('change', async (e) => {
  const url = e.target.value.trim() || DEFAULT_SERVER_URL;
  await browser.storage.local.set({ [STORAGE_KEYS.SERVER_URL]: url });
  currentServerUrl = url;
});

document.getElementById('login-btn').addEventListener('click', () => doAuth(false));
document.getElementById('register-btn').addEventListener('click', () => doAuth(true));

document.getElementById('password').addEventListener('keydown', (e) => {
  if (e.key === 'Enter') doAuth(false);
});

document.getElementById('logout-btn').addEventListener('click', doLogout);

document.getElementById('devices-list').addEventListener('click', async (e) => {
  const btn = e.target.closest('.remove-device-btn');
  if (!btn) return;

  const deviceId = btn.dataset.id;
  btn.disabled = true;
  btn.textContent = '...';

  try {
    await deleteDevice(currentServerUrl, currentDeviceToken, deviceId);
    await loadDevices(currentServerUrl, currentDeviceToken, currentDeviceId);
  } catch (err) {
    btn.disabled = false;
    btn.textContent = 'Remove';
    alert('Failed to remove device: ' + err.message);
  }
});

init();
