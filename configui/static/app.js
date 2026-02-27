let currentConfig = null;

function showSection(sectionName) {
    document.querySelectorAll('.section-content').forEach(el => el.classList.remove('active'));
    document.querySelectorAll('.sidebar-menu a').forEach(el => el.classList.remove('active'));
    
    document.getElementById(sectionName + '-section').classList.add('active');
    document.querySelector(`.sidebar-menu a[onclick="showSection('${sectionName}')"]`).classList.add('active');
    
    if (sectionName === 'provider') {
        const defaultProvider = document.getElementById('provider').value;
        document.getElementById('providerSelect').value = defaultProvider;
        loadProviderConfig(defaultProvider);
    } else if (sectionName === 'cron') {
        loadCronJobs();
    } else if (sectionName === 'sessions') {
        loadSessions();
    } else if (sectionName === 'skills') {
        loadSkills();
    } else if (sectionName === 'chat') {
        initChat();
    }
}

function loadProviderConfig(provider) {
    if (currentConfig && currentConfig.providers && currentConfig.providers[provider]) {
        const p = currentConfig.providers[provider];
        document.getElementById('providerModel').value = p.model || '';
        document.getElementById('providerApiKey').value = p.apiKey || '';
        document.getElementById('providerApiBase').value = p.apiBase || '';
    } else {
        document.getElementById('providerModel').value = '';
        document.getElementById('providerApiKey').value = '';
        document.getElementById('providerApiBase').value = '';
    }
}

async function loadVisualConfig() {
    try {
        const response = await fetch('/api/config');
        const data = await response.json();
        
        if (!data.success) {
            showMessage('加载配置失败: ' + data.error, 'error');
            return;
        }
        
        const cfg = data.data;
        currentConfig = cfg;
        
        if (cfg.agents && cfg.agents.defaults) {
            document.getElementById('workspace').value = cfg.agents.defaults.workspace || '';
            document.getElementById('provider').value = cfg.agents.defaults.provider || 'openai';
            document.getElementById('maxTokens').value = cfg.agents.defaults.maxTokens || 4096;
            document.getElementById('temperature').value = cfg.agents.defaults.temperature || 0.7;
            document.getElementById('maxToolIterations').value = cfg.agents.defaults.maxToolIterations || 15;
        }
        
        if (cfg.channels && cfg.channels.feishu) {
            document.getElementById('feishuEnabled').value = String(cfg.channels.feishu.enabled);
            document.getElementById('feishuAppId').value = cfg.channels.feishu.appId || '';
            document.getElementById('feishuAppSecret').value = cfg.channels.feishu.appSecret || '';
        }
        
        if (cfg.channels && cfg.channels.mochat) {
            document.getElementById('mochatEnabled').value = String(cfg.channels.mochat.enabled);
            document.getElementById('mochatBaseUrl').value = cfg.channels.mochat.baseUrl || '';
            document.getElementById('mochatSocketUrl').value = cfg.channels.mochat.socketUrl || '';
        }
        
        onProviderChange();
    } catch (e) {
        console.error('Load config error:', e);
        showMessage('加载配置失败: ' + e, 'error');
    }
}

function onProviderChange() {
    const providerSelect = document.getElementById('providerSelect');
    const provider = providerSelect ? providerSelect.value : document.getElementById('provider').value;
    
    if (currentConfig && currentConfig.providers && currentConfig.providers[provider]) {
        const p = currentConfig.providers[provider];
        document.getElementById('providerModel').value = p.model || '';
        document.getElementById('providerApiKey').value = p.apiKey || '';
        document.getElementById('providerApiBase').value = p.apiBase || '';
    } else {
        document.getElementById('providerModel').value = '';
        document.getElementById('providerApiKey').value = '';
        document.getElementById('providerApiBase').value = '';
    }
}

