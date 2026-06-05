const API_BASE = import.meta.env.VITE_API_BASE || '/api/v1';

class ApiError extends Error {
  constructor(status, message, data) {
    super(message);
    this.status = status;
    this.data = data;
  }
}

function getToken() {
  return localStorage.getItem('token');
}

async function request(path, options = {}) {
  const headers = { ...options.headers };

  if (!(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json';
  }

  const token = getToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  });

  const data = await res.json().catch(() => ({}));

  if (res.status === 401 && !window.location.pathname.startsWith('/login')) {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    window.location.href = '/login';
    throw new ApiError(401, 'Unauthorized', data);
  }

  if (!res.ok) {
    throw new ApiError(res.status, data.error || 'Request failed', data);
  }

  return data;
}

export const api = {
  // Auth
  register: (body) => request('/auth/register', { method: 'POST', body: JSON.stringify(body) }),
  login: (body) => request('/auth/login', { method: 'POST', body: JSON.stringify(body) }),
  me: () => request('/auth/me'),
  getCaptcha: () => request('/auth/captcha'),

  // Files
  listFiles: (parentId) => {
    const params = parentId ? `?parent_id=${parentId}` : '';
    return request(`/files${params}`);
  },
  uploadFile: (file, parentId) => {
    const form = new FormData();
    form.append('file', file);
    if (parentId) form.append('parent_id', parentId);
    return request('/files/upload', { method: 'POST', body: form });
  },
  mkdir: (name, parentId) => request('/files/mkdir', {
    method: 'POST',
    body: JSON.stringify({ name, parent_id: parentId }),
  }),
  downloadUrl: (id) => `${API_BASE}/files/${id}/download`,
  deleteFile: (id) => request(`/files/${id}`, { method: 'DELETE' }),
  shareFile: (id, password, expireHours) => request(`/files/${id}/share`, {
    method: 'POST',
    body: JSON.stringify({ password, expire_hours: expireHours }),
  }),
  getShare: (token) => request(`/share/${token}`),

  // Videos
  getRandomVideos: () => request("/videos/random"),

  // Forum
  listBoards: () => request('/boards'),
  listPosts: (boardId, page = 1) => request(`/boards/${boardId}/posts?page=${page}`),
  createPost: (boardId, title, content) => request(`/boards/${boardId}/posts`, {
    method: 'POST',
    body: JSON.stringify({ title, content }),
  }),
  getPost: (id) => request(`/posts/${id}`),
  replyPost: (id, content) => request(`/posts/${id}/reply`, {
    method: 'POST',
    body: JSON.stringify({ content }),
  }),
  toggleLike: (id) => request(`/posts/${id}/like`, { method: 'POST' }),
};

