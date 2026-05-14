// player-core.js - единый модуль плеера
const PlayerCore = (function () {
	let audioElement = null
	let currentMusicId = null
	let currentMusicName = ''
	let currentArtistName = ''
	let repeatEnabled = false
	let updateInterval = null
	const apiBaseUrl = 'http://localhost:3003'

	// Инициализация
	function init() {
		console.log('PlayerCore инициализирован')
	}

	// Загрузка и воспроизведение трека
	async function play(musicId, musicName, artistName, seekTime = 0) {
		try {
			currentMusicId = musicId
			currentMusicName = musicName
			currentArtistName = artistName

			// Получаем presigned URL
			const response = await fetch(`${apiBaseUrl}/music/play`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ music_id: musicId }),
				credentials: 'include',
			})
			const data = await response.json()
			const presignedUrl = data.presign_url || data.presigned_url || data.url

			// Загружаем из сети (кеширование отключено)
			console.log('Загрузка из сети...')
			const fetchResponse = await fetch(presignedUrl)
			const audioBlob = await fetchResponse.blob()
			const blobUrl = URL.createObjectURL(audioBlob)
			console.log('Загружено из сети')

			// Останавливаем старый трек
			if (audioElement) {
				audioElement.pause()
				audioElement = null
			}

			// Создаём новый аудио элемент
			audioElement = new Audio(blobUrl)

			if (seekTime > 0) {
				audioElement.currentTime = seekTime
			}

			audioElement.addEventListener('ended', () => {
				if (repeatEnabled) {
					audioElement.currentTime = 0
					audioElement.play()
				}
				notifyListeners('ended', { musicId: currentMusicId })
			})

			await audioElement.load()
			await audioElement.play()

			notifyListeners('play', {
				musicId: currentMusicId,
				musicName: currentMusicName,
				artistName: currentArtistName,
				currentTime: audioElement.currentTime,
				duration: audioElement.duration,
				isPlaying: true,
				volume: audioElement.volume,
			})

			// Запускаем синхронизацию времени
			if (updateInterval) clearInterval(updateInterval)
			updateInterval = setInterval(() => {
				if (audioElement && !audioElement.paused) {
					notifyListeners('timeupdate', {
						currentTime: audioElement.currentTime,
						duration: audioElement.duration,
					})
				}
			}, 500)

			// Добавляем в историю
			addToListeningHistory(musicId)

			return true
		} catch (error) {
			console.error('Ошибка воспроизведения:', error)
			notifyListeners('error', { message: error.message })
			return false
		}
	}

	function pause() {
		if (audioElement) {
			audioElement.pause()
			notifyListeners('pause', {})
		}
	}

	function resume() {
		if (audioElement) {
			audioElement.play()
			notifyListeners('play', {})
		}
	}

	function seek(percent) {
		if (audioElement && audioElement.duration) {
			audioElement.currentTime = percent * audioElement.duration
		}
	}

	function setVolume(volume) {
		if (audioElement) {
			audioElement.volume = volume
			notifyListeners('volumechange', { volume: volume })
		}
	}

	function toggleRepeat() {
		repeatEnabled = !repeatEnabled
		notifyListeners('repeatchange', { repeatEnabled: repeatEnabled })
		return repeatEnabled
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
			repeatEnabled: repeatEnabled,
		}
	}

	// Слушатели для синхронизации с UI
	let listeners = []
	function addListener(callback) {
		listeners.push(callback)
	}

	function notifyListeners(event, data) {
		listeners.forEach(cb => cb(event, data))
	}

	// Добавление в историю
	async function addToListeningHistory(musicId) {
		try {
			const response = await fetch(`${apiBaseUrl}/listening-history`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ musicID: musicId }),
				credentials: 'include',
			})
			return response.ok
		} catch (error) {
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
		getState,
		addListener,
	}
})()
