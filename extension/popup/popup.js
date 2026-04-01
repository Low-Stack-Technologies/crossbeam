let state = { isLoggedIn: false, isConnected: false, deviceName: '', deviceId: '', devices: [], recentPushes: [] };
let loadMoreCursor = null;
let inMemoryOlder = [];  // older pushes loaded via "load more"
let linkModeActive = false;
let selectedFile = null;

const isStandalone = new URLSearchParams(location.search).has('standalone');

async function init() {
  if (isStandalone) {
    document.body.classList.add('standalone');
    document.getElementById('popout-btn').classList.add('hidden');
  }

  const response = await browser.runtime.sendMessage({ type: MSG_TYPES.GET_STATE });
  state = response;
  render();
}

function render() {
  renderHeader();
  renderAuthOverlay();
  renderDeviceOptions();
  renderChat();
}

function renderHeader() {
  const dot = document.getElementById('conn-dot');
  dot.classList.toggle('conn-dot--on', state.isConnected);
  dot.classList.toggle('conn-dot--off', !state.isConnected);
  dot.title = state.isConnected ? 'Connected' : 'Disconnected';
  document.getElementById('device-name').textContent = state.deviceName || 'Crossbeam';
}

function renderAuthOverlay() {
  document.getElementById('auth-overlay').classList.toggle('hidden', state.isLoggedIn);
}

function renderDeviceOptions() {
  const select = document.getElementById('target-device');
  const current = select.value;
  while (select.options.length > 1) select.remove(1);

  for (const device of (state.devices || [])) {
    if (device.id === state.deviceId) continue;
    const opt = document.createElement('option');
    opt.value = device.id;
    opt.textContent = device.name;
    select.appendChild(opt);
  }

  // Hide device selector if there's no one else to target
  select.style.display = select.options.length <= 1 ? 'none' : '';

  if ([...select.options].some(o => o.value === current)) select.value = current;
}

// ── Chat rendering ────────────────────────────────────────────────────────────

function renderChat() {
  const container = document.getElementById('chat-messages');
  const empty = document.getElementById('chat-empty');
  const loadMoreBtn = document.getElementById('load-more-btn');

  // Build ordered list: older (load-more) first, then cache (newest last = bottom)
  const allPushes = [...inMemoryOlder, ...[...state.recentPushes].reverse()];

  // Clear dynamic bubbles (keep load-more btn and empty notice)
  container.querySelectorAll('.bubble-row').forEach(el => el.remove());

  empty.classList.toggle('hidden', allPushes.length > 0);
  loadMoreBtn.classList.toggle('hidden', allPushes.length < 50 && inMemoryOlder.length === 0);

  const fragment = document.createDocumentFragment();
  for (const push of allPushes) {
    fragment.appendChild(buildBubble(push));
  }
  container.appendChild(fragment);

  // Update cursor from oldest push in cache
  const oldest = allPushes[0];
  loadMoreCursor = oldest ? oldest.created_at : null;

  scrollToBottom();
}

