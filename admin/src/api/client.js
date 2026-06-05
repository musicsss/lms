const API_BASE = '/api/v1'

function getToken() {
  return localStorage.getItem('admin_token')
}

async function request(path, options = {}) {
  const headers = { ...options.headers }

  if (!(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }

  const token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE}${path}`, { ...options, headers })

  const data = await res.json().catch(() => ({}))

  if (res.status === 401 && !window.location.pathname.startsWith('/login')) {
    localStorage.removeItem('admin_token')
    localStorage.removeItem('admin_user')
    window.location.href = '/login'
    throw new Error('Unauthorized')
  }

  if (res.status === 403) {
    throw new Error('Admin access required')
  }

  if (!res.ok) {
    const err = new Error(data.error || 'Request failed')
    err.data = data
    throw err
  }

  return data
}

export const api = {
  login: (body) => request('/auth/login', { method: 'POST', body: JSON.stringify(body) }),
  getCaptcha: () => request('/auth/captcha'),

  // generic methods for low-level access
  get: (path) => request(path),
  post: (path, body) => request(path, { method: 'POST', body: JSON.stringify(body) }),
  put: (path, body) => request(path, { method: 'PUT', body: JSON.stringify(body) }),
  del: (path) => request(path, { method: 'DELETE' }),

  stats: () => request('/admin/stats'),

  listUsers: (page = 1, search = '') =>
    request(`/admin/users?page=${page}&page_size=20&search=${encodeURIComponent(search)}`),
  updateUserRole: (id, role) =>
    request(`/admin/users/${id}`, { method: 'PUT', body: JSON.stringify({ role }) }),
  deleteUser: (id) =>
    request(`/admin/users/${id}`, { method: 'DELETE' }),

  listFiles: (page = 1) =>
    request(`/admin/files?page=${page}&page_size=20`),
  deleteFile: (id) =>
    request(`/admin/files/${id}`, { method: 'DELETE' }),

  listBoards: () => request('/admin/boards'),
  createBoard: (data) =>
    request('/admin/boards', { method: 'POST', body: JSON.stringify(data) }),
  updateBoard: (id, data) =>
    request(`/admin/boards/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteBoard: (id) =>
    request(`/admin/boards/${id}`, { method: 'DELETE' }),
  listPosts: (boardId, page = 1) =>
    request(`/admin/boards/${boardId}/posts?page=${page}&page_size=20`),
  deletePost: (id) =>
    request(`/admin/posts/${id}`, { method: 'DELETE' }),

  // Config
  getConfigTargets: () => request('/admin/config/targets'),
  execConfigCommand: (command) =>
    request('/admin/config/exec', { method: 'POST', body: JSON.stringify({ command }) }),
}
