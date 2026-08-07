import { useState, useEffect } from 'react';
import { useToast } from '../hooks/useToast';
import apiClient from '../api/client';
import Spinner from '../components/Spinner';
import { Trophy, ExternalLink, ArrowUpRight } from 'lucide-react';
import { Link } from 'react-router-dom';
import { truncateUrl } from '../utils/format';

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

  const getMedalColor = (index) => {
    if (index === 0) return 'text-yellow-500 bg-yellow-50 dark:bg-yellow-900/30';
    if (index === 1) return 'text-gray-400 bg-gray-50 dark:bg-gray-800';
    if (index === 2) return 'text-orange-600 bg-orange-50 dark:bg-orange-900/30';
    return 'text-gray-500 bg-gray-50 dark:bg-gray-800';
  };

  if (loading) {
    return <Spinner size="lg" className="py-20" />;
  }

  return (
    <div className="py-8 space-y-8 animate-fade-in">
      <div className="text-center space-y-2">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white flex items-center justify-center">
          <Trophy className="w-8 h-8 mr-3 text-yellow-500" />
          Top Performing Links
        </h1>
        <p className="text-gray-600 dark:text-gray-400">The most clicked links across the platform.</p>
      </div>

      <div className="max-w-4xl mx-auto space-y-4">
        {links.length === 0 ? (
          <div className="card text-center py-12">
            <p className="text-gray-500 dark:text-gray-400">No links have been clicked yet. Be the first!</p>
          </div>
        ) : (
          links.map((link, index) => (
            <div 
              key={link.short_code} 
              className="card flex items-center justify-between hover:shadow-md transition-all duration-200 group"
            >
              <div className="flex items-center space-x-4 flex-grow min-w-0">
                <div className={`flex-shrink-0 w-10 h-10 rounded-full flex items-center justify-center font-bold text-lg ${getMedalColor(index)}`}>
                  {index + 1}
                </div>
                <div className="min-w-0 flex-grow">
                  <a 
                    href={`${import.meta.env.VITE_API_URL || 'http://localhost:8080'}/${link.short_code}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-lg font-semibold text-brand-600 dark:text-brand-400 hover:underline flex items-center space-x-1"
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
                <div className="text-right">
                  <p className="text-2xl font-bold text-gray-900 dark:text-white">{link.total_clicks.toLocaleString()}</p>
                  <p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wider">Clicks</p>
                </div>
                <Link 
                  to={`/stats/${link.short_code}`}
                  className="p-2 rounded-lg bg-gray-100 dark:bg-gray-700 hover:bg-brand-50 dark:hover:bg-brand-900/30 text-gray-600 dark:text-gray-300 hover:text-brand-600 dark:hover:text-brand-400 transition-colors"
                  title="View Stats"
                >
                  <ArrowUpRight className="w-5 h-5" />
                </Link>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
