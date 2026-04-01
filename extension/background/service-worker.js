// When running as a Chrome service worker, load dependencies via importScripts.
// When running as a Firefox background page (scripts[]), all deps are already
// loaded from the manifest scripts list — importScripts is not available there.
if (typeof importScripts !== 'undefined') {
  if (typeof browser === 'undefined') {
    importScripts('../browser-polyfill.min.js');
  }
  importScripts('../shared/constants.js');
  importScripts('../shared/storage.js');
  importScripts('../shared/api.js');
}

let ws = null;
let reconnectDelay = RECONNECT_BASE_MS;
let reconnectTimer = null;
let cachedDevices = [];

// ── WebSocket ────────────────────────────────────────────────────────────────

async function connectWS() {
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return;

  const creds = await getCredentials();
  if (!creds.deviceToken) return;

  const wsUrl = creds.serverUrl.replace(/^http/, 'ws') + '/gateway?token=' + creds.deviceToken;

  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    reconnectDelay = RECONNECT_BASE_MS;
    reconnectTimer = null;
  };

  ws.onmessage = async (event) => {
    let msg;
    try { msg = JSON.parse(event.data); } catch { return; }
    await handleWsMessage(msg);
  };

  ws.onclose = () => scheduleReconnect();
  ws.onerror = () => {
    ws.close();
  };
}

function scheduleReconnect() {
  if (reconnectTimer) return;
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    connectWS();
  }, reconnectDelay);
  reconnectDelay = Math.min(reconnectDelay * 2, RECONNECT_MAX_MS);
}

async function handleWsMessage(msg) {
  switch (msg.op) {
    case WS_OPS.READY: {
      const pending = msg.d && msg.d.pending_pushes ? msg.d.pending_pushes : [];
      // Prepend all pending pushes to cache (oldest first so newest ends up at top)
      for (let i = pending.length - 1; i >= 0; i--) {
        await prependPush(pending[i]);
      }
      // Reconcile local cache with server (fire-and-forget)
      const readyCreds = await getCredentials();
      syncWithServer(readyCreds.serverUrl, readyCreds.deviceToken);
      break;
    }

    case WS_OPS.PUSH_CREATE: {
      const push = msg.d;
      const creds = await getCredentials();
      const existing = await getRecentPushes();
      const alreadyExists = existing.some(p => p.id === push.id);
      await prependPush(push);
      // Only notify for pushes we did NOT just send from this device
      const fromThisDevice = push.source_device_id && push.source_device_id === creds.deviceId;
      if (!alreadyExists && !fromThisDevice) {
        showPushNotification(push);
      }
      browser.runtime.sendMessage({ type: MSG_TYPES.PUSHES_UPDATED }).catch(() => {});
      break;
    }

    case WS_OPS.PUSH_DELETE: {
      const { id } = msg.d;
      await removePush(id);
      browser.runtime.sendMessage({ type: MSG_TYPES.PUSHES_UPDATED }).catch(() => {});
      break;
    }

    case WS_OPS.DEVICE_UPDATE: {
      const creds = await getCredentials();
      try {
        cachedDevices = await listDevices(creds.serverUrl, creds.deviceToken);
      } catch {
        // ignore
      }
      break;
    }
  }
}

function showPushNotification(push) {
  let title = push.title || 'Crossbeam';
  let message = '';

  if (push.type === PUSH_TYPES.NOTE) {
    message = push.body || '';
  } else if (push.type === PUSH_TYPES.LINK) {
    message = push.url || push.body || '';
  } else if (push.type === PUSH_TYPES.FILE) {
    message = push.file_name || 'File received';
  }

  browser.notifications.create({
    type: 'basic',
    iconUrl: '../icons/icon-48.png',
    title,
    message: message.slice(0, 200),
  }).catch(() => {});
}

// ── Alarms (keepalive) ───────────────────────────────────────────────────────

browser.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === WS_ALARM_NAME) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      connectWS();
    }
  }
});

// ── Context menu ─────────────────────────────────────────────────────────────

browser.runtime.onInstalled.addListener(() => {
  browser.alarms.create(WS_ALARM_NAME, { periodInMinutes: WS_ALARM_PERIOD_MINUTES });

  browser.contextMenus.create({
    id: 'send-note',
    title: 'Send as note',
    contexts: ['selection'],
  });

  browser.contextMenus.create({
    id: 'send-link',
    title: 'Send page link',
    contexts: ['page', 'link'],
  });
});

browser.contextMenus.onClicked.addListener(async (info) => {
  const creds = await getCredentials();
  if (!creds.deviceToken) return;

  try {
    if (info.menuItemId === 'send-note' && info.selectionText) {
      const push = await createNotePush(creds.serverUrl, creds.deviceToken, info.selectionText);
      await prependPush(push);
    } else if (info.menuItemId === 'send-link') {
      const url = info.linkUrl || info.pageUrl;
      if (!url) return;
      const push = await createLinkPush(creds.serverUrl, creds.deviceToken, url);
      await prependPush(push);
    }
  } catch (err) {
    if (err.status === 401) handleSessionExpired();
  }
});

