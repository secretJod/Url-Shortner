import { useState } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import { KeyRound, Mail, Copy, Check, AlertTriangle, ArrowRight } from 'lucide-react';
import { isValidApiKey } from '../utils/format';
import { useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';

export default function LoginPage() {
  const { login, importKey } = useAuth();
  const { showToast } = useToast();
  const navigate = useNavigate();

  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [generatedKey, setGeneratedKey] = useState(null);
  const [copied, setCopied] = useState(false);
  const [emailFocused, setEmailFocused] = useState(false);

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

  return (
    <div className="min-h-[70vh] flex items-center justify-center py-12 relative overflow-hidden">
      {/* Animated background orbs */}
      <motion.div
        className="absolute top-0 left-1/4 w-72 h-72 bg-brand-400/20 dark:bg-brand-500/10 rounded-full blur-3xl pointer-events-none"
        animate={{ y: [0, -30, 0], scale: [1, 1.1, 1] }}
        transition={{ duration: 8, repeat: Infinity, ease: 'easeInOut' }}
      />
      <motion.div
        className="absolute bottom-0 right-1/4 w-80 h-80 bg-purple-400/20 dark:bg-purple-500/10 rounded-full blur-3xl pointer-events-none"
        animate={{ y: [0, 30, 0], scale: [1, 1.15, 1] }}
        transition={{ duration: 10, repeat: Infinity, ease: 'easeInOut' }}
      />

      <div className="max-w-md w-full mx-auto space-y-8 relative z-10">
        <AnimatePresence mode="wait">
          {generatedKey ? (
            <motion.div
              key="key-display"
              initial={{ opacity: 0, scale: 0.85, rotateY: -90 }}
              animate={{ opacity: 1, scale: 1, rotateY: 0 }}
              exit={{ opacity: 0, scale: 0.85, rotateY: 90 }}
              transition={{ type: 'spring', stiffness: 200, damping: 25 }}
              className="card text-center space-y-6 relative overflow-hidden"
            >
              <motion.div
                initial={{ scale: 0, rotate: -180 }}
                animate={{ scale: 1, rotate: 0 }}
                transition={{ type: 'spring', stiffness: 300, damping: 15, delay: 0.2 }}
                className="mx-auto w-16 h-16 bg-gradient-to-br from-green-400 to-emerald-600 rounded-2xl flex items-center justify-center shadow-glow-green"
              >
                <KeyRound className="w-8 h-8 text-white" />
              </motion.div>
              <h2 className="text-2xl font-bold text-gradient">Your API Key is Ready</h2>

              <div className="bg-yellow-50/80 dark:bg-yellow-900/20 backdrop-blur-md border border-yellow-300/50 dark:border-yellow-700/50 rounded-xl p-4 flex items-start space-x-3 text-left">
                <AlertTriangle className="w-5 h-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-0.5 animate-pulse" />
                <div>
                  <p className="text-sm font-semibold text-yellow-800 dark:text-yellow-200">Warning: Save this key now!</p>
                  <p className="text-xs text-yellow-700 dark:text-yellow-300 mt-1">
                    For security reasons, this key will only be shown once. If you lose it, you will need to generate a new one.
                  </p>
                </div>
              </div>

              <div className="bg-gray-50/80 dark:bg-gray-900/50 backdrop-blur-md border border-gray-200/50 dark:border-gray-700/50 rounded-xl p-4 break-all font-mono text-sm text-gray-800 dark:text-gray-200 relative">
                {generatedKey}
                <div className="absolute inset-0 shimmer-surface opacity-30 pointer-events-none rounded-xl" />
              </div>

              <motion.button
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.98 }}
                onClick={copyToClipboard}
                className="btn-primary w-full flex items-center justify-center space-x-2"
              >
                {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
                <span>{copied ? 'Copied!' : 'Copy to Clipboard'}</span>
              </motion.button>

              <motion.button
                whileHover={{ x: 4 }}
                onClick={() => navigate('/dashboard')}
                className="text-gradient hover:opacity-80 text-sm font-medium flex items-center justify-center space-x-1 mx-auto transition-opacity"
              >
                <span>I've saved it, go to Dashboard</span>
                <ArrowRight className="w-4 h-4" />
              </motion.button>
            </motion.div>
          ) : (
            <motion.div
              key="login-form"
              initial={{ opacity: 0, y: 30 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -30 }}
              className="space-y-8"
            >
              <div className="text-center space-y-2">
                <motion.h1
                  initial={{ opacity: 0, y: -20 }}
                  animate={{ opacity: 1, y: 0 }}
                  className="text-3xl font-bold text-gradient"
                >
                  Welcome to LinkSnip
                </motion.h1>
                <p className="text-gray-600 dark:text-gray-400">Get your API key to unlock enterprise features.</p>
              </div>

              <div className="card space-y-6 relative overflow-hidden">
                <div
                  className="absolute top-0 left-0 right-0 h-1"
                  style={{ backgroundImage: 'linear-gradient(90deg, #0ea5e9, #8b5cf6)' }}
                />
                <form onSubmit={handleGenerateKey} className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                      Email Address
                    </label>
                    <div className="relative">
                      <div
                        className={`absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none transition-colors duration-200 ${emailFocused ? 'text-brand-600 dark:text-brand-400' : 'text-gray-400'}`}
                      >
                        <Mail className="h-5 w-5" />
                      </div>
                      <input
                        type="email"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        onFocus={() => setEmailFocused(true)}
                        onBlur={() => setEmailFocused(false)}
                        placeholder="you@company.com"
                        className="input-field pl-11"
                        required
                      />
                    </div>
                  </div>
                  <motion.button
                    whileHover={{ scale: 1.01 }}
                    whileTap={{ scale: 0.99 }}
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
                  </motion.button>
                </form>

                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <div className="w-full border-t border-gray-200 dark:border-gray-700"></div>
                  </div>
                  <div className="relative flex justify-center text-sm">
                    <span className="px-3 glass text-gray-500 dark:text-gray-400 rounded-full">Or</span>
                  </div>
                </div>

                <form onSubmit={handleImportKey} className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
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
                  <motion.button
                    whileHover={{ scale: 1.01 }}
                    whileTap={{ scale: 0.99 }}
                    type="submit"
                    className="btn-secondary w-full flex items-center justify-center space-x-2"
                  >
                    <span>Import Existing Key</span>
                    <ArrowRight className="w-4 h-4" />
                  </motion.button>
                </form>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
