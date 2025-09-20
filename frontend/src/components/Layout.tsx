import React, { useState, useEffect } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { 
  Home, 
  Users, 
  Plus, 
  Search, 
  Brain,
  Menu,
  X,
  Activity,
  Play
} from 'lucide-react';
import AgentHubAPI from '../services/api';

interface LayoutProps {
  children: React.ReactNode;
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  const location = useLocation();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [systemStatus, setSystemStatus] = useState<'healthy' | 'error' | 'loading'>('loading');

  // 检查系统健康状态
  useEffect(() => {
    const checkHealth = async () => {
      try {
        await AgentHubAPI.healthCheck();
        setSystemStatus('healthy');
      } catch (error) {
        setSystemStatus('error');
      }
    };

    checkHealth();
    // 每30秒检查一次
    const interval = setInterval(checkHealth, 30000);
    return () => clearInterval(interval);
  }, []);

  const navigation = [
    { name: '概览', href: '/', icon: Home },
    { name: 'Agent管理', href: '/agents', icon: Users },
    { name: '注册Agent', href: '/register', icon: Plus },
    { name: '能力发现', href: '/discover', icon: Search },
    { name: '智能分析', href: '/analyze', icon: Brain },
    { name: 'Playground', href: '/playground', icon: Play },
  ];

  const isCurrentPath = (path: string) => {
    return location.pathname === path;
  };

  return (
    <div className="h-screen flex overflow-hidden bg-gray-100">
      {/* Mobile sidebar */}
      <div className={`fixed inset-0 flex z-40 md:hidden ${sidebarOpen ? '' : 'hidden'}`}>
        <div className="fixed inset-0 bg-gray-600 bg-opacity-75" onClick={() => setSidebarOpen(false)} />
        <div className="relative flex-1 flex flex-col max-w-xs w-full bg-white">
          <div className="absolute top-0 right-0 -mr-12 pt-2">
            <button
              type="button"
              className="ml-1 flex items-center justify-center h-10 w-10 rounded-full focus:outline-none focus:ring-2 focus:ring-inset focus:ring-white"
              onClick={() => setSidebarOpen(false)}
            >
              <X className="h-6 w-6 text-white" />
            </button>
          </div>
          <SidebarContent navigation={navigation} isCurrentPath={isCurrentPath} />
        </div>
      </div>

      {/* Static sidebar for desktop */}
      <div className="hidden md:flex md:w-64 md:flex-col md:fixed md:inset-y-0">
        <SidebarContent navigation={navigation} isCurrentPath={isCurrentPath} />
      </div>

      {/* Main content */}
      <div className="md:pl-64 flex flex-col flex-1">
        {/* Top nav */}
        <div className="sticky top-0 z-10 flex-shrink-0 flex h-16 bg-white shadow">
          <button
            type="button"
            className="px-4 border-r border-gray-200 text-gray-500 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-primary-500 md:hidden"
            onClick={() => setSidebarOpen(true)}
          >
            <Menu className="h-6 w-6" />
          </button>
          <div className="flex-1 px-4 flex justify-between items-center">
            <div>
              <h1 className="text-2xl font-semibold text-gray-900">AgentHub</h1>
            </div>
            <div className="ml-4 flex items-center md:ml-6">
              {/* System status indicator */}
              <div className="flex items-center space-x-2">
                <Activity className={`h-5 w-5 ${
                  systemStatus === 'healthy' ? 'text-success-500' : 
                  systemStatus === 'error' ? 'text-error-500' : 'text-gray-400'
                }`} />
                <span className={`text-sm font-medium ${
                  systemStatus === 'healthy' ? 'text-success-600' : 
                  systemStatus === 'error' ? 'text-error-600' : 'text-gray-400'
                }`}>
                  {systemStatus === 'healthy' ? '系统正常' : 
                   systemStatus === 'error' ? '系统异常' : '检查中...'}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Page content */}
        <main className="flex-1 relative overflow-y-auto focus:outline-none">
          <div className="py-6">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 md:px-8">
              {children}
            </div>
          </div>
        </main>
      </div>
    </div>
  );
};

const SidebarContent: React.FC<{
  navigation: Array<{ name: string; href: string; icon: any }>;
  isCurrentPath: (path: string) => boolean;
}> = ({ navigation, isCurrentPath }) => {
  return (
    <div className="flex-1 flex flex-col min-h-0 bg-white border-r border-gray-200">
      <div className="flex-1 flex flex-col pt-5 pb-4 overflow-y-auto">
        <div className="flex items-center flex-shrink-0 px-4">
          <div className="flex items-center">
            <div className="bg-primary-500 rounded-lg p-2">
              <Brain className="h-8 w-8 text-white" />
            </div>
            <div className="ml-3">
              <h1 className="text-xl font-bold text-gray-900">AgentHub</h1>
              <p className="text-sm text-gray-500">智能Agent平台</p>
            </div>
          </div>
        </div>
        <nav className="mt-8 flex-1 px-2 space-y-1">
          {navigation.map((item) => {
            const Icon = item.icon;
            const current = isCurrentPath(item.href);
            return (
              <Link
                key={item.name}
                to={item.href}
                className={`group flex items-center px-2 py-2 text-sm font-medium rounded-md transition-colors duration-150 ${
                  current
                    ? 'bg-primary-100 text-primary-900'
                    : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                }`}
              >
                <Icon
                  className={`mr-3 h-5 w-5 ${
                    current ? 'text-primary-500' : 'text-gray-400 group-hover:text-gray-500'
                  }`}
                />
                {item.name}
              </Link>
            );
          })}
        </nav>
      </div>
    </div>
  );
};

export default Layout;