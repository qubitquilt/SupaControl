import axios from 'axios';

const api = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add auth token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Handle 401 errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// Auth API
export const authAPI = {
  login: (username, password) =>
    axios.post('/api/v1/auth/login', { username, password }),
  getMe: () => api.get('/auth/me'),
  createAPIKey: (name, expiresAt = null) =>
    api.post('/auth/api-keys', { name, expires_at: expiresAt }),
  listAPIKeys: () => api.get('/auth/api-keys'),
  deleteAPIKey: (id) => api.delete(`/auth/api-keys/${id}`),
};

// Instances API
export const instancesAPI = {
  create: (name) => api.post('/instances', { name }),
  list: () => api.get('/instances'),
  get: (name) => api.get(`/instances/${name}`),
  delete: (name) => api.delete(`/instances/${name}`),
};

export default api;
