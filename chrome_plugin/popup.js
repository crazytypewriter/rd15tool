// popup.js

function getAllSubdomains(domain) {
  const parts = domain.split('.');
  const subdomains = [];
  for (let i = 0; i < parts.length - 1; i++) {
    subdomains.push(parts.slice(i).join('.'));
  }
  subdomains.push(domain);
  return [...new Set(subdomains)].reverse();
}

function extractDomainsFromPage(tabId) {
  return new Promise((resolve, reject) => {
    try {
      chrome.scripting.executeScript({
        target: { tabId: tabId },
        func: () => {
          const elements = [
            ...document.querySelectorAll('a[href]'),
            ...document.querySelectorAll('iframe[src]'),
            ...document.querySelectorAll('img[src]'),
            ...document.querySelectorAll('script[src]'),
            ...document.querySelectorAll('link[href]'),
            ...document.querySelectorAll('video[src]'),
            ...document.querySelectorAll('audio[src]')
          ];
          const urlsFromElements = elements.map(el => el.href || el.src);

          const urlsFromInlineScripts = [...document.querySelectorAll('script:not([src])')]
            .map(s => s.textContent)
            .flatMap(code => {
              // Более строгая регулярка
              const matches = code.match(/https?:\/\/[a-zA-Z0-9.-]+\.[a-z]{2,}(\/[^\s'"`<>]*)?/g);
              return matches || [];
            });

          const allUrls = urlsFromElements.concat(urlsFromInlineScripts);

          const allHosts = allUrls.map(url => {
            try { return new URL(url).hostname; } catch { return null; }
          }).filter(Boolean);

          // Вот сюда вставляем фильтрацию:
          const cleanedHosts = allHosts.filter(host =>
            host.includes('.')
          );

          return [...new Set(allHosts)];
        }
      }).then(injectionResults => {
        const result = injectionResults[0]?.result || [];
        resolve(result);
      }).catch(err => reject(err));
    } catch (e) {
      reject(new Error('chrome.scripting API не поддерживается в этом браузере или манифесте.'));
    }
  });
}



function getSecondLevelDomain(hostname) {
  const parts = hostname.split('.');
  if (parts.length <= 2) return hostname;

  const secondLevelTlds = ['co.uk', 'com.au', 'co.jp', 'com.br', 'com.cn'];
  const lastTwo = parts.slice(-2).join('.');
  const lastThree = parts.slice(-3).join('.');

  if (secondLevelTlds.includes(lastTwo)) {
    return lastThree;
  } else {
    return lastTwo;
  }
}

async function loadDomains() {
  chrome.tabs.query({ active: true, currentWindow: true }, async tabs => {
    const listEl = document.getElementById('domainList');
    listEl.innerHTML = 'Загрузка...';
    if (tabs[0]?.id) {
      try {
        const pageDomains = await extractDomainsFromPage(tabs[0].id);
        const mainUrl = new URL(tabs[0].url);
        const combined = [...new Set([mainUrl.hostname, ...pageDomains])];

        // Прогоняем через getSecondLevelDomain и удаляем дубли
        const secondLevelDomains = [...new Set(combined.map(getSecondLevelDomain))];

        listEl.innerHTML = '';
        secondLevelDomains.forEach(domain => {
          const li = document.createElement('li');
          li.innerHTML = `<label><input type="checkbox" value="${domain}" checked> ${domain}</label>`;
          listEl.appendChild(li);
        });
        if (secondLevelDomains.length === 0) listEl.textContent = 'Нет доменов';
      } catch (err) {
        listEl.textContent = 'Ошибка получения доменов: ' + err.message;
      }
    } else {
      listEl.textContent = 'Нет активной вкладки';
    }
  });
}


document.getElementById('selectAll').addEventListener('change', e => {
  document.querySelectorAll('#domainList input[type="checkbox"]').forEach(cb => cb.checked = e.target.checked);
});

document.getElementById('sendSelected').addEventListener('click', async () => {
  const selected = [...document.querySelectorAll('#domainList input[type="checkbox"]:checked')]
    .map(cb => '.' + cb.value).join('\n');

  if (selected) {
    try {
      const response = await fetch('http://192.168.31.1:8090/suffixes', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: selected
      });
      document.getElementById('result').textContent = response.ok ? 'Отправлено успешно' : 'Ошибка отправки';
    } catch (err) {
      document.getElementById('result').textContent = 'Ошибка отправки: ' + err;
    }
  } else {
    alert('Не выбраны домены!');
  }
});

document.getElementById('sendSelectedDel').addEventListener('click', async () => {
  const selected = [...document.querySelectorAll('#domainList input[type="checkbox"]:checked')]
    .map(cb => '.' + cb.value).join('\n');

  if (selected) {
    try {
      const response = await fetch('http://192.168.31.1:8090/suffixes', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: selected
      });
      document.getElementById('result').textContent = response.ok ? 'Отправлено успешно' : 'Ошибка отправки';
    } catch (err) {
      document.getElementById('result').textContent = 'Ошибка отправки: ' + err;
    }
  } else {
    alert('Не выбраны домены!');
  }
});

document.addEventListener('DOMContentLoaded', loadDomains);
