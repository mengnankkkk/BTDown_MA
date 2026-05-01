import type { AppView } from '../../../types/appView'
import { navigationItems } from '../constants/navigationItems'

interface SideNavigationProps {
  activeView: AppView
  onChange: (view: AppView) => void
}

export function SideNavigation({ activeView, onChange }: SideNavigationProps) {
  return (
    <aside className="side-navigation">
      <div className="app-branding">
        <h1>BTDown_MA</h1>
        <p>桌面端边下边播控制台</p>
      </div>
      <nav className="nav-list">
        {navigationItems.map((item) => (
          <button
            key={item.key}
            type="button"
            className={item.key === activeView ? 'nav-item active' : 'nav-item'}
            onClick={() => onChange(item.key)}
          >
            <strong>{item.title}</strong>
            <span>{item.description}</span>
          </button>
        ))}
      </nav>
    </aside>
  )
}
