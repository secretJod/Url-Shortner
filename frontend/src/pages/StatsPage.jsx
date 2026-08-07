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
import { motion } from 'framer-motion';

const container = {
  hidden: { opacity: 0 },
  show: { opacity: 1, transition: { staggerChildren: 0.08, delayChildren: 0.1 } },
};
const item = {
  hidden: { opacity: 0, y: 20 },
  show: { opacity: 1, y: 0 },
};

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
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        className="card text-center py-20"
      >
        <p className="text-gray-500 dark:text-gray-400">Link not found or you don't have permission to view it.</p>
        <Link to="/dashboard" className="btn-primary mt-4 inline-block">Back to Dashboard</Link>
      </motion.div>
    );
  }

  return (
    <motion.div
      variants={container}
      initial="hidden"
      animate="show"
      className="py-8 space-y-8"
    >
      <motion.div variants={item} className="flex items-center space-x-4">
        <motion.div whileHover={{ x: -4, scale: 1.1 }} whileTap={{ scale: 0.9 }}>
          <Link to="/dashboard" className="p-3 rounded-xl glass hover:shadow-glow-brand transition-all">
            <ArrowLeft className="w-5 h-5 text-gray-600 dark:text-gray-300" />
          </Link>
        </motion.div>
        <div>
          <h1 className="text-2xl font-bold text-gradient">Analytics for /{stats.short_code}</h1>
          <a
            href={stats.long_url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-sm text-brand-600 dark:text-brand-400 hover:underline truncate block max-w-md"
          >
            {stats.long_url}
          </a>
        </div>
      </motion.div>

      <motion.div variants={item} className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <StatCard title="Total Clicks" value={stats.total_clicks} icon={MousePointerClick} color="brand" />
        <StatCard title="Unique Visitors" value={stats.unique_ips} icon={Users} color="green" />
      </motion.div>

      <motion.div variants={item} className="card">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Daily Clicks</h2>
        <DailyChart data={stats.daily_clicks} />
      </motion.div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <motion.div variants={item} className="card">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
            <Globe className="w-5 h-5 mr-2 text-brand-600" />
            Top Referrers
          </h2>
          {stats.top_referrers && stats.top_referrers.length > 0 ? (
            <motion.ul
              className="space-y-3"
              variants={{ show: { transition: { staggerChildren: 0.08 } } }}
            >
              {stats.top_referrers.map((ref, idx) => (
                <motion.li
                  key={idx}
                  variants={{ hidden: { opacity: 0, x: -20 }, show: { opacity: 1, x: 0 } }}
                  whileHover={{ x: 4, scale: 1.02 }}
                  className="flex items-center justify-between p-3 bg-gray-50/70 dark:bg-gray-700/40 backdrop-blur-sm rounded-xl border border-gray-200/50 dark:border-gray-700/50"
                >
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-200 truncate">{ref}</span>
                  <span className="text-xs font-semibold text-white bg-gradient-to-r from-brand-500 to-indigo-500 px-2.5 py-1 rounded-full shadow-glow-brand">
                    #{idx + 1}
                  </span>
                </motion.li>
              ))}
            </motion.ul>
          ) : (
            <p className="text-gray-500 dark:text-gray-400 text-sm">No referrer data available.</p>
          )}
        </motion.div>

        <motion.div variants={item} className="card">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
            <Clock className="w-5 h-5 mr-2 text-brand-600" />
            Recent Clicks
          </h2>
          <div className="overflow-x-auto max-h-64 overflow-y-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50/80 dark:bg-gray-800/50 backdrop-blur-sm sticky top-0">
                <tr>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Time</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Country</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Device</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {clicks.length > 0 ? clicks.map((click, idx) => (
                  <motion.tr
                    key={idx}
                    initial={{ opacity: 0, x: 10 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: idx * 0.03 }}
                    className="hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
                  >
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
                  </motion.tr>
                )) : (
                  <tr>
                    <td colSpan="3" className="px-4 py-8 text-center text-sm text-gray-500">No recent clicks</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </motion.div>
      </div>
    </motion.div>
  );
}
