// WebSocket connection
let ws;
let reconnectInterval = 3000;
let charts = {};
let chartData = {};

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
                    display: true,
                    position: 'top',
                    labels: {
                        color: CHART_COLORS.text,
                        usePointStyle: true,
                        padding: 15,
                        font: {
                            size: 12
                        }
                    }
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
}

function updateDisplay(data) {
    // Update timestamp
    const time = new Date(data.timestamp).toLocaleString();
    document.getElementById('timestamp').textContent = 'Last update: ' + time;

    // Update interfaces
    const container = document.getElementById('interfaces');

    for (const [name, stats] of Object.entries(data.interfaces)) {
        let card = document.getElementById('card-' + name);

        // Create card if it doesn't exist
        if (!card) {
            card = createInterfaceCard(name);
            container.appendChild(card);

            // Create chart
            const canvasId = 'chart-' + name;
            createChart(canvasId, name);
        }

        // Update current stats badges
        card.querySelector('.current-upload').textContent = formatBytes(stats.upload_rate);
        card.querySelector('.current-download').textContent = formatBytes(stats.download_rate);

        // Update metrics
        card.querySelector('.avg-upload').textContent = formatBytes(stats.upload_avg);
        card.querySelector('.avg-download').textContent = formatBytes(stats.download_avg);
        card.querySelector('.peak-upload').textContent = formatBytes(stats.upload_peak);
        card.querySelector('.peak-download').textContent = formatBytes(stats.download_peak);

        // Update chart
        updateChart(name, data.timestamp, stats.upload_rate, stats.download_rate);
    }
}

function createInterfaceCard(name) {
    const card = document.createElement('div');
    card.className = 'interface-card';
    card.id = 'card-' + name;

    card.innerHTML = `
        <div class="interface-header">
            <div class="interface-name">${name}</div>
            <div class="interface-stats">
                <div class="stat-badge upload">
                    <div class="stat-label">↑ Upload</div>
                    <div class="stat-value current-upload">0 Mbps</div>
                </div>
                <div class="stat-badge download">
                    <div class="stat-label">↓ Download</div>
                    <div class="stat-value current-download">0 Mbps</div>
                </div>
            </div>
        </div>

        <div class="chart-container">
            <canvas id="chart-${name}"></canvas>
        </div>

        <div class="metrics-row">
            <div class="metric-item upload">
                <span class="metric-label">↑ Average Upload</span>
                <span class="metric-value avg-upload">0 Mbps</span>
            </div>
            <div class="metric-item download">
                <span class="metric-label">↓ Average Download</span>
                <span class="metric-value avg-download">0 Mbps</span>
            </div>
            <div class="metric-item upload">
                <span class="metric-label">↑ Peak Upload</span>
                <span class="metric-value peak-upload">0 Mbps</span>
            </div>
            <div class="metric-item download">
                <span class="metric-label">↓ Peak Download</span>
                <span class="metric-value peak-download">0 Mbps</span>
            </div>
        </div>
    `;

    return card;
}

// Start connection when page loads
connect();