function saveVisualConfig() {
    try {
        const cfg = currentConfig || {};
        
        if (!cfg.agents) cfg.agents = {};
        if (!cfg.agents.defaults) cfg.agents.defaults = {};
        if (!cfg.channels) cfg.channels = {};
        
        cfg.agents.defaults.workspace = document.getElementById('workspace').value;
        cfg.agents.defaults.provider = document.getElementById('provider').value;
        cfg.agents.defaults.maxTokens = parseInt(document.getElementById('maxTokens').value) || 4096;
        cfg.agents.defaults.temperature = parseFloat(document.getElementById('temperature').value) || 0.7;
        cfg.agents.defaults.maxToolIterations = parseInt(document.getElementById('maxToolIterations').value) || 15;
        
        if (!cfg.channels.feishu) cfg.channels.feishu = {};
        cfg.channels.feishu.enabled = document.getElementById('feishuEnabled').value === 'true';
        cfg.channels.feishu.appId = document.getElementById('feishuAppId').value;
        cfg.channels.feishu.appSecret = document.getElementById('feishuAppSecret').value;
        
        if (!cfg.channels.mochat) cfg.channels.mochat = {};
        cfg.channels.mochat.enabled = document.getElementById('mochatEnabled').value === 'true';
        cfg.channels.mochat.baseUrl = document.getElementById('mochatBaseUrl').value;
        cfg.channels.mochat.socketUrl = document.getElementById('mochatSocketUrl').value;
        
        const provider = document.getElementById('provider').value;
        const providerSelect = document.getElementById('providerSelect');
        const providerForSave = providerSelect ? providerSelect.value : provider;
        
        if (!cfg.providers) cfg.providers = {};
        if (!cfg.providers[providerForSave]) cfg.providers[providerForSave] = {};
        cfg.providers[providerForSave].model = document.getElementById('providerModel').value;
        cfg.providers[providerForSave].apiKey = document.getElementById('providerApiKey').value;
        cfg.providers[providerForSave].apiBase = document.getElementById('providerApiBase').value;
        
        const jsonStr = JSON.stringify(cfg, null, 4);
        
        fetch('/api/config', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: jsonStr
        })
        .then(r => r.json())
        .then(d => {
            if (d.success) {
                showMessage('配置保存成功！', 'success');
                currentConfig = cfg;
            } else {
                showMessage('保存失败: ' + d.error, 'error');
            }
        })
        .catch(e => showMessage('请求失败: ' + e, 'error'));
    } catch (e) {
        showMessage('保存失败: ' + e, 'error');
    }
}

function restartService() {
    if (!confirm('确定要重启服务吗？')) return;
    
    document.getElementById('status').textContent = '正在重启...';
    fetch('/api/restart', {method: 'POST'})
        .then(r => r.json())
        .then(d => {
            if (d.success) {
                showMessage('服务已重启', 'success');
            } else {
                showMessage('重启失败: ' + d.error, 'error');
            }
            document.getElementById('status').textContent = '就绪';
        })
        .catch(e => {
            showMessage('请求失败: ' + e, 'error');
            document.getElementById('status').textContent = '就绪';
        });
}

function showMessage(msg, type) {
    const msgDiv = document.createElement('div');
    msgDiv.className = 'message message-' + type;
    msgDiv.textContent = msg;
    
    const container = document.querySelector('.container');
    container.insertBefore(msgDiv, container.firstChild);
    
    setTimeout(() => msgDiv.remove(), 3000);
}

function loadCronJobs() {
    fetch('/api/cron')
        .then(res => res.json())
        .then(data => {
            const container = document.getElementById('cron-list');
            if (!data.success) {
                container.innerHTML = '<div class="error">加载失败: ' + data.error + '</div>';
                return;
            }
            const jobs = data.data || [];
            if (jobs.length === 0) {
                container.innerHTML = '<div class="empty">暂无 Cron 任务</div>';
                return;
            }
            let html = '<table class="data-table"><thead><tr><th>ID</th><th>名称</th><th>调度</th><th>启用</th><th>下次运行</th><th>状态</th><th>操作</th></tr></thead><tbody>';
            jobs.forEach(job => {
                const nextRun = job.state && job.state.next_run_at_ms ? new Date(job.state.next_run_at_ms).toLocaleString() : '-';
                const lastStatus = job.state && job.state.last_status ? job.state.last_status : '-';
                const enabled = job.enabled ? '✓' : '✗';
                html += `<tr><td>${job.id}</td><td>${job.name}</td><td>${job.schedule}</td><td>${enabled}</td><td>${nextRun}</td><td>${lastStatus}</td><td><button class="btn btn-danger btn-sm" onclick="deleteCronJob('${job.id}')">删除</button></td></tr>`;
            });
            html += '</tbody></table>';
            container.innerHTML = html;
        })
        .catch(e => {
            document.getElementById('cron-list').innerHTML = '<div class="error">加载失败: ' + e + '</div>';
        });
}

function deleteCronJob(id) {
    if (!confirm('确定要删除这个 Cron 任务吗？')) {
        return;
    }
    fetch('/api/cron/' + id, {
        method: 'DELETE'
    })
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            showMessage('Cron 任务删除成功', 'success');
            loadCronJobs();
        } else {
            showMessage('删除失败: ' + data.error, 'error');
        }
    })
    .catch(e => {
        showMessage('请求失败: ' + e, 'error');
    });
}

function loadSessions() {
    fetch('/api/sessions')
        .then(res => res.json())
        .then(data => {
            const container = document.getElementById('sessions-list');
            if (!data.success) {
                container.innerHTML = '<div class="error">加载失败: ' + data.error + '</div>';
                return;
            }
            const sessions = data.data || [];
            if (sessions.length === 0) {
                container.innerHTML = '<div class="empty">暂无 Session</div>';
                return;
            }
            let html = '<table class="data-table"><thead><tr><th>Key</th><th>创建时间</th><th>更新时间</th><th>操作</th></tr></thead><tbody>';
            sessions.forEach(s => {
                const key = encodeURIComponent(s.key);
                html += `<tr><td>${s.key}</td><td>${s.created_at}</td><td>${s.updated_at}</td><td><button class="btn btn-danger btn-sm" onclick="deleteSession('${key}')">删除</button></td></tr>`;
            });
            html += '</tbody></table>';
            container.innerHTML = html;
        })
        .catch(e => {
            document.getElementById('sessions-list').innerHTML = '<div class="error">加载失败: ' + e + '</div>';
        });
}

