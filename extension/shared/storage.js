async function storageGet(keys) {
  return browser.storage.local.get(keys);
}

async function storageSet(items) {
  return browser.storage.local.set(items);
}

async function storageRemove(keys) {
  return browser.storage.local.remove(keys);
}

async function getServerUrl() {
  const data = await storageGet(STORAGE_KEYS.SERVER_URL);
  return data[STORAGE_KEYS.SERVER_URL] || DEFAULT_SERVER_URL;
}

async function getCredentials() {
  const data = await storageGet([
    STORAGE_KEYS.SERVER_URL,
    STORAGE_KEYS.USER_TOKEN,
    STORAGE_KEYS.DEVICE_TOKEN,
    STORAGE_KEYS.DEVICE_ID,
    STORAGE_KEYS.DEVICE_NAME,
    STORAGE_KEYS.USER,
  ]);
  return {
    serverUrl: data[STORAGE_KEYS.SERVER_URL] || DEFAULT_SERVER_URL,
    userToken: data[STORAGE_KEYS.USER_TOKEN] || null,
    deviceToken: data[STORAGE_KEYS.DEVICE_TOKEN] || null,
    deviceId: data[STORAGE_KEYS.DEVICE_ID] || null,
    deviceName: data[STORAGE_KEYS.DEVICE_NAME] || null,
    user: data[STORAGE_KEYS.USER] || null,
  };
}

async function setCredentials({ serverUrl, userToken, deviceToken, deviceId, deviceName, user }) {
  const items = {};
  if (serverUrl !== undefined) items[STORAGE_KEYS.SERVER_URL] = serverUrl;
  if (userToken !== undefined) items[STORAGE_KEYS.USER_TOKEN] = userToken;
  if (deviceToken !== undefined) items[STORAGE_KEYS.DEVICE_TOKEN] = deviceToken;
  if (deviceId !== undefined) items[STORAGE_KEYS.DEVICE_ID] = deviceId;
  if (deviceName !== undefined) items[STORAGE_KEYS.DEVICE_NAME] = deviceName;
  if (user !== undefined) items[STORAGE_KEYS.USER] = user;
  return storageSet(items);
}

async function clearCredentials() {
  return storageRemove([
    STORAGE_KEYS.USER_TOKEN,
    STORAGE_KEYS.DEVICE_TOKEN,
    STORAGE_KEYS.DEVICE_ID,
    STORAGE_KEYS.DEVICE_NAME,
    STORAGE_KEYS.USER,
  ]);
}

async function getRecentPushes() {
  const data = await storageGet(STORAGE_KEYS.RECENT_PUSHES);
  return data[STORAGE_KEYS.RECENT_PUSHES] || [];
}

async function setRecentPushes(pushes) {
  return storageSet({ [STORAGE_KEYS.RECENT_PUSHES]: pushes });
}

async function prependPush(push) {
  const existing = await getRecentPushes();
  if (existing.some(p => p.id === push.id)) return existing;
  const updated = [push, ...existing].slice(0, MAX_RECENT_PUSHES);
  await setRecentPushes(updated);
  return updated;
}

async function removePush(pushId) {
  const existing = await getRecentPushes();
  const updated = existing.filter(p => p.id !== pushId);
  await setRecentPushes(updated);
  return updated;
}
