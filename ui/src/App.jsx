import { useState } from 'react';
import { ConnectionProvider } from './context/ConnectionContext';
import Layout from './components/Layout';
import AlertsTab from './components/AlertsTab';
import RulesTab from './components/RulesTab';
import HistoryTab from './components/HistoryTab';
import SilencesTab from './components/SilencesTab';
import ChannelsTab from './components/ChannelsTab';
import ConnectionsTab from './components/ConnectionsTab';

const TABS = [
  { id: 'alerts', label: 'Alerts' },
  { id: 'rules', label: 'Rules' },
  { id: 'history', label: 'History' },
  { id: 'silences', label: 'Silences' },
  { id: 'channels', label: 'Channels' },
  { id: 'connections', label: 'Connections' },
];

function TabContent({ activeTab }) {
  switch (activeTab) {
    case 'alerts': return <AlertsTab />;
    case 'rules': return <RulesTab />;
    case 'history': return <HistoryTab />;
    case 'silences': return <SilencesTab />;
    case 'channels': return <ChannelsTab />;
    case 'connections': return <ConnectionsTab />;
    default: return <AlertsTab />;
  }
}

export default function App() {
  const [activeTab, setActiveTab] = useState('alerts');

  return (
    <ConnectionProvider>
      <Layout tabs={TABS} activeTab={activeTab} onTabChange={setActiveTab}>
        <TabContent activeTab={activeTab} />
      </Layout>
    </ConnectionProvider>
  );
}
