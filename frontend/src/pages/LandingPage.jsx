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