function buildBubble(push) {
  const isMine = push.source_device_id === state.deviceId || !push.source_device_id;
  const row = document.createElement('div');
  row.className = `bubble-row ${isMine ? 'outgoing' : 'incoming'}`;
  row.dataset.id = push.id;

  const bubble = document.createElement('div');
  bubble.className = 'bubble';

  // Title
  if (push.title) {
    const t = document.createElement('div');
    t.className = 'bubble-title';
    t.textContent = push.title;
    bubble.appendChild(t);
  }

  // Content by type
  if (push.type === PUSH_TYPES.NOTE) {
    const b = document.createElement('div');
    b.className = 'bubble-body';
    b.textContent = push.body || '';
    bubble.appendChild(b);

  } else if (push.type === PUSH_TYPES.LINK) {
    const u = document.createElement('div');
    u.className = 'bubble-url';
    const a = document.createElement('a');
    a.href = push.url || '#';
    a.target = '_blank';
    a.rel = 'noopener';
    a.textContent = push.url || '';
    u.appendChild(a);
    bubble.appendChild(u);
    if (push.body) {
      const b = document.createElement('div');
      b.className = 'bubble-body';
      b.textContent = push.body;
      bubble.appendChild(b);
    }

  } else if (push.type === PUSH_TYPES.FILE) {
    const f = document.createElement('div');
    f.className = 'bubble-file';
    f.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>`;
    if (push.file_url) {
      const a = document.createElement('a');
      a.href = push.file_url;
      a.target = '_blank';
      a.rel = 'noopener';
      a.textContent = push.file_name || 'File';
      f.appendChild(a);
    } else {
      const span = document.createElement('span');
      span.textContent = push.file_name || 'File';
      f.appendChild(span);
    }
    bubble.appendChild(f);
    if (push.file_size) {
      const s = document.createElement('div');
      s.className = 'bubble-file-size';
      s.textContent = formatBytes(push.file_size);
      bubble.appendChild(s);
    }
  }

  row.appendChild(bubble);

  // Meta row (time + actions)
  const meta = document.createElement('div');
  meta.className = 'bubble-meta';

  const time = document.createElement('span');
  time.className = 'bubble-time';
  time.textContent = timeAgo(push.created_at);
  meta.appendChild(time);

  const actions = document.createElement('div');
  actions.className = 'bubble-actions';

  // Copy action
  let copyText = null;
  if (push.type === PUSH_TYPES.NOTE) copyText = push.body || '';
  else if (push.type === PUSH_TYPES.LINK) copyText = push.url || '';

  if (copyText !== null) {
    const copyBtn = document.createElement('button');
    copyBtn.className = 'bubble-action-btn copy-btn';
    copyBtn.dataset.copy = copyText;
    copyBtn.textContent = 'Copy';
    actions.appendChild(copyBtn);
  }

  const delBtn = document.createElement('button');
  delBtn.className = 'bubble-action-btn delete';
  delBtn.dataset.delete = push.id;
  delBtn.textContent = 'Delete';
  actions.appendChild(delBtn);

  meta.appendChild(actions);
  row.appendChild(meta);

  return row;
}

function scrollToBottom() {
  const container = document.getElementById('chat-messages');
  container.scrollTop = container.scrollHeight;
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function timeAgo(isoString) {
  const diff = Date.now() - new Date(isoString).getTime();
  const s = Math.floor(diff / 1000);
  if (s < 60) return 'Now';
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  return `${Math.floor(s / 86400)}d ago`;
}

function formatBytes(bytes) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1048576).toFixed(1)} MB`;
}

// ── Input helpers ─────────────────────────────────────────────────────────────

function setLinkMode(active) {
  linkModeActive = active;
  document.getElementById('link-extras').classList.toggle('hidden', !active);
  document.getElementById('link-btn').classList.toggle('active', active);
  if (active) {
    document.getElementById('link-url').focus();
  }
}

function setSelectedFile(file) {
  selectedFile = file;
  const chip = document.getElementById('file-chip');
  if (file) {
    document.getElementById('file-chip-name').textContent = file.name;
    chip.classList.remove('hidden');
  } else {
    chip.classList.add('hidden');
    document.getElementById('file-input').value = '';
  }
}

// ── Send ──────────────────────────────────────────────────────────────────────

async function doSend() {
  const sendBtn = document.getElementById('send-btn');
  const targetDeviceId = document.getElementById('target-device').value || undefined;
  sendBtn.disabled = true;

  try {
    let res;

    if (selectedFile) {
      const fileBuffer = await new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = e => resolve(e.target.result);
        reader.onerror = reject;
        reader.readAsArrayBuffer(selectedFile);
      });
      const title = document.getElementById('message-input').value.trim() || undefined;
      res = await browser.runtime.sendMessage({
        type: MSG_TYPES.SEND_PUSH,
        pushType: PUSH_TYPES.FILE,
        data: { fileBuffer, fileName: selectedFile.name, fileType: selectedFile.type || 'application/octet-stream', title, targetDeviceId },
      });
      if (res && res.error) throw new Error(res.error);
      setSelectedFile(null);
      document.getElementById('message-input').value = '';

    } else if (linkModeActive) {
      const url = document.getElementById('link-url').value.trim();
      if (!url) { document.getElementById('link-url').focus(); return; }
      const title = document.getElementById('link-title').value.trim() || undefined;
      const body = document.getElementById('message-input').value.trim() || undefined;
      res = await browser.runtime.sendMessage({
        type: MSG_TYPES.SEND_PUSH,
        pushType: PUSH_TYPES.LINK,
        data: { url, title, body, targetDeviceId },
      });
      if (res && res.error) throw new Error(res.error);
      document.getElementById('link-url').value = '';
      document.getElementById('link-title').value = '';
      document.getElementById('message-input').value = '';
      setLinkMode(false);

    } else {
      const body = document.getElementById('message-input').value.trim();
      if (!body) { document.getElementById('message-input').focus(); return; }
      res = await browser.runtime.sendMessage({
        type: MSG_TYPES.SEND_PUSH,
        pushType: PUSH_TYPES.NOTE,
        data: { body, targetDeviceId },
      });
      if (res && res.error) throw new Error(res.error);
      document.getElementById('message-input').value = '';
      autoResizeTextarea();
    }

    hideSendError();

    // Refresh and re-render
    const fresh = await browser.runtime.sendMessage({ type: MSG_TYPES.GET_STATE });
    state = fresh;
    renderChat();

  } catch (err) {
    showSendError(err.message || 'Failed to send. Please try again.');
  } finally {
    sendBtn.disabled = false;
  }
}

