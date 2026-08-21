const headers = { 'Content-Type': 'application/json', 'X-Actor-ID': 'local-quality', 'X-Actor-Name': '本地质量负责人', 'X-Actor-Roles': 'quality_manager,calibration_operator' };
async function request(path, init = {}) { const response = await fetch(path, { ...init, headers: { ...headers, ...init.headers } }); if (!response.ok) {
    const body = await response.json();
    throw Object.assign(new Error(body.message), body);
} return response.json(); }
export const api = {
    listInstruments: (params) => request(`/api/v1/instruments?${params}`),
    getInstrument: (id) => request(`/api/v1/instruments/${id}`),
    checkUsage: (id, batchNumber) => request(`/api/v1/instruments/${id}/usage-checks`, { method: 'POST', body: JSON.stringify({ batch_number: batchNumber, operator_id: 'local-operator' }) }),
    listAlerts: () => request('/api/v1/alerts?page=1&size=100&sort=created_at&order=desc'),
    createExecution: (payload) => request('/api/v1/executions', { method: 'POST', body: JSON.stringify(payload) }),
    restore: (id, version, comment) => request(`/api/v1/nonconformances/${id}/restore`, { method: 'POST', headers: { 'If-Match': String(version) }, body: JSON.stringify({ comment }) }),
    trace: (id) => request(`/api/v1/instruments/${id}/trace`),
};
