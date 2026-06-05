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

  // User Profile
  getUserProfile: () => request('/user/profile'),
  getPublicUserProfile: (id) => request('/users/' + id + '/profile'),
  updateUserProfile: (data) => request('/user/profile', { method: 'PUT', body: JSON.stringify(data) }),
  updateUserPassword: (data) => request('/user/password', { method: 'PUT', body: JSON.stringify(data) }),
  getUserFiles: (type, page = 1) => request('/user/files?type=' + type + '&page=' + page),
  getUserPosts: (page = 1) => request('/user/posts?page=' + page),
  getUserLikedVideos: (page = 1) => request('/user/liked-videos?page=' + page),
  uploadAvatar: (file) => {
    const form = new FormData();
    form.append('avatar', file);
    return request('/user/avatar', { method: 'POST', body: form });
  },

  // Files
  listFiles: (parentId) => {
    const params = parentId ? `?parent_id=${parentId}` : '';
    return request(`/files${params}`);
  },
  uploadWithProgress: (file, parentId, onProgress) => {
    return new Promise((resolve, reject) => {
      const form = new FormData();
      form.append('file', file);
      if (parentId) form.append('parent_id', parentId);
      const xhr = new XMLHttpRequest();
      xhr.open('POST', API_BASE + '/files/upload');
      const token = getToken();
      if (token) xhr.setRequestHeader('Authorization', 'Bearer ' + token);
      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable && onProgress) {
          onProgress(Math.round((e.loaded / e.total) * 100));
        }
      };
      xhr.onload = () => {
        try {
          const data = JSON.parse(xhr.responseText);
          if (xhr.status >= 200 && xhr.status < 300) {
            resolve(data);
          } else {
            reject(new ApiError(xhr.status, data.error || 'Upload failed', data));
          }
        } catch (e) {
          reject(new Error('Invalid response'));
        }
      };
      xhr.onerror = () => reject(new Error('Network error'));
      xhr.send(form);
    });
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
  getVideoInfo: (id) => request(`/videos/${id}/info`),
  thumbnailUrl: (id) => API_BASE + '/videos/' + id + '/thumbnail',
  playVideoUrl: (id) => `${API_BASE}/video-play/${id}`,
  getComments: (id) => request(`/videos/${id}/comments`),
  createComment: (id, content, parentId) => request(`/videos/${id}/comments`, {
    method: 'POST',
    body: JSON.stringify({ content, parent_id: parentId }),
  }),
  getDanmaku: (id) => request(`/videos/${id}/danmaku`),
  sendDanmaku: (id, content, timeSec, color, fontSize, dmType) => request(`/videos/${id}/danmaku`, {
    method: 'POST',
    body: JSON.stringify({ content, time_sec: timeSec, color, font_size: fontSize, type: dmType }),
  }),
  toggleVideoLike: (id) => request(`/videos/${id}/like-toggle`, { method: 'POST' }),
  getVideoLikeStatus: (id) => request(`/videos/${id}/like-status`),

  // Presence
  videoHeartbeat: (id) => request(`/videos/${id}/heartbeat`, { method: 'POST' }),
  videoWatchers: (id) => request(`/videos/${id}/watchers`),

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
