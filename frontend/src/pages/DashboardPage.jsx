import { useState, useEffect } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import apiClient from '../api/client';
import LinkTable from '../components/LinkTable';
import ShortenForm from '../components/ShortenForm';
import Spinner from '../components/Spinner';
import { Search, Plus, X } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';

const pageVariants = {
  hidden: { opacity: 0, y: 20 },
  show: { opacity: 1, y: 0, transition: { duration: 0.4, staggerChildren: 0.08 } },
};
const child = {
  hidden: { opacity: 0, y: 15 },
  show: { opacity: 1, y: 0 },
};

export default function DashboardPage() {
  const { apiKey } = useAuth();
  const { showToast } = useToast();

  const [links, setLinks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [searchFocused, setSearchFocused] = useState(false);

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
    <motion.div
      variants={pageVariants}
      initial="hidden"
      animate="show"
      className="py-8 space-y-8"
    >
      <motion.div variants={child} className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold text-gradient">Dashboard</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">Manage and track your shortened links.</p>
        </div>
        <motion.button
          whileHover={{ scale: 1.03 }}
          whileTap={{ scale: 0.97 }}
          onClick={() => setShowForm(!showForm)}
          className="btn-primary flex items-center space-x-2"
        >
          <motion.div animate={{ rotate: showForm ? 90 : 0 }} transition={{ duration: 0.3 }}>
            {showForm ? <X className="w-4 h-4" /> : <Plus className="w-4 h-4" />}
          </motion.div>
          <span>{showForm ? 'Cancel' : 'Shorten New URL'}</span>
        </motion.button>
      </motion.div>

      <AnimatePresence>
        {showForm && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.3 }}
            className="overflow-hidden"
          >
            <ShortenForm onSuccess={() => {
              fetchLinks();
              setShowForm(false);
            }} />
          </motion.div>
        )}
      </AnimatePresence>

      <motion.div variants={child} className="card p-4">
        <div className="relative mb-4">
          <div
            className={`absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none transition-colors duration-200 ${searchFocused ? 'text-brand-600 dark:text-brand-400' : 'text-gray-400'}`}
          >
            <Search className="h-5 w-5" />
          </div>
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            onFocus={() => setSearchFocused(true)}
            onBlur={() => setSearchFocused(false)}
            placeholder="Search by short code or URL..."
            className="input-field pl-11"
          />
        </div>

        {loading ? (
          <Spinner size="lg" />
        ) : (
          <LinkTable links={filteredLinks} onDelete={handleDelete} />
        )}
      </motion.div>
    </motion.div>
  );
}
