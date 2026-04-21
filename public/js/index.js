(function () {
    var map = L.map('server-map', { zoomControl: true, scrollWheelZoom: false }).setView([30, 20], 2);
    L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
        attribution: '&copy; <a href="https://carto.com/">CARTO</a>',
        subdomains: 'abcd', maxZoom: 19
    }).addTo(map);

    var requestedRegions = {};

    function typeLabel(t) {
        if (t === 'amneziawg') return 'AmneziaWG';
        if (t === 'ovpn') return 'OpenVPN';
        return t;
    }

    function flagEmoji(code) {
        if (!code || code.length !== 2) return '🌍';
        return code.toUpperCase().replace(/./g, function(c) {
            return String.fromCodePoint(c.charCodeAt(0) + 127397);
        });
    }

    function sendRequest(regionCode, btn, onDone) {
        btn.disabled = true;
        fetch('/api/server/request', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ region: regionCode })
        }).then(function(r) { return r.json(); }).then(function() {
            requestedRegions[regionCode] = true;
            onDone();
        }).catch(function() {
            btn.disabled = false;
        });
    }

    function buildPopupContent(region) {
        var html = '<div class="map-popup">';
        html += '<div class="map-popup__title">' + (region.flag_url ? '' : flagEmoji(region.code)) + ' ' + region.name + '</div>';
        if (region.servers && region.servers.length > 0) {
            html += '<ul class="map-popup__servers">';
            region.servers.forEach(function(s) {
                html += '<li>▶ ' + s.name + ' · ' + typeLabel(s.type) + '</li>';
            });
            html += '</ul>';
        } else {
            html += '<div style="font-size:0.82rem;opacity:0.7;margin-bottom:6px">Серверов пока нет</div>';
            if (requestedRegions[region.code]) {
                html += '<div class="map-popup__done">✓ Запрос отправлен</div>';
            } else {
                html += '<button class="map-popup__request" data-region="' + region.code + '">Запросить сервер</button>';
            }
        }
        html += '</div>';
        return html;
    }

    function renderGrid(regions) {
        var grid = document.getElementById('servers-grid');
        if (!regions || regions.length === 0) {
            grid.innerHTML = '<p class="servers__loading">Нет данных</p>';
            return;
        }
        grid.innerHTML = '';
        regions.forEach(function(region) {
            var card = document.createElement('div');
            card.className = 'server-card';
            var head = '<div class="server-card__head"><span class="server-card__flag">' + flagEmoji(region.code) + '</span><span class="server-card__name">' + region.name + '</span></div>';
            var body = '';
            if (region.servers && region.servers.length > 0) {
                body += '<ul class="server-card__list">';
                region.servers.forEach(function(s) {
                    body += '<li><span>▶</span>' + s.name + ' · ' + typeLabel(s.type) + '</li>';
                });
                body += '</ul>';
            } else {
                body += '<span class="server-card__empty">Серверов пока нет</span>';
                if (requestedRegions[region.code]) {
                    body += '<span class="server-card__requested">✓ Запрос отправлен</span>';
                } else {
                    body += '<button class="server-card__request" data-region="' + region.code + '">Запросить сервер</button>';
                }
            }
            card.innerHTML = head + body;
            var btn = card.querySelector('.server-card__request');
            if (btn) {
                btn.addEventListener('click', function() {
                    var code = btn.getAttribute('data-region');
                    sendRequest(code, btn, function() {
                        btn.outerHTML = '<span class="server-card__requested">✓ Запрос отправлен</span>';
                    });
                });
            }
            grid.appendChild(card);
        });
    }

    fetch('/api/regions').then(function(r) { return r.json(); }).then(function(regions) {
        regions.forEach(function(region) {
            if (!region.lat && !region.lng) return;
            var icon = L.divIcon({
                html: '<div style="width:14px;height:14px;border-radius:50%;background:' +
                    (region.servers && region.servers.length > 0 ? '#7c6ef7' : '#555') +
                    ';border:2px solid #fff;box-shadow:0 0 6px rgba(0,0,0,0.5)"></div>',
                className: '', iconSize: [14, 14], iconAnchor: [7, 7]
            });
            var marker = L.marker([region.lat, region.lng], { icon: icon }).addTo(map);
            marker.bindPopup(buildPopupContent(region), { maxWidth: 260 });
            marker.on('popupopen', function() {
                var btn = document.querySelector('.map-popup__request[data-region="' + region.code + '"]');
                if (btn) {
                    btn.addEventListener('click', function() {
                        sendRequest(region.code, btn, function() {
                            btn.outerHTML = '<div class="map-popup__done">✓ Запрос отправлен</div>';
                        });
                    });
                }
            });
        });
        renderGrid(regions);
    }).catch(function() {
        document.getElementById('servers-grid').innerHTML = '<p class="servers__loading">Ошибка загрузки</p>';
    });
})();
