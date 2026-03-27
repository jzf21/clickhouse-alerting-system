import { useConnection } from '../context/ConnectionContext';

export default function ConnectionSelector() {
  const { connections, selectedConnectionId, setSelectedConnectionId } = useConnection();

  return (
    <div className="connection-selector">
      <label>Profile:</label>
      <select
        value={selectedConnectionId}
        onChange={e => setSelectedConnectionId(e.target.value)}
      >
        <option value="">All Connections</option>
        {connections.map(c => (
          <option key={c.id} value={c.id}>
            {c.name} ({c.host}:{c.port}/{c.database})
          </option>
        ))}
      </select>
    </div>
  );
}
