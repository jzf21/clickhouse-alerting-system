import { useState, useEffect, useCallback } from 'react';
import { api, withConnection } from '../api';
import { useConnection } from '../context/ConnectionContext';

const LIMIT = 20;

export default function HistoryTab() {
  const { selectedConnectionId } = useConnection();
  const [events, setEvents] = useState([]);
  const [offset, setOffset] = useState(0);

  const load = useCallback(async () => {
    try {
      let path = `/api/alerts/history?limit=${LIMIT}&offset=${offset}`;
      path = withConnection(path, selectedConnectionId);
      const data = await api(path);
      setEvents(data || []);
    } catch (e) {
      console.error('Failed to load history:', e);
    }
  }, [selectedConnectionId, offset]);

  useEffect(() => { setOffset(0); }, [selectedConnectionId]);
  useEffect(() => { load(); }, [load]);

  return (
    <div className="card">
      <h2>Alert History</h2>
      {!events.length ? (
        <div className="empty">No alert history</div>
      ) : (
        <>
          <table>
            <thead>
              <tr><th>Time</th><th>Rule</th><th>State</th><th>Severity</th><th>Value</th></tr>
            </thead>
            <tbody>
              {events.map(e => (
                <tr key={e.id}>
                  <td>{new Date(e.created_at).toLocaleString()}</td>
                  <td>{e.rule_name}</td>
                  <td><span className={`badge badge-${e.state}`}>{e.state}</span></td>
                  <td><span className={`badge badge-${e.severity}`}>{e.severity}</span></td>
                  <td>{Number(e.value).toFixed(2)}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="pagination">
            {offset > 0 && (
              <button className="btn btn-sm btn-primary" onClick={() => setOffset(o => o - LIMIT)}>Previous</button>
            )}
            {events.length === LIMIT && (
              <button className="btn btn-sm btn-primary" onClick={() => setOffset(o => o + LIMIT)}>Next</button>
            )}
          </div>
        </>
      )}
    </div>
  );
}
