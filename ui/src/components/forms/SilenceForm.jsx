import { useState } from 'react';
import { api } from '../../api';
import { useConnection } from '../../context/ConnectionContext';

export default function SilenceForm({ defaultMatchers, defaultConnectionId, onSave, onCancel }) {
  const { connections } = useConnection();
  const defaultEnd = new Date(Date.now() + 2 * 3600 * 1000).toISOString().slice(0, 16);

  const [matchers, setMatchers] = useState(
    JSON.stringify(defaultMatchers || [{ label: '', value: '' }], null, 2)
  );
  const [comment, setComment] = useState(defaultMatchers ? 'Silenced from dashboard' : '');
  const [createdBy, setCreatedBy] = useState('');
  const [endsAt, setEndsAt] = useState(defaultEnd);
  const [connectionId, setConnectionId] = useState(defaultConnectionId || '');

  const handleSubmit = async (e) => {
    e.preventDefault();
    const body = {
      matchers: JSON.parse(matchers),
      comment,
      created_by: createdBy,
      ends_at: new Date(endsAt).toISOString(),
      connection_id: connectionId || null,
    };
    await api('/api/silences', { method: 'POST', body: JSON.stringify(body) });
    onSave();
  };

  return (
    <form onSubmit={handleSubmit}>
      <div className="form-group">
        <label>Matchers (JSON array)</label>
        <textarea value={matchers} onChange={e => setMatchers(e.target.value)} />
      </div>
      <div className="form-group">
        <label>Comment</label>
        <input value={comment} onChange={e => setComment(e.target.value)} />
      </div>
      <div className="form-group">
        <label>Created By</label>
        <input value={createdBy} onChange={e => setCreatedBy(e.target.value)} />
      </div>
      <div className="form-group">
        <label>Ends At</label>
        <input type="datetime-local" value={endsAt} onChange={e => setEndsAt(e.target.value)} required />
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
      <div className="form-actions">
        <button type="submit" className="btn btn-primary">Create</button>
        <button type="button" className="btn" onClick={onCancel}>Cancel</button>
      </div>
    </form>
  );
}
