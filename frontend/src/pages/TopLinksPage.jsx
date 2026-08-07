import { useState, useEffect } from 'react';
import { useToast } from '../hooks/useToast';
import apiClient from '../api/client';
import Spinner from '../components/Spinner';
import { Trophy, ExternalLink, ArrowUpRight } from 'lucide-react';
import { Link } from 'react-router-dom';
import { truncateUrl } from '../utils/format';
import { motion } from 'framer-motion';

const getMedalStyle = (index) => {
  if (index === 0) return {
    bg: 'bg-gradient-to-br from-yellow-300 to-amber-500',
    text: 'text-white',
    glow: 'rgba(250, 204, 21, 0.5)',
    ring: 'shadow-[0_0_25px_rgba(250,204,21,0.5)]',
  };
  if (index === 1) return {
    bg: 'bg-gradient-to-br from-gray-200 to-gray-400',
    text: 'text-white',
    glow: 'rgba(156, 163, 175, 0.5)',
    ring: 'shadow-[0_0_25px_rgba(156,163,175,0.4)]',
  };
  if (index === 2) return {
    bg: 'bg-gradient-to-br from-orange-300 to-amber-700',
    text: 'text-white',
    glow: 'rgba(217, 119, 6, 0.5)',
    ring: 'shadow-[0_0_25px_rgba(217,119,6,0.4)]',
  };
  return {
    bg: 'bg-gradient-to-br from-gray-100 to-gray-300 dark:from-gray-700 dark:to-gray-800',
    text: 'text-gray-600 dark:text-gray-300',
    glow: 'rgba(107, 114, 128, 0.3)',
    ring: '',
  };
};

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

  if (loading) {
    return <Spinner size="lg" className="py-20" />;
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4 }}
      className="py-8 space-y-8"
    >
      <div className="text-center space-y-3">
        <motion.div
          initial={{ scale: 0, rotate: -180 }}
          animate={{ scale: 1, rotate: 0 }}
          transition={{ type: 'spring', stiffness: 200, damping: 15 }}
          className="inline-block"
        >
          <Trophy className="w-12 h-12 mx-auto text-yellow-500 drop-shadow-[0_0_20px_rgba(234,179,8,0.6)]" />
        </motion.div>
        <h1 className="text-3xl font-bold text-gradient">Top Performing Links</h1>
        <p className="text-gray-600 dark:text-gray-400">The most clicked links across the platform.</p>
      </div>

      <div className="max-w-4xl mx-auto space-y-4">
        {links.length === 0 ? (
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            className="card text-center py-12"
          >
            <div className="text-5xl mb-3">🏆</div>
            <p className="text-gray-500 dark:text-gray-400">No links have been clicked yet. Be the first!</p>
          </motion.div>
        ) : (
          links.map((link, index) => {
            const medal = getMedalStyle(index);
            return (
              <motion.div
                key={link.short_code}
                initial={{ opacity: 0, x: -30 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: index * 0.07, type: 'spring', stiffness: 200, damping: 20 }}
                whileHover={{ scale: 1.02, x: 6 }}
                className="card flex items-center justify-between group relative overflow-hidden"
              >
                <div className="flex items-center space-x-4 flex-grow min-w-0">
                  <div className={`relative flex-shrink-0 w-12 h-12 rounded-2xl flex items-center justify-center font-bold text-lg ${medal.bg} ${medal.text} ${medal.ring}`}>
                    {index + 1}
                    {index < 3 && <div className="absolute inset-0 rounded-2xl shimmer-surface opacity-40" />}
                  </div>
                  <div className="min-w-0 flex-grow">
                    <a
                      href={`${import.meta.env.VITE_API_URL || 'http://localhost:8080'}/${link.short_code}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-lg font-semibold text-gradient hover:opacity-80 flex items-center space-x-1 transition-opacity"
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
                  <motion.div
                    whileHover={{ scale: 1.15 }}
                    className="text-right"
                  >
                    <p className="text-2xl font-bold text-gradient">{link.total_clicks.toLocaleString()}</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wider">Clicks</p>
                  </motion.div>
                  <motion.div whileHover={{ scale: 1.15, rotate: 10 }} whileTap={{ scale: 0.9 }}>
                    <Link
                      to={`/stats/${link.short_code}`}
                      className="p-2.5 rounded-xl bg-gray-100 dark:bg-gray-700 hover:bg-gradient-to-br hover:from-brand-500 hover:to-indigo-500 text-gray-600 dark:text-gray-300 hover:text-white transition-all"
                      title="View Stats"
                    >
                      <ArrowUpRight className="w-5 h-5" />
                    </Link>
                  </motion.div>
                </div>
              </motion.div>
            );
          })
        )}
      </div>
    </motion.div>
  );
}
