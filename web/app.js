async function j(url, opts) {
  const r = await fetch(url, opts);
  const t = await r.text();
  try { return JSON.parse(t); } catch { return t; }
}
async function refresh() {
  document.getElementById('stats').textContent = JSON.stringify(await j('/api/stats'), null, 2);
  document.getElementById('queue').textContent = JSON.stringify(await j('/api/queue'), null, 2);
  document.getElementById('bounces').textContent = JSON.stringify(await j('/api/bounces'), null, 2);
}
document.getElementById('refresh').onclick = refresh;
document.getElementById('deliver').onclick = async () => {
  await j('/api/deliver', { method: 'POST' });
  await refresh();
};
document.getElementById('submit').onclick = async () => {
  const body = {
    mail_from: document.getElementById('from').value,
    to: document.getElementById('to').value.split(/[,;\s]+/).filter(Boolean),
    subject: document.getElementById('subject').value,
    body: document.getElementById('body').value,
  };
  await j('/api/submit', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  await refresh();
};
refresh();
