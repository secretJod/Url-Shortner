import { createContext, useState, useEffect, useCallback } from 'react';
import apiClient from '../api/client';

export const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [apiKey, setApiKey] = useState(localStorage.getItem('urlshortener_api_key') || null);
  const [isAuthenticated, setIsAuthenticated] = useState(!!apiKey);

  useEffect(() => {
    if (apiKey) {
      localStorage.setItem('urlshortener_api_key', apiKey);
      setIsAuthenticated(true);
    } else {
      localStorage.removeItem('urlshortener_api_key');
      setIsAuthenticated(false);
    }
  }, [apiKey]);

  const login = useCallback(async (email) => {
    const response = await apiClient.post('/api/keys', { email });
    const newKey = response.data.api_key;
    setApiKey(newKey);
    return newKey;
  }, []);

  const importKey = useCallback((key) => {
    setApiKey(key);
  }, []);

  const logout = useCallback(() => {
    setApiKey(null);
  }, []);

  return (
    <AuthContext.Provider value={{ apiKey, isAuthenticated, login, importKey, logout }}>
      {children}
    </AuthContext.Provider>
  );
}
