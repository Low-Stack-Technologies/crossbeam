async function apiFetch(baseUrl, token, method, path, body) {
  const headers = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const opts = { method, headers };
  if (body !== undefined) opts.body = JSON.stringify(body);

  const res = await fetch(`${baseUrl}${path}`, opts);

  if (res.status === 401) {
    const err = new Error('Unauthorized');
    err.status = 401;
    throw err;
  }

  if (!res.ok) {
    const text = await res.text().catch(() => '');
    const err = new Error(text || `HTTP ${res.status}`);
    err.status = res.status;
    throw err;
  }

  if (res.status === 204) return null;
  return res.json();
}

async function apiFormFetch(baseUrl, token, path, formData) {
  const headers = {};
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res = await fetch(`${baseUrl}${path}`, {
    method: 'POST',
    headers,
    body: formData,
  });

  if (res.status === 401) {
    const err = new Error('Unauthorized');
    err.status = 401;
    throw err;
  }

  if (!res.ok) {
    const text = await res.text().catch(() => '');
    const err = new Error(text || `HTTP ${res.status}`);
    err.status = res.status;
    throw err;
  }

  return res.json();
}

async function login(baseUrl, email, password) {
  return apiFetch(baseUrl, null, 'POST', '/api/v1/auth/login', { email, password });
}

async function register(baseUrl, email, password, name) {
  return apiFetch(baseUrl, null, 'POST', '/api/v1/auth/register', { email, password, name });
}

async function getMe(baseUrl, token) {
  return apiFetch(baseUrl, token, 'GET', '/api/v1/users/@me');
}

async function listDevices(baseUrl, token) {
  const result = await apiFetch(baseUrl, token, 'GET', '/api/v1/devices');
  return result && result.devices ? result.devices : [];
}

async function registerDevice(baseUrl, token, name, type) {
  return apiFetch(baseUrl, token, 'POST', '/api/v1/devices', { name, type });
}

async function deleteDevice(baseUrl, token, deviceId) {
  return apiFetch(baseUrl, token, 'DELETE', `/api/v1/devices/${deviceId}`);
}

async function listPushes(baseUrl, token, cursor, limit) {
  let path = '/api/v1/pushes';
  const params = new URLSearchParams();
  if (cursor) params.set('cursor', cursor);
  if (limit) params.set('limit', String(limit));
  if ([...params].length) path += '?' + params.toString();
  return apiFetch(baseUrl, token, 'GET', path);
}

async function createNotePush(baseUrl, token, body, title, targetDeviceId) {
  const payload = { type: PUSH_TYPES.NOTE, body };
  if (title) payload.title = title;
  if (targetDeviceId) payload.target_device_id = targetDeviceId;
  const result = await apiFetch(baseUrl, token, 'POST', '/api/v1/pushes', payload);
  return result.push;
}

async function createLinkPush(baseUrl, token, url, title, body, targetDeviceId) {
  const payload = { type: PUSH_TYPES.LINK, url };
  if (title) payload.title = title;
  if (body) payload.body = body;
  if (targetDeviceId) payload.target_device_id = targetDeviceId;
  const result = await apiFetch(baseUrl, token, 'POST', '/api/v1/pushes', payload);
  return result.push;
}

async function createFilePush(baseUrl, token, fileBuffer, fileName, fileType, title, body, targetDeviceId) {
  const formData = new FormData();
  formData.append('type', PUSH_TYPES.FILE);
  formData.append('file', new Blob([fileBuffer], { type: fileType }), fileName);
  if (title) formData.append('title', title);
  if (body) formData.append('body', body);
  if (targetDeviceId) formData.append('target_device_id', targetDeviceId);
  const result = await apiFormFetch(baseUrl, token, '/api/v1/pushes', formData);
  return result.push;
}

async function deletePush(baseUrl, token, pushId) {
  return apiFetch(baseUrl, token, 'DELETE', `/api/v1/pushes/${pushId}`);
}
