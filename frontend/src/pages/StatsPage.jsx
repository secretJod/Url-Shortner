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
      <div className="card text-center py-20">
        <p className="text-gray-500 dark:text-gray-400">Link not found or you don't have permission to view it.</p>
        <Link to="/dashboard" className="btn-primary mt-4 inline-block">Back to Dashboard</Link>
      </div>
    );
  }

  return (
    <div className="py-8 space-y-8 animate-fade-in">
      <div className="flex items-center space-x-4">
        <Link to="/dashboard" className="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors">
          <ArrowLeft className="w-5 h-5 text-gray-600 dark:text-gray-300" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Analytics for /{stats.short_code}</h1>
          <a 
            href={stats.long_url} 
            target="_blank" 
            rel="noopener noreferrer"
            className="text-sm text-brand-600 dark:text-brand-400 hover:underline truncate block max-w-md"
          >
            {stats.long_url}
          </a>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <StatCard title="Total Clicks" value={stats.total_clicks} icon={MousePointerClick} color="brand" />
        <StatCard title="Unique Visitors" value={stats.unique_ips} icon={Users} color="green" />
      </div>

      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Daily Clicks</h2>
        <DailyChart data={stats.daily_clicks} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="card">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
            <Globe className="w-5 h-5 mr-2 text-brand-600" />
            Top Referrers
          </h2>
          {stats.top_referrers && stats.top_referrers.length > 0 ? (
            <ul className="space-y-3">
              {stats.top_referrers.map((ref, idx) => (
                <li key={idx} className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-200 truncate">{ref}</span>
                  <span className="text-xs font-semibold text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30 px-2 py-1 rounded-full">
                    #{idx + 1}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-gray-500 dark:text-gray-400 text-sm">No referrer data available.</p>
          )}
        </div>

        <div className="card">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
            <Clock className="w-5 h-5 mr-2 text-brand-600" />
            Recent Clicks
          </h2>
          <div className="overflow-x-auto max-h-64 overflow-y-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-800/50 sticky top-0">
                <tr>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Time</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Country</th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Device</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {clicks.length > 0 ? clicks.map((click, idx) => (
                  <tr key={idx} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
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
                  </tr>
                )) : (
                  <tr>
                    <td colSpan="3" className="px-4 py-8 text-center text-sm text-gray-500">No recent clicks</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
