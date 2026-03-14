const CACHE_NAME = 'openoms-v3';
const STATIC_ASSETS = ['/', '/manifest.json'];

self.addEventListener('install', (e) => {
  e.waitUntil(caches.open(CACHE_NAME).then(c => c.addAll(STATIC_ASSETS)));
  self.skipWaiting();
});

self.addEventListener('activate', (e) => {
  e.waitUntil(caches.keys().then(keys =>
    Promise.all(keys.filter(k => k !== CACHE_NAME).map(k => caches.delete(k)))
  ));
  self.clients.claim();
});

self.addEventListener('fetch', (e) => {
  // Only handle same-origin requests — let cross-origin API calls pass through
  if (new URL(e.request.url).origin !== self.location.origin) return;

  // Network-first for navigation (HTML pages) — ensures fresh chunks after deploy
  if (e.request.mode === 'navigate') {
    e.respondWith(fetch(e.request).catch(() => caches.match(e.request)));
    return;
  }

  // Cache-first for static assets
  e.respondWith(caches.match(e.request).then(r => r || fetch(e.request)));
});
