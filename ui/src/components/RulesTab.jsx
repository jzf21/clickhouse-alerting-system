import { useState, useEffect, useCallback } from 'react';
import { api, withConnection, opSym } from '../api';
import { useConnection } from '../context/ConnectionContext';
import Modal from './Modal';
import RuleForm from './forms/RuleForm';
import TemplatesModal from './TemplatesModal';

export default function RulesTab() {
  const { selectedConnectionId, connections } = useConnection();
  const [rules, setRules] = useState([]);
  const [editingRule, setEditingRule] = useState(undefined); // undefined=closed, null=new, object=edit
  const [showTemplates, setShowTemplates] = useState(false);

  const connMap = {};
  connections.forEach(c => { connMap[c.id] = c.name; });

  const load = useCallback(async () => {
    try {
      const data = await api(withConnection('/api/rules', selectedConnectionId));
      setRules(data || []);
    } catch (e) {
      console.error('Failed to load rules:', e);
    }
  }, [selectedConnectionId]);

  useEffect(() => { load(); }, [load]);

  const toggleRule = async (id, enabled) => {
    await api(`/api/rules/${id}`, { method: 'PUT', body: JSON.stringify({ enabled }) });
    load();
  };

  const deleteRule = async (id) => {
    if (!window.confirm('Delete this rule?')) return;
    await api(`/api/rules/${id}`, { method: 'DELETE' });
    load();
  };

  const handleEdit = async (id) => {
    const r = await api(`/api/rules/${id}`);
    setEditingRule(r);
  };

  const handleSaved = () => {
    setEditingRule(undefined);
    load();
  };

  return (
    <div className="card">
      <div className="card-header">
        <h2>Alert Rules</h2>
        <div style={{ display: 'flex', gap: '0.5rem' }}>
          <button className="btn" onClick={() => setShowTemplates(true)}>Templates</button>
          <button className="btn btn-primary" onClick={() => setEditingRule(null)}>New Rule</button>
        </div>
      </div>

      {!rules.length ? (
        <div className="empty">No rules configured</div>
      ) : (
        <table>
          <thead>
            <tr><th>Name</th><th>Connection</th><th>Severity</th><th>Condition</th><th>Interval</th><th>Enabled</th><th>Actions</th></tr>
          </thead>
          <tbody>
            {rules.map(r => (
              <tr key={r.id}>
                <td>{r.name}</td>
                <td>{r.connection_id ? (connMap[r.connection_id] || 'Unknown') : <span className="text-muted">None</span>}</td>
                <td><span className={`badge badge-${r.severity}`}>{r.severity}</span></td>
                <td><code>{r.column} {opSym(r.operator)} {r.threshold}</code></td>
                <td>{r.eval_interval}s</td>
                <td><span className="toggle" onClick={() => toggleRule(r.id, !r.enabled)}>{r.enabled ? 'On' : 'Off'}</span></td>
                <td className="actions">
                  <button className="btn btn-sm btn-primary" onClick={() => handleEdit(r.id)}>Edit</button>
                  <button className="btn btn-sm btn-danger" onClick={() => deleteRule(r.id)}>Delete</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <Modal title={editingRule ? 'Edit Rule' : 'New Rule'} isOpen={editingRule !== undefined} onClose={() => setEditingRule(undefined)}>
        <RuleForm
          rule={editingRule}
          defaultConnectionId={selectedConnectionId}
          onSave={handleSaved}
          onCancel={() => setEditingRule(undefined)}
        />
      </Modal>

      <TemplatesModal
        isOpen={showTemplates}
        onClose={() => setShowTemplates(false)}
        onApplied={() => { setShowTemplates(false); load(); }}
      />
    </div>
  );
}
