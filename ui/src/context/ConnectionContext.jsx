import { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { api } from '../api';

const ConnectionContext = createContext(null);

export function ConnectionProvider({ children }) {
  const [connections, setConnections] = useState([]);
  const [selectedConnectionId, setSelectedConnectionId] = useState(
    () => localStorage.getItem('selectedConnectionId') || ''
  );

  const refreshConnections = useCallback(async () => {
    try {
      const conns = await api('/api/connections');
      setConnections(conns || []);
    } catch (e) {
      console.error('Failed to load connections:', e);
    }
  }, []);

  useEffect(() => {
    refreshConnections();
  }, [refreshConnections]);

  useEffect(() => {
    localStorage.setItem('selectedConnectionId', selectedConnectionId);
  }, [selectedConnectionId]);

  const selectedConnection = connections.find(c => c.id === selectedConnectionId) || null;

  return (
    <ConnectionContext.Provider value={{
      connections,
      selectedConnectionId,
      selectedConnection,
      setSelectedConnectionId,
      refreshConnections,
    }}>
      {children}
    </ConnectionContext.Provider>
  );
}

export function useConnection() {
  const ctx = useContext(ConnectionContext);
  if (!ctx) throw new Error('useConnection must be used within ConnectionProvider');
  return ctx;
}
