// ============================================================================
// WebSocket Real-time Connection
// ============================================================================

let ws;
let reconnectInterval = 3000;
let charts = {};
let chartData = {};
let availableInterfaces = new Set();
let interfaceLabels = {}; // Store custom labels for interfaces
let modalChart = null;
let currentZoomedInterface = null;
let interfaceStats = {}; // Store current statistics for each interface

// Frontend statistics calculation
const STATS_WINDOW = 10; // 10 seconds window for avg/peak calculation
let statsHistory = {}; // Store historical data points for each interface

// Chart configuration
const MAX_DATA_POINTS = 60; // Show last 60 seconds
const CHART_COLORS = {
    upload: 'rgb(239, 68, 68)',      // Red
    download: 'rgb(16, 185, 129)',   // Green
    grid: 'rgba(71, 85, 105, 0.3)',
    text: 'rgb(148, 163, 184)'
};

function connect() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = protocol + '//' + window.location.host + '/api/realtime';

    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        updateStatus(true);
    };

    ws.onclose = () => {
        updateStatus(false);
        setTimeout(connect, reconnectInterval);
    };

    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };

    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        updateDisplay(data);
    };
}

// ============================================================================
// Frontend Statistics Calculation
// ============================================================================

function updateStatsHistory(interfaceName, timestamp, uploadRate, downloadRate) {
    if (!statsHistory[interfaceName]) {
        statsHistory[interfaceName] = [];
    }

    const history = statsHistory[interfaceName];

    // Add new data point
    history.push({
        time: new Date(timestamp),
        upload: uploadRate,
        download: downloadRate
    });

    // Remove data points older than STATS_WINDOW seconds
    const cutoffTime = new Date(timestamp) - (STATS_WINDOW * 1000);
    while (history.length > 0 && history[0].time < cutoffTime) {
        history.shift();
    }
}

function calculateStats(interfaceName) {
    const history = statsHistory[interfaceName];

    if (!history || history.length === 0) {
        return { avgUpload: 0, avgDownload: 0, peakUpload: 0, peakDownload: 0 };
    }

    let sumUpload = 0;
    let sumDownload = 0;
    let maxUpload = 0;
    let maxDownload = 0;

    for (const point of history) {
        sumUpload += point.upload;
        sumDownload += point.download;
        maxUpload = Math.max(maxUpload, point.upload);
        maxDownload = Math.max(maxDownload, point.download);
    }

    return {
        avgUpload: sumUpload / history.length,
        avgDownload: sumDownload / history.length,
        peakUpload: maxUpload,
        peakDownload: maxDownload
    };
}

function updateStatus(connected) {
    const statusEl = document.getElementById('status');
    const statusText = statusEl.querySelector('.status-text');

    if (connected) {
        statusEl.className = 'status connected';
        statusText.textContent = 'Connected';
    } else {
        statusEl.className = 'status disconnected';
        statusText.textContent = 'Reconnecting...';
    }
}

// ============================================================================
// Real-time Display Functions
// ============================================================================

function formatBytes(bytes) {
    const mbps = (bytes * 8 / 1000000).toFixed(2);
    return mbps + ' Mbps';
}

