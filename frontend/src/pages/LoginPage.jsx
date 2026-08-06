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
