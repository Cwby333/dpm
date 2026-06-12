const AUDIO_CACHE = 'audio-cache-v3'

self.addEventListener('install', () => self.skipWaiting())

self.addEventListener('activate', (event) => {
    event.waitUntil(
        caches.keys().then(keys =>
            Promise.all(keys
                .filter(k => k !== AUDIO_CACHE)
                .map(k => caches.delete(k))
            )
        )
    )
})

self.addEventListener('fetch', (event) => {
    event.respondWith(handleFetch(event.request))
})

async function handleFetch(request) {
    const cache = await caches.open(AUDIO_CACHE)
    const cached = await cache.match(request)
    if (cached) {
        console.log('[SW] cache HIT:', request.url.slice(-40))
        return cached
    }

    console.log('[SW] cache MISS, fetching:', request.url.slice(-40))
    const response = await fetch(request)

    const contentType = response.headers.get('Content-Type') || ''
    if (contentType.startsWith('audio/') && response.ok) {
        const clone = response.clone()
        cache.put(request, clone)
            .then(() => console.log('[SW] cached:', request.url.slice(-40)))
            .catch(() => {})
    }

    return response
}
