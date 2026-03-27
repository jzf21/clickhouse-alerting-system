import { useState, useEffect, useCallback } from 'react';
import { api, withConnection } from '../api';
import { useConnection } from '../context/ConnectionContext';
import Modal from './Modal';
import ChannelForm from './forms/ChannelForm';

export default function ChannelsTab() {
  const { selectedConnectionId, connections } = useConnection();
  const [channels, setChannels] = useState([]);
  const [editingChannel, setEditingChannel] = useState(undefined);

  const connMap = {};
  connections.forEach(c => { connMap[c.id] = c.name; });

  const load = useCallback(async () => {
    try {
      const data = await api(withConnection('/api/channels', selectedConnectionId));
      setChannels(data || []);
    } catch (e) {
      console.error('Failed to load channels:', e);
    }
  }, [selectedConnectionId]);

  useEffect(() => { load(); }, [load]);

  const handleEdit = async (id) => {
    const c = await api(`/api/channels/${id}`);
    setEditingChannel(c);
  };

  const deleteChannel = async (id) => {
    if (!window.confirm('Delete this channel?')) return;
    await api(`/api/channels/${id}`, { method: 'DELETE' });
    load();
  };

  const testChannel = async (id) => {
    try {
      await api(`/api/channels/${id}/test`, { method: 'POST' });
      window.alert('Test notification sent!');
    } catch (e) {
      window.alert('Test failed: ' + e.message);
    }
  };

  return (
    <div className="card">
      <div className="card-header">
        <h2>Notification Channels</h2>
        <button className="btn btn-primary" onClick={() => setEditingChannel(null)}>New Channel</button>
      </div>

      {!channels.length ? (
        <div className="empty">No channels configured</div>
      ) : (
        <table>
          <thead>
            <tr><th>Name</th><th>Type</th><th>Scope</th><th>Enabled</th><th>Actions</th></tr>
          </thead>
          <tbody>
            {channels.map(c => (
              <tr key={c.id}>
                <td>{c.name}</td>
                <td>{c.type}</td>
                <td>{c.connection_id ? (connMap[c.connection_id] || 'Unknown') : <span className="text-muted">Global</span>}</td>
                <td>{c.enabled ? 'Yes' : 'No'}</td>
                <td className="actions">
                  <button className="btn btn-sm btn-success" onClick={() => testChannel(c.id)}>Test</button>
                  <button className="btn btn-sm btn-primary" onClick={() => handleEdit(c.id)}>Edit</button>
                  <button className="btn btn-sm btn-danger" onClick={() => deleteChannel(c.id)}>Delete</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <Modal title={editingChannel ? 'Edit Channel' : 'New Channel'} isOpen={editingChannel !== undefined} onClose={() => setEditingChannel(undefined)}>
        <ChannelForm
          channel={editingChannel}
          defaultConnectionId={selectedConnectionId}
          onSave={() => { setEditingChannel(undefined); load(); }}
          onCancel={() => setEditingChannel(undefined)}
        />
      </Modal>
    </div>
  );
}
