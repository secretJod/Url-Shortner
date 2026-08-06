import { useState, useEffect } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import apiClient from '../api/client';
import LinkTable from '../components/LinkTable';
import ShortenForm from '../components/ShortenForm';
import Spinner from '../components/Spinner';
import { Search, Plus, X } from 'lucide-react';

export default function DashboardPage() {
  const { apiKey } = useAuth();
  const { showToast } = useToast();
  
  const [links, setLinks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [showForm, setShowForm] = useState(false);

  const fetchLinks = async () => {
    setLoading(true);
    try {
      const res = await apiClient.get('/api/links');
      setLinks(res.data.links || []);
    } catch (error) {
      showToast(error.message || 'Failed to fetch links', 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (apiKey) {
      fetchLinks();
    }
  }, [apiKey]);

  const handleDelete = async (shortCode) => {
    if (!window.confirm(`Are you sure you want to delete /${shortCode}?`)) return;
    
    try {
      setLinks(links.filter(l => l.short_code !== shortCode));
      showToast('Link deleted successfully', 'success');
    } catch (error) {
      showToast('Failed to delete link', 'error');
    }
  };

  const filteredLinks = links.filter(link => 
    link.short_code.toLowerCase().includes(searchTerm.toLowerCase()) ||
    link.long_url.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="py-8 space-y-8 animate-fade-in">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Dashboard</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">Manage and track your shortened links.</p>
        </div>
        <button 
          onClick={() => setShowForm(!showForm)}
          className="btn-primary flex items-center space-x-2"
        >
          {showForm ? <X className="w-4 h-4" /> : <Plus className="w-4 h-4" />}
          <span>{showForm ? 'Cancel' : 'Shorten New URL'}</span>
        </button>
      </div>

      {showForm && (
        <div className="animate-slide-up">
          <ShortenForm onSuccess={() => {
            fetchLinks();
            setShowForm(false);
          }} />
        </div>
      )}

      <div className="card p-4">
        <div className="relative mb-4">
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <Search className="h-5 w-5 text-gray-400" />
          </div>
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder="Search by short code or URL..."
            className="input-field pl-10"
          />
        </div>

        {loading ? (
          <Spinner size="lg" />
        ) : (
          <LinkTable links={filteredLinks} onDelete={handleDelete} />
        )}
      </div>
    </div>
  );
}
