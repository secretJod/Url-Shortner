import { useState, useEffect } from 'react';
import { Link2, MousePointerClick, Zap, TrendingUp, BarChart3, ShieldCheck } from 'lucide-react';
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
    <div className="py-12 space-y-16 overflow-hidden">
      {/* Hero Section */}
      <div className="text-center space-y-6 max-w-4xl mx-auto animate-slide-up">
        <div className="inline-flex items-center px-3 py-1 rounded-full bg-brand-100 dark:bg-brand-900/50 text-brand-800 dark:text-brand-200 text-sm font-medium mb-4 animate-bounce-slow">
          <Zap className="w-4 h-4 mr-1" />
          Enterprise-Grade URL Shortening
        </div>
        <h1 className="text-4xl md:text-6xl font-extrabold text-gray-900 dark:text-white tracking-tight">
          Shorten, Share, and{' '}
          <span className="text-brand-600 dark:text-brand-400 animate-pulse-slow">Analyze</span>{' '}
          Your Links
        </h1>
        <p className="text-xl text-gray-600 dark:text-gray-300 max-w-2xl mx-auto animate-fade-in">
          Create short, memorable links in seconds. Track every click with detailed analytics 
          and powerful enterprise features.
        </p>
      </div>

      {/* Shorten Form */}
      <div className="max-w-3xl mx-auto animate-scale-in">
        <ShortenForm />
      </div>

      {/* Feature Highlights */}
      <div className="max-w-5xl mx-auto grid grid-cols-1 md:grid-cols-3 gap-6 animate-slide-up">
        <div className="card p-6 text-center hover:shadow-lg transition-all duration-300 hover:-translate-y-1">
          <div className="w-12 h-12 mx-auto mb-4 rounded-full bg-brand-100 dark:bg-brand-900/50 flex items-center justify-center animate-float">
            <TrendingUp className="w-6 h-6 text-brand-600 dark:text-brand-400" />
          </div>
          <h3 className="font-semibold text-gray-900 dark:text-white mb-2">Real-Time Analytics</h3>
          <p className="text-sm text-gray-600 dark:text-gray-400">Track every click with detailed charts and insights.</p>
        </div>
        <div className="card p-6 text-center hover:shadow-lg transition-all duration-300 hover:-translate-y-1">
          <div className="w-12 h-12 mx-auto mb-4 rounded-full bg-purple-100 dark:bg-purple-900/50 flex items-center justify-center animate-float" style={{ animationDelay: '0.5s' }}>
            <BarChart3 className="w-6 h-6 text-purple-600 dark:text-purple-400" />
          </div>
          <h3 className="font-semibold text-gray-900 dark:text-white mb-2">Smart Insights</h3>
          <p className="text-sm text-gray-600 dark:text-gray-400">See top referrers, devices, and daily click trends.</p>
        </div>
        <div className="card p-6 text-center hover:shadow-lg transition-all duration-300 hover:-translate-y-1">
          <div className="w-12 h-12 mx-auto mb-4 rounded-full bg-green-100 dark:bg-green-900/50 flex items-center justify-center animate-float" style={{ animationDelay: '1s' }}>
            <ShieldCheck className="w-6 h-6 text-green-600 dark:text-green-400" />
          </div>
          <h3 className="font-semibold text-gray-900 dark:text-white mb-2">Enterprise Security</h3>
          <p className="text-sm text-gray-600 dark:text-gray-400">API keys, rate limiting, and privacy-first IP hashing.</p>
        </div>
      </div>

      {/* Stats Preview */}
      <div className="max-w-5xl mx-auto">
        <h2 className="text-2xl font-bold text-center text-gray-900 dark:text-white mb-8 animate-fade-in">
          Trusted by Thousands
        </h2>
        {loadingStats ? (
          <Spinner />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 max-w-2xl mx-auto">
            <div className="animate-scale-in">
              <StatCard 
                title="Links Shortened" 
                value={stats.totalLinks.toLocaleString()} 
                icon={Link2} 
                color="brand" 
              />
            </div>
            <div className="animate-scale-in" style={{ animationDelay: '0.2s' }}>
              <StatCard 
                title="Total Clicks Tracked" 
                value={stats.totalClicks.toLocaleString()} 
                icon={MousePointerClick} 
                color="purple" 
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}