function formatTime(date) {
    return date.toLocaleTimeString('en-US', {
        hour12: false,
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
}

function createChart(canvasId, interfaceName) {
    const ctx = document.getElementById(canvasId).getContext('2d');

    // Initialize data storage
    chartData[interfaceName] = {
        labels: [],
        upload: [],
        download: []
    };

    const chart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [
                {
                    label: 'Upload',
                    data: [],
                    borderColor: CHART_COLORS.upload,
                    backgroundColor: 'rgba(239, 68, 68, 0.1)',
                    borderWidth: 2,
                    tension: 0.4,
                    fill: true,
                    pointRadius: 0,
                    pointHoverRadius: 4
                },
                {
                    label: 'Download',
                    data: [],
                    borderColor: CHART_COLORS.download,
                    backgroundColor: 'rgba(16, 185, 129, 0.1)',
                    borderWidth: 2,
                    tension: 0.4,
                    fill: true,
                    pointRadius: 0,
                    pointHoverRadius: 4
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            animation: {
                duration: 300
            },
            interaction: {
                intersect: false,
                mode: 'index'
            },
            plugins: {
                legend: {
                    display: false
                },
                tooltip: {
                    backgroundColor: 'rgba(30, 41, 59, 0.95)',
                    titleColor: CHART_COLORS.text,
                    bodyColor: '#f1f5f9',
                    borderColor: 'rgba(71, 85, 105, 0.5)',
                    borderWidth: 1,
                    padding: 12,
                    displayColors: true,
                    callbacks: {
                        label: function(context) {
                            return context.dataset.label + ': ' + formatBytes(context.parsed.y);
                        }
                    }
                }
            },
            scales: {
                x: {
                    display: true,
                    grid: {
                        color: CHART_COLORS.grid,
                        drawBorder: false
                    },
                    ticks: {
                        color: CHART_COLORS.text,
                        maxTicksLimit: 8,
                        maxRotation: 0,
                        font: {
                            size: 10
                        }
                    }
                },
                y: {
                    display: true,
                    beginAtZero: true,
                    grid: {
                        color: CHART_COLORS.grid,
                        drawBorder: false
                    },
                    ticks: {
                        color: CHART_COLORS.text,
                        callback: function(value) {
                            return (value * 8 / 1000000).toFixed(0) + ' Mbps';
                        },
                        font: {
                            size: 10
                        }
                    }
                }
            }
        }
    });

    charts[interfaceName] = chart;
    return chart;
}

function updateChart(interfaceName, timestamp, uploadRate, downloadRate) {
    const data = chartData[interfaceName];
    const chart = charts[interfaceName];

    if (!data || !chart) return;

    const time = formatTime(new Date(timestamp));

    // Add new data
    data.labels.push(time);
    data.upload.push(uploadRate);
    data.download.push(downloadRate);

    // Keep only last MAX_DATA_POINTS
    if (data.labels.length > MAX_DATA_POINTS) {
        data.labels.shift();
        data.upload.shift();
        data.download.shift();
    }

    // Update chart
    chart.data.labels = data.labels;
    chart.data.datasets[0].data = data.upload;
    chart.data.datasets[1].data = data.download;
    chart.update('none'); // Update without animation for smooth real-time display

    // Update modal chart if this interface is currently zoomed
    if (currentZoomedInterface === interfaceName) {
        updateModalChart();
    }
}

function updateDisplay(data) {
    // Update timestamp
    const time = new Date(data.timestamp).toLocaleString();
    document.getElementById('timestamp').textContent = 'Last update: ' + time;

    // Update interfaces
    const container = document.getElementById('interfaces');

    for (const [name, stats] of Object.entries(data.interfaces)) {
        // Track available interfaces
        availableInterfaces.add(name);

        let card = document.getElementById('card-' + name);

        // Create card if it doesn't exist
        if (!card) {
            card = createInterfaceCard(name);
            container.appendChild(card);

            // Create chart
            const canvasId = 'chart-' + name;
            createChart(canvasId, name);
        }

        // Update stats history for frontend calculation
        updateStatsHistory(name, data.timestamp, stats.upload_rate, stats.download_rate);

        // Calculate frontend stats
        const calculatedStats = calculateStats(name);

        // Update current display
        card.querySelector('.current-upload').textContent = formatBytes(stats.upload_rate);
        card.querySelector('.current-download').textContent = formatBytes(stats.download_rate);

        // Update calculated avg/peak
        card.querySelector('.avg-upload').textContent = formatBytes(calculatedStats.avgUpload);
        card.querySelector('.avg-download').textContent = formatBytes(calculatedStats.avgDownload);
        card.querySelector('.peak-upload').textContent = formatBytes(calculatedStats.peakUpload);
        card.querySelector('.peak-download').textContent = formatBytes(calculatedStats.peakDownload);

        // Store combined stats for modal
        interfaceStats[name] = {
            upload_rate: stats.upload_rate,
            download_rate: stats.download_rate,
            upload_avg: calculatedStats.avgUpload,
            download_avg: calculatedStats.avgDownload,
            upload_peak: calculatedStats.peakUpload,
            download_peak: calculatedStats.peakDownload
        };

        // Update chart
        updateChart(name, data.timestamp, stats.upload_rate, stats.download_rate);

        // Update modal stats if this interface is currently zoomed
        if (currentZoomedInterface === name) {
            updateModalStats(interfaceStats[name]);
        }
    }

    // Update interface selector in history panel
    updateInterfaceSelector();
}

function createInterfaceCard(name) {
    const card = document.createElement('div');
    card.className = 'interface-card';
    card.id = 'card-' + name;

    const displayName = getInterfaceDisplayName(name);
    const hasCustomLabel = interfaceLabels[name] && interfaceLabels[name] !== name;

    card.innerHTML = `
        <div class="interface-header">
            <div class="interface-name-wrapper">
                <span class="interface-name" data-interface="${name}">${displayName}</span>
                ${hasCustomLabel ? `<span class="original-name">(${name})</span>` : ''}
                <button class="edit-btn" data-interface="${name}" title="Edit label">✏️</button>
            </div>
            <div class="interface-actions">
                <button class="history-btn" onclick="openHistoryPanel('${name}')">📊</button>
            </div>
        </div>

        <div class="chart-container">
            <canvas id="chart-${name}"></canvas>
        </div>

        <div class="stats-detail">
            <div class="stats-main">
                <div class="stat-current">
                    <span class="stat-upload current-upload">0</span>
                    <span class="stat-download current-download">0</span>
                </div>
                <button class="stats-toggle" onclick="toggleStats('${name}')">
                    <span class="toggle-icon">▼</span>
                </button>
            </div>
            <div class="stats-content" id="stats-${name}">
                <div class="stat-row">
                    <span class="stat-label">平均 (10s)</span>
                    <div class="stat-values">
                        <span class="stat-upload avg-upload">0</span>
                        <span class="stat-download avg-download">0</span>
                    </div>
                </div>
                <div class="stat-row">
                    <span class="stat-label">峰值 (10s)</span>
                    <div class="stat-values">
                        <span class="stat-upload peak-upload">0</span>
                        <span class="stat-download peak-download">0</span>
                    </div>
                </div>
            </div>
        </div>
    `;

    // Make interface name editable after DOM is created
    setTimeout(() => {
        const nameElement = card.querySelector('.interface-name');
        const editBtn = card.querySelector('.edit-btn');
        if (nameElement && editBtn) {
            makeInterfaceNameEditable(name, nameElement, editBtn);
        }

        // Add click event to chart container for zoom
        const chartContainer = card.querySelector('.chart-container');
        if (chartContainer) {
            chartContainer.addEventListener('click', () => {
                openChartModal(name);
            });
        }
    }, 0);

    return card;
}

// ============================================================================
// Historical Data Functions
// ============================================================================

function updateInterfaceSelector() {
    const select = document.getElementById('historyInterface');
    // This element only exists in history.html, not in index.html
    if (!select) return;

    const currentValue = select.value;

    // Clear and repopulate
    select.innerHTML = '';
    availableInterfaces.forEach(iface => {
        const option = document.createElement('option');
        option.value = iface;
        option.textContent = iface;
        select.appendChild(option);
    });

    // Restore selection if possible
    if (currentValue && availableInterfaces.has(currentValue)) {
        select.value = currentValue;
    }
}

function openHistoryPanel(interfaceName) {
    // Open history page in new window
    const width = 1200;
    const height = 800;
    const left = (screen.width - width) / 2;
    const top = (screen.height - height) / 2;

    const url = `/history.html?interface=${encodeURIComponent(interfaceName)}`;
    const features = `width=${width},height=${height},left=${left},top=${top},resizable=yes,scrollbars=yes`;

    window.open(url, `history_${interfaceName}`, features);
}

// ============================================================================
// Interface Label Management
// ============================================================================

// Load interface labels from server
async function loadInterfaceLabels() {
    try {
        const response = await fetch('/api/config/labels');
        if (response.ok) {
            interfaceLabels = await response.json();
            console.log('Loaded interface labels:', interfaceLabels);
        }
    } catch (error) {
        console.error('Error loading interface labels:', error);
    }
}

// Get display name for an interface
function getInterfaceDisplayName(interfaceName) {
    return interfaceLabels[interfaceName] || interfaceName;
}

// Update interface name display after editing
function updateInterfaceNameDisplay(interfaceName, wrapper) {
    const displayName = getInterfaceDisplayName(interfaceName);
    const hasCustomLabel = interfaceLabels[interfaceName] && interfaceLabels[interfaceName] !== interfaceName;

    // Rebuild the wrapper content
    wrapper.innerHTML = `
        <span class="interface-name" data-interface="${interfaceName}">${displayName}</span>
        ${hasCustomLabel ? `<span class="original-name">(${interfaceName})</span>` : ''}
        <button class="edit-btn" data-interface="${interfaceName}" title="Edit label">✏️</button>
    `;

    // Re-attach event listeners
    const nameElement = wrapper.querySelector('.interface-name');
    const editBtn = wrapper.querySelector('.edit-btn');
    if (nameElement && editBtn) {
        makeInterfaceNameEditable(interfaceName, nameElement, editBtn);
    }
}

// Make interface name editable on double-click or edit button click
function makeInterfaceNameEditable(interfaceName, nameElement, editBtn) {
    nameElement.title = 'Double-click to edit label';

    const startEdit = () => {
        const currentLabel = getInterfaceDisplayName(interfaceName);
        const wrapper = nameElement.parentElement;

        // Create input element
        const input = document.createElement('input');
        input.type = 'text';
        input.value = currentLabel;
        input.className = 'interface-name-input';
        input.style.cssText = `
            flex: 1;
            font-size: inherit;
            font-weight: inherit;
            background: rgba(255, 255, 255, 0.1);
            border: 1px solid rgba(255, 255, 255, 0.3);
            border-radius: 4px;
            padding: 4px 8px;
            color: inherit;
        `;

        // Save original content
        const originalContent = wrapper.innerHTML;

        // Replace content with input
        wrapper.innerHTML = '';
        wrapper.appendChild(input);
        input.focus();
        input.select();

        // Save on blur or Enter key
        const saveLabel = async () => {
            const newLabel = input.value.trim();

            // Update label in memory
            interfaceLabels[interfaceName] = newLabel;

            // Save to server
            try {
                const response = await fetch('/api/config/labels', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ [interfaceName]: newLabel })
                });

                if (response.ok) {
                    console.log(`Label saved: ${interfaceName} -> ${newLabel}`);
                } else {
                    console.error('Failed to save label');
                }
            } catch (error) {
                console.error('Error saving label:', error);
            }

            // Update the display without recreating the card
            updateInterfaceNameDisplay(interfaceName, wrapper);
        };

        let saved = false;
        input.addEventListener('blur', () => {
            if (!saved) {
                saved = true;
                saveLabel();
            }
        });
        input.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                saved = true;
                saveLabel();
            }
        });
    };

    // Double-click on name to edit
    nameElement.addEventListener('dblclick', startEdit);

    // Click edit button to edit
    editBtn.addEventListener('click', startEdit);
}

