// player-core.js - единый модуль плеера
const PlayerCore = (function () {
	let audioElement = null
	let currentMusicId = null
	let currentMusicName = ''
	let currentArtistName = ''
	let repeatMode = 0 // 0=off, 1=repeat one, 2=repeat all
	let updateInterval = null
	let playlist = null
	let currentIndex = -1
	const apiBaseUrl = 'https://172.17.110.58:443'

	function log(...args) {
		console.log('[PlayerCore]', ...args)
	}

	function warn(...args) {
		console.warn('[PlayerCore]', ...args)
	}

	function error(...args) {
		console.error('[PlayerCore]', ...args)
	}

	// Инициализация
	function init() {
		log('Инициализирован')
	}

	// Загрузка и воспроизведение трека
	async function play(musicId, musicName, artistName, seekTime = 0) {
		try {
			log(`play: "${musicName}" (id=${musicId}, seek=${seekTime})`)
			currentMusicId = musicId
			currentMusicName = musicName
			currentArtistName = artistName

			log('play: запрос presigned URL')
			const response = await fetch(`${apiBaseUrl}/music/play`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ music_id: musicId }),
				credentials: 'include',
			})
			const data = await response.json()
			const presignedUrl = data.presign_url || data.presigned_url || data.url
			log(`play: presigned URL ${presignedUrl ? 'получен' : 'пустой'}`)

			let blobUrl = null
			const cachedBlob = await getCachedAudioById(musicId)

			if (cachedBlob && cachedBlob.size > 0) {
				blobUrl = URL.createObjectURL(cachedBlob)
				log('play: загружено из кеша')
			} else {
				log('play: загрузка из сети')
				const fetchResponse = await fetch(presignedUrl)
				const audioBlob = await fetchResponse.blob()
				await cacheAudioById(musicId, audioBlob)
				blobUrl = URL.createObjectURL(audioBlob)
				log(`play: загружено (${audioBlob.size} байт)`)
			}

			if (audioElement) {
				log('play: остановка старого аудио')
				audioElement.pause()
				audioElement = null
			}

			audioElement = new Audio(blobUrl)

			if (seekTime > 0) {
				audioElement.currentTime = seekTime
			}

			audioElement.addEventListener('ended', () => {
				log(`ended: "${currentMusicName}" завершился (repeatMode=${repeatMode}, playlist=${!!playlist}, index=${currentIndex}/${playlist ? playlist.length - 1 : 'n/a'})`)
				if (repeatMode === 1) {
					log('ended: повтор одного трека')
					audioElement.currentTime = 0
					audioElement.play()
				} else if (playlist) {
					log('ended: переключение на следующий трек')
					next()
				} else {
					log('ended: воспроизведение завершено')
					notifyListeners('ended', { musicId: currentMusicId })
				}
			})

			await audioElement.load()
			await audioElement.play()
			log('play: воспроизведение начато')

			notifyListeners('play', {
				musicId: currentMusicId,
				musicName: currentMusicName,
				artistName: currentArtistName,
				currentTime: audioElement.currentTime,
				duration: audioElement.duration,
				isPlaying: true,
				volume: audioElement.volume,
			})

			notifyListeners('trackchange', {
				musicId: currentMusicId,
				musicName: currentMusicName,
				artistName: currentArtistName,
				currentIndex: currentIndex,
				totalTracks: playlist ? playlist.length : 0,
			})

			if (updateInterval) clearInterval(updateInterval)
			updateInterval = setInterval(() => {
				if (audioElement && !audioElement.paused) {
					notifyListeners('timeupdate', {
						currentTime: audioElement.currentTime,
						duration: audioElement.duration,
					})
				}
			}, 500)

			addToListeningHistory(musicId)

			return true
		} catch (err) {
			error('Ошибка воспроизведения:', err)
			notifyListeners('error', { message: err.message })
			return false
		}
	}

	function pause() {
		if (audioElement) {
			log('pause')
			audioElement.pause()
			notifyListeners('pause', {})
		}
	}

	function resume() {
		if (audioElement) {
			log('resume')
			audioElement.play()
			notifyListeners('play', {})
		}
	}

	function seek(percent) {
		if (audioElement && audioElement.duration) {
			const time = percent * audioElement.duration
			log(`seek: ${(percent * 100).toFixed(1)}% -> ${time.toFixed(1)}с`)
			audioElement.currentTime = time
		}
	}

	function setVolume(volume) {
		if (audioElement) {
			audioElement.volume = volume
			notifyListeners('volumechange', { volume: volume })
		}
	}

	function toggleRepeat() {
		repeatMode = (repeatMode + 1) % 3
		const labels = ['off', 'repeat one', 'repeat all']
		log(`toggleRepeat: ${labels[repeatMode]} (${repeatMode})`)
		notifyListeners('repeatchange', { repeatMode: repeatMode })
		return repeatMode
	}

	function setPlaylist(tracks, startIndex) {
		if (!tracks || tracks.length === 0) {
			warn('setPlaylist: пустой плейлист')
			return false
		}
		playlist = tracks
		currentIndex = startIndex || 0
		log(`setPlaylist: ${tracks.length} треков, startIndex=${currentIndex}`)
		const track = playlist[currentIndex]
		return play(track.musicId, track.musicName, track.artistName)
	}

	function next() {
		if (!playlist) {
			warn('next: нет плейлиста')
			return false
		}
		if (currentIndex >= playlist.length - 1) {
			if (repeatMode === 2) {
				log('next: последний трек, repeat all — переход к первому')
				currentIndex = 0
			} else {
				log('next: последний трек в плейлисте')
				pause()
				return false
			}
		} else {
			currentIndex++
		}
		const track = playlist[currentIndex]
		log(`next: → трек ${currentIndex + 1}/${playlist.length} — "${track.musicName}"`)
		return play(track.musicId, track.musicName, track.artistName)
	}

	function prev() {
		if (!playlist) {
			warn('prev: нет плейлиста')
			return false
		}
		if (currentIndex <= 0) {
			log('prev: первый трек, перемотка в начало')
			seek(0)
			return true
		}
		if (audioElement && audioElement.currentTime > 3) {
			log('prev: перемотка текущего трека (прошло >3с)')
			seek(0)
			return true
		}
		currentIndex--
		const track = playlist[currentIndex]
		log(`prev: ← трек ${currentIndex + 1}/${playlist.length} — "${track.musicName}"`)
		return play(track.musicId, track.musicName, track.artistName)
	}

	function getState() {
		return {
			musicId: currentMusicId,
			musicName: currentMusicName,
			artistName: currentArtistName,
			isPlaying: audioElement ? !audioElement.paused : false,
			currentTime: audioElement ? audioElement.currentTime : 0,
			duration: audioElement ? audioElement.duration : 0,
			volume: audioElement ? audioElement.volume : 1,
			repeatMode: repeatMode,
			hasPlaylist: playlist !== null,
			currentIndex: currentIndex,
			totalTracks: playlist ? playlist.length : 0,
		}
	}

	let listeners = []
	function addListener(callback) {
		listeners.push(callback)
	}

	function notifyListeners(event, data) {
		listeners.forEach(cb => cb(event, data))
	}

	const CACHE_NAME = 'audio-cache-v2'

	async function cacheAudioById(musicId, audioBlob) {
		const cache = await caches.open(CACHE_NAME)
		const cacheKey = `/audio/${musicId}`
		const response = new Response(audioBlob, {
			headers: {
				'Content-Type': 'audio/mpeg',
				'Content-Length': audioBlob.size.toString(),
			},
		})
		await cache.put(cacheKey, response)
	}

	async function getCachedAudioById(musicId) {
		const cache = await caches.open(CACHE_NAME)
		const cacheKey = `/audio/${musicId}`
		const cachedResponse = await cache.match(cacheKey)
		if (cachedResponse && cachedResponse.ok) {
			return await cachedResponse.blob()
		}
		return null
	}

	async function addToListeningHistory(musicId) {
		try {
			log(`addToListeningHistory: ${musicId}`)
			const response = await fetch(`${apiBaseUrl}/listening-history`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ musicID: musicId }),
				credentials: 'include',
			})
			if (response.ok) {
				log('addToListeningHistory: OK')
			} else {
				warn(`addToListeningHistory: статус ${response.status}`)
			}
			return response.ok
		} catch (err) {
			warn('addToListeningHistory: ошибка:', err)
			return false
		}
	}

	return {
		init,
		play,
		pause,
		resume,
		seek,
		setVolume,
		toggleRepeat,
		setPlaylist,
		next,
		prev,
		getState,
		addListener,
	}
})()
