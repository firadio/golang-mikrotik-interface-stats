// ============================================================================
// Historical Data Query Page
// ============================================================================

let historyChart = null;
let navigationStack = [];  // 导航栈，用于下钻和返回
let currentState = null;   // 当前状态

// Chart configuration
const CHART_COLORS = {
    upload: 'rgb(239, 68, 68)',      // Red
    download: 'rgb(16, 185, 129)',   // Green
    grid: 'rgba(71, 85, 105, 0.3)',
    text: 'rgb(148, 163, 184)'
};

// ============================================================================
// 采样配置和间隔管理
// ============================================================================

const SAMPLING_CONFIG = {
    MIN_POINTS: 100,  // 最小采样点
    MAX_POINTS: 500,  // 最大采样点
    DEFAULT_PREFER: 'min'  // 默认偏好最少点数（最快）
};

// 所有可用的间隔选项（从大到小排序）
const ALL_INTERVALS = [
    { value: '1d',  seconds: 86400, label: '1天' },
    { value: '12h', seconds: 43200, label: '12小时' },
    { value: '6h',  seconds: 21600, label: '6小时' },
    { value: '4h',  seconds: 14400, label: '4小时' },
    { value: '3h',  seconds: 10800, label: '3小时' },
    { value: '2h',  seconds: 7200,  label: '2小时' },
    { value: '1h',  seconds: 3600,  label: '1小时' },
    { value: '30m', seconds: 1800,  label: '30分钟' },
    { value: '20m', seconds: 1200,  label: '20分钟' },
    { value: '15m', seconds: 900,   label: '15分钟' },
    { value: '10m', seconds: 600,   label: '10分钟' },
    { value: '5m',  seconds: 300,   label: '5分钟' },
    { value: '4m',  seconds: 240,   label: '4分钟' },
    { value: '3m',  seconds: 180,   label: '3分钟' },
    { value: '2m',  seconds: 120,   label: '2分钟' },
    { value: '1m',  seconds: 60,    label: '1分钟' },
    { value: '30s', seconds: 30,    label: '30秒' },
    { value: '20s', seconds: 20,    label: '20秒' },
    { value: '10s', seconds: 10,    label: '10秒' }
];

// 下钻配置
const DRILL_DOWN_CONFIG = {
    '30d': { target: '24h', groupBy: 'day' },
    '7d':  { target: '24h', groupBy: 'day' },
    '6h':  { target: '1h',  groupBy: 'hour' },
    '24h': { target: '1h',  groupBy: 'hour' },
    '1h':  null  // 无下钻
};

/**
 * 根据时间范围自动过滤可用的间隔选项
 */
function getAvailableIntervals(startTime, endTime) {
    const durationSeconds = (endTime - startTime) / 1000;

    const intervals = ALL_INTERVALS.filter(interval => {
        const points = Math.ceil(durationSeconds / interval.seconds);
        return points >= SAMPLING_CONFIG.MIN_POINTS &&
               points <= SAMPLING_CONFIG.MAX_POINTS;
    }).map(interval => {
        const points = Math.ceil(durationSeconds / interval.seconds);
        return {
            ...interval,
            points: points,
            isDefault: false
        };
    });

    // 没有符合条件的间隔，放宽限制
    if (intervals.length === 0) {
        console.warn('No intervals in range, using fallback');
        const fallback = ALL_INTERVALS.filter(interval => {
            const points = Math.ceil(durationSeconds / interval.seconds);
            return points >= 10 && points <= 500;
        }).map(interval => ({
            ...interval,
            points: Math.ceil(durationSeconds / interval.seconds),
            isDefault: false
        }));

        if (fallback.length > 0) {
            fallback[0].isDefault = true;
        }
        return fallback;
    }

    // 设置默认值（最少点数 = 最快）
    const defaultInterval = intervals.reduce((min, curr) =>
        curr.points < min.points ? curr : min
    );
    defaultInterval.isDefault = true;

    return intervals;
}

/**
 * 获取速度指示器
 */
function getSpeedIndicator(points) {
    if (points <= 60) return '⚡⚡⚡ 极快';
    if (points <= 120) return '⚡⚡ 快';
    if (points <= 180) return '⚡ 中等';
    return '🐌 较慢';
}

/**
 * 将时间戳对齐到指定的间隔边界
 * @param {Date} date - 要对齐的时间
 * @param {string} interval - 间隔字符串 (如 "5m", "1h", "10s")
 * @param {string} direction - 对齐方向: "floor" (向下), "ceil" (向上), "round" (四舍五入)
 * @returns {Date} 对齐后的时间
 */
