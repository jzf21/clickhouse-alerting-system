import { useState, useEffect, useCallback } from 'react';
import { api } from '../api';
import { useConnection } from '../context/ConnectionContext';
import Modal from './Modal';
import ConnectionForm from './forms/ConnectionForm';

export default function ConnectionsTab() {
  const { refreshConnections } = useConnection();
  const [connections, setConnections] = useState([]);
  const [editingConn, setEditingConn] = useState(undefined);

  const load = useCallback(async () => {
    try {
      const data = await api('/api/connections');
      setConnections(data || []);
    } catch (e) {
      console.error('Failed to load connections:', e);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleEdit = async (id) => {
    const c = await api(`/api/connections/${id}`);
    setEditingConn(c);
  };

  const deleteConnection = async (id) => {
    if (!window.confirm('Delete this connection?')) return;
    try {
      await api(`/api/connections/${id}`, { method: 'DELETE' });
      load();
      refreshConnections();
    } catch (e) {
      window.alert('Cannot delete: ' + e.message);
    }
  };

  const testConnection = async (id) => {
    try {
      await api(`/api/connections/${id}/test`, { method: 'POST' });
      window.alert('Connection test successful!');
    } catch (e) {
      window.alert('Test failed: ' + e.message);
    }
  };

  const handleSaved = () => {
    setEditingConn(undefined);
    load();
    refreshConnections();
  };

  return (
    <div className="card">
      <div className="card-header">
        <h2>ClickHouse Connections</h2>
        <button className="btn btn-primary" onClick={() => setEditingConn(null)}>New Connection</button>
      </div>

      {!connections.length ? (
        <div className="empty">No connections configured</div>
      ) : (
        <table>
          <thead>
            <tr><th>Name</th><th>Host</th><th>Port</th><th>Database</th><th>Enabled</th><th>Actions</th></tr>
          </thead>
          <tbody>
            {connections.map(c => (
              <tr key={c.id}>
                <td>{c.name}</td>
                <td>{c.host}</td>
                <td>{c.port}</td>
                <td>{c.database}</td>
                <td>{c.enabled ? 'Yes' : 'No'}</td>
                <td className="actions">
                  <button className="btn btn-sm btn-success" onClick={() => testConnection(c.id)}>Test</button>
                  <button className="btn btn-sm btn-primary" onClick={() => handleEdit(c.id)}>Edit</button>
                  <button className="btn btn-sm btn-danger" onClick={() => deleteConnection(c.id)}>Delete</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <Modal title={editingConn ? 'Edit Connection' : 'New Connection'} isOpen={editingConn !== undefined} onClose={() => setEditingConn(undefined)}>
        <ConnectionForm
          connection={editingConn}
          onSave={handleSaved}
          onCancel={() => setEditingConn(undefined)}
        />
      </Modal>
    </div>
  );
}
