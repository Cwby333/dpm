// api.js - единый helper для fetch с обработкой 401/403
const apiFetch = async (url, options = {}) => {
	const response = await fetch(url, options)
	if (response.status === 401 || response.status === 403) {
		alert('Для доступа к этой функции необходимо авторизоваться')
		window.location.href = '/login.html'
		throw new Error('Unauthorized')
	}
	return response
}

const apiFetchOptional = async (url, options = {}) => {
	const response = await fetch(url, options)
	if (response.status === 401 || response.status === 403) {
		console.warn(`apiFetchOptional: 401/403 on ${url} — returning null`)
		return null
	}
	return response
}
