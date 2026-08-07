import { useState, useEffect } from 'react';
import { Link2, MousePointerClick, Zap, TrendingUp, BarChart3, ShieldCheck } from 'lucide-react';
import ShortenForm from '../components/ShortenForm';
import StatCard from '../components/StatCard';
import apiClient from '../api/client';
import Spinner from '../components/Spinner';
import { motion } from 'framer-motion';

const container = {
  hidden: { opacity: 0 },
  show: { opacity: 1, transition: { staggerChildren: 0.12, delayChildren: 0.1 } },
};
const item = {
  hidden: { opacity: 0, y: 30 },
  show: { opacity: 1, y: 0, transition: { type: 'spring', stiffness: 200, damping: 20 } },
};

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
    <div className="py-12 space-y-20 overflow-hidden relative">
      {/* Decorative floating orbs */}
      <motion.div
        className="absolute top-20 left-10 w-72 h-72 bg-brand-400/20 dark:bg-brand-500/10 rounded-full blur-3xl pointer-events-none"
        animate={{ y: [0, -30, 0], x: [0, 20, 0] }}
        transition={{ duration: 8, repeat: Infinity, ease: 'easeInOut' }}
      />
      <motion.div
        className="absolute top-40 right-10 w-80 h-80 bg-purple-400/20 dark:bg-purple-500/10 rounded-full blur-3xl pointer-events-none"
        animate={{ y: [0, 40, 0], x: [0, -20, 0] }}
        transition={{ duration: 10, repeat: Infinity, ease: 'easeInOut' }}
      />

      {/* Hero Section */}
      <motion.div
        variants={container}
        initial="hidden"
        animate="show"
        className="text-center space-y-6 max-w-4xl mx-auto relative z-10"
      >
        <motion.div
          variants={item}
          className="inline-flex items-center px-4 py-1.5 rounded-full glass border border-brand-400/30 text-brand-700 dark:text-brand-300 text-sm font-medium shadow-glow-brand"
        >
          <Zap className="w-4 h-4 mr-1.5 animate-pulse" />
          Enterprise-Grade URL Shortening
        </motion.div>

        <motion.h1
          variants={item}
          className="text-4xl md:text-6xl font-extrabold text-gray-900 dark:text-white tracking-tight"
        >
          Shorten, Share, and{' '}
          <span className="text-gradient-animated">Analyze</span>{' '}
          Your Links
        </motion.h1>

        <motion.p
          variants={item}
          className="text-xl text-gray-600 dark:text-gray-300 max-w-2xl mx-auto"
        >
          Create short, memorable links in seconds. Track every click with detailed analytics
          and powerful enterprise features.
        </motion.p>
      </motion.div>

      {/* Shorten Form */}
      <motion.div
        initial={{ opacity: 0, y: 30 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.4, duration: 0.5 }}
        className="max-w-3xl mx-auto relative z-10"
      >
        <ShortenForm />
      </motion.div>

      {/* Feature Highlights */}
      <motion.div
        variants={container}
        initial="hidden"
        whileInView="show"
        viewport={{ once: true, amount: 0.2 }}
        className="max-w-5xl mx-auto grid grid-cols-1 md:grid-cols-3 gap-6 relative z-10"
      >
        <FeatureCard
          icon={TrendingUp}
          title="Real-Time Analytics"
          desc="Track every click with detailed charts and insights."
          color="brand"
        />
        <FeatureCard
          icon={BarChart3}
          title="Smart Insights"
          desc="See top referrers, devices, and daily click trends."
          color="purple"
        />
        <FeatureCard
          icon={ShieldCheck}
          title="Enterprise Security"
          desc="API keys, rate limiting, and privacy-first IP hashing."
          color="green"
        />
      </motion.div>

      {/* Stats Preview */}
      <motion.div
        initial={{ opacity: 0 }}
        whileInView={{ opacity: 1 }}
        viewport={{ once: true }}
        transition={{ duration: 0.5 }}
        className="max-w-5xl mx-auto relative z-10"
      >
        <motion.h2
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          className="text-3xl font-bold text-center text-gradient mb-8"
        >
          Trusted by Thousands
        </motion.h2>
        {loadingStats ? (
          <Spinner />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 max-w-2xl mx-auto">
            <motion.div
              initial={{ opacity: 0, scale: 0.9 }}
              whileInView={{ opacity: 1, scale: 1 }}
              viewport={{ once: true }}
              transition={{ delay: 0.1, type: 'spring', stiffness: 200 }}
            >
              <StatCard
                title="Links Shortened"
                value={stats.totalLinks.toLocaleString()}
                icon={Link2}
                color="brand"
              />
            </motion.div>
            <motion.div
              initial={{ opacity: 0, scale: 0.9 }}
              whileInView={{ opacity: 1, scale: 1 }}
              viewport={{ once: true }}
              transition={{ delay: 0.25, type: 'spring', stiffness: 200 }}
            >
              <StatCard
                title="Total Clicks Tracked"
                value={stats.totalClicks.toLocaleString()}
                icon={MousePointerClick}
                color="purple"
              />
            </motion.div>
          </div>
        )}
      </motion.div>
    </div>
  );
}

function FeatureCard({ icon: Icon, title, desc, color }) {
  const colorMap = {
    brand: { bg: 'bg-brand-50 dark:bg-brand-900/30', text: 'text-brand-600 dark:text-brand-400', glow: 'rgba(14,165,233,0.4)' },
    purple: { bg: 'bg-purple-50 dark:bg-purple-900/30', text: 'text-purple-600 dark:text-purple-400', glow: 'rgba(139,92,246,0.4)' },
    green: { bg: 'bg-green-50 dark:bg-green-900/30', text: 'text-green-600 dark:text-green-400', glow: 'rgba(16,185,129,0.4)' },
  };
  const c = colorMap[color];

  return (
    <motion.div
      variants={item}
      whileHover={{ y: -8, rotateX: 5, rotateY: -5 }}
      transition={{ type: 'spring', stiffness: 300, damping: 20 }}
      style={{ transformPerspective: 800 }}
      className="card p-6 text-center group relative overflow-hidden"
    >
      <div
        className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none"
        style={{ background: `radial-gradient(circle at 50% 0%, ${c.glow}, transparent 70%)` }}
      />
      <motion.div
        animate={{ y: [0, -8, 0] }}
        transition={{ duration: 3, repeat: Infinity, ease: 'easeInOut' }}
        className={`relative w-14 h-14 mx-auto mb-4 rounded-2xl ${c.bg} flex items-center justify-center`}
        style={{ boxShadow: `0 0 25px ${c.glow}` }}
      >
        <Icon className={`w-7 h-7 ${c.text}`} />
      </motion.div>
      <h3 className="relative font-semibold text-gray-900 dark:text-white mb-2">{title}</h3>
      <p className="relative text-sm text-gray-600 dark:text-gray-400">{desc}</p>
    </motion.div>
  );
}
