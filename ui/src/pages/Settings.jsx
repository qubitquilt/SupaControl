import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { authAPI } from '../api';
import './Settings.css';

function Settings({ onLogout }) {
  const navigate = useNavigate();
  const [apiKeys, setApiKeys] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [creating, setCreating] = useState(false);
  const [newKey, setNewKey] = useState(null);

  const loadAPIKeys = async () => {
    try {
      const response = await authAPI.listAPIKeys();
      setApiKeys(response.data.api_keys || []);
    } catch (err) {
      setError('Failed to load API keys');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadAPIKeys();
  }, []);

  const handleCreateAPIKey = async (e) => {
    e.preventDefault();
    setCreating(true);
    setError('');

    try {
      const response = await authAPI.createAPIKey(newKeyName);
      setNewKey(response.data.key);
      setNewKeyName('');
      loadAPIKeys();
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to create API key');
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteAPIKey = async (id) => {
    if (!confirm('Are you sure you want to delete this API key?')) return;

    try {
      await authAPI.deleteAPIKey(id);
      setSuccess('API key deleted successfully');
      loadAPIKeys();
      setTimeout(() => setSuccess(''), 3000);
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to delete API key');
    }
  };

  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text);
    setSuccess('Copied to clipboard!');
    setTimeout(() => setSuccess(''), 2000);
  };

  return (
    <div className="settings">
      <header className="settings-header">
        <div>
          <h1>Settings</h1>
          <p>Manage API keys and configuration</p>
        </div>
        <div className="header-actions">
          <button onClick={() => navigate('/')} className="btn-secondary">
            Back to Dashboard
          </button>
          <button onClick={onLogout} className="btn-secondary">
            Logout
          </button>
        </div>
      </header>

      <main className="settings-main">
        {error && <div className="error-banner">{error}</div>}
        {success && <div className="success-banner">{success}</div>}

        <section className="settings-section">
          <div className="section-header">
            <div>
              <h2>API Keys</h2>
              <p>Manage API keys for supactl CLI and programmatic access</p>
            </div>
            <button onClick={() => setShowCreateModal(true)} className="btn-primary">
              + Generate API Key
            </button>
          </div>

          {loading ? (
            <div className="loading">Loading API keys...</div>
          ) : apiKeys.length === 0 ? (
            <div className="empty-state">
              <p>No API keys yet. Generate one to use with supactl.</p>
            </div>
          ) : (
            <table className="api-keys-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Created</th>
                  <th>Last Used</th>
                  <th>Expires</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {apiKeys.map((key) => (
                  <tr key={key.id}>
                    <td>{key.name}</td>
                    <td>{new Date(key.created_at).toLocaleDateString()}</td>
                    <td>
                      {key.last_used
                        ? new Date(key.last_used).toLocaleDateString()
                        : 'Never'}
                    </td>
                    <td>
                      {key.expires_at
                        ? new Date(key.expires_at).toLocaleDateString()
                        : 'Never'}
                    </td>
                    <td>
                      <button
                        onClick={() => handleDeleteAPIKey(key.id)}
                        className="btn-danger"
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </main>

      {/* Create API Key Modal */}
      {showCreateModal && !newKey && (
        <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h2>Generate API Key</h2>
            <form onSubmit={handleCreateAPIKey}>
              <div className="form-group">
                <label htmlFor="keyName">Key Name</label>
                <input
                  id="keyName"
                  type="text"
                  value={newKeyName}
                  onChange={(e) => setNewKeyName(e.target.value)}
                  placeholder="Development Key"
                  required
                  autoFocus
                />
                <small>A descriptive name to identify this key</small>
              </div>
              <div className="modal-actions">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="btn-secondary"
                  disabled={creating}
                >
                  Cancel
                </button>
                <button type="submit" className="btn-primary" disabled={creating}>
                  {creating ? 'Generating...' : 'Generate'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Show New Key Modal */}
      {newKey && (
        <div className="modal-overlay">
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h2>API Key Generated</h2>
            <p className="warning">
              Save this API key now. You will not be able to see it again!
            </p>
            <div className="key-display">
              <code>{newKey}</code>
              <button onClick={() => copyToClipboard(newKey)} className="btn-secondary">
                Copy
              </button>
            </div>
            <div className="modal-actions">
              <button
                onClick={() => {
                  setNewKey(null);
                  setShowCreateModal(false);
                }}
                className="btn-primary"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default Settings;