function alignTimestamp(date, interval, direction = 'floor') {
    const timestamp = Math.floor(date.getTime() / 1000); // 转为Unix秒

    // 解析间隔为秒数
    let intervalSeconds;
    const match = interval.match(/^(\d+)([smhd])$/);
    if (!match) {
        console.warn(`Invalid interval format: ${interval}`);
        return date;
    }

    const value = parseInt(match[1]);
    const unit = match[2];

    switch (unit) {
        case 's': intervalSeconds = value; break;
        case 'm': intervalSeconds = value * 60; break;
        case 'h': intervalSeconds = value * 3600; break;
        case 'd': intervalSeconds = value * 86400; break;
        default: return date;
    }

    // 对齐时间戳
    let alignedTimestamp;
    switch (direction) {
        case 'ceil':
            alignedTimestamp = Math.ceil(timestamp / intervalSeconds) * intervalSeconds;
            break;
        case 'round':
            alignedTimestamp = Math.round(timestamp / intervalSeconds) * intervalSeconds;
            break;
        case 'floor':
        default:
            alignedTimestamp = Math.floor(timestamp / intervalSeconds) * intervalSeconds;
            break;
    }

    return new Date(alignedTimestamp * 1000);
}

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

    // Initialize with default time range (24h)
    const defaultRange = '24h';
    document.querySelector(`.time-btn[data-range="${defaultRange}"]`).classList.add('active');
    onTimeRangeChange(defaultRange);

    // Auto-load data if interface is specified
    if (interfaceName) {
        setTimeout(() => {
            const { start, end } = currentState;
            const intervals = getAvailableIntervals(start, end);
            const defaultInterval = intervals.find(i => i.isDefault);
            if (defaultInterval) {
                loadHistoricalData();
            }
        }, 500);
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

            const range = this.dataset.range;

            // Show/hide custom range inputs
            if (range === 'custom') {
                customRange.style.display = 'block';
                initializeDateInputs();
            } else {
                customRange.style.display = 'none';
                // 更新间隔选择器
                onTimeRangeChange(range);
            }
        });
    });
}

/**
 * 时间范围改变时触发
 */
function onTimeRangeChange(range) {
    const { start, end } = getTimeRange(range);
    const intervals = getAvailableIntervals(start, end);

    // 渲染间隔选择器
    renderIntervalSelector(intervals);

    // 更新当前状态
    const defaultInterval = intervals.find(i => i.isDefault);
    currentState = {
        range: range,
        start: start,
        end: end,
        interval: defaultInterval ? defaultInterval.value : intervals[0].value,
        canDrillDown: DRILL_DOWN_CONFIG[range] !== null
    };

    // 清空导航栈（因为切换了时间范围）
    navigationStack = [currentState];
}

/**
 * 渲染间隔选择器
 */
function renderIntervalSelector(intervals) {
    const container = document.getElementById('historyInterval');
    container.innerHTML = '';

    intervals.forEach(interval => {
        const option = document.createElement('option');
        option.value = interval.value;
        option.textContent = `${interval.label} (${interval.points}点 ${getSpeedIndicator(interval.points)})`;
        if (interval.isDefault) {
            option.selected = true;
        }
        container.appendChild(option);
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

function getTimeRange(range) {
    if (!range) {
        const activeBtn = document.querySelector('.time-btn.active');
        range = activeBtn ? activeBtn.dataset.range : '24h';
    }

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

    if (!currentState) {
        alert('Please select a time range');
        return;
    }

    // 更新当前状态的间隔
    currentState.interval = interval;

    let { start, end } = currentState;

    // 对齐时间戳到间隔边界
    start = alignTimestamp(start, interval, 'floor');  // 开始时间向下对齐
    end = alignTimestamp(end, interval, 'ceil');       // 结束时间向上对齐

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

    // Check if datapoints is null or empty
    if (!data.datapoints || data.datapoints.length === 0) {
        console.warn('No data points available for chart');
        // Create empty chart with message
        historyChart = new Chart(ctx, {
            type: 'line',
            data: { labels: [], datasets: [] },
            options: {
                plugins: {
                    title: {
                        display: true,
                        text: 'No data available for the selected time range and interval',
                        color: CHART_COLORS.text,
                        font: { size: 14 }
                    }
                }
            }
        });
        return;
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

    // Check if datapoints is null or empty
    if (!data.datapoints || data.datapoints.length === 0) {
        container.innerHTML = '<p style="text-align: center; color: var(--text-secondary);">No data available for the selected time range.</p>';
        return;
    }

    // Use server-calculated statistics if available, otherwise calculate from datapoints
    let stats;
    if (data.stats && (data.stats.upload_avg > 0 || data.stats.download_avg > 0 ||
                       data.stats.upload_peak > 0 || data.stats.download_peak > 0)) {
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
            <div class="stat-card-label">Sustained Peak Upload</div>
            <div class="stat-card-value" style="color: var(--upload-color)">${formatBytes(stats.uploadAvg)}</div>
        </div>
        <div class="stat-card">
            <div class="stat-card-label">Sustained Peak Download</div>
            <div class="stat-card-value" style="color: var(--download-color)">${formatBytes(stats.downloadAvg)}</div>
        </div>
        <div class="stat-card">
            <div class="stat-card-label">Burst Peak Upload</div>
            <div class="stat-card-value" style="color: var(--upload-color)">${formatBytes(stats.uploadMax)}</div>
        </div>
        <div class="stat-card">
            <div class="stat-card-label">Burst Peak Download</div>
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
