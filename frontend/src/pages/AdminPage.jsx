import { useState, useEffect } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import apiClient from '../api/client';
import StatCard from '../components/StatCard';
import Spinner from '../components/Spinner';
import { Clock, Activity, Server, Zap, AlertTriangle, CheckCircle, XCircle } from 'lucide-react';
import { formatUptime, formatNumber } from '../utils/format';
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts';

const COLORS = ['#10b981', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6', '#6b7280'];

export default function AdminPage() {
  const { apiKey } = useAuth();
  const { showToast } = useToast();
  const [metrics, setMetrics] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchMetrics = async () => {
      try {
        const res = await apiClient.get('/metrics');
        setMetrics(res.data);
      } catch (error) {
        showToast(error.message || 'Failed to fetch metrics', 'error');
      } finally {
        setLoading(false);
      }
    };
    fetchMetrics();
  }, [apiKey]);

  if (loading) {
    return <Spinner size="lg" className="py-20" />;
  }

  if (!metrics) {
    return (
      <div className="card text-center py-20">
        <p className="text-gray-500 dark:text-gray-400">Failed to load system metrics.</p>
      </div>
    );
  }

  const statusCodesData = Object.entries(metrics.status_codes || {}).map(([name, value]) => ({
    name: `HTTP ${name}`,
    value: value
  }));

  const cacheHitRate = metrics.cache_hit_rate || 0;

  return (
    <div className="py-8 space-y-8 animate-fade-in">
      <div className="flex items-center space-x-3">
        <Server className="w-8 h-8 text-brand-600" />
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">System Metrics</h1>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard title="Uptime" value={formatUptime(metrics.uptime_seconds)} icon={Clock} color="green" />
        <StatCard title="Total Requests" value={formatNumber(metrics.total_requests)} icon={Activity} color="blue" />
        <StatCard title="Avg Latency" value={`${metrics.avg_latency_ms?.toFixed(2)} ms`} icon={Zap} color="purple" />
        <StatCard title="Max Latency" value={`${metrics.max_latency_ms?.toFixed(2)} ms`} icon={AlertTriangle} color="orange" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="card">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Status Code Breakdown</h2>
          <div className="h-64 flex items-center justify-center">
            {statusCodesData.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={statusCodesData}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={80}
                    paddingAngle={5}
                    dataKey="value"
                    label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                    labelLine={false}
                  >
                    {statusCodesData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip 
                    contentStyle={{ 
                      backgroundColor: '#1f2937', 
                      border: 'none', 
                      borderRadius: '8px',
                      color: '#fff'
                    }}
                  />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <p className="text-gray-500 dark:text-gray-400">No request data available.</p>
            )}
          </div>
        </div>

        <div className="card space-y-6">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">Cache Performance</h2>
            <div className="flex justify-between mb-2 text-sm">
              <span className="text-gray-600 dark:text-gray-400">Hit Rate</span>
              <span className="font-semibold text-gray-900 dark:text-white">{cacheHitRate.toFixed(1)}%</span>
            </div>
            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-4 overflow-hidden">
              <div 
                className="h-full bg-gradient-to-r from-green-400 to-blue-500 rounded-full transition-all duration-500"
                style={{ width: `${cacheHitRate}%` }}
              />
            </div>
            <div className="flex justify-between mt-4 text-sm text-gray-600 dark:text-gray-400">
              <div className="flex items-center space-x-1">
                <CheckCircle className="w-4 h-4 text-green-500" />
                <span>{formatNumber(metrics.cache_hits)} Hits</span>
              </div>
              <div className="flex items-center space-x-1">
                <XCircle className="w-4 h-4 text-red-500" />
                <span>{formatNumber(metrics.cache_misses)} Misses</span>
              </div>
            </div>
          </div>

          <div className="pt-4 border-t border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Event Processing</h2>
            <div className="grid grid-cols-2 gap-4">
              <div className="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg text-center">
                <p className="text-2xl font-bold text-green-700 dark:text-green-400">{formatNumber(metrics.events_processed)}</p>
                <p className="text-sm text-green-600 dark:text-green-300">Processed</p>
              </div>
              <div className="p-4 bg-red-50 dark:bg-red-900/20 rounded-lg text-center">
                <p className="text-2xl font-bold text-red-700 dark:text-red-400">{formatNumber(metrics.events_failed)}</p>
                <p className="text-sm text-red-600 dark:text-red-300">Failed</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