// ============================================================================
// Chart Zoom Modal Functions
// ============================================================================

function openChartModal(interfaceName) {
    currentZoomedInterface = interfaceName;
    const modal = document.getElementById('chartModal');
    const modalTitle = document.getElementById('modalTitle');
    const displayName = getInterfaceDisplayName(interfaceName);

    modalTitle.textContent = `${displayName} - Real-time Traffic`;
    modal.classList.add('show');

    // Create modal chart if not exists
    if (!modalChart) {
        const ctx = document.getElementById('modalChart').getContext('2d');
        modalChart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [
                    {
                        label: 'Upload',
                        data: [],
                        borderColor: CHART_COLORS.upload,
                        backgroundColor: 'rgba(239, 68, 68, 0.1)',
                        borderWidth: 3,
                        tension: 0.4,
                        fill: true,
                        pointRadius: 0,
                        pointHoverRadius: 6
                    },
                    {
                        label: 'Download',
                        data: [],
                        borderColor: CHART_COLORS.download,
                        backgroundColor: 'rgba(16, 185, 129, 0.1)',
                        borderWidth: 3,
                        tension: 0.4,
                        fill: true,
                        pointRadius: 0,
                        pointHoverRadius: 6
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                animation: {
                    duration: 300
                },
                interaction: {
                    intersect: false,
                    mode: 'index'
                },
                plugins: {
                    legend: {
                        display: false
                    },
                    tooltip: {
                        backgroundColor: 'rgba(30, 41, 59, 0.95)',
                        titleColor: CHART_COLORS.text,
                        bodyColor: '#f1f5f9',
                        borderColor: 'rgba(71, 85, 105, 0.5)',
                        borderWidth: 1,
                        padding: 16,
                        displayColors: true,
                        titleFont: {
                            size: 14
                        },
                        bodyFont: {
                            size: 13
                        },
                        callbacks: {
                            label: function(context) {
                                return context.dataset.label + ': ' + formatBytes(context.parsed.y);
                            }
                        }
                    }
                },
                scales: {
                    x: {
                        display: true,
                        grid: {
                            color: CHART_COLORS.grid,
                            drawBorder: false
                        },
                        ticks: {
                            color: CHART_COLORS.text,
                            maxTicksLimit: 12,
                            maxRotation: 0,
                            font: {
                                size: 12
                            }
                        }
                    },
                    y: {
                        display: true,
                        beginAtZero: true,
                        grid: {
                            color: CHART_COLORS.grid,
                            drawBorder: false
                        },
                        ticks: {
                            color: CHART_COLORS.text,
                            callback: function(value) {
                                return (value * 8 / 1000000).toFixed(0) + ' Mbps';
                            },
                            font: {
                                size: 12
                            }
                        }
                    }
                }
            }
        });
    }

    // Copy data from original chart
    updateModalChart();

    // Update modal stats
    if (interfaceStats[interfaceName]) {
        updateModalStats(interfaceStats[interfaceName]);
    }

    // Prevent body scroll
    document.body.style.overflow = 'hidden';
}

