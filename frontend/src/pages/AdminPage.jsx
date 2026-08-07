import { useState, useEffect } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import apiClient from '../api/client';
import StatCard from '../components/StatCard';
import Spinner from '../components/Spinner';
import { Clock, Activity, Server, Zap, AlertTriangle, CheckCircle, XCircle } from 'lucide-react';
import { formatUptime, formatNumber } from '../utils/format';
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts';
import { motion } from 'framer-motion';

const COLORS = ['#10b981', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6', '#6b7280'];

const container = {
  hidden: { opacity: 0 },
  show: { opacity: 1, transition: { staggerChildren: 0.1, delayChildren: 0.1 } },
};
const item = {
  hidden: { opacity: 0, y: 20 },
  show: { opacity: 1, y: 0 },
};

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
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        className="card text-center py-20"
      >
        <p className="text-gray-500 dark:text-gray-400">Failed to load system metrics.</p>
      </motion.div>
    );
  }

  const statusCodesData = Object.entries(metrics.status_codes || {}).map(([name, value]) => ({
    name: `HTTP ${name}`,
    value: value
  }));

  const cacheHitRate = metrics.cache_hit_rate || 0;

  return (
    <motion.div
      variants={container}
      initial="hidden"
      animate="show"
      className="py-8 space-y-8"
    >
      <motion.div variants={item} className="flex items-center space-x-3">
        <motion.div
          animate={{ rotate: [0, 5, -5, 0] }}
          transition={{ duration: 4, repeat: Infinity, ease: 'easeInOut' }}
        >
          <Server className="w-8 h-8 text-gradient" />
        </motion.div>
        <h1 className="text-3xl font-bold text-gradient">System Metrics</h1>
      </motion.div>

      <motion.div variants={item} className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard title="Uptime" value={formatUptime(metrics.uptime_seconds)} icon={Clock} color="green" />
        <StatCard title="Total Requests" value={formatNumber(metrics.total_requests)} icon={Activity} color="blue" />
        <StatCard title="Avg Latency" value={`${metrics.avg_latency_ms?.toFixed(2)} ms`} icon={Zap} color="purple" />
        <StatCard title="Max Latency" value={`${metrics.max_latency_ms?.toFixed(2)} ms`} icon={AlertTriangle} color="orange" />
      </motion.div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <motion.div variants={item} className="card">
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
                    outerRadius={90}
                    paddingAngle={5}
                    dataKey="value"
                    label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                    labelLine={false}
                    isAnimationActive
                    animationDuration={800}
                  >
                    {statusCodesData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={{
                      backgroundColor: 'rgba(31, 41, 55, 0.9)',
                      backdropFilter: 'blur(10px)',
                      border: '1px solid rgba(255,255,255,0.1)',
                      borderRadius: '12px',
                      color: '#fff',
                      boxShadow: '0 8px 32px rgba(0,0,0,0.3)',
                    }}
                  />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <p className="text-gray-500 dark:text-gray-400">No request data available.</p>
            )}
          </div>
        </motion.div>

        <motion.div variants={item} className="card space-y-6">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">Cache Performance</h2>
            <div className="flex justify-between mb-2 text-sm">
              <span className="text-gray-600 dark:text-gray-400">Hit Rate</span>
              <span className="font-semibold text-gradient">{cacheHitRate.toFixed(1)}%</span>
            </div>
            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-4 overflow-hidden relative">
              <motion.div
                initial={{ width: 0 }}
                animate={{ width: `${cacheHitRate}%` }}
                transition={{ duration: 1, ease: 'easeOut', delay: 0.3 }}
                className="h-full rounded-full relative overflow-hidden"
                style={{ backgroundImage: 'linear-gradient(90deg, #10b981, #3b82f6)' }}
              >
                <div className="absolute inset-0 shimmer-surface opacity-50" />
              </motion.div>
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
              <motion.div
                whileHover={{ scale: 1.03, y: -2 }}
                className="p-4 bg-green-50/70 dark:bg-green-900/20 backdrop-blur-sm rounded-xl text-center border border-green-200/50 dark:border-green-800/50"
                style={{ boxShadow: '0 0 25px rgba(16, 185, 129, 0.15)' }}
              >
                <p className="text-2xl font-bold text-green-700 dark:text-green-400">{formatNumber(metrics.events_processed)}</p>
                <p className="text-sm text-green-600 dark:text-green-300 mt-1">Processed</p>
              </motion.div>
              <motion.div
                whileHover={{ scale: 1.03, y: -2 }}
                className="p-4 bg-red-50/70 dark:bg-red-900/20 backdrop-blur-sm rounded-xl text-center border border-red-200/50 dark:border-red-800/50"
                style={{ boxShadow: '0 0 25px rgba(239, 68, 68, 0.15)' }}
              >
                <p className="text-2xl font-bold text-red-700 dark:text-red-400">{formatNumber(metrics.events_failed)}</p>
                <p className="text-sm text-red-600 dark:text-red-300 mt-1">Failed</p>
              </motion.div>
            </div>
          </div>
        </motion.div>
      </div>
    </motion.div>
  );
}
