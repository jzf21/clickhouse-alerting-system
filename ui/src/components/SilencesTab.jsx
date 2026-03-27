import { useState, useEffect, useCallback } from 'react';
import { api, withConnection } from '../api';
import { useConnection } from '../context/ConnectionContext';
import Modal from './Modal';
import SilenceForm from './forms/SilenceForm';

export default function SilencesTab() {
  const { selectedConnectionId } = useConnection();
  const [silences, setSilences] = useState([]);
  const [showForm, setShowForm] = useState(false);

  const load = useCallback(async () => {
    try {
      const data = await api(withConnection('/api/silences', selectedConnectionId));
      setSilences(data || []);
    } catch (e) {
      console.error('Failed to load silences:', e);
    }
  }, [selectedConnectionId]);

  useEffect(() => { load(); }, [load]);

  const deleteSilence = async (id) => {
    if (!window.confirm('Delete this silence?')) return;
    await api(`/api/silences/${id}`, { method: 'DELETE' });
    load();
  };

  const now = new Date();

  return (
    <div className="card">
      <div className="card-header">
        <h2>Silences</h2>
        <button className="btn btn-primary" onClick={() => setShowForm(true)}>New Silence</button>
      </div>

      {!silences.length ? (
        <div className="empty">No silences</div>
      ) : (
        <table>
          <thead>
            <tr><th>Matchers</th><th>Comment</th><th>Ends At</th><th>Status</th><th>Actions</th></tr>
          </thead>
          <tbody>
            {silences.map(s => {
              const active = new Date(s.starts_at) <= now && new Date(s.ends_at) > now;
              return (
                <tr key={s.id}>
                  <td><code>{typeof s.matchers === 'string' ? s.matchers : JSON.stringify(s.matchers)}</code></td>
                  <td>{s.comment}</td>
                  <td>{new Date(s.ends_at).toLocaleString()}</td>
                  <td><span className={`badge ${active ? 'badge-firing' : 'badge-inactive'}`}>{active ? 'active' : 'expired'}</span></td>
                  <td className="actions">
                    <button className="btn btn-sm btn-danger" onClick={() => deleteSilence(s.id)}>Delete</button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}

      <Modal title="New Silence" isOpen={showForm} onClose={() => setShowForm(false)}>
        <SilenceForm
          defaultConnectionId={selectedConnectionId}
          onSave={() => { setShowForm(false); load(); }}
          onCancel={() => setShowForm(false)}
        />
      </Modal>
    </div>
  );
}