function closeChartModal() {
    const modal = document.getElementById('chartModal');
    modal.classList.remove('show');
    currentZoomedInterface = null;

    // Restore body scroll
    document.body.style.overflow = '';
}

function updateModalChart() {
    if (!modalChart || !currentZoomedInterface) return;

    const data = chartData[currentZoomedInterface];
    if (!data) return;

    // Update modal chart with current data
    modalChart.data.labels = [...data.labels];
    modalChart.data.datasets[0].data = [...data.upload];
    modalChart.data.datasets[1].data = [...data.download];
    modalChart.update('none');
}

function updateModalStats(stats) {
    // Update current stats
    document.getElementById('modalCurrentUpload').textContent = formatBytes(stats.upload_rate);
    document.getElementById('modalCurrentDownload').textContent = formatBytes(stats.download_rate);

    // Update sustained peak (average) stats
    document.getElementById('modalAvgUpload').textContent = formatBytes(stats.upload_avg);
    document.getElementById('modalAvgDownload').textContent = formatBytes(stats.download_avg);

    // Update burst peak stats
    document.getElementById('modalPeakUpload').textContent = formatBytes(stats.upload_peak);
    document.getElementById('modalPeakDownload').textContent = formatBytes(stats.download_peak);
}

// Close modal on Escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && currentZoomedInterface) {
        closeChartModal();
    }
});

// Close modal on background click
document.getElementById('chartModal')?.addEventListener('click', (e) => {
    if (e.target.id === 'chartModal') {
        closeChartModal();
    }
});

// ============================================================================
// Stats Toggle Function
// ============================================================================

function toggleStats(interfaceName) {
    const statsContent = document.getElementById(`stats-${interfaceName}`);
    const toggleBtn = statsContent.previousElementSibling;
    const toggleIcon = toggleBtn.querySelector('.toggle-icon');

    if (statsContent.style.display === 'none' || !statsContent.style.display) {
        statsContent.style.display = 'block';
        toggleIcon.textContent = '▲';
    } else {
        statsContent.style.display = 'none';
        toggleIcon.textContent = '▼';
    }
}

// ============================================================================
// Initialize
// ============================================================================

// Start connection when page loads
loadInterfaceLabels().then(() => {
    connect();
});
