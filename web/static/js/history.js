// ============================================================================
// Historical Data Query Page
// ============================================================================

let historyChart = null;

// Chart configuration
const CHART_COLORS = {
    upload: 'rgb(239, 68, 68)',      // Red
    download: 'rgb(16, 185, 129)',   // Green
    grid: 'rgba(71, 85, 105, 0.3)',
    text: 'rgb(148, 163, 184)'
};

// ============================================================================
// Initialization
// ============================================================================

document.addEventListener('DOMContentLoaded', () => {
    // Get interface from URL parameter
    const urlParams = new URLSearchParams(window.location.search);
    const interfaceName = urlParams.get('interface');

    if (interfaceName) {
        const select = document.getElementById('historyInterface');
        const option = document.createElement('option');
        option.value = interfaceName;
        option.textContent = interfaceName;
        select.appendChild(option);
        select.value = interfaceName;
    }

    // Load available interfaces
    loadAvailableInterfaces();

    // Setup time range buttons
    setupTimeRangeButtons();

    // Auto-load data if interface is specified
    if (interfaceName) {
        setTimeout(loadHistoricalData, 500);
    }
});

// ============================================================================
// Interface Loading
// ============================================================================

async function loadAvailableInterfaces() {
    try {
        const response = await fetch('/api/current');
        const data = await response.json();

        const select = document.getElementById('historyInterface');
        const currentValue = select.value;

        // Clear existing options
        select.innerHTML = '';

        // Add all interfaces
        const interfaces = Object.keys(data.interfaces).sort();
        interfaces.forEach(iface => {
            const option = document.createElement('option');
            option.value = iface;
            option.textContent = iface;
            select.appendChild(option);
        });

        // Restore selection if possible
        if (currentValue && interfaces.includes(currentValue)) {
            select.value = currentValue;
        }
    } catch (error) {
        console.error('Failed to load interfaces:', error);
    }
}

// ============================================================================
// Time Range Handling
// ============================================================================

function setupTimeRangeButtons() {
    const timeButtons = document.querySelectorAll('.time-btn');
    const customRange = document.getElementById('customRange');

    timeButtons.forEach(btn => {
        btn.addEventListener('click', function() {
            // Remove active class from all buttons
            timeButtons.forEach(b => b.classList.remove('active'));
            // Add active class to clicked button
            this.classList.add('active');

            // Show/hide custom range inputs
            if (this.dataset.range === 'custom') {
                customRange.style.display = 'block';
                initializeDateInputs();
            } else {
                customRange.style.display = 'none';
            }
        });
    });
}

function initializeDateInputs() {
    const now = new Date();
    const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000);

    // Format for datetime-local input: YYYY-MM-DDTHH:MM
    const formatDateTime = (date) => {
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        const hours = String(date.getHours()).padStart(2, '0');
        const minutes = String(date.getMinutes()).padStart(2, '0');
        return `${year}-${month}-${day}T${hours}:${minutes}`;
    };

    document.getElementById('customStart').value = formatDateTime(yesterday);
    document.getElementById('customEnd').value = formatDateTime(now);
}

function getTimeRange() {
    const activeBtn = document.querySelector('.time-btn.active');
    const range = activeBtn.dataset.range;

    const end = new Date();
    let start;

    switch (range) {
        case '1h':
            start = new Date(end.getTime() - 1 * 60 * 60 * 1000);
            break;
        case '6h':
            start = new Date(end.getTime() - 6 * 60 * 60 * 1000);
            break;
        case '24h':
            start = new Date(end.getTime() - 24 * 60 * 60 * 1000);
            break;
        case '7d':
            start = new Date(end.getTime() - 7 * 24 * 60 * 60 * 1000);
            break;
        case '30d':
            start = new Date(end.getTime() - 30 * 24 * 60 * 60 * 1000);
            break;
        case 'custom':
            start = new Date(document.getElementById('customStart').value);
            // Don't modify end for custom range
            const customEnd = document.getElementById('customEnd').value;
            if (customEnd) {
                return { start, end: new Date(customEnd) };
            }
            break;
        default:
            start = new Date(end.getTime() - 24 * 60 * 60 * 1000);
    }

    return { start, end };
}

// ============================================================================
// Data Loading
// ============================================================================