function deleteSession(key) {
    if (!confirm('确定要删除这个 Session 吗？')) {
        return;
    }
    fetch('/api/sessions/' + key, {
        method: 'DELETE'
    })
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            showMessage('Session 删除成功', 'success');
            loadSessions();
        } else {
            showMessage('删除失败: ' + data.error, 'error');
        }
    })
    .catch(e => {
        showMessage('请求失败: ' + e, 'error');
    });
}

function loadSkills() {
    fetch('/api/skills')
        .then(res => res.json())
        .then(data => {
            const container = document.getElementById('skills-list');
            if (!data.success) {
                container.innerHTML = '<div class="error">加载失败: ' + data.error + '</div>';
                return;
            }
            const skills = data.data || [];
            if (skills.length === 0) {
                container.innerHTML = '<div class="empty">暂无 Skills</div>';
                return;
            }
            let html = '<table class="data-table"><thead><tr><th>名称</th><th>来源</th><th>路径</th></tr></thead><tbody>';
            skills.forEach(skill => {
                html += `<tr><td>${skill.name}</td><td>${skill.source}</td><td>${skill.path}</td></tr>`;
            });
            html += '</tbody></table>';
            container.innerHTML = html;
        })
        .catch(e => {
            document.getElementById('skills-list').innerHTML = '<div class="error">加载失败: ' + e + '</div>';
        });
}

function initChat() {
    const container = document.getElementById('chat-messages');
    if (!container.dataset.initialized) {
        container.dataset.initialized = 'true';
    }
}

function handleChatKeyDown(event) {
    if (event.key === 'Enter' && !event.shiftKey) {
        event.preventDefault();
        sendChatMessage();
    }
}

function autoResizeTextarea(textarea) {
    textarea.style.height = 'auto';
    textarea.style.height = Math.min(textarea.scrollHeight, 300) + 'px';
}

function sendChatMessage() {
    const input = document.getElementById('chat-input');
    const content = input.value.trim();
    if (!content) return;

    addMessage('user', '我', content);
    input.value = '';
    input.style.height = 'auto';

    fetch('/api/chat', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({content: content})
    })
    .then(res => res.json())
    .then(data => {
        if (!data.success) {
            addMessage('system', '系统', '发送失败: ' + data.error);
        }
    })
    .catch(e => {
        addMessage('system', '系统', '请求失败: ' + e);
    });
}

function handleImageUpload(event) {
    const file = event.target.files[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = function(e) {
        const imgData = e.target.result;
        addMessage('user', '我', '<img src="' + imgData + '" alt="图片">');
        
        fetch('/api/chat', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({content: '[图片]', image: imgData})
        })
        .then(res => res.json())
        .then(data => {
            if (!data.success) {
                addMessage('system', '系统', '发送失败: ' + data.error);
            }
        })
        .catch(err => {
            addMessage('system', '系统', '请求失败: ' + err);
        });
    };
    reader.readAsDataURL(file);
    event.target.value = '';
}

function addMessage(type, sender, content, time) {
    const container = document.getElementById('chat-messages');
    const msgDiv = document.createElement('div');
    msgDiv.className = 'chat-message chat-message-' + type;
    
    const now = time || new Date().toLocaleTimeString('zh-CN', {hour: '2-digit', minute: '2-digit'});
    
    let avatar = '';
    if (type === 'user') {
        avatar = '<div class="chat-avatar">我</div>';
    } else if (type === 'assistant' || type === 'bot') {
        avatar = '<div class="chat-avatar">🤖</div>';
    }
    
    const bubbleHtml = `
        <div class="chat-bubble">
            ${type !== 'system' ? '<div class="chat-sender">' + sender + '</div>' : ''}
            <div class="chat-message-content">${content}</div>
            ${type !== 'system' ? '<div class="chat-time">' + now + '</div>' : ''}
        </div>
    `;
    
    if (type === 'user') {
        msgDiv.innerHTML = bubbleHtml + avatar;
    } else if (type === 'system') {
        msgDiv.innerHTML = bubbleHtml;
    } else {
        msgDiv.innerHTML = avatar + bubbleHtml;
    }
    
    container.appendChild(msgDiv);
    container.scrollTop = container.scrollHeight;
}

function addDivider(text) {
    const container = document.getElementById('chat-messages');
    const divider = document.createElement('div');
    divider.className = 'chat-divider';
    divider.innerHTML = '<span>' + text + '</span>';
    container.appendChild(divider);
    container.scrollTop = container.scrollHeight;
}

window.onload = loadVisualConfig;
