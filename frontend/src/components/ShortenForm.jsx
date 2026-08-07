import { useState, useRef } from 'react';
import { Link2, Scissors, Copy, Check, ChevronDown, ChevronUp } from 'lucide-react';
import apiClient from '../api/client';
import { useToast } from '../hooks/useToast';
import { motion, AnimatePresence } from 'framer-motion';

export default function ShortenForm({ onSuccess }) {
  const [url, setUrl] = useState('');
  const [customAlias, setCustomAlias] = useState('');
  const [expiresAt, setExpiresAt] = useState('');
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState(null);
  const [copied, setCopied] = useState(false);
  const [focused, setFocused] = useState(false);
  const { showToast } = useToast();
  const inputRef = useRef(null);

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
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
      className="card w-full relative overflow-hidden"
    >
      {/* Decorative gradient orb in background */}
      <div className="absolute -top-20 -right-20 w-60 h-60 bg-brand-400/10 dark:bg-brand-500/10 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute -bottom-20 -left-20 w-60 h-60 bg-purple-400/10 dark:bg-purple-500/10 rounded-full blur-3xl pointer-events-none" />

      <form onSubmit={handleSubmit} className="space-y-4 relative">
        <div className="flex flex-col sm:flex-row gap-3">
          {/* ─── INPUT — Google Material outlined style ─── */}
          <div className="relative flex-grow">
            <div className="relative">
              <div
                className={`absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none transition-colors duration-200 ${focused ? 'text-brand-600 dark:text-brand-400' : 'text-gray-400'}`}
              >
                <Link2 className="h-5 w-5" />
              </div>
              <input
                ref={inputRef}
                type="url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                onFocus={() => setFocused(true)}
                onBlur={() => setFocused(false)}
                placeholder="Paste your long URL here..."
                className="input-field pl-11"
                required
              />
            </div>
          </div>

          <motion.button
            type="submit"
            disabled={loading}
            whileHover={{ scale: loading ? 1 : 1.03 }}
            whileTap={{ scale: loading ? 1 : 0.95 }}
            transition={{ type: 'spring', stiffness: 400, damping: 17 }}
            className="btn-primary flex items-center justify-center space-x-2 min-w-[140px]"
          >
            {loading ? (
              <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin" />
            ) : (
              <>
                <Scissors className="w-4 h-4" />
                <span>Shorten</span>
              </>
            )}
          </motion.button>
        </div>

        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          className="flex items-center text-sm text-gray-500 dark:text-gray-400 hover:text-brand-600 dark:hover:text-brand-400 transition-colors group"
        >
          <motion.div animate={{ rotate: showAdvanced ? 180 : 0 }} transition={{ duration: 0.3 }}>
            {showAdvanced ? <ChevronUp className="w-4 h-4 mr-1" /> : <ChevronDown className="w-4 h-4 mr-1" />}
          </motion.div>
          Advanced Options
        </button>

        <AnimatePresence>
          {showAdvanced && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: 'auto' }}
              exit={{ opacity: 0, height: 0 }}
              transition={{ duration: 0.3 }}
              className="overflow-hidden"
            >
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2">
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
            </motion.div>
          )}
        </AnimatePresence>
      </form>

      <AnimatePresence>
        {result && (
          <motion.div
            initial={{ opacity: 0, y: 20, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -20, scale: 0.95 }}
            transition={{ type: 'spring', stiffness: 300, damping: 25 }}
            className="mt-6 p-5 bg-green-50/80 dark:bg-green-900/20 backdrop-blur-md border border-green-300/50 dark:border-green-700/50 rounded-2xl relative overflow-hidden"
            style={{ boxShadow: '0 0 30px rgba(16, 185, 129, 0.2)' }}
          >
            <p className="text-sm font-medium text-green-800 dark:text-green-200 mb-2">✨ Your shortened URL:</p>
            <div className="flex items-center gap-2">
              <a
                href={result.short_url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-lg font-semibold text-gradient hover:opacity-80 truncate flex-grow"
              >
                {result.short_url}
              </a>
              <motion.button
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
                onClick={() => copyToClipboard(result.short_url)}
                className="btn-secondary flex items-center space-x-1 flex-shrink-0"
              >
                {copied ? <Check className="w-4 h-4 text-green-600" /> : <Copy className="w-4 h-4" />}
                <span>{copied ? 'Copied' : 'Copy'}</span>
              </motion.button>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}