// ── Send error display ────────────────────────────────────────────────────────

let sendErrorTimer = null;

function showSendError(msg) {
  const el = document.getElementById('send-error');
  el.textContent = msg;
  el.classList.remove('hidden');
  clearTimeout(sendErrorTimer);
  sendErrorTimer = setTimeout(() => el.classList.add('hidden'), 4000);
}

function hideSendError() {
  clearTimeout(sendErrorTimer);
  document.getElementById('send-error').classList.add('hidden');
}

// ── Auto-resize textarea ──────────────────────────────────────────────────────

function autoResizeTextarea() {
  const el = document.getElementById('message-input');
  el.style.height = 'auto';
  el.style.height = Math.min(el.scrollHeight, 100) + 'px';
}

// ── Event listeners ───────────────────────────────────────────────────────────

document.getElementById('attach-btn').addEventListener('click', () => {
  document.getElementById('file-input').click();
});

document.getElementById('file-input').addEventListener('change', e => {
  setSelectedFile(e.target.files[0] || null);
});

document.getElementById('clear-file-btn').addEventListener('click', () => {
  setSelectedFile(null);
});

document.getElementById('link-btn').addEventListener('click', () => {
  setLinkMode(!linkModeActive);
});

document.getElementById('message-input').addEventListener('input', autoResizeTextarea);

document.getElementById('message-input').addEventListener('keydown', e => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault();
    doSend();
  }
});

document.getElementById('send-btn').addEventListener('click', doSend);

document.getElementById('load-more-btn').addEventListener('click', async () => {
  const btn = document.getElementById('load-more-btn');
  btn.disabled = true;
  btn.textContent = 'Loading...';

  try {
    const result = await browser.runtime.sendMessage({
      type: MSG_TYPES.LOAD_MORE_PUSHES,
      cursor: loadMoreCursor,
      limit: 20,
    });
    const older = (result && result.pushes) || [];
    inMemoryOlder = [...older.reverse(), ...inMemoryOlder];
    renderChat();
  } catch { /* ignore */ } finally {
    btn.disabled = false;
    btn.textContent = 'Load older messages';
  }
});

document.getElementById('chat-messages').addEventListener('click', async e => {
  const copyBtn = e.target.closest('.copy-btn');
  if (copyBtn) {
    await navigator.clipboard.writeText(copyBtn.dataset.copy).catch(() => {});
    const orig = copyBtn.textContent;
    copyBtn.textContent = 'Copied!';
    setTimeout(() => { copyBtn.textContent = orig; }, 1500);
    return;
  }

  const delBtn = e.target.closest('[data-delete]');
  if (delBtn) {
    const pushId = delBtn.dataset.delete;
    const row = document.querySelector(`.bubble-row[data-id="${pushId}"]`);
    if (row) row.style.opacity = '0.4';
    try {
      await browser.runtime.sendMessage({ type: MSG_TYPES.DELETE_PUSH, pushId });
      inMemoryOlder = inMemoryOlder.filter(p => p.id !== pushId);
      const fresh = await browser.runtime.sendMessage({ type: MSG_TYPES.GET_STATE });
      state = fresh;
      renderChat();
    } catch {
      if (row) row.style.opacity = '';
    }
    return;
  }
});

document.getElementById('popout-btn').addEventListener('click', () => {
  browser.windows.create({
    url: browser.runtime.getURL('popup/popup.html') + '?standalone=1',
    type: 'popup',
    width: 420,
    height: 620,
  });
  window.close();
});

document.getElementById('settings-btn').addEventListener('click', () => {
  browser.runtime.openOptionsPage();
});

document.getElementById('go-settings-btn').addEventListener('click', () => {
  browser.runtime.openOptionsPage();
});

browser.runtime.onMessage.addListener(async msg => {
  if (msg.type === MSG_TYPES.SESSION_EXPIRED) {
    state.isLoggedIn = false;
    renderAuthOverlay();
  }
  if (msg.type === MSG_TYPES.PUSHES_UPDATED) {
    const fresh = await browser.runtime.sendMessage({ type: MSG_TYPES.GET_STATE });
    state = fresh;
    renderChat();
  }
});

init();
