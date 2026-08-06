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
