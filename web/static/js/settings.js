// Settings Page JavaScript

let interfaceLabels = {};
let monitoredInterfaces = [];

// Load current settings on page load
window.addEventListener('DOMContentLoaded', async () => {
    // Load interface list first, then labels
    await loadCurrentData();
    await loadLabels();
});

// Load current monitoring data to get interface list
async function loadCurrentData() {
    try {
        const response = await fetch('/api/current');
        if (!response.ok) throw new Error('Failed to fetch current data');

        const data = await response.json();
        // data.interfaces is an object with interface names as keys
        monitoredInterfaces = Object.keys(data.interfaces);

        console.log('Monitored interfaces:', monitoredInterfaces);
    } catch (error) {
        console.error('Error loading monitored interfaces:', error);
        showStatus('Error loading interface list', true);
    }
}

// Load existing labels from server
async function loadLabels() {
    try {
        const response = await fetch('/api/config/labels');
        if (!response.ok) throw new Error('Failed to fetch labels');

        interfaceLabels = await response.json();
        console.log('Loaded labels:', interfaceLabels);

        renderLabelEditor();
    } catch (error) {
        console.error('Error loading labels:', error);
        showStatus('Error loading settings', true);
    }
}

// Render label editor UI
function renderLabelEditor() {
    const container = document.getElementById('labelEditor');

    if (monitoredInterfaces.length === 0) {
        container.innerHTML = '<p style="color: var(--text-secondary);">Loading interfaces...</p>';
        return;
    }

    container.innerHTML = '';

    monitoredInterfaces.forEach(ifaceName => {
        const currentLabel = interfaceLabels[ifaceName] || '';

        const item = document.createElement('div');
        item.className = 'label-item';
        item.innerHTML = `
            <label>${ifaceName}</label>
            <input
                type="text"
                id="label_${ifaceName}"
                value="${escapeHtml(currentLabel)}"
                placeholder="${ifaceName}"
                onchange="markUnsaved()"
            />
            <button class="reset-btn" onclick="resetLabel('${ifaceName}')">Reset</button>
        `;

        container.appendChild(item);
    });
}

// Reset label to original interface name
function resetLabel(ifaceName) {
    const input = document.getElementById(`label_${ifaceName}`);
    if (input) {
        input.value = '';
        markUnsaved();
    }
}

// Mark form as having unsaved changes
function markUnsaved() {
    const saveBtn = document.getElementById('saveBtn');
    saveBtn.textContent = 'Save Changes *';
}

// Save labels to server
async function saveLabels() {
    const saveBtn = document.getElementById('saveBtn');
    saveBtn.disabled = true;
    showStatus('Saving...');

    // Collect all labels from inputs
    const updatedLabels = {};
    monitoredInterfaces.forEach(ifaceName => {
        const input = document.getElementById(`label_${ifaceName}`);
        if (input) {
            updatedLabels[ifaceName] = input.value.trim();
        }
    });

    try {
        const response = await fetch('/api/config/labels', {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(updatedLabels)
        });

        if (!response.ok) throw new Error('Failed to save labels');

        interfaceLabels = updatedLabels;
        saveBtn.textContent = 'Save Changes';
        showStatus('Settings saved successfully!');

        // Reload after 1 second to show updated labels
        setTimeout(() => {
            window.location.reload();
        }, 1000);

    } catch (error) {
        console.error('Error saving labels:', error);
        showStatus('Error saving settings', true);
    } finally {
        saveBtn.disabled = false;
    }
}

// Show status message
function showStatus(message, isError = false) {
    const statusEl = document.getElementById('statusMessage');
    statusEl.textContent = message;
    statusEl.className = isError ? 'status-message error' : 'status-message';

    // Clear message after 3 seconds
    if (!isError && message) {
        setTimeout(() => {
            statusEl.textContent = '';
        }, 3000);
    }
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
