import { useState, useEffect } from 'react';
import { api } from '../../api';
import { useConnection } from '../../context/ConnectionContext';

export default function ChannelForm({ channel, defaultConnectionId, onSave, onCancel }) {
  const { connections } = useConnection();
  const isEdit = !!channel;

  const [name, setName] = useState('');
  const [type, setType] = useState('slack');
  const [config, setConfig] = useState('{"webhook_url":""}');
  const [enabled, setEnabled] = useState(true);
  const [connectionId, setConnectionId] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isEdit) {
      setName(channel.name || '');
      setType(channel.type || 'slack');
      setConfig(JSON.stringify(typeof channel.config === 'string' ? JSON.parse(channel.config) : (channel.config || {}), null, 2));
      setEnabled(channel.enabled !== false);
      setConnectionId(channel.connection_id || '');
    } else {
      setConnectionId(defaultConnectionId || '');
    }
  }, [channel, isEdit, defaultConnectionId]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setSaving(true);
    try {
      const body = {
        name, type,
        config: JSON.parse(config),
        enabled,
        connection_id: connectionId || null,
      };

      if (isEdit) {
        await api(`/api/channels/${channel.id}`, { method: 'PUT', body: JSON.stringify(body) });
      } else {
        await api('/api/channels', { method: 'POST', body: JSON.stringify(body) });
      }
      onSave();
    } catch (err) {
      setError(err.message || 'Failed to save channel');
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <div className="form-group">
        <label>Name</label>
        <input value={name} onChange={e => setName(e.target.value)} required />
      </div>
      <div className="form-group">
        <label>Type</label>
        <select value={type} onChange={e => setType(e.target.value)}>
          <option value="slack">Slack</option>
          <option value="webhook">Webhook</option>
        </select>
      </div>
      <div className="form-group">
        <label>Config (JSON)</label>
        <textarea value={config} onChange={e => setConfig(e.target.value)} />
      </div>
      <div className="form-group">
        <label>Scope</label>
        <select value={connectionId} onChange={e => setConnectionId(e.target.value)}>
          <option value="">Global (all connections)</option>
          {connections.map(c => (
            <option key={c.id} value={c.id}>{c.name} ({c.host}:{c.port}/{c.database})</option>
          ))}
        </select>
      </div>
      <div className="form-group">
        <label><input type="checkbox" checked={enabled} onChange={e => setEnabled(e.target.checked)} /> Enabled</label>
      </div>
      <div className="form-actions">
        <button type="submit" className="btn btn-primary">{isEdit ? 'Update' : 'Create'}</button>
        <button type="button" className="btn" onClick={onCancel}>Cancel</button>
      </div>
    </form>
  );
}
