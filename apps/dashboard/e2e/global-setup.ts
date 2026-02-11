async function globalSetup() {
  const maxRetries = 30;
  for (let i = 0; i < maxRetries; i++) {
    try {
      const res = await fetch('http://localhost:8080/health');
      if (res.ok) return;
    } catch {}
    await new Promise(r => setTimeout(r, 1000));
  }
  throw new Error('API server not available at http://localhost:8080 after 30s');
}
export default globalSetup;
