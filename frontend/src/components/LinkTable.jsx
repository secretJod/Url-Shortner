import { Link } from 'react-router-dom';
import { Copy, BarChart2, ExternalLink, Trash2, Check } from 'lucide-react';
import { formatDate, truncateUrl } from '../utils/format';
import { useToast } from '../hooks/useToast';
import { useState } from 'react';
import { motion } from 'framer-motion';

export default function LinkTable({ links, onDelete }) {
  const { showToast } = useToast();
  const [copiedId, setCopiedId] = useState(null);

  const copyToClipboard = (text, id) => {
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    showToast('Copied to clipboard!', 'success');
    setTimeout(() => setCopiedId(null), 2000);
  };

  if (!links || links.length === 0) {
    return (
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        className="card text-center py-16"
      >
        <div className="text-5xl mb-3">🔗</div>
        <p className="text-gray-500 dark:text-gray-400">No links found. Shorten your first URL!</p>
      </motion.div>
    );
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4 }}
      className="card-flush"
    >
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead className="bg-gray-50/80 dark:bg-gray-800/50 backdrop-blur-sm">
            <tr>
              <th scope="col" className="px-6 py-4 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Short URL</th>
              <th scope="col" className="px-6 py-4 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Original URL</th>
              <th scope="col" className="px-6 py-4 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Created</th>
              <th scope="col" className="px-6 py-4 text-right text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {links.map((link, index) => (
              <motion.tr
                key={link.short_code}
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: index * 0.05, duration: 0.3 }}
                whileHover={{
                  backgroundColor: 'rgba(14, 165, 233, 0.05)',
                }}
                className="transition-colors"
              >
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  <div className="flex items-center space-x-2">
                    <a
                      href={`${import.meta.env.VITE_API_URL || 'http://localhost:8080'}/${link.short_code}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="font-medium text-gradient hover:opacity-80 transition-opacity"
                    >
                      /{link.short_code}
                    </a>
                    <ExternalLink className="w-3 h-3 text-gray-400" />
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 max-w-xs truncate" title={link.long_url}>
                  {truncateUrl(link.long_url, 40)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                  {formatDate(link.created_at)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                  <div className="flex justify-end space-x-2">
                    <motion.button
                      whileHover={{ scale: 1.2, y: -2 }}
                      whileTap={{ scale: 0.9 }}
                      onClick={() => copyToClipboard(`${import.meta.env.VITE_API_URL || 'http://localhost:8080'}/${link.short_code}`, link.short_code)}
                      className="p-2 text-gray-500 hover:text-brand-600 dark:text-gray-400 dark:hover:text-brand-400 rounded-lg hover:bg-brand-50 dark:hover:bg-brand-900/30 transition-colors"
                      title="Copy"
                    >
                      {copiedId === link.short_code ? <Check className="w-4 h-4 text-green-500" /> : <Copy className="w-4 h-4" />}
                    </motion.button>
                    <motion.div whileHover={{ scale: 1.2, y: -2 }} whileTap={{ scale: 0.9 }}>
                      <Link
                        to={`/stats/${link.short_code}`}
                        className="block p-2 text-gray-500 hover:text-blue-600 dark:text-gray-400 dark:hover:text-blue-400 rounded-lg hover:bg-blue-50 dark:hover:bg-blue-900/30 transition-colors"
                        title="View Stats"
                      >
                        <BarChart2 className="w-4 h-4" />
                      </Link>
                    </motion.div>
                    {onDelete && (
                      <motion.button
                        whileHover={{ scale: 1.2, y: -2 }}
                        whileTap={{ scale: 0.9 }}
                        onClick={() => onDelete(link.short_code)}
                        className="p-2 text-gray-500 hover:text-red-600 dark:text-gray-400 dark:hover:text-red-400 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/30 transition-colors"
                        title="Delete"
                      >
                        <Trash2 className="w-4 h-4" />
                      </motion.button>
                    )}
                  </div>
                </td>
              </motion.tr>
            ))}
          </tbody>
        </table>
      </div>
    </motion.div>
  );
}
