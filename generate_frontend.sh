#!/bin/bash
set -e

echo "🚀 Generating LinkSnip Enterprise Frontend..."

# Create directory structure
mkdir -p frontend/public
mkdir -p frontend/src/api
mkdir -p frontend/src/components
mkdir -p frontend/src/context
mkdir -p frontend/src/hooks
mkdir -p frontend/src/pages
mkdir -p frontend/src/utils

echo "📦 Creating configuration files..."

cat << 'FILE_EOF' > frontend/package.json
{
  "name": "url-shortener-frontend",
  "private": true,
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "axios": "^1.6.7",
    "clsx": "^2.1.0",
    "lucide-react": "^0.344.0",
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.22.1",
    "recharts": "^2.12.2",
    "tailwind-merge": "^2.2.1"
  },
  "devDependencies": {
    "@types/react": "^18.2.56",
    "@types/react-dom": "^18.2.19",
    "@vitejs/plugin-react": "^4.2.1",
    "autoprefixer": "^10.4.17",
    "postcss": "^8.4.35",
    "tailwindcss": "^3.4.1",
    "vite": "^5.1.4"
  }
}
FILE_EOF

cat << 'FILE_EOF' > frontend/vite.config.js
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/metrics': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      }
    }
  }
});
FILE_EOF

cat << 'FILE_EOF' > frontend/tailwind.config.js
/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#f0f9ff',
          100: '#e0f2fe',
          200: '#bae6fd',
          300: '#7dd3fc',
          400: '#38bdf8',
          500: '#0ea5e9',
          600: '#0284c7',
          700: '#0369a1',
          800: '#075985',
          900: '#0c4a6e',
          950: '#082f49',
        }
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-in-out',
        'slide-up': 'slideUp 0.4s ease-out',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { transform: 'translateY(10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        }
      }
    },
  },
  plugins: [],
}
FILE_EOF

cat << 'FILE_EOF' > frontend/postcss.config.js
export default {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
}
FILE_EOF

