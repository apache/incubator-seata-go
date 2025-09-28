import React from 'react';
import {Route, Routes} from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import AgentList from './pages/AgentList';
import AgentRegister from './pages/AgentRegister';
import ContextAnalysis from './pages/ContextAnalysis';
import AgentDiscover from './pages/AgentDiscover';
import AgentDetail from './pages/AgentDetail';
import Playground from './pages/Playground';

function App() {
    return (
        <Layout>
            <Routes>
                <Route path="/" element={<Dashboard/>}/>
                <Route path="/agents" element={<AgentList/>}/>
                <Route path="/agents/:id" element={<AgentDetail/>}/>
                <Route path="/register" element={<AgentRegister/>}/>
                <Route path="/discover" element={<AgentDiscover/>}/>
                <Route path="/analyze" element={<ContextAnalysis/>}/>
                <Route path="/playground" element={<Playground/>}/>
            </Routes>
        </Layout>
    );
}

export default App;