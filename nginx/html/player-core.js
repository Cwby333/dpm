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
	const apiBaseUrl = 'https://mutestreamingservice.ru'

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

			log('play: запрос HLS плейлиста')
			const response = await apiFetch(`${apiBaseUrl}/music/play`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ music_id: musicId }),
				credentials: 'include',
			})
			const data = await response.json()
			const hlsPlaylist = data.presign_url || data.hls_playlist
			if (!hlsPlaylist) throw new Error('HLS playlist not found')

			const m3u8Blob = new Blob([hlsPlaylist], { type: 'application/vnd.apple.mpegurl' })
			const m3u8Url = URL.createObjectURL(m3u8Blob)

			if (audioElement) {
				audioElement.pause()
				audioElement = null
			}

			audioElement = new Audio()
			audioElement.addEventListener('ended', () => {
				if (repeatMode === 1) {
					audioElement.currentTime = 0
					audioElement.play()
				} else if (playlist) {
					next()
				} else {
					notifyListeners('ended', { musicId: currentMusicId })
				}
			})
			audioElement.addEventListener('playing', () => {
				notifyListeners('statechange', {})
			})
			audioElement.addEventListener('pause', () => {
				notifyListeners('statechange', {})
			})

			if (typeof Hls !== 'undefined' && Hls.isSupported()) {
				log('play: HLS supported, using hls.js')
				try {
					const hls = new Hls()
					hls.on(Hls.Events.MANIFEST_PARSED, () => {
						log('play: манифест загружен')
						if (seekTime > 0) audioElement.currentTime = seekTime
						audioElement.play()
					})
					hls.on(Hls.Events.ERROR, (event, data) => {
						if (data && data.fatal) {
							error('HLS fatal error:', data.type, data.details)
							if (data.type === Hls.ErrorTypes.MEDIA_ERROR) {
								hls.recoverMediaError()
							} else {
								notifyListeners('error', { message: 'HLS error: ' + data.details })
							}
						}
					})
					hls.loadSource(m3u8Url)
					hls.attachMedia(audioElement)
				} catch (err) {
					error('HLS init error:', err.message, err.stack)
				}
			} else {
				log('play: HLS not supported, fallback to native')
				audioElement.src = m3u8Url
				audioElement.load()
				if (seekTime > 0) audioElement.currentTime = seekTime
				await audioElement.play()
			}

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
			incrementListenCount(musicId)

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
		log(
			`next: → трек ${currentIndex + 1}/${playlist.length} — "${track.musicName}"`,
		)
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
		log(
			`prev: ← трек ${currentIndex + 1}/${playlist.length} — "${track.musicName}"`,
		)
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

	async function addToListeningHistory(musicId) {
		try {
			log(`addToListeningHistory: ${musicId}`)
			const response = await apiFetch(`${apiBaseUrl}/listening-history`, {
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

	async function incrementListenCount(musicId) {
		try {
			log(`incrementListenCount: ${musicId}`)
			const response = await apiFetch(`${apiBaseUrl}/music/inc-lis-count`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ music_id: musicId }),
				credentials: 'include',
			})
			if (response.ok) {
				log('incrementListenCount: OK')
			} else {
				warn(`incrementListenCount: статус ${response.status}`)
			}
		} catch (err) {
			warn('incrementListenCount: ошибка:', err)
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
