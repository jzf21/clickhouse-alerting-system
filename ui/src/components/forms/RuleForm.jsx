import { useState, useEffect } from 'react';
import { api } from '../../api';
import { useConnection } from '../../context/ConnectionContext';

export default function RuleForm({ rule, defaultConnectionId, onSave, onCancel }) {
  const { connections } = useConnection();
  const isEdit = !!rule;

  const [name, setName] = useState('');
  const [query, setQuery] = useState('');
  const [column, setColumn] = useState('');
  const [operator, setOperator] = useState('gt');
  const [threshold, setThreshold] = useState('');
  const [severity, setSeverity] = useState('warning');
  const [evalInterval, setEvalInterval] = useState(60);
  const [forDuration, setForDuration] = useState(0);
  const [labels, setLabels] = useState('{}');
  const [connectionId, setConnectionId] = useState('');
  const [channelIds, setChannelIds] = useState([]);
  const [channels, setChannels] = useState([]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    api('/api/channels').then(setChannels).catch(() => {});
  }, []);

  useEffect(() => {
    if (isEdit) {
      setName(rule.name || '');
      setQuery(rule.query || '');
      setColumn(rule.column || '');
      setOperator(rule.operator || 'gt');
      setThreshold(rule.threshold ?? '');
      setSeverity(rule.severity || 'warning');
      setEvalInterval(rule.eval_interval || 60);
      setForDuration(rule.for_duration || 0);
      setLabels(JSON.stringify(rule.labels || {}, null, 2));
      setConnectionId(rule.connection_id || '');
      setChannelIds(rule.channel_ids || []);
    } else {
      setConnectionId(defaultConnectionId || '');
    }
  }, [rule, isEdit, defaultConnectionId]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setSaving(true);
    try {
      const body = {
        name, query, column, operator,
        threshold: parseFloat(threshold),
        severity,
        eval_interval: parseInt(evalInterval),
        for_duration: parseInt(forDuration),
        labels: JSON.parse(labels),
        channel_ids: channelIds,
        connection_id: connectionId,
        enabled: true,
      };

      if (isEdit) {
        await api(`/api/rules/${rule.id}`, { method: 'PUT', body: JSON.stringify(body) });
      } else {
        await api('/api/rules', { method: 'POST', body: JSON.stringify(body) });
      }
      onSave();
    } catch (err) {
      setError(err.message || 'Failed to save rule');
    } finally {
      setSaving(false);
    }
  };

  const toggleChannel = (id) => {
    setChannelIds(prev => prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]);
  };

  return (
    <form onSubmit={handleSubmit}>
      {error && <div style={{ color: 'var(--danger)', marginBottom: '1rem', fontSize: '0.875rem' }}>{error}</div>}
      <div className="form-group">
        <label>Name</label>
        <input value={name} onChange={e => setName(e.target.value)} required />
      </div>
      <div className="form-group">
        <label>ClickHouse Connection</label>
        <select value={connectionId} onChange={e => setConnectionId(e.target.value)}>
          <option value="">-- Select Connection --</option>
          {connections.filter(c => c.enabled).map(c => (
            <option key={c.id} value={c.id}>{c.name} ({c.host}:{c.port}/{c.database})</option>
          ))}
        </select>
      </div>
      <div className="form-group">
        <label>Query (SQL)</label>
        <textarea value={query} onChange={e => setQuery(e.target.value)} required />
      </div>
      <div className="form-row">
        <div className="form-group">
          <label>Column</label>
          <input value={column} onChange={e => setColumn(e.target.value)} required />
        </div>
        <div className="form-group">
          <label>Operator</label>
          <select value={operator} onChange={e => setOperator(e.target.value)}>
            {['gt', 'gte', 'lt', 'lte', 'eq', 'neq'].map(o => (
              <option key={o} value={o}>{{ gt: '>', gte: '>=', lt: '<', lte: '<=', eq: '==', neq: '!=' }[o]}</option>
            ))}
          </select>
        </div>
      </div>
      <div className="form-row">
        <div className="form-group">
          <label>Threshold</label>
          <input type="number" step="any" value={threshold} onChange={e => setThreshold(e.target.value)} required />
        </div>
        <div className="form-group">
          <label>Severity</label>
          <select value={severity} onChange={e => setSeverity(e.target.value)}>
            <option value="warning">warning</option>
            <option value="critical">critical</option>
            <option value="info">info</option>
          </select>
        </div>
      </div>
      <div className="form-row">
        <div className="form-group">
          <label>Eval Interval (seconds)</label>
          <input type="number" value={evalInterval} onChange={e => setEvalInterval(e.target.value)} />
        </div>
        <div className="form-group">
          <label>For Duration (seconds)</label>
          <input type="number" value={forDuration} onChange={e => setForDuration(e.target.value)} />
        </div>
      </div>
      <div className="form-group">
        <label>Labels (JSON)</label>
        <textarea value={labels} onChange={e => setLabels(e.target.value)} />
      </div>
      <div className="form-group">
        <label>Channels</label>
        <div className="channel-checklist">
          {channels.length ? channels.map(c => (
            <label key={c.id} className="channel-option">
              <input type="checkbox" checked={channelIds.includes(c.id)} onChange={() => toggleChannel(c.id)} />
              {c.name} <span className="channel-type">({c.type})</span>
            </label>
          )) : <div className="text-muted">No channels configured yet</div>}
        </div>
      </div>
      <div className="form-actions">
        <button type="submit" className="btn btn-primary" disabled={saving}>{saving ? 'Saving...' : (isEdit ? 'Update' : 'Create')}</button>
        <button type="button" className="btn" onClick={onCancel}>Cancel</button>
      </div>
    </form>
  );
}
