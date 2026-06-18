const API_BASE_URL = 'https://mutestreamingservice.ru'

const apiFetch = async (url, options = {}) => {
	let response = await fetch(url, options)
	if (response.status === 401 || response.status === 403) {
		const refreshResp = await fetch(`${API_BASE_URL}/refresh`, {
			method: 'POST',
			credentials: 'include',
		})
		if (refreshResp.ok) {
			response = await fetch(url, options)
			if (response.ok) return response
		}
		alert('Для доступа к этой функции необходимо авторизоваться')
		window.location.href = '/login.html'
		throw new Error('Unauthorized')
	}
	return response
}

const apiFetchOptional = async (url, options = {}) => {
	let response = await fetch(url, options)
	if (response.status === 401 || response.status === 403) {
		const refreshResp = await fetch(`${API_BASE_URL}/refresh`, {
			method: 'POST',
			credentials: 'include',
		})
		if (refreshResp.ok) {
			response = await fetch(url, options)
			if (response.ok) return response
		}
		return null
	}
	return response
}