// ── Message broker ───────────────────────────────────────────────────────────

browser.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  handleMessage(msg).then(sendResponse).catch(err => {
    sendResponse({ error: err.message, status: err.status });
  });
  return true; // keep message channel open for async response
});

async function handleMessage(msg) {
  switch (msg.type) {
    case MSG_TYPES.GET_STATE: {
      const creds = await getCredentials();
      const recentPushes = await getRecentPushes();
      const isConnected = ws && ws.readyState === WebSocket.OPEN;

      if (creds.deviceToken && cachedDevices.length === 0) {
        try {
          cachedDevices = await listDevices(creds.serverUrl, creds.deviceToken);
        } catch { /* ignore */ }
      }

      return {
        isConnected: !!isConnected,
        deviceName: creds.deviceName,
        deviceId: creds.deviceId,
        user: creds.user,
        devices: cachedDevices,
        recentPushes,
        isLoggedIn: !!creds.deviceToken,
      };
    }

    case MSG_TYPES.SEND_PUSH: {
      const creds = await getCredentials();
      if (!creds.deviceToken) throw new Error('Not logged in');

      const { pushType, data } = msg;
      let push;

      if (pushType === PUSH_TYPES.NOTE) {
        push = await createNotePush(creds.serverUrl, creds.deviceToken, data.body, data.title, data.targetDeviceId);
      } else if (pushType === PUSH_TYPES.LINK) {
        push = await createLinkPush(creds.serverUrl, creds.deviceToken, data.url, data.title, data.body, data.targetDeviceId);
      } else if (pushType === PUSH_TYPES.FILE) {
        push = await createFilePush(
          creds.serverUrl, creds.deviceToken,
          data.fileBuffer, data.fileName, data.fileType,
          data.title, data.body, data.targetDeviceId
        );
      }

      await prependPush(push);
      return { success: true, push };
    }

    case MSG_TYPES.DELETE_PUSH: {
      const creds = await getCredentials();
      if (!creds.deviceToken) throw new Error('Not logged in');
      await deletePush(creds.serverUrl, creds.deviceToken, msg.pushId);
      await removePush(msg.pushId);
      return { success: true };
    }

    case MSG_TYPES.LOAD_MORE_PUSHES: {
      const creds = await getCredentials();
      if (!creds.deviceToken) throw new Error('Not logged in');
      const result = await listPushes(creds.serverUrl, creds.deviceToken, msg.cursor, msg.limit);
      return result;
    }

    case MSG_TYPES.CREDENTIALS_UPDATED: {
      cachedDevices = [];
      if (ws) {
        ws.onclose = null;
        ws.onerror = null;
        ws.close();
        ws = null;
      }
      reconnectDelay = RECONNECT_BASE_MS;
      await connectWS();
      return { success: true };
    }

    default:
      throw new Error(`Unknown message type: ${msg.type}`);
  }
}

async function syncWithServer(serverUrl, deviceToken) {
  try {
    const result = await listPushes(serverUrl, deviceToken, null, 50);
    const serverPushes = result.pushes || [];
    const local = await getRecentPushes();

    if (serverPushes.length === 0) {
      await setRecentPushes([]);
      return;
    }

    // Server returns newest-first; last item is the oldest in this page
    const oldestServerTime = new Date(serverPushes[serverPushes.length - 1].created_at).getTime();
    const serverIds = new Set(serverPushes.map(p => p.id));

    // Keep local pushes that are either outside the sync window or still on the server
    const reconciled = local.filter(p => {
      const t = new Date(p.created_at).getTime();
      return t < oldestServerTime || serverIds.has(p.id);
    });

    // Add server pushes not already in local cache
    const localIds = new Set(reconciled.map(p => p.id));
    const toAdd = serverPushes.filter(p => !localIds.has(p.id));

    const merged = [...toAdd, ...reconciled]
      .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
      .slice(0, MAX_RECENT_PUSHES);

    await setRecentPushes(merged);
  } catch { /* best-effort — don't disrupt WS flow */ }
}

async function handleSessionExpired() {
  await clearCredentials();
  browser.runtime.sendMessage({ type: MSG_TYPES.SESSION_EXPIRED }).catch(() => {});
}

async function verifyAuth() {
  const creds = await getCredentials();
  if (!creds.deviceToken) return;
  try {
    await getMe(creds.serverUrl, creds.deviceToken);
  } catch (err) {
    // Any HTTP error response means the credentials are bad (401, 404, 500 when
    // user doesn't exist, etc.). Skip if err.status is unset — that means the
    // server was unreachable (network error) and we should keep credentials.
    if (err.status) handleSessionExpired();
  }
}

// ── Startup ──────────────────────────────────────────────────────────────────

browser.alarms.create(WS_ALARM_NAME, { periodInMinutes: WS_ALARM_PERIOD_MINUTES });
connectWS();
verifyAuth();
