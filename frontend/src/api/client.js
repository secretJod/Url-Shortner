import axios from 'axios';

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '',
  headers: {
    'Content-Type': 'application/json',
  },
});

apiClient.interceptors.request.use(
  (config) => {
    const apiKey = localStorage.getItem('urlshortener_api_key');
    if (apiKey) {
      config.headers.Authorization = `Bearer ${apiKey}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response) {
      const { status, data } = error.response;
      
      if (status === 401) {
        localStorage.removeItem('urlshortener_api_key');
        if (window.location.pathname !== '/login') {
          window.location.href = '/login';
        }
      }
      
      let errorMessage = data?.error || data?.message || 'An unexpected error occurred';
      
      if (status === 429) {
        errorMessage = 'Too many requests. Please wait a moment and try again.';
      } else if (status === 500) {
        errorMessage = 'Server error. Please try again later.';
      }
      
      return Promise.reject({ status, message: errorMessage, data });
    }
    return Promise.reject({ status: 0, message: 'Network error. Please check your connection.' });
  }
);

export default apiClient;
