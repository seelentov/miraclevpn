const TOKEN_KEY = 'miiboost_chat_token';
let token = localStorage.getItem(TOKEN_KEY);

function now() {
    const d = new Date();
    return d.getHours().toString().padStart(2,'0') + ':' + d.getMinutes().toString().padStart(2,'0');
}

function parseMd(text) {
    return text
        .replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
        .replace(/\[([^\]]+)\]\((https?:\/\/[^)]+)\)/g,'<a href="$2" target="_blank" rel="noopener" class="md-link">$1</a>')
        .replace(/\*([^*\n]+)\*/g,'<b>$1</b>')
        .replace(/`([^`\n]+)`/g,'<code>$1</code>');
}

function showTyping() {
    document.getElementById('typing').classList.add('visible');
    scrollBottom();
}

function hideTyping() {
    document.getElementById('typing').classList.remove('visible');
}

function scrollBottom() {
    const el = document.getElementById('chat-messages');
    el.scrollTop = el.scrollHeight;
}

function renderMsg(msg, fromUser) {
    const wrap = document.createElement('div');
    wrap.className = fromUser ? 'msg msg--user' : 'msg msg--bot';

    const bubble = document.createElement('div');
    bubble.className = 'msg__bubble';
    bubble.innerHTML = parseMd(msg.text || '');
    wrap.appendChild(bubble);

    if (!fromUser) {
        if (msg.file) {
            const a = document.createElement('a');
            a.className = 'msg__file';
            a.href = msg.file.url;
            a.download = msg.file.name;
            a.innerHTML = `
                <span class="msg__file-icon">📄</span>
                <div class="msg__file-info">
                    <div class="msg__file-name">${msg.file.name}</div>
                    <div class="msg__file-hint">Нажмите, чтобы скачать</div>
                </div>`;
            wrap.appendChild(a);
        }

        if (msg.buttons && msg.buttons.length) {
            const btns = document.createElement('div');
            btns.className = 'msg__buttons';
            msg.buttons.forEach(row => {
                const rowEl = document.createElement('div');
                rowEl.className = 'msg__btn-row';
                row.forEach(btn => {
                    if (btn.url) {
                        const a = document.createElement('a');
                        a.className = 'msg__btn msg__btn--url';
                        a.href = btn.url;
                        a.target = '_blank';
                        a.rel = 'noopener';
                        a.textContent = btn.text;
                        rowEl.appendChild(a);
                    } else {
                        const b = document.createElement('button');
                        b.className = 'msg__btn';
                        b.textContent = btn.text;
                        b.addEventListener('click', () => {
                            addUserMsg(btn.text);
                            sendAction(btn.action);
                        });
                        rowEl.appendChild(b);
                    }
                });
                btns.appendChild(rowEl);
            });
            wrap.appendChild(btns);
        }
    }

    const time = document.createElement('div');
    time.className = 'msg__time';
    time.textContent = now();
    wrap.appendChild(time);

    const container = document.getElementById('chat-messages');
    container.insertBefore(wrap, document.getElementById('typing'));
    scrollBottom();
}

function addUserMsg(text) { renderMsg({ text }, true); }
function addBotMsgs(msgs) { hideTyping(); msgs.forEach(m => renderMsg(m, false)); }

async function apiStart(type, key) {
    const body = { type };
    if (key) body.key = key;
    return fetch('/api/chat/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });
}

async function sendAction(action, serverID) {
    showTyping();
    const body = { action };
    if (serverID) body.server_id = serverID;
    try {
        const res = await fetch('/api/chat/action', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': 'Bearer ' + token,
            },
            body: JSON.stringify(body),
        });
        const data = await res.json();
        if (data.messages) addBotMsgs(data.messages);
        else hideTyping();
    } catch (e) {
        hideTyping();
        renderMsg({ text: '⚠️ Ошибка соединения. Попробуйте позже.' }, false);
    }
}

document.getElementById('btn-new').addEventListener('click', async () => {
    const btn = document.getElementById('btn-new');
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner"></span>';
    const res = await apiStart('new');
    if (res.ok) {
        const data = await res.json();
        token = data.token;
        localStorage.setItem(TOKEN_KEY, token);
        document.getElementById('welcome').style.display = 'none';
        addBotMsgs(data.messages);
    } else {
        btn.disabled = false;
        btn.textContent = '🆕 Новый пользователь';
    }
});

document.getElementById('btn-has-key').addEventListener('click', () => {
    document.getElementById('key-wrap').classList.toggle('visible');
    document.getElementById('key-input').focus();
});

document.getElementById('btn-submit-key').addEventListener('click', async () => {
    const key = document.getElementById('key-input').value.trim();
    const errEl = document.getElementById('key-error');
    const btn = document.getElementById('btn-submit-key');
    errEl.classList.remove('visible');
    if (!key) { errEl.textContent = 'Введите ключ'; errEl.classList.add('visible'); return; }
    btn.disabled = true;
    btn.innerHTML = '<span class="spinner"></span>';
    const res = await apiStart('key', key);
    const data = await res.json();
    if (res.ok) {
        token = data.token;
        localStorage.setItem(TOKEN_KEY, token);
        document.getElementById('welcome').style.display = 'none';
        addBotMsgs(data.messages);
    } else {
        errEl.textContent = data.error || 'Неверный ключ';
        errEl.classList.add('visible');
        btn.disabled = false;
        btn.textContent = 'Войти';
    }
});

document.getElementById('btn-logout').addEventListener('click', () => {
    localStorage.removeItem(TOKEN_KEY);
    token = null;
    document.getElementById('chat-messages').innerHTML =
        '<div class="typing" id="typing"><span></span><span></span><span></span></div>';
    document.getElementById('welcome').style.display = 'flex';
});

if (token) {
    document.getElementById('welcome').style.display = 'none';
    showTyping();
    sendAction('menu');
}
