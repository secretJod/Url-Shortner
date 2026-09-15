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

  // requestVerification kicks off email verification (SEC-01) — the
  // backend no longer returns a key from this call, it sends a magic
  // link instead. The key is only available once that link is clicked
  // (which redeems it via GET /api/keys/verify), so the caller pastes it
  // back in via importKey.
  const requestVerification = useCallback(async (email) => {
    const response = await apiClient.post('/api/keys', { email });
    return response.data.message;
  }, []);

  const importKey = useCallback((key) => {
    setApiKey(key);
  }, []);

  const logout = useCallback(() => {
    setApiKey(null);
  }, []);

  return (
    <AuthContext.Provider value={{ apiKey, isAuthenticated, requestVerification, importKey, logout }}>
      {children}
    </AuthContext.Provider>
  );
}
