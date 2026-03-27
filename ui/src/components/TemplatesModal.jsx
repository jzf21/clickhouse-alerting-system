import { useState, useEffect } from 'react';
import { api, opSym } from '../api';
import { useConnection } from '../context/ConnectionContext';
import Modal from './Modal';

export default function TemplatesModal({ isOpen, onClose, onApplied }) {
  const { selectedConnectionId, connections } = useConnection();
  const [templates, setTemplates] = useState([]);
  const [selected, setSelected] = useState([]);
  const [applying, setApplying] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!isOpen) return;
    api('/api/rule-templates').then(data => {
      setTemplates(data || []);
      setSelected([]);
      setError('');
    }).catch(() => {});
  }, [isOpen]);

  const categories = [...new Set(templates.map(t => t.category))];

  const toggle = (id) => {
    setSelected(prev => prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]);
  };

  const selectCategory = (cat) => {
    const ids = templates.filter(t => t.category === cat).map(t => t.id);
    const allSelected = ids.every(id => selected.includes(id));
    if (allSelected) {
      setSelected(prev => prev.filter(id => !ids.includes(id)));
    } else {
      setSelected(prev => [...new Set([...prev, ...ids])]);
    }
  };

  const selectAll = () => {
    if (selected.length === templates.length) {
      setSelected([]);
    } else {
      setSelected(templates.map(t => t.id));
    }
  };

  const connName = connections.find(c => c.id === selectedConnectionId)?.name || 'selected connection';

  const handleApply = async () => {
    if (!selectedConnectionId) {
      setError('Please select a connection first');
      return;
    }
    if (!selected.length) {
      setError('Select at least one template');
      return;
    }
    setApplying(true);
    setError('');
    try {
      await api('/api/rule-templates/apply', {
        method: 'POST',
        body: JSON.stringify({
          connection_id: selectedConnectionId,
          template_ids: selected,
        }),
      });
      onApplied();
    } catch (err) {
      setError(err.message || 'Failed to apply templates');
    } finally {
      setApplying(false);
    }
  };

  return (
    <Modal title="Rule Templates" isOpen={isOpen} onClose={onClose}>
      <div style={{ marginBottom: '1rem' }}>
        <p style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', margin: '0 0 0.75rem' }}>
          Select templates to create as rules for <strong>{connName}</strong>.
          Duplicates (same name + connection) will be skipped.
        </p>
        {error && <div style={{ color: 'var(--danger)', marginBottom: '0.75rem', fontSize: '0.875rem' }}>{error}</div>}

        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem' }}>
          <button className="btn btn-sm" onClick={selectAll}>
            {selected.length === templates.length ? 'Deselect All' : 'Select All'}
          </button>
          <span style={{ fontSize: '0.875rem', color: 'var(--text-secondary)' }}>
            {selected.length} of {templates.length} selected
          </span>
        </div>
      </div>

      <div style={{ maxHeight: '400px', overflowY: 'auto' }}>
        {categories.map(cat => {
          const catTemplates = templates.filter(t => t.category === cat);
          const allCatSelected = catTemplates.every(t => selected.includes(t.id));
          return (
            <div key={cat} style={{ marginBottom: '1rem' }}>
              <div
                onClick={() => selectCategory(cat)}
                style={{
                  display: 'flex', alignItems: 'center', gap: '0.5rem',
                  cursor: 'pointer', fontWeight: 600, fontSize: '0.9rem',
                  padding: '0.35rem 0', borderBottom: '1px solid var(--border)',
                  marginBottom: '0.5rem', color: 'var(--text-primary)',
                }}
              >
                <input type="checkbox" checked={allCatSelected} readOnly style={{ cursor: 'pointer' }} />
                {cat}
                <span style={{ fontWeight: 400, color: 'var(--text-secondary)', fontSize: '0.8rem' }}>
                  ({catTemplates.length})
                </span>
              </div>
              {catTemplates.map(t => (
                <label
                  key={t.id}
                  style={{
                    display: 'flex', alignItems: 'flex-start', gap: '0.5rem',
                    padding: '0.35rem 0 0.35rem 1.25rem', cursor: 'pointer',
                  }}
                >
                  <input
                    type="checkbox"
                    checked={selected.includes(t.id)}
                    onChange={() => toggle(t.id)}
                    style={{ marginTop: '0.2rem', cursor: 'pointer' }}
                  />
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                      <span>{t.name}</span>
                      <span className={`badge badge-${t.severity}`} style={{ fontSize: '0.7rem' }}>{t.severity}</span>
                    </div>
                    <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>{t.description}</div>
                    <code style={{ fontSize: '0.75rem' }}>{t.column} {opSym(t.operator)} {t.threshold}</code>
                  </div>
                </label>
              ))}
            </div>
          );
        })}
      </div>

      <div className="form-actions" style={{ marginTop: '1rem' }}>
        <button
          className="btn btn-primary"
          onClick={handleApply}
          disabled={applying || !selected.length || !selectedConnectionId}
        >
          {applying ? 'Applying...' : `Apply ${selected.length} Template${selected.length !== 1 ? 's' : ''}`}
        </button>
        <button className="btn" onClick={onClose}>Cancel</button>
      </div>
    </Modal>
  );
}