async function loadHistoricalData() {
    const interfaceEl = document.getElementById('historyInterface');
    const intervalEl = document.getElementById('historyInterval');
    const queryBtn = document.getElementById('queryBtn');
    const btnText = queryBtn.querySelector('.btn-text');
    const spinner = queryBtn.querySelector('.loading-spinner');

    const interfaceName = interfaceEl.value;
    const interval = intervalEl.value;

    if (!interfaceName) {
        alert('Please select an interface');
        return;
    }

    // Get time range
    const { start, end } = getTimeRange();

    // Show loading state
    queryBtn.disabled = true;
    btnText.style.display = 'none';
    spinner.style.display = 'inline';

    try {
        // Build API URL
        const params = new URLSearchParams({
            interface: interfaceName,
            start: Math.floor(start.getTime() / 1000),
            end: Math.floor(end.getTime() / 1000),
            interval: interval
        });

        const response = await fetch(`/api/history?${params}`);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${await response.text()}`);
        }

        const data = await response.json();
        displayHistoricalChart(data);
        displayHistoricalStats(data);

    } catch (error) {
        console.error('Failed to load historical data:', error);
        alert('Failed to load historical data: ' + error.message);
    } finally {
        // Hide loading state
        queryBtn.disabled = false;
        btnText.style.display = 'inline';
        spinner.style.display = 'none';
    }
}

// ============================================================================
// Chart Display
// ============================================================================

function formatBytes(bytes) {
    const mbps = (bytes * 8 / 1000000).toFixed(2);
    return mbps + ' Mbps';
}

function displayHistoricalChart(data) {
    const canvas = document.getElementById('historyChart');
    const ctx = canvas.getContext('2d');

    // Destroy existing chart
    if (historyChart) {
        historyChart.destroy();
    }

    // Prepare data
    const labels = data.datapoints.map(dp => new Date(dp.timestamp));
    const uploadAvg = data.datapoints.map(dp => dp.upload_avg);
    const downloadAvg = data.datapoints.map(dp => dp.download_avg);
    const uploadPeak = data.datapoints.map(dp => dp.upload_peak);
    const downloadPeak = data.datapoints.map(dp => dp.download_peak);

    // Create chart
    historyChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [
                {
                    label: 'Upload (Avg)',
                    data: uploadAvg,
                    borderColor: CHART_COLORS.upload,
                    backgroundColor: 'rgba(239, 68, 68, 0.1)',
                    borderWidth: 2,
                    tension: 0.4,
                    fill: true,
                    pointRadius: 0,
                    pointHoverRadius: 4
                },
                {
                    label: 'Download (Avg)',
                    data: downloadAvg,
                    borderColor: CHART_COLORS.download,
                    backgroundColor: 'rgba(16, 185, 129, 0.1)',
                    borderWidth: 2,
                    tension: 0.4,
                    fill: true,
                    pointRadius: 0,
                    pointHoverRadius: 4
                },
                {
                    label: 'Upload (Peak)',
                    data: uploadPeak,
                    borderColor: 'rgba(239, 68, 68, 0.5)',
                    borderWidth: 1,
                    borderDash: [5, 5],
                    tension: 0.4,
                    fill: false,
                    pointRadius: 0,
                    pointHoverRadius: 4
                },
                {
                    label: 'Download (Peak)',
                    data: downloadPeak,
                    borderColor: 'rgba(16, 185, 129, 0.5)',
                    borderWidth: 1,
                    borderDash: [5, 5],
                    tension: 0.4,
                    fill: false,
                    pointRadius: 0,
                    pointHoverRadius: 4
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
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
                    type: 'time',
                    time: {
                        displayFormats: {
                            hour: 'MMM d, HH:mm',
                            day: 'MMM d'
                        }
                    },
                    grid: {
                        color: CHART_COLORS.grid,
                        drawBorder: false
                    },
                    ticks: {
                        color: CHART_COLORS.text,
                        maxTicksLimit: 12,
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
}

// ============================================================================
// Statistics Display
// ============================================================================

function displayHistoricalStats(data) {
    const container = document.getElementById('historyStats');

    if (data.datapoints.length === 0) {
        container.innerHTML = '<p style="text-align: center; color: var(--text-secondary);">No data available for the selected time range.</p>';
        return;
    }

    // Use server-calculated statistics if available, otherwise calculate from datapoints
    let stats;
    if (data.stats) {
        // Use accurate statistics from VictoriaMetrics aggregation queries
        stats = {
            uploadAvg: data.stats.upload_avg || 0,
            downloadAvg: data.stats.download_avg || 0,
            uploadMax: data.stats.upload_peak || 0,
            downloadMax: data.stats.download_peak || 0,
            dataPoints: data.datapoints.length
        };
    } else {
        // Fallback: calculate from returned datapoints (may be inaccurate due to sampling)
        const uploadAvgs = data.datapoints.map(dp => dp.upload_avg).filter(v => v > 0);
        const downloadAvgs = data.datapoints.map(dp => dp.download_avg).filter(v => v > 0);
        const uploadPeaks = data.datapoints.map(dp => dp.upload_peak).filter(v => v > 0);
        const downloadPeaks = data.datapoints.map(dp => dp.download_peak).filter(v => v > 0);

        stats = {
            uploadAvg: uploadAvgs.length > 0 ? uploadAvgs.reduce((a, b) => a + b, 0) / uploadAvgs.length : 0,
            downloadAvg: downloadAvgs.length > 0 ? downloadAvgs.reduce((a, b) => a + b, 0) / downloadAvgs.length : 0,
            uploadMax: uploadPeaks.length > 0 ? Math.max(...uploadPeaks) : 0,
            downloadMax: downloadPeaks.length > 0 ? Math.max(...downloadPeaks) : 0,
            dataPoints: data.datapoints.length
        };
    }

    container.innerHTML = `
        <div class="stat-card">
            <div class="stat-card-label">Average Upload</div>
            <div class="stat-card-value" style="color: var(--upload-color)">${formatBytes(stats.uploadAvg)}</div>
        </div>
        <div class="stat-card">
            <div class="stat-card-label">Average Download</div>
            <div class="stat-card-value" style="color: var(--download-color)">${formatBytes(stats.downloadAvg)}</div>
        </div>
        <div class="stat-card">
            <div class="stat-card-label">Peak Upload</div>
            <div class="stat-card-value" style="color: var(--upload-color)">${formatBytes(stats.uploadMax)}</div>
        </div>
        <div class="stat-card">
            <div class="stat-card-label">Peak Download</div>
            <div class="stat-card-value" style="color: var(--download-color)">${formatBytes(stats.downloadMax)}</div>
        </div>
        <div class="stat-card">
            <div class="stat-card-label">Data Points</div>
            <div class="stat-card-value">${stats.dataPoints}</div>
        </div>
        <div class="stat-card">
            <div class="stat-card-label">Interval</div>
            <div class="stat-card-value">${data.interval}</div>
        </div>
    `;
}