cat << 'FILE_EOF' > frontend/index.html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>LinkSnip — Enterprise URL Shortener</title>
    <script>
      // Prevent flash of unstyled content / dark mode flicker
      if (localStorage.theme === 'dark' || (!('theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
        document.documentElement.classList.add('dark')
      } else {
        document.documentElement.classList.remove('dark')
      }
    </script>
  </head>
  <body class="bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100 transition-colors duration-200">
    <div id="root"></div>
    <script type="module" src="/src/main.jsx"></script>
  </body>
</html>
FILE_EOF

cat << 'FILE_EOF' > frontend/public/favicon.svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-brand-600"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path></svg>
FILE_EOF

cat << 'FILE_EOF' > frontend/public/_redirects
/*    /index.html   200
FILE_EOF

echo "🎨 Creating global styles and entry points..."

cat << 'FILE_EOF' > frontend/src/index.css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  body {
    @apply antialiased;
  }
  
  /* Custom Scrollbar */
  ::-webkit-scrollbar {
    width: 8px;
    height: 8px;
  }
  ::-webkit-scrollbar-track {
    @apply bg-gray-100 dark:bg-gray-800;
  }
  ::-webkit-scrollbar-thumb {
    @apply bg-gray-300 dark:bg-gray-600 rounded-full;
  }
  ::-webkit-scrollbar-thumb:hover {
    @apply bg-gray-400 dark:bg-gray-500;
  }
}

@layer components {
  .btn-primary {
    @apply px-4 py-2 bg-brand-600 hover:bg-brand-700 text-white font-medium rounded-lg shadow-sm hover:shadow-md transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900 disabled:opacity-50 disabled:cursor-not-allowed;
  }
  
  .btn-secondary {
    @apply px-4 py-2 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-200 font-medium rounded-lg border border-gray-300 dark:border-gray-600 shadow-sm hover:shadow-md transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900;
  }
  
  .input-field {
    @apply block w-full px-4 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg shadow-sm focus:ring-2 focus:ring-brand-500 focus:border-brand-500 dark:focus:ring-brand-400 dark:focus:border-brand-400 text-gray-900 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500 transition-colors;
  }

  .card {
    @apply bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6 transition-all duration-200;
  }
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/main.jsx
import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App.jsx';
import { AuthProvider } from './context/AuthContext.jsx';
import { ToastProvider } from './context/ToastContext.jsx';
import './index.css';

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <BrowserRouter>
      <ToastProvider>
        <AuthProvider>
          <App />
        </AuthProvider>
      </ToastProvider>
    </BrowserRouter>
  </React.StrictMode>
);
FILE_EOF

cat << 'FILE_EOF' > frontend/src/App.jsx
import { Routes, Route } from 'react-router-dom';
import Navbar from './components/Navbar';
import Footer from './components/Footer';
import Toast from './components/Toast';
import LandingPage from './pages/LandingPage';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import StatsPage from './pages/StatsPage';
import TopLinksPage from './pages/TopLinksPage';
import AdminPage from './pages/AdminPage';
import ProtectedRoute from './components/ProtectedRoute';

function App() {
  return (
    <div className="flex flex-col min-h-screen">
      <Navbar />
      <main className="flex-grow container mx-auto px-4 sm:px-6 lg:px-8 py-8 animate-fade-in">
        <Routes>
          <Route path="/" element={<LandingPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/top" element={<TopLinksPage />} />
          
          {/* Protected Routes */}
          <Route path="/dashboard" element={
            <ProtectedRoute>
              <DashboardPage />
            </ProtectedRoute>
          } />
          <Route path="/stats/:shortCode" element={
            <ProtectedRoute>
              <StatsPage />
            </ProtectedRoute>
          } />
          <Route path="/admin" element={
            <ProtectedRoute>
              <AdminPage />
            </ProtectedRoute>
          } />
        </Routes>
      </main>
      <Footer />
      <Toast />
    </div>
  );
}

export default App;
FILE_EOF

echo "🛠️ Creating utilities and API client..."

cat << 'FILE_EOF' > frontend/src/utils/format.js
export function formatDate(dateString) {
  if (!dateString) return 'N/A';
  const date = new Date(dateString);
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date);
}

export function formatNumber(num) {
  if (num === undefined || num === null) return '0';
  return new Intl.NumberFormat('en-US').format(num);
}

export function formatUptime(seconds) {
  if (!seconds) return '0s';
  const d = Math.floor(seconds / (3600 * 24));
  const h = Math.floor((seconds % (3600 * 24)) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  
  let str = '';
  if (d > 0) str += `${d}d `;
  if (h > 0) str += `${h}h `;
  if (m > 0) str += `${m}m `;
  if (s > 0 && d === 0) str += `${s}s`;
  
  return str.trim() || '0s';
}

export function truncateUrl(url, maxLength = 50) {
  if (!url) return '';
  if (url.length <= maxLength) return url;
  return url.substring(0, maxLength) + '...';
}

export function isValidApiKey(key) {
  return /^usk_[a-f0-9]{64}$/.test(key);
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/api/client.js
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
FILE_EOF

echo "🔐 Creating Auth and Toast contexts..."

cat << 'FILE_EOF' > frontend/src/context/AuthContext.jsx
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
FILE_EOF

cat << 'FILE_EOF' > frontend/src/hooks/useAuth.js
import { useContext } from 'react';
import { AuthContext } from '../context/AuthContext';

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/context/ToastContext.jsx
import { createContext, useState, useCallback } from 'react';

export const ToastContext = createContext(null);

export function ToastProvider({ children }) {
  const [toasts, setToasts] = useState([]);

  const showToast = useCallback((message, type = 'info', duration = 4000) => {
    const id = Date.now() + Math.random();
    setToasts((prev) => [...prev, { id, message, type }]);
    
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, duration);
  }, []);

  const removeToast = useCallback((id) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  return (
    <ToastContext.Provider value={{ toasts, showToast, removeToast }}>
      {children}
    </ToastContext.Provider>
  );
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/hooks/useToast.js
import { useContext } from 'react';
import { ToastContext } from '../context/ToastContext';

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return context;
}
FILE_EOF

echo "🧩 Creating reusable UI components..."

cat << 'FILE_EOF' > frontend/src/components/Toast.jsx
import { useToast } from '../hooks/useToast';
import { X, CheckCircle, AlertCircle, Info } from 'lucide-react';
import { clsx } from 'clsx';

export default function Toast() {
  const { toasts, removeToast } = useToast();

  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 w-full max-w-sm">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={clsx(
            "flex items-center p-4 rounded-lg shadow-lg border animate-slide-up",
            toast.type === 'success' && "bg-green-50 dark:bg-green-900/30 border-green-200 dark:border-green-800 text-green-800 dark:text-green-200",
            toast.type === 'error' && "bg-red-50 dark:bg-red-900/30 border-red-200 dark:border-red-800 text-red-800 dark:text-red-200",
            toast.type === 'info' && "bg-blue-50 dark:bg-blue-900/30 border-blue-200 dark:border-blue-800 text-blue-800 dark:text-blue-200"
          )}
        >
          <div className="flex-shrink-0 mr-3">
            {toast.type === 'success' && <CheckCircle className="w-5 h-5" />}
            {toast.type === 'error' && <AlertCircle className="w-5 h-5" />}
            {toast.type === 'info' && <Info className="w-5 h-5" />}
          </div>
          <p className="text-sm font-medium flex-grow">{toast.message}</p>
          <button
            onClick={() => removeToast(toast.id)}
            className="flex-shrink-0 ml-3 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      ))}
    </div>
  );
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/components/Spinner.jsx
import { clsx } from 'clsx';

export default function Spinner({ size = 'md', className = '' }) {
  const sizes = {
    sm: 'w-4 h-4',
    md: 'w-8 h-8',
    lg: 'w-12 h-12'
  };

  return (
    <div className={clsx("flex justify-center items-center p-4", className)}>
      <div
        className={clsx(
          "animate-spin rounded-full border-2 border-gray-200 dark:border-gray-700 border-t-brand-600 dark:border-t-brand-400",
          sizes[size]
        )}
      />
    </div>
  );
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/components/Navbar.jsx
import { Link, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { Link2, LayoutDashboard, Trophy, ShieldCheck, LogIn, LogOut, Moon, Sun, Menu, X } from 'lucide-react';
import { useState } from 'react';
import { clsx } from 'clsx';

export default function Navbar() {
  const { isAuthenticated, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const toggleTheme = () => {
    if (document.documentElement.classList.contains('dark')) {
      document.documentElement.classList.remove('dark');
      localStorage.theme = 'light';
    } else {
      document.documentElement.classList.add('dark');
      localStorage.theme = 'dark';
    }
  };

  const handleLogout = () => {
    logout();
    navigate('/');
  };

  const navLinks = [
    { name: 'Home', path: '/', icon: Link2 },
    { name: 'Top Links', path: '/top', icon: Trophy },
  ];

  if (isAuthenticated) {
    navLinks.push({ name: 'Dashboard', path: '/dashboard', icon: LayoutDashboard });
    navLinks.push({ name: 'Admin', path: '/admin', icon: ShieldCheck });
  }

  const isActive = (path) => location.pathname === path;

  return (
    <nav className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 sticky top-0 z-40">
      <div className="container mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          <div className="flex items-center">
            <Link to="/" className="flex items-center space-x-2">
              <Link2 className="w-8 h-8 text-brand-600" />
              <span className="text-xl font-bold text-gray-900 dark:text-white">LinkSnip</span>
            </Link>
          </div>

          {/* Desktop Nav */}
          <div className="hidden md:flex items-center space-x-4">
            {navLinks.map((link) => (
              <Link
                key={link.path}
                to={link.path}
                className={clsx(
                  "flex items-center space-x-1 px-3 py-2 rounded-md text-sm font-medium transition-colors",
                  isActive(link.path)
                    ? "bg-brand-50 dark:bg-brand-900/50 text-brand-700 dark:text-brand-300"
                    : "text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
                )}
              >
                <link.icon className="w-4 h-4" />
                <span>{link.name}</span>
              </Link>
            ))}

            <button
              onClick={toggleTheme}
              className="p-2 rounded-md text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              aria-label="Toggle theme"
            >
              <Sun className="w-5 h-5 hidden dark:block" />
              <Moon className="w-5 h-5 block dark:hidden" />
            </button>

            {isAuthenticated ? (
              <button onClick={handleLogout} className="btn-secondary flex items-center space-x-1">
                <LogOut className="w-4 h-4" />
                <span>Logout</span>
              </button>
            ) : (
              <Link to="/login" className="btn-primary flex items-center space-x-1">
                <LogIn className="w-4 h-4" />
                <span>Login</span>
              </Link>
            )}
          </div>

          {/* Mobile menu button */}
          <div className="flex items-center md:hidden">
             <button
              onClick={toggleTheme}
              className="p-2 mr-2 rounded-md text-gray-600 dark:text-gray-300"
            >
              <Sun className="w-5 h-5 hidden dark:block" />
              <Moon className="w-5 h-5 block dark:hidden" />
            </button>
            <button
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              className="p-2 rounded-md text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
            >
              {mobileMenuOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
            </button>
          </div>
        </div>
      </div>

      {/* Mobile Nav */}
      {mobileMenuOpen && (
        <div className="md:hidden bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 animate-fade-in">
          <div className="px-2 pt-2 pb-3 space-y-1">
            {navLinks.map((link) => (
              <Link
                key={link.path}
                to={link.path}
                onClick={() => setMobileMenuOpen(false)}
                className={clsx(
                  "flex items-center space-x-2 px-3 py-2 rounded-md text-base font-medium",
                  isActive(link.path)
                    ? "bg-brand-50 dark:bg-brand-900/50 text-brand-700 dark:text-brand-300"
                    : "text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
                )}
              >
                <link.icon className="w-5 h-5" />
                <span>{link.name}</span>
              </Link>
            ))}
            {isAuthenticated ? (
              <button onClick={() => { handleLogout(); setMobileMenuOpen(false); }} className="w-full text-left flex items-center space-x-2 px-3 py-2 rounded-md text-base font-medium text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30">
                <LogOut className="w-5 h-5" />
                <span>Logout</span>
              </button>
            ) : (
              <Link to="/login" onClick={() => setMobileMenuOpen(false)} className="flex items-center space-x-2 px-3 py-2 rounded-md text-base font-medium text-brand-600 dark:text-brand-400 hover:bg-brand-50 dark:hover:bg-brand-900/30">
                <LogIn className="w-5 h-5" />
                <span>Login</span>
              </Link>
            )}
          </div>
        </div>
      )}
    </nav>
  );
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/components/Footer.jsx
import { Github, Twitter, Heart } from 'lucide-react';

export default function Footer() {
  return (
    <footer className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 py-8 mt-auto">
      <div className="container mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col md:flex-row justify-between items-center space-y-4 md:space-y-0">
          <div className="text-gray-600 dark:text-gray-400 text-sm flex items-center">
            Built with <Heart className="w-4 h-4 mx-1 text-red-500 fill-current" /> by LinkSnip Enterprise
          </div>
          <div className="flex space-x-6">
            <a href="#" className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors">
              <Github className="w-5 h-5" />
            </a>
            <a href="#" className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors">
              <Twitter className="w-5 h-5" />
            </a>
          </div>
          <div className="text-gray-500 dark:text-gray-400 text-sm">
            &copy; {new Date().getFullYear()} LinkSnip. All rights reserved.
          </div>
        </div>
      </div>
    </footer>
  );
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/components/ProtectedRoute.jsx
import { Navigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';

export default function ProtectedRoute({ children }) {
  const { isAuthenticated } = useAuth();

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return children;
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/components/StatCard.jsx
import { clsx } from 'clsx';

export default function StatCard({ title, value, icon: Icon, trend, color = 'brand' }) {
  const colors = {
    brand: 'text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30',
    green: 'text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/30',
    blue: 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/30',
    purple: 'text-purple-600 dark:text-purple-400 bg-purple-50 dark:bg-purple-900/30',
    orange: 'text-orange-600 dark:text-orange-400 bg-orange-50 dark:bg-orange-900/30',
  };

  return (
    <div className="card hover:shadow-md transition-shadow">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-gray-500 dark:text-gray-400">{title}</p>
          <p className="mt-1 text-3xl font-bold text-gray-900 dark:text-white">{value}</p>
          {trend && (
            <p className="mt-1 text-xs text-green-600 dark:text-green-400 font-medium">
              {trend}
            </p>
          )}
        </div>
        <div className={clsx("p-3 rounded-lg", colors[color])}>
          <Icon className="w-6 h-6" />
        </div>
      </div>
    </div>
  );
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/components/ShortenForm.jsx
import { useState } from 'react';
import { Link2, Scissors, Copy, Check, ChevronDown, ChevronUp } from 'lucide-react';
import apiClient from '../api/client';
import { useToast } from '../hooks/useToast';
import { clsx } from 'clsx';

export default function ShortenForm({ onSuccess }) {
  const [url, setUrl] = useState('');
  const [customAlias, setCustomAlias] = useState('');
  const [expiresAt, setExpiresAt] = useState('');
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState(null);
  const [copied, setCopied] = useState(false);
  const { showToast } = useToast();

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!url) return;

    setLoading(true);
    setResult(null);
    setCopied(false);

    try {
      const body = { url };
      if (customAlias) body.custom_alias = customAlias;
      if (expiresAt) body.expires_at = new Date(expiresAt).toISOString();

      const response = await apiClient.post('/api/shorten', body);
      setResult(response.data);
      if (onSuccess) onSuccess();
      showToast('Link shortened successfully!', 'success');
      setUrl('');
      setCustomAlias('');
      setExpiresAt('');
    } catch (error) {
      showToast(error.message || 'Failed to shorten URL', 'error');
    } finally {
      setLoading(false);
    }
  };

  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    showToast('Copied to clipboard!', 'success');
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="card w-full">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="flex flex-col sm:flex-row gap-3">
          <div className="relative flex-grow">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <Link2 className="h-5 w-5 text-gray-400" />
            </div>
            <input
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="Paste your long URL here..."
              className="input-field pl-10"
              required
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="btn-primary flex items-center justify-center space-x-2 min-w-[120px]"
          >
            {loading ? (
              <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin" />
            ) : (
              <>
                <Scissors className="w-4 h-4" />
                <span>Shorten</span>
              </>
            )}
          </button>
        </div>

        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          className="flex items-center text-sm text-gray-500 dark:text-gray-400 hover:text-brand-600 dark:hover:text-brand-400 transition-colors"
        >
          {showAdvanced ? <ChevronUp className="w-4 h-4 mr-1" /> : <ChevronDown className="w-4 h-4 mr-1" />}
          Advanced Options
        </button>

        {showAdvanced && (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2 animate-fade-in">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Custom Alias (optional)
              </label>
              <input
                type="text"
                value={customAlias}
                onChange={(e) => setCustomAlias(e.target.value)}
                placeholder="my-custom-link"
                className="input-field"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Expiration Date (optional)
              </label>
              <input
                type="datetime-local"
                value={expiresAt}
                onChange={(e) => setExpiresAt(e.target.value)}
                className="input-field"
              />
            </div>
          </div>
        )}
      </form>

      {result && (
        <div className="mt-6 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg animate-slide-up">
          <p className="text-sm text-green-800 dark:text-green-200 mb-2">Your shortened URL:</p>
          <div className="flex items-center gap-2">
            <a
              href={result.short_url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-lg font-semibold text-brand-600 dark:text-brand-400 hover:underline truncate flex-grow"
            >
              {result.short_url}
            </a>
            <button
              onClick={() => copyToClipboard(result.short_url)}
              className="btn-secondary flex items-center space-x-1 flex-shrink-0"
            >
              {copied ? <Check className="w-4 h-4 text-green-600" /> : <Copy className="w-4 h-4" />}
              <span>{copied ? 'Copied' : 'Copy'}</span>
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/components/LinkTable.jsx
import { Link } from 'react-router-dom';
import { Copy, BarChart2, ExternalLink, Trash2, Check } from 'lucide-react';
import { formatDate, truncateUrl } from '../utils/format';
import { useToast } from '../hooks/useToast';
import { useState } from 'react';

export default function LinkTable({ links, onDelete }) {
  const { showToast } = useToast();
  const [copiedId, setCopiedId] = useState(null);

  const copyToClipboard = (text, id) => {
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    showToast('Copied to clipboard!', 'success');
    setTimeout(() => setCopiedId(null), 2000);
  };

  if (!links || links.length === 0) {
    return (
      <div className="card text-center py-12">
        <p className="text-gray-500 dark:text-gray-400">No links found. Shorten your first URL!</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto card p-0">
      <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        <thead className="bg-gray-50 dark:bg-gray-800/50">
          <tr>
            <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Short URL</th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Original URL</th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Created</th>
            <th scope="col" className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Actions</th>
          </tr>
        </thead>
        <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
          {links.map((link) => (
            <tr key={link.short_code} className="hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
              <td className="px-6 py-4 whitespace-nowrap text-sm">
                <div className="flex items-center space-x-2">
                  <a
                    href={`${window.location.origin}/${link.short_code}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-brand-600 dark:text-brand-400 hover:underline font-medium"
                  >
                    /{link.short_code}
                  </a>
                  <ExternalLink className="w-3 h-3 text-gray-400" />
                </div>
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 max-w-xs truncate" title={link.long_url}>
                {truncateUrl(link.long_url, 40)}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {formatDate(link.created_at)}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <div className="flex justify-end space-x-2">
                  <button
                    onClick={() => copyToClipboard(`${window.location.origin}/${link.short_code}`, link.short_code)}
                    className="p-1.5 text-gray-500 hover:text-brand-600 dark:text-gray-400 dark:hover:text-brand-400 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                    title="Copy"
                  >
                    {copiedId === link.short_code ? <Check className="w-4 h-4 text-green-500" /> : <Copy className="w-4 h-4" />}
                  </button>
                  <Link
                    to={`/stats/${link.short_code}`}
                    className="p-1.5 text-gray-500 hover:text-blue-600 dark:text-gray-400 dark:hover:text-blue-400 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                    title="View Stats"
                  >
                    <BarChart2 className="w-4 h-4" />
                  </Link>
                  {onDelete && (
                    <button
                      onClick={() => onDelete(link.short_code)}
                      className="p-1.5 text-gray-500 hover:text-red-600 dark:text-gray-400 dark:hover:text-red-400 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                      title="Delete"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  )}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/components/DailyChart.jsx
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

export default function DailyChart({ data }) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-gray-500 dark:text-gray-400">
        No click data available yet.
      </div>
    );
  }

  const chartData = data.map(d => ({
    ...d,
    date: new Date(d.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
  }));

  return (
    <div className="h-64 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={chartData} margin={{ top: 10, right: 10, left: -10, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" className="dark:stroke-gray-700" />
          <XAxis 
            dataKey="date" 
            tick={{ fill: '#6b7280', fontSize: 12 }} 
            axisLine={{ stroke: '#e5e7eb' }}
            tickLine={false}
          />
          <YAxis 
            tick={{ fill: '#6b7280', fontSize: 12 }} 
            axisLine={{ stroke: '#e5e7eb' }}
            tickLine={false}
          />
          <Tooltip 
            contentStyle={{ 
              backgroundColor: '#1f2937', 
              border: 'none', 
              borderRadius: '8px',
              color: '#fff'
            }}
            cursor={{ fill: 'rgba(14, 165, 233, 0.1)' }}
          />
          <Bar dataKey="count" fill="#0ea5e9" radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
FILE_EOF

echo "✅ Configuration and core components created. Continuing with pages..."
echo "📄 Creating application pages..."

cat << 'FILE_EOF' > frontend/src/pages/LandingPage.jsx
import { useState, useEffect } from 'react';
import { Link2, MousePointerClick, Zap } from 'lucide-react';
import ShortenForm from '../components/ShortenForm';
import StatCard from '../components/StatCard';
import apiClient from '../api/client';
import Spinner from '../components/Spinner';

export default function LandingPage() {
  const [stats, setStats] = useState({ totalLinks: 0, totalClicks: 0 });
  const [loadingStats, setLoadingStats] = useState(true);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const res = await apiClient.get('/api/stats/top?limit=100');
        const links = res.data.top_links || [];
        const totalClicks = links.reduce((sum, link) => sum + link.total_clicks, 0);
        setStats({ totalLinks: links.length, totalClicks });
      } catch (error) {
        console.error('Failed to fetch stats', error);
      } finally {
        setLoadingStats(false);
      }
    };
    fetchStats();
  }, []);

  return (
    <div className="py-12 space-y-16">
      {/* Hero Section */}
      <div className="text-center space-y-6 max-w-4xl mx-auto">
        <div className="inline-flex items-center px-3 py-1 rounded-full bg-brand-100 dark:bg-brand-900/50 text-brand-800 dark:text-brand-200 text-sm font-medium mb-4">
          <Zap className="w-4 h-4 mr-1" />
          Enterprise-Grade URL Shortening
        </div>
        <h1 className="text-4xl md:text-6xl font-extrabold text-gray-900 dark:text-white tracking-tight">
          Shorten, Share, and <span className="text-transparent bg-clip-text bg-gradient-to-r from-brand-500 to-purple-600">Analyze</span> Your Links
        </h1>
        <p className="text-xl text-gray-600 dark:text-gray-300 max-w-2xl mx-auto">
          Create short, memorable links in seconds. Track every click with detailed analytics 
          and powerful enterprise features.
        </p>
      </div>

      {/* Shorten Form */}
      <div className="max-w-3xl mx-auto">
        <ShortenForm />
      </div>

      {/* Stats Preview */}
      <div className="max-w-5xl mx-auto">
        <h2 className="text-2xl font-bold text-center text-gray-900 dark:text-white mb-8">
          Trusted by Thousands
        </h2>
        {loadingStats ? (
          <Spinner />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 max-w-2xl mx-auto">
            <StatCard 
              title="Links Shortened" 
              value={stats.totalLinks.toLocaleString()} 
              icon={Link2} 
              color="brand" 
            />
            <StatCard 
              title="Total Clicks Tracked" 
              value={stats.totalClicks.toLocaleString()} 
              icon={MousePointerClick} 
              color="purple" 
            />
          </div>
        )}
      </div>
    </div>
  );
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/pages/LoginPage.jsx
import { useState } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import { KeyRound, Mail, Copy, Check, AlertTriangle, ArrowRight } from 'lucide-react';
import { isValidApiKey } from '../utils/format';
import { useNavigate } from 'react-router-dom';

export default function LoginPage() {
  const { login, importKey } = useAuth();
  const { showToast } = useToast();
  const navigate = useNavigate();
  
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [generatedKey, setGeneratedKey] = useState(null);
  const [copied, setCopied] = useState(false);
  
  const [existingKey, setExistingKey] = useState('');

  const handleGenerateKey = async (e) => {
    e.preventDefault();
    if (!email) return;

    setLoading(true);
    try {
      const key = await login(email);
      setGeneratedKey(key);
      showToast('API Key generated! Save it now.', 'success');
    } catch (error) {
      showToast(error.message || 'Failed to generate API key', 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleImportKey = (e) => {
    e.preventDefault();
    if (!isValidApiKey(existingKey)) {
      showToast('Invalid API key format. Must start with "usk_" and be 68 characters long.', 'error');
      return;
    }
    importKey(existingKey);
    showToast('API Key imported successfully!', 'success');
    navigate('/dashboard');
  };

  const copyToClipboard = () => {
    navigator.clipboard.writeText(generatedKey);
    setCopied(true);
    showToast('Copied to clipboard!', 'success');
    setTimeout(() => setCopied(false), 2000);
  };

  if (generatedKey) {
    return (
      <div className="max-w-lg mx-auto py-12 animate-fade-in">
        <div className="card text-center space-y-6">
          <div className="mx-auto w-16 h-16 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center">
            <KeyRound className="w-8 h-8 text-green-600 dark:text-green-400" />
          </div>
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">Your API Key is Ready</h2>
          
          <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4 flex items-start space-x-3 text-left">
            <AlertTriangle className="w-5 h-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-semibold text-yellow-800 dark:text-yellow-200">Warning: Save this key now!</p>
              <p className="text-xs text-yellow-700 dark:text-yellow-300 mt-1">
                For security reasons, this key will only be shown once. If you lose it, you will need to generate a new one.
              </p>
            </div>
          </div>

          <div className="bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg p-4 break-all font-mono text-sm text-gray-800 dark:text-gray-200">
            {generatedKey}
          </div>

          <button 
            onClick={copyToClipboard}
            className="btn-primary w-full flex items-center justify-center space-x-2"
          >
            {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
            <span>{copied ? 'Copied!' : 'Copy to Clipboard'}</span>
          </button>

          <button 
            onClick={() => navigate('/dashboard')}
            className="text-brand-600 dark:text-brand-400 hover:underline text-sm font-medium flex items-center justify-center space-x-1 mx-auto"
          >
            <span>I've saved it, go to Dashboard</span>
            <ArrowRight className="w-4 h-4" />
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-md mx-auto py-12 space-y-8 animate-fade-in">
      <div className="text-center space-y-2">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Welcome to LinkSnip</h1>
        <p className="text-gray-600 dark:text-gray-400">Get your API key to unlock enterprise features.</p>
      </div>

      <div className="card space-y-6">
        <form onSubmit={handleGenerateKey} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Email Address
            </label>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <Mail className="h-5 w-5 text-gray-400" />
              </div>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="you@company.com"
                className="input-field pl-10"
                required
              />
            </div>
          </div>
          <button
            type="submit"
            disabled={loading}
            className="btn-primary w-full flex items-center justify-center space-x-2"
          >
            {loading ? (
              <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin" />
            ) : (
              <>
                <KeyRound className="w-4 h-4" />
                <span>Generate API Key</span>
              </>
            )}
          </button>
        </form>

        <div className="relative">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-gray-200 dark:border-gray-700"></div>
          </div>
          <div className="relative flex justify-center text-sm">
            <span className="px-2 bg-white dark:bg-gray-800 text-gray-500 dark:text-gray-400">Or</span>
          </div>
        </div>

        <form onSubmit={handleImportKey} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Already have a key?
            </label>
            <input
              type="text"
              value={existingKey}
              onChange={(e) => setExistingKey(e.target.value)}
              placeholder="usk_..."
              className="input-field font-mono text-sm"
            />
          </div>
          <button
            type="submit"
            className="btn-secondary w-full flex items-center justify-center space-x-2"
          >
            <span>Import Existing Key</span>
            <ArrowRight className="w-4 h-4" />
          </button>
        </form>
      </div>
    </div>
  );
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/pages/DashboardPage.jsx
import { useState, useEffect } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import apiClient from '../api/client';
import LinkTable from '../components/LinkTable';
import ShortenForm from '../components/ShortenForm';
import Spinner from '../components/Spinner';
import { Search, Plus, X } from 'lucide-react';

export default function DashboardPage() {
  const { apiKey } = useAuth();
  const { showToast } = useToast();
  
  const [links, setLinks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [showForm, setShowForm] = useState(false);

  const fetchLinks = async () => {
    setLoading(true);
    try {
      const res = await apiClient.get('/api/links');
      setLinks(res.data.links || []);
    } catch (error) {
      showToast(error.message || 'Failed to fetch links', 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (apiKey) {
      fetchLinks();
    }
  }, [apiKey]);

  const handleDelete = async (shortCode) => {
    if (!window.confirm(`Are you sure you want to delete /${shortCode}?`)) return;
    
    try {
      setLinks(links.filter(l => l.short_code !== shortCode));
      showToast('Link deleted successfully', 'success');
    } catch (error) {
      showToast('Failed to delete link', 'error');
    }
  };

  const filteredLinks = links.filter(link => 
    link.short_code.toLowerCase().includes(searchTerm.toLowerCase()) ||
    link.long_url.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="py-8 space-y-8 animate-fade-in">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Dashboard</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">Manage and track your shortened links.</p>
        </div>
        <button 
          onClick={() => setShowForm(!showForm)}
          className="btn-primary flex items-center space-x-2"
        >
          {showForm ? <X className="w-4 h-4" /> : <Plus className="w-4 h-4" />}
          <span>{showForm ? 'Cancel' : 'Shorten New URL'}</span>
        </button>
      </div>

      {showForm && (
        <div className="animate-slide-up">
          <ShortenForm onSuccess={() => {
            fetchLinks();
            setShowForm(false);
          }} />
        </div>
      )}

      <div className="card p-4">
        <div className="relative mb-4">
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <Search className="h-5 w-5 text-gray-400" />
          </div>
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder="Search by short code or URL..."
            className="input-field pl-10"
          />
        </div>

        {loading ? (
          <Spinner size="lg" />
        ) : (
          <LinkTable links={filteredLinks} onDelete={handleDelete} />
        )}
      </div>
    </div>
  );
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/pages/StatsPage.jsx
import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import apiClient from '../api/client';
import StatCard from '../components/StatCard';
import DailyChart from '../components/DailyChart';
import Spinner from '../components/Spinner';
import { ArrowLeft, MousePointerClick, Users, Globe, Smartphone, Monitor, Tablet, Clock } from 'lucide-react';
import { formatDate } from '../utils/format';

export default function StatsPage() {
  const { shortCode } = useParams();
  const { apiKey } = useAuth();
  const { showToast } = useToast();

  const [stats, setStats] = useState(null);
  const [clicks, setClicks] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      if (!apiKey) return;
      try {
        const [statsRes, clicksRes] = await Promise.all([
          apiClient.get(`/api/stats/${shortCode}`),
          apiClient.get(`/api/links/${shortCode}/clicks?limit=20`)
        ]);
        setStats(statsRes.data);
        setClicks(clicksRes.data.clicks || []);
      } catch (error) {
        showToast(error.message || 'Failed to fetch stats', 'error');
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [shortCode, apiKey]);

  const getDeviceIcon = (type) => {
    switch (type?.toLowerCase()) {
      case 'mobile': return <Smartphone className="w-4 h-4" />;
      case 'tablet': return <Tablet className="w-4 h-4" />;
      default: return <Monitor className="w-4 h-4" />;
    }
  };

  if (loading) {
    return <Spinner size="lg" className="py-20" />;
  }

  if (!stats) {
    return (
      <div className="card text-center py-20">
        <p className="text-gray-500 dark:text-gray-400">Link not found or you don't have permission to view it.</p>
        <Link to="/dashboard" className="btn-primary mt-4 inline-block">Back to Dashboard</Link>
      </div>
    );
  }

  return (
    <div className="py-8 space-y-8 animate-fade-in">
      <div className="flex items-center space-x-4">
        <Link to="/dashboard" className="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors">
          <ArrowLeft className="w-5 h-5 text-gray-600 dark:text-gray-300" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Analytics for /{stats.short_code}</h1>
          <a 
            href={stats.long_url} 
            target="_blank" 
            rel="noopener noreferrer"
            className="text-sm text-brand-600 dark:text-brand-400 hover:underline truncate block max-w-md"
          >
            {stats.long_url}
          </a>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <StatCard title="Total Clicks" value={stats.total_clicks} icon={MousePointerClick} color="brand" />
        <StatCard title="Unique Visitors" value={stats.unique_ips} icon={Users} color="green" />
      </div>

      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Daily Clicks</h2>
        <DailyChart data={stats.daily_clicks} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="card">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
            <Globe className="w-5 h-5 mr-2 text-brand-600" />
            Top Referrers
          </h2>
          {stats.top_referrers && stats.top_referrers.length > 0 ? (
            <ul className="space-y-3">
              {stats.top_referrers.map((ref, idx) => (
                <li key={idx} className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-200 truncate">{ref}</span>
                  <span className="text-xs font-semibold text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30 px-2 py-1 rounded-full">
                    #{idx + 1}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-gray-500 dark:text-gray-400 text-sm">No referrer data available.</p>
          )}
        </div>

        <div className="card">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
            <Clock className="w-5 h-5 mr-2 text-brand-600" />
            Recent Clicks
          </h2>
          <div className="overflow-x-auto max-h-64 overflow-y-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-800/50 sticky top-0">
                <tr>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Time</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Country</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Device</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {clicks.length > 0 ? clicks.map((click, idx) => (
                  <tr key={idx} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                    <td className="px-4 py-2 text-xs text-gray-600 dark:text-gray-300 whitespace-nowrap">
                      {formatDate(click.timestamp)}
                    </td>
                    <td className="px-4 py-2 text-xs text-gray-600 dark:text-gray-300">
                      {click.country || 'Unknown'}
                    </td>
                    <td className="px-4 py-2 text-xs text-gray-600 dark:text-gray-300 flex items-center space-x-1">
                      {getDeviceIcon(click.device_type)}
                      <span className="capitalize">{click.device_type || 'Unknown'}</span>
                    </td>
                  </tr>
                )) : (
                  <tr>
                    <td colSpan="3" className="px-4 py-8 text-center text-sm text-gray-500">No recent clicks</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/pages/TopLinksPage.jsx
import { useState, useEffect } from 'react';
import { useToast } from '../hooks/useToast';
import apiClient from '../api/client';
import Spinner from '../components/Spinner';
import { Trophy, ExternalLink, ArrowUpRight } from 'lucide-react';
import { Link } from 'react-router-dom';
import { truncateUrl } from '../utils/format';

export default function TopLinksPage() {
  const { showToast } = useToast();
  const [links, setLinks] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchTopLinks = async () => {
      try {
        const res = await apiClient.get('/api/stats/top?limit=10');
        setLinks(res.data.top_links || []);
      } catch (error) {
        showToast(error.message || 'Failed to fetch top links', 'error');
      } finally {
        setLoading(false);
      }
    };
    fetchTopLinks();
  }, []);

  const getMedalColor = (index) => {
    if (index === 0) return 'text-yellow-500 bg-yellow-50 dark:bg-yellow-900/30';
    if (index === 1) return 'text-gray-400 bg-gray-50 dark:bg-gray-800';
    if (index === 2) return 'text-orange-600 bg-orange-50 dark:bg-orange-900/30';
    return 'text-gray-500 bg-gray-50 dark:bg-gray-800';
  };

  if (loading) {
    return <Spinner size="lg" className="py-20" />;
  }

  return (
    <div className="py-8 space-y-8 animate-fade-in">
      <div className="text-center space-y-2">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white flex items-center justify-center">
          <Trophy className="w-8 h-8 mr-3 text-yellow-500" />
          Top Performing Links
        </h1>
        <p className="text-gray-600 dark:text-gray-400">The most clicked links across the platform.</p>
      </div>

      <div className="max-w-4xl mx-auto space-y-4">
        {links.length === 0 ? (
          <div className="card text-center py-12">
            <p className="text-gray-500 dark:text-gray-400">No links have been clicked yet. Be the first!</p>
          </div>
        ) : (
          links.map((link, index) => (
            <div 
              key={link.short_code} 
              className="card flex items-center justify-between hover:shadow-md transition-all duration-200 group"
            >
              <div className="flex items-center space-x-4 flex-grow min-w-0">
                <div className={`flex-shrink-0 w-10 h-10 rounded-full flex items-center justify-center font-bold text-lg ${getMedalColor(index)}`}>
                  {index + 1}
                </div>
                <div className="min-w-0 flex-grow">
                  <a 
                    href={`${window.location.origin}/${link.short_code}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-lg font-semibold text-brand-600 dark:text-brand-400 hover:underline flex items-center space-x-1"
                  >
                    <span>/{link.short_code}</span>
                    <ExternalLink className="w-4 h-4 opacity-0 group-hover:opacity-100 transition-opacity" />
                  </a>
                  <p className="text-sm text-gray-500 dark:text-gray-400 truncate" title={link.long_url}>
                    {truncateUrl(link.long_url, 60)}
                  </p>
                </div>
              </div>
              <div className="flex items-center space-x-4 flex-shrink-0 ml-4">
                <div className="text-right">
                  <p className="text-2xl font-bold text-gray-900 dark:text-white">{link.total_clicks.toLocaleString()}</p>
                  <p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wider">Clicks</p>
                </div>
                <Link 
                  to={`/stats/${link.short_code}`}
                  className="p-2 rounded-lg bg-gray-100 dark:bg-gray-700 hover:bg-brand-50 dark:hover:bg-brand-900/30 text-gray-600 dark:text-gray-300 hover:text-brand-600 dark:hover:text-brand-400 transition-colors"
                  title="View Stats"
                >
                  <ArrowUpRight className="w-5 h-5" />
                </Link>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
FILE_EOF

cat << 'FILE_EOF' > frontend/src/pages/AdminPage.jsx
import { useState, useEffect } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import apiClient from '../api/client';
import StatCard from '../components/StatCard';
import Spinner from '../components/Spinner';
import { Clock, Activity, Server, Zap, AlertTriangle, CheckCircle, XCircle } from 'lucide-react';
import { formatUptime, formatNumber } from '../utils/format';
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts';

const COLORS = ['#10b981', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6', '#6b7280'];

export default function AdminPage() {
  const { apiKey } = useAuth();
  const { showToast } = useToast();
  const [metrics, setMetrics] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchMetrics = async () => {
      try {
        const res = await apiClient.get('/metrics');
        setMetrics(res.data);
      } catch (error) {
        showToast(error.message || 'Failed to fetch metrics', 'error');
      } finally {
        setLoading(false);
      }
    };
    fetchMetrics();
  }, [apiKey]);

  if (loading) {
    return <Spinner size="lg" className="py-20" />;
  }

  if (!metrics) {
    return (
      <div className="card text-center py-20">
        <p className="text-gray-500 dark:text-gray-400">Failed to load system metrics.</p>
      </div>
    );
  }

  const statusCodesData = Object.entries(metrics.status_codes || {}).map(([name, value]) => ({
    name: `HTTP ${name}`,
    value: value
  }));

  const cacheHitRate = metrics.cache_hit_rate || 0;

  return (
    <div className="py-8 space-y-8 animate-fade-in">
      <div className="flex items-center space-x-3">
        <Server className="w-8 h-8 text-brand-600" />
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">System Metrics</h1>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard title="Uptime" value={formatUptime(metrics.uptime_seconds)} icon={Clock} color="green" />
        <StatCard title="Total Requests" value={formatNumber(metrics.total_requests)} icon={Activity} color="blue" />
        <StatCard title="Avg Latency" value={`${metrics.avg_latency_ms?.toFixed(2)} ms`} icon={Zap} color="purple" />
        <StatCard title="Max Latency" value={`${metrics.max_latency_ms?.toFixed(2)} ms`} icon={AlertTriangle} color="orange" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="card">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Status Code Breakdown</h2>
          <div className="h-64 flex items-center justify-center">
            {statusCodesData.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={statusCodesData}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={80}
                    paddingAngle={5}
                    dataKey="value"
                    label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                    labelLine={false}
                  >
                    {statusCodesData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip 
                    contentStyle={{ 
                      backgroundColor: '#1f2937', 
                      border: 'none', 
                      borderRadius: '8px',
                      color: '#fff'
                    }}
                  />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <p className="text-gray-500 dark:text-gray-400">No request data available.</p>
            )}
          </div>
        </div>

        <div className="card space-y-6">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">Cache Performance</h2>
            <div className="flex justify-between mb-2 text-sm">
              <span className="text-gray-600 dark:text-gray-400">Hit Rate</span>
              <span className="font-semibold text-gray-900 dark:text-white">{cacheHitRate.toFixed(1)}%</span>
            </div>
            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-4 overflow-hidden">
              <div 
                className="h-full bg-gradient-to-r from-green-400 to-blue-500 rounded-full transition-all duration-500"
                style={{ width: `${cacheHitRate}%` }}
              />
            </div>
            <div className="flex justify-between mt-4 text-sm text-gray-600 dark:text-gray-400">
              <div className="flex items-center space-x-1">
                <CheckCircle className="w-4 h-4 text-green-500" />
                <span>{formatNumber(metrics.cache_hits)} Hits</span>
              </div>
              <div className="flex items-center space-x-1">
                <XCircle className="w-4 h-4 text-red-500" />
                <span>{formatNumber(metrics.cache_misses)} Misses</span>
              </div>
            </div>
          </div>

          <div className="pt-4 border-t border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Event Processing</h2>
            <div className="grid grid-cols-2 gap-4">
              <div className="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg text-center">
                <p className="text-2xl font-bold text-green-700 dark:text-green-400">{formatNumber(metrics.events_processed)}</p>
                <p className="text-sm text-green-600 dark:text-green-300">Processed</p>
              </div>
              <div className="p-4 bg-red-50 dark:bg-red-900/20 rounded-lg text-center">
                <p className="text-2xl font-bold text-red-700 dark:text-red-400">{formatNumber(metrics.events_failed)}</p>
                <p className="text-sm text-red-600 dark:text-red-300">Failed</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
FILE_EOF

echo "📚 Creating documentation..."

cat << 'FILE_EOF' > frontend/FRONTEND_README.md
# LinkSnip — Enterprise URL Shortener Frontend

A modern, enterprise-grade frontend application for the LinkSnip URL Shortener. Built with React, Vite, Tailwind CSS, and Recharts, it connects seamlessly to the Go/Fiber backend API.

## 🚀 Getting Started

### Prerequisites
- Node.js >= 18.x
- npm or yarn

### Installation & Running

1. Navigate to the frontend directory:

2. Install dependencies:
   ```bash
   npm install
