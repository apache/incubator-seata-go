/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { createContext, useContext, useEffect, useState } from 'react';
import type { ReactNode } from 'react';

type Theme = 'dark' | 'light' | 'neon';

interface ThemeContextType {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextType | null>(null);

interface ThemeProviderProps {
  children: ReactNode;
}

const themes = {
  dark: {
    '--bg-primary': '#0f0f23',
    '--panel-bg': '#1a1a2e',
    '--header-bg': '#16213e',
    '--card-bg': '#16213e',
    '--input-bg': '#0e1a3c',
    '--text-primary': '#ffffff',
    '--text-secondary': '#a0a0a0',
    '--accent-color': '#00d4ff',
    '--success-color': '#00ff88',
    '--warning-color': '#ffa500',
    '--error-color': '#ff4444',
    '--border-color': '#333',
    '--hover-bg': '#2a2a4a',
  },
  light: {
    '--bg-primary': '#f8f9fa',
    '--panel-bg': '#ffffff',
    '--header-bg': '#f1f3f4',
    '--card-bg': '#ffffff',
    '--input-bg': '#f8f9fa',
    '--text-primary': '#202124',
    '--text-secondary': '#5f6368',
    '--accent-color': '#1a73e8',
    '--success-color': '#137333',
    '--warning-color': '#f29900',
    '--error-color': '#d93025',
    '--border-color': '#dadce0',
    '--hover-bg': '#f1f3f4',
  },
  neon: {
    '--bg-primary': '#000011',
    '--panel-bg': '#0a0a1a',
    '--header-bg': '#1a0a2e',
    '--card-bg': '#2a1a3e',
    '--input-bg': '#1a0a2e',
    '--text-primary': '#00ffff',
    '--text-secondary': '#ff00ff',
    '--accent-color': '#00ff00',
    '--success-color': '#ffff00',
    '--warning-color': '#ff8800',
    '--error-color': '#ff0080',
    '--border-color': '#ff00ff',
    '--hover-bg': '#3a2a4e',
  }
};

export const ThemeProvider: React.FC<ThemeProviderProps> = ({ children }) => {
  const [theme, setTheme] = useState<Theme>(() => {
    const saved = localStorage.getItem('workflow-agent-theme');
    return (saved as Theme) || 'dark';
  });

  useEffect(() => {
    const root = document.documentElement;
    const themeVars = themes[theme];
    
    // 应用 CSS 变量
    Object.entries(themeVars).forEach(([property, value]) => {
      root.style.setProperty(property, value);
    });
    
    // 保存到 localStorage
    localStorage.setItem('workflow-agent-theme', theme);
    
    // 更新 body class 以便额外的样式处理
    document.body.className = `theme-${theme}`;
  }, [theme]);

  const toggleTheme = () => {
    const themeOrder: Theme[] = ['dark', 'light', 'neon'];
    const currentIndex = themeOrder.indexOf(theme);
    const nextIndex = (currentIndex + 1) % themeOrder.length;
    setTheme(themeOrder[nextIndex]);
  };

  const value: ThemeContextType = {
    theme,
    setTheme,
    toggleTheme,
  };

  return (
    <ThemeContext.Provider value={value}>
      {children}
    </ThemeContext.Provider>
  );
};

export const useTheme = (): ThemeContextType => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};