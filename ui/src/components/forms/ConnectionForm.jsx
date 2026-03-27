import { useState, useEffect } from 'react';
import { api } from '../../api';

export default function ConnectionForm({ connection, onSave, onCancel }) {
  const isEdit = !!connection;

  const [name, setName] = useState('');
  const [host, setHost] = useState('');
  const [port, setPort] = useState(9000);
  const [database, setDatabase] = useState('default');
  const [username, setUsername] = useState('default');
  const [password, setPassword] = useState('');
  const [secure, setSecure] = useState(false);
  const [maxOpenConns, setMaxOpenConns] = useState(5);
  const [enabled, setEnabled] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isEdit) {
      setName(connection.name || '');
      setHost(connection.host || '');
      setPort(connection.port || 9000);
      setDatabase(connection.database || 'default');
      setUsername(connection.username || 'default');
      setPassword(connection.password || '');
      setSecure(connection.secure || false);
      setMaxOpenConns(connection.max_open_conns || 5);
      setEnabled(connection.enabled !== false);
    }
  }, [connection, isEdit]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setSaving(true);
    try {
      const body = {
        name, host, port: parseInt(port), database, username, password,
        secure, max_open_conns: parseInt(maxOpenConns), enabled,
      };

      if (isEdit) {
        await api(`/api/connections/${connection.id}`, { method: 'PUT', body: JSON.stringify(body) });
      } else {
        await api('/api/connections', { method: 'POST', body: JSON.stringify(body) });
      }
      onSave();
    } catch (err) {
      setError(err.message || 'Failed to save connection');
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      {error && <div style={{ color: 'var(--danger)', marginBottom: '1rem', fontSize: '0.875rem' }}>{error}</div>}
      <div className="form-group">
        <label>Name</label>
        <input value={name} onChange={e => setName(e.target.value)} required />
      </div>
      <div className="form-row">
        <div className="form-group">
          <label>Host</label>
          <input value={host} onChange={e => setHost(e.target.value)} required />
        </div>
        <div className="form-group">
          <label>Port</label>
          <input type="number" value={port} onChange={e => setPort(e.target.value)} />
        </div>
      </div>
      <div className="form-row">
        <div className="form-group">
          <label>Database</label>
          <input value={database} onChange={e => setDatabase(e.target.value)} />
        </div>
        <div className="form-group">
          <label>Username</label>
          <input value={username} onChange={e => setUsername(e.target.value)} />
        </div>
      </div>
      <div className="form-group">
        <label>Password</label>
        <input type="password" value={password} onChange={e => setPassword(e.target.value)} />
      </div>
      <div className="form-row">
        <div className="form-group">
          <label><input type="checkbox" checked={secure} onChange={e => setSecure(e.target.checked)} /> Secure (TLS)</label>
        </div>
        <div className="form-group">
          <label>Max Open Conns</label>
          <input type="number" value={maxOpenConns} onChange={e => setMaxOpenConns(e.target.value)} />
        </div>
      </div>
      <div className="form-group">
        <label><input type="checkbox" checked={enabled} onChange={e => setEnabled(e.target.checked)} /> Enabled</label>
      </div>
      <div className="form-actions">
        <button type="submit" className="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : (isEdit ? 'Update' : 'Create')}</button>
        <button type="button" className="btn" onClick={onCancel}>Cancel</button>
      </div>
    </form>
  );
}
