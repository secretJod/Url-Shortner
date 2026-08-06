import { Link } from 'react-router-dom';
import { Copy, BarChart2, ExternalLink, Trash2, Check } from 'lucide-react';
import { formatDate, truncateUrl } from '../utils/format';
import { useToast } from '../hooks/useToast';
import { useState } from 'react';

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
      <div className="card text-center py-12">
        <p className="text-gray-500 dark:text-gray-400">No links found. Shorten your first URL!</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto card p-0">
      <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        <thead className="bg-gray-50 dark:bg-gray-800/50">
          <tr>
            <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Short URL</th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Original URL</th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Created</th>
            <th scope="col" className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Actions</th>
          </tr>
        </thead>
        <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
          {links.map((link) => (
            <tr key={link.short_code} className="hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
              <td className="px-6 py-4 whitespace-nowrap text-sm">
                <div className="flex items-center space-x-2">
                  <a
                    href={`${window.location.origin}/${link.short_code}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-brand-600 dark:text-brand-400 hover:underline font-medium"
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
                  <button
                    onClick={() => copyToClipboard(`${window.location.origin}/${link.short_code}`, link.short_code)}
                    className="p-1.5 text-gray-500 hover:text-brand-600 dark:text-gray-400 dark:hover:text-brand-400 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                    title="Copy"
                  >
                    {copiedId === link.short_code ? <Check className="w-4 h-4 text-green-500" /> : <Copy className="w-4 h-4" />}
                  </button>
                  <Link
                    to={`/stats/${link.short_code}`}
                    className="p-1.5 text-gray-500 hover:text-blue-600 dark:text-gray-400 dark:hover:text-blue-400 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                    title="View Stats"
                  >
                    <BarChart2 className="w-4 h-4" />
                  </Link>
                  {onDelete && (
                    <button
                      onClick={() => onDelete(link.short_code)}
                      className="p-1.5 text-gray-500 hover:text-red-600 dark:text-gray-400 dark:hover:text-red-400 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                      title="Delete"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  )}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
