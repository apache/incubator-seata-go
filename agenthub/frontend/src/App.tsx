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