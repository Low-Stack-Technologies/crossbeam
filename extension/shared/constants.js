const DEFAULT_SERVER_URL = 'https://crossbeam.low-stack.tech';

const PUSH_TYPES = {
  NOTE: 'note',
  LINK: 'link',
  FILE: 'file',
};

const WS_OPS = {
  READY: 'READY',
  PUSH_CREATE: 'PUSH_CREATE',
  PUSH_DELETE: 'PUSH_DELETE',
  DEVICE_UPDATE: 'DEVICE_UPDATE',
};

const MSG_TYPES = {
  GET_STATE: 'GET_STATE',
  SEND_PUSH: 'SEND_PUSH',
  DELETE_PUSH: 'DELETE_PUSH',
  LOAD_MORE_PUSHES: 'LOAD_MORE_PUSHES',
  CREDENTIALS_UPDATED: 'CREDENTIALS_UPDATED',
  SESSION_EXPIRED: 'SESSION_EXPIRED',
  PUSHES_UPDATED: 'PUSHES_UPDATED',
};

const STORAGE_KEYS = {
  SERVER_URL: 'serverUrl',
  USER_TOKEN: 'userToken',
  DEVICE_TOKEN: 'deviceToken',
  DEVICE_ID: 'deviceId',
  DEVICE_NAME: 'deviceName',
  USER: 'user',
  RECENT_PUSHES: 'recentPushes',
};

const DEVICE_TYPE = 'browser_extension';
const MAX_RECENT_PUSHES = 50;
const RECONNECT_BASE_MS = 2000;
const RECONNECT_MAX_MS = 60000;
const WS_ALARM_NAME = 'ws-keepalive';
const WS_ALARM_PERIOD_MINUTES = 1;
