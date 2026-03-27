import ConnectionSelector from './ConnectionSelector';

export default function Layout({ tabs, activeTab, onTabChange, children }) {
  return (
    <>
      <header>
        <div className="container header-inner">
          <h1>ClickHouse Alerting System</h1>
          <ConnectionSelector />
        </div>
      </header>
      <div className="container">
        <nav id="nav">
          {tabs.map(tab => (
            <button
              key={tab.id}
              className={activeTab === tab.id ? 'active' : ''}
              onClick={() => onTabChange(tab.id)}
            >
              {tab.label}
            </button>
          ))}
        </nav>
        {children}
      </div>
    </>
  );
}
