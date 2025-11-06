import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { instancesAPI } from '../api';
import './Dashboard.css';

function Dashboard({ onLogout }) {
  const navigate = useNavigate();
  const [instances, setInstances] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newInstanceName, setNewInstanceName] = useState('');
  const [creating, setCreating] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState(null);

  const loadInstances = async () => {
    try {
      const response = await instancesAPI.list();
      setInstances(response.data.instances || []);
    } catch (err) {
      setError('Failed to load instances');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadInstances();
    // Refresh instances every 10 seconds
    const interval = setInterval(loadInstances, 10000);
    return () => clearInterval(interval);
  }, []);

  const handleCreateInstance = async (e) => {
    e.preventDefault();
    setCreating(true);
    setError('');

    try {
      await instancesAPI.create(newInstanceName);
      setShowCreateModal(false);
      setNewInstanceName('');
      loadInstances();
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to create instance');
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteInstance = async (name) => {
    try {
      await instancesAPI.delete(name);
      setDeleteConfirm(null);
      loadInstances();
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to delete instance');
    }
  };

  const getStatusBadge = (status) => {
    const classes = {
      RUNNING: 'status-running',
      PROVISIONING: 'status-provisioning',
      DELETING: 'status-deleting',
      FAILED: 'status-failed',
    };
    return <span className={`status-badge ${classes[status]}`}>{status}</span>;
  };

  return (
    <div className="dashboard">
      <header className="dashboard-header">
        <div>
          <h1>SupaControl</h1>
          <p>Manage your Supabase instances</p>
        </div>
        <div className="header-actions">
          <button onClick={() => navigate('/settings')} className="btn-secondary">
            Settings
          </button>
          <button onClick={onLogout} className="btn-secondary">
            Logout
          </button>
        </div>
      </header>

      <main className="dashboard-main">
        {error && <div className="error-banner">{error}</div>}

        <div className="instances-header">
          <h2>Instances ({instances.length})</h2>
          <button onClick={() => setShowCreateModal(true)} className="btn-primary">
            + Create Instance
          </button>
        </div>

        {loading ? (
          <div className="loading">Loading instances...</div>
        ) : instances.length === 0 ? (
          <div className="empty-state">
            <p>No instances yet. Create your first Supabase instance!</p>
          </div>
        ) : (
          <table className="instances-table">
            <thead>
              <tr>
                <th>Project Name</th>
                <th>Status</th>
                <th>Studio URL</th>
                <th>API URL</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {instances.map((instance) => (
                <tr key={instance.id}>
                  <td>{instance.project_name}</td>
                  <td>{getStatusBadge(instance.status)}</td>
                  <td>
                    {instance.studio_url ? (
                      <a href={instance.studio_url} target="_blank" rel="noopener noreferrer">
                        Open Studio
                      </a>
                    ) : (
                      '-'
                    )}
                  </td>
                  <td>
                    {instance.api_url ? (
                      <a href={instance.api_url} target="_blank" rel="noopener noreferrer">
                        View API
                      </a>
                    ) : (
                      '-'
                    )}
                  </td>
                  <td>{new Date(instance.created_at).toLocaleDateString()}</td>
                  <td>
                    {instance.status !== 'DELETING' && (
                      <button
                        onClick={() => setDeleteConfirm(instance.project_name)}
                        className="btn-danger"
                        disabled={instance.status === 'PROVISIONING'}
                      >
                        Delete
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </main>

      {/* Create Instance Modal */}
      {showCreateModal && (
        <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h2>Create New Instance</h2>
            <form onSubmit={handleCreateInstance}>
              <div className="form-group">
                <label htmlFor="instanceName">Project Name</label>
                <input
                  id="instanceName"
                  type="text"
                  value={newInstanceName}
                  onChange={(e) => setNewInstanceName(e.target.value)}
                  placeholder="my-app"
                  pattern="[a-z0-9-]+"
                  title="Only lowercase letters, numbers, and hyphens"
                  required
                  autoFocus
                />
                <small>Only lowercase letters, numbers, and hyphens (3-63 chars)</small>
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
                  {creating ? 'Creating...' : 'Create'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {deleteConfirm && (
        <div className="modal-overlay" onClick={() => setDeleteConfirm(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h2>Confirm Deletion</h2>
            <p>
              Are you sure you want to delete <strong>{deleteConfirm}</strong>?
            </p>
            <p className="warning">This action cannot be undone. All data will be permanently lost.</p>
            <div className="modal-actions">
              <button onClick={() => setDeleteConfirm(null)} className="btn-secondary">
                Cancel
              </button>
              <button onClick={() => handleDeleteInstance(deleteConfirm)} className="btn-danger">
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default Dashboard;
