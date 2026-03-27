import { useState, useEffect, useCallback } from 'react';
import { api, withConnection, timeAgo } from '../api';
import { useConnection } from '../context/ConnectionContext';
import Modal from './Modal';
import SilenceForm from './forms/SilenceForm';

export default function AlertsTab() {
  const { selectedConnectionId } = useConnection();
  const [alerts, setAlerts] = useState([]);
  const [silenceTarget, setSilenceTarget] = useState(null);

  const load = useCallback(async () => {
    try {
      const data = await api(withConnection('/api/alerts', selectedConnectionId));
      setAlerts(data || []);
    } catch (e) {
      console.error('Failed to load alerts:', e);
    }
  }, [selectedConnectionId]);

  useEffect(() => {
    load();
    const interval = setInterval(load, 30000);
    return () => clearInterval(interval);
  }, [load]);

  const handleSilenceCreated = () => {
    setSilenceTarget(null);
    load();
  };

  if (!alerts.length) {
    return <div className="card"><h2>Active Alerts</h2><div className="empty">No active alerts</div></div>;
  }

  return (
    <div className="card">
      <h2>Active Alerts</h2>
      <table>
        <thead>
          <tr><th>Status</th><th>Rule</th><th>Severity</th><th>Value</th><th>Since</th><th>Actions</th></tr>
        </thead>
        <tbody>
          {alerts.map(a => (
            <tr key={a.rule_id}>
              <td><span className={`badge badge-${a.state}`}>{a.state}</span></td>
              <td>{a.rule_name}</td>
              <td><span className={`badge badge-${a.severity}`}>{a.severity}</span></td>
              <td>{a.last_eval_value != null ? Number(a.last_eval_value).toFixed(2) : '-'}</td>
              <td>{a.firing_since ? timeAgo(a.firing_since) : a.pending_since ? timeAgo(a.pending_since) : '-'}</td>
              <td className="actions">
                {(a.state === 'firing' || a.state === 'pending') && (
                  <button className="btn btn-sm btn-danger" onClick={() => setSilenceTarget(a)}>Silence</button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <Modal title="Create Silence" isOpen={!!silenceTarget} onClose={() => setSilenceTarget(null)}>
        {silenceTarget && (
          <SilenceForm
            defaultMatchers={[{ label: 'alertname', value: silenceTarget.rule_name }]}
            defaultConnectionId={silenceTarget.connection_id || selectedConnectionId}
            onSave={handleSilenceCreated}
            onCancel={() => setSilenceTarget(null)}
          />
        )}
      </Modal>
    </div>
  );
}
