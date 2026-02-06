// Yao Sandbox - Chrome Stealth Initialization Script
// Injected before page load to mask automation fingerprints
// Location: /usr/local/share/yao/stealth-init.js

// Remove webdriver flag
Object.defineProperty(navigator, 'webdriver', { get: () => undefined });

// Fake chrome.runtime (Chrome Extension API)
if (!window.chrome) window.chrome = {};
if (!window.chrome.runtime) {
  window.chrome.runtime = {
    connect: function() {},
    sendMessage: function() {},
    onMessage: { addListener: function() {} },
    id: undefined
  };
}

// Fake navigator.plugins (simulate Chrome default plugins)
Object.defineProperty(navigator, 'plugins', {
  get: () => [
    { name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer', description: 'Portable Document Format' },
    { name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai', description: '' },
    { name: 'Native Client', filename: 'internal-nacl-plugin', description: '' }
  ]
});

// Fake navigator.languages
Object.defineProperty(navigator, 'languages', { get: () => ['en-US', 'en'] });

// Fix permissions API behavior
const originalQuery = window.navigator.permissions.query;
window.navigator.permissions.query = (parameters) =>
  parameters.name === 'notifications'
    ? Promise.resolve({ state: Notification.permission })
    : originalQuery(parameters);

// WebGL vendor/renderer spoofing
const getParameter = WebGLRenderingContext.prototype.getParameter;
WebGLRenderingContext.prototype.getParameter = function(parameter) {
  if (parameter === 37445) return 'Google Inc. (Intel)';           // UNMASKED_VENDOR_WEBGL
  if (parameter === 37446) return 'ANGLE (Intel, Mesa Intel(R) UHD Graphics, OpenGL 4.6)'; // UNMASKED_RENDERER_WEBGL
  return getParameter.call(this, parameter);
};
