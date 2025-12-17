package report

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>kantra-ai Migration Plan</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.1/dist/chart.umd.min.js"></script>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background-color: #f5f5f5;
            color: #333;
            line-height: 1.6;
        }

        #app {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            position: relative;
        }

        header {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
        }

        header h1 {
            font-size: 24px;
            color: #2c3e50;
        }

        header p {
            font-size: 14px;
            color: #7f8c8d;
        }

        .dashboard {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin-bottom: 20px;
        }

        .metric-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            text-align: center;
        }

        .metric-value {
            font-size: 32px;
            font-weight: bold;
            color: #2c3e50;
            margin-bottom: 5px;
        }

        .metric-label {
            font-size: 14px;
            color: #7f8c8d;
        }

        .charts-container {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
            margin-bottom: 20px;
        }

        .chart-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }

        .chart-card h3 {
            font-size: 16px;
            font-weight: 600;
            color: #2c3e50;
            margin-bottom: 15px;
        }

        .chart-card canvas {
            max-height: 250px;
        }

        #phases-container {
            display: flex;
            flex-direction: column;
            gap: 15px;
        }

        .phase {
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            overflow: hidden;
            transition: all 0.2s;
        }

        .phase.approved {
            border-left: 4px solid #27ae60;
        }

        .phase.deferred {
            border-left: 4px solid #f39c12;
            opacity: 0.7;
        }

        .phase-header {
            padding: 20px;
            border-bottom: 1px solid #ecf0f1;
            display: flex;
            justify-content: space-between;
            align-items: center;
            cursor: pointer;
        }

        .phase-header h3 {
            font-size: 18px;
            color: #2c3e50;
            margin: 0;
            display: flex;
            align-items: center;
            gap: 10px;
        }

        .phase-order {
            background: #3498db;
            color: white;
            width: 28px;
            height: 28px;
            border-radius: 50%;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            font-weight: bold;
            font-size: 14px;
        }

        .status-badge {
            padding: 5px 12px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
        }

        .status-badge.approved {
            background-color: #d5f4e6;
            color: #27ae60;
        }

        .status-badge.deferred {
            background-color: #fdebd0;
            color: #f39c12;
        }

        .status-badge.pending {
            background-color: #e8f4f8;
            color: #3498db;
        }

        .phase-content {
            padding: 20px;
            display: none;
        }

        .phase.expanded .phase-content {
            display: block;
        }

        .risk-warning {
            background-color: #fff3cd;
            border: 2px solid #ff6b6b;
            border-radius: 6px;
            padding: 15px;
            margin-bottom: 20px;
        }

        .risk-warning-header {
            display: flex;
            align-items: center;
            gap: 8px;
            color: #d63031;
            font-size: 15px;
            font-weight: 700;
            margin-bottom: 12px;
        }

        .risk-warning-content p {
            margin: 0 0 15px 0;
            color: #856404;
            font-size: 14px;
            line-height: 1.6;
        }

        .phase-meta {
            display: flex;
            gap: 15px;
            margin-bottom: 15px;
            flex-wrap: wrap;
        }

        .phase-meta span {
            padding: 4px 10px;
            background-color: #ecf0f1;
            border-radius: 4px;
            font-size: 13px;
        }

        .risk {
            font-weight: 600;
        }

        .risk.high {
            background-color: #fadbd8 !important;
            color: #c0392b;
        }

        .risk.medium {
            background-color: #fdebd0 !important;
            color: #f39c12;
        }

        .risk.low {
            background-color: #d5f4e6 !important;
            color: #27ae60;
        }

        .phase-stats {
            display: flex;
            gap: 15px;
            margin-bottom: 15px;
            font-size: 14px;
            color: #7f8c8d;
            flex-wrap: wrap;
        }

        .phase-explanation {
            margin-bottom: 15px;
            color: #34495e;
            line-height: 1.6;
            padding: 12px;
            background-color: #f8f9fa;
            border-left: 3px solid #3498db;
            border-radius: 4px;
        }

        .phase-violations h4 {
            font-size: 14px;
            font-weight: 600;
            margin-bottom: 10px;
            color: #2c3e50;
        }

        .violation-detail {
            background-color: #f8f9fa;
            border-left: 3px solid #3498db;
            padding: 15px;
            margin-bottom: 15px;
            border-radius: 4px;
        }

        .violation-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
        }

        .violation-header strong {
            color: #2c3e50;
            font-size: 14px;
        }

        .violation-meta {
            font-size: 12px;
            color: #7f8c8d;
        }

        .violation-description {
            color: #555;
            font-size: 13px;
            line-height: 1.6;
            margin-bottom: 15px;
        }

        .incidents-list h5 {
            font-size: 13px;
            font-weight: 600;
            color: #34495e;
            margin-bottom: 10px;
            margin-top: 15px;
        }

        .incident-item {
            background-color: white;
            padding: 10px;
            margin-bottom: 8px;
            border-radius: 3px;
            border: 1px solid #e1e8ed;
        }

        .incident-location {
            display: flex;
            align-items: center;
            gap: 5px;
            margin-bottom: 5px;
        }

        .incident-location code {
            background-color: #ecf0f1;
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 12px;
            font-family: 'Monaco', 'Courier New', monospace;
            color: #2c3e50;
        }

        .line-number {
            color: #e74c3c;
            font-weight: 600;
            font-size: 12px;
        }

        .diff-container {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 10px;
            padding: 15px;
            background-color: #fafafa;
            margin: 12px 0;
        }

        .diff-pane {
            border-radius: 4px;
            overflow: hidden;
            border: 1px solid #ddd;
        }

        .diff-header {
            padding: 8px 12px;
            font-weight: 600;
            font-size: 12px;
            display: flex;
            align-items: center;
            gap: 6px;
        }

        .before-pane .diff-header {
            background-color: #fadbd8;
            color: #c0392b;
            border-bottom: 2px solid #e74c3c;
        }

        .after-pane .diff-header {
            background-color: #c8e6c9;
            color: #1b5e20;
            border-bottom: 2px solid #27ae60;
        }

        .diff-pane pre {
            margin: 0;
            padding: 12px;
            overflow-x: auto;
            font-size: 12px;
            line-height: 1.5;
        }

        .before-pane pre {
            background-color: #ffeef0 !important;
        }

        .after-pane pre {
            background-color: #e8f5e9 !important;
        }

        .diff-pane code {
            font-family: 'Monaco', 'Courier New', 'Consolas', monospace;
            color: #2c3e50;
        }

        .code-snippet {
            background: #2c3e50;
            color: #ecf0f1;
            padding: 12px;
            border-radius: 4px;
            overflow-x: auto;
            font-family: 'Monaco', 'Courier New', 'Consolas', monospace;
            font-size: 12px;
            line-height: 1.5;
            margin: 8px 0;
        }

        .highlighted-line {
            background: #3d2a1f;
            border-left: 3px solid #f0ab00;
            display: block;
            margin-left: -12px;
            padding-left: 9px;
            margin-right: -12px;
            padding-right: 12px;
        }

        .expand-toggle {
            color: #3498db;
            cursor: pointer;
            font-size: 13px;
            text-decoration: none;
            display: inline-block;
            margin-top: 8px;
        }

        .expand-toggle:hover {
            text-decoration: underline;
        }

        .arrow {
            display: inline-block;
            transition: transform 0.2s;
            font-size: 10px;
        }

        .phase.expanded .phase-header .arrow {
            transform: rotate(90deg);
        }

        .show-first-3 .incident-item:nth-child(n+4) {
            display: none;
        }

        footer {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-top: 20px;
            text-align: center;
            color: #7f8c8d;
            font-size: 14px;
        }

        @media (max-width: 768px) {
            header {
                flex-direction: column;
                gap: 15px;
                align-items: stretch;
            }

            header h1 {
                font-size: 20px;
                text-align: center;
            }

            .dashboard {
                grid-template-columns: repeat(2, 1fr);
            }

            .charts-container {
                grid-template-columns: 1fr;
            }

            .chart-card canvas {
                max-height: 300px;
            }

            .phase-meta,
            .phase-stats {
                flex-wrap: wrap;
            }

            .diff-container {
                grid-template-columns: 1fr;
            }
        }

        @media (max-width: 480px) {
            #app {
                padding: 10px;
            }

            .dashboard {
                grid-template-columns: 1fr;
                gap: 10px;
            }

            .metric-card {
                padding: 15px;
            }

            .metric-value {
                font-size: 24px;
            }
        }
    </style>
</head>
<body>
    <div id="app">
        <header>
            <div>
                <h1>kantra-ai Migration Plan</h1>
                <p>Generated: {{.Plan.Metadata.CreatedAt.Format "January 2, 2006 at 3:04 PM"}}</p>
            </div>
        </header>

        <!-- Dashboard -->
        <div class="dashboard">
            <div class="metric-card">
                <div class="metric-value">{{len .Plan.Phases}}</div>
                <div class="metric-label">Total Phases</div>
            </div>

            <div class="metric-card">
                <div class="metric-value">{{.Plan.Metadata.TotalViolations}}</div>
                <div class="metric-label">Total Violations</div>
            </div>

            <div class="metric-card">
                <div class="metric-value">{{.TotalIncidents}}</div>
                <div class="metric-label">Total Incidents</div>
            </div>

            <div class="metric-card">
                <div class="metric-value">${{printf "%.2f" .TotalCost}}</div>
                <div class="metric-label">Estimated Cost</div>
            </div>
        </div>

        <!-- Charts -->
        <div class="charts-container">
            <div class="chart-card">
                <h3>Complexity Distribution</h3>
                <canvas id="complexity-chart"></canvas>
            </div>
            <div class="chart-card">
                <h3>Violations by Category</h3>
                <canvas id="category-chart"></canvas>
            </div>
            <div class="chart-card">
                <h3>Risk Distribution</h3>
                <canvas id="risk-chart"></canvas>
            </div>
        </div>

        <!-- Phases -->
        <div id="phases-container">
            {{range $index, $phase := .Plan.Phases}}
            <div class="phase{{if $phase.Deferred}} deferred{{else}} approved{{end}}" id="phase-{{$phase.ID}}">
                <div class="phase-header" onclick="togglePhase('{{$phase.ID}}')">
                    <h3>
                        <span class="phase-order">{{$phase.Order}}</span>
                        {{$phase.Name}}
                    </h3>
                    <div style="display: flex; align-items: center; gap: 10px;">
                        {{if $phase.Deferred}}
                        <span class="status-badge deferred">↷ Deferred</span>
                        {{else}}
                        <span class="status-badge approved">✓ Approved</span>
                        {{end}}
                        <span class="arrow">▶</span>
                    </div>
                </div>

                <div class="phase-content">
                    {{if eq $phase.Risk "high"}}
                    <div class="risk-warning">
                        <div class="risk-warning-header">
                            ⚠️ High Risk Phase - Review Carefully
                        </div>
                        <div class="risk-warning-content">
                            <p>This phase contains complex changes that may require manual review.</p>
                        </div>
                    </div>
                    {{end}}

                    <div class="phase-meta">
                        <span class="risk {{$phase.Risk}}">{{$phase.Risk}} risk</span>
                        <span>{{$phase.Category}}</span>
                        <span>Effort: {{index $phase.EffortRange 0}}-{{index $phase.EffortRange 1}}</span>
                    </div>

                    <div class="phase-stats">
                        <span>💰 Cost: ${{printf "%.2f" $phase.EstimatedCost}}</span>
                        <span>⏱️ Duration: {{$phase.EstimatedDurationMinutes}} min</span>
                        <span>🔧 Violations: {{len $phase.Violations}}</span>
                    </div>

                    <div class="phase-explanation">
                        <strong>Why this grouping:</strong> {{$phase.Explanation}}
                    </div>

                    <div class="phase-violations">
                        <h4>Violations ({{len $phase.Violations}})</h4>
                        {{range $vIndex, $violation := $phase.Violations}}
                        <div class="violation-detail">
                            <div class="violation-header">
                                <strong>{{$violation.ViolationID}}</strong>
                                <span class="violation-meta">{{$violation.IncidentCount}} incidents • Effort: {{$violation.Effort}}</span>
                            </div>
                            <div class="violation-description">{{$violation.Description}}</div>

                            <div class="incidents-list">
                                <h5>Incidents (showing first 3 of {{$violation.IncidentCount}})</h5>
                                <div class="incidents show-first-3" id="incidents-{{$phase.ID}}-{{$vIndex}}">
                                    {{range $iIndex, $incident := $violation.Incidents}}
                                    <div class="incident-item">
                                        <div class="incident-location">
                                            <code>{{$incident.URI}}</code>
                                            <span class="line-number">:{{$incident.LineNumber}}</span>
                                        </div>
                                        {{if $incident.Message}}
                                        {{formatDiff $incident.Message}}
                                        {{end}}
                                        {{if $incident.CodeSnip}}
                                        {{highlightLine $incident.CodeSnip $incident.LineNumber}}
                                        {{end}}
                                    </div>
                                    {{end}}
                                </div>
                                {{if gt $violation.IncidentCount 3}}
                                <a class="expand-toggle" onclick="toggleAllIncidents('{{$phase.ID}}-{{$vIndex}}', {{$violation.IncidentCount}}); return false;" href="#">
                                    ▼ Show all {{$violation.IncidentCount}} incidents
                                </a>
                                {{end}}
                            </div>
                        </div>
                        {{end}}
                    </div>
                </div>
            </div>
            {{end}}
        </div>

        <footer>
            Generated by kantra-ai | Provider: {{.Plan.Metadata.Provider}}
        </footer>
    </div>

    <script>
        function togglePhase(phaseId) {
            const phase = document.getElementById('phase-' + phaseId);
            phase.classList.toggle('expanded');
        }

        function toggleAllIncidents(violationId, totalCount) {
            const incidents = document.getElementById('incidents-' + violationId);
            const toggle = event.target;

            incidents.classList.toggle('show-first-3');

            if (incidents.classList.contains('show-first-3')) {
                toggle.textContent = '▼ Show all ' + totalCount + ' incidents';
            } else {
                toggle.textContent = '▲ Show less';
            }
        }

        // Expand first non-deferred phase by default
        document.addEventListener('DOMContentLoaded', function() {
            const firstPhase = document.querySelector('.phase:not(.deferred)');
            if (firstPhase) {
                firstPhase.classList.add('expanded');
            }

            // Render charts
            renderCharts();
        });

        function renderCharts() {
            renderComplexityChart();
            renderCategoryChart();
            renderRiskChart();
        }

        function renderComplexityChart() {
            // Map effort to complexity levels
            const effortToComplexity = (effort) => {
                if (effort <= 1) return 'trivial';
                if (effort <= 3) return 'low';
                if (effort <= 5) return 'medium';
                if (effort <= 7) return 'high';
                return 'expert';
            };

            // Aggregate violations by complexity from template data
            const complexityCount = {
                'trivial': 0,
                'low': 0,
                'medium': 0,
                'high': 0,
                'expert': 0
            };

            // Build effort array from template
            const efforts = [
                {{range $pIndex, $phase := .Plan.Phases}}
                    {{range $vIndex, $violation := $phase.Violations}}
                        {{$violation.Effort}},
                    {{end}}
                {{end}}
            ];

            efforts.forEach(effort => {
                const complexity = effortToComplexity(effort);
                complexityCount[complexity]++;
            });

            // Filter out zero counts and prepare data
            const labels = [];
            const data = [];
            const colors = {
                'trivial': '#27ae60',
                'low': '#2ecc71',
                'medium': '#f39c12',
                'high': '#e74c3c',
                'expert': '#c0392b'
            };
            const backgroundColors = [];

            Object.keys(complexityCount).forEach(key => {
                if (complexityCount[key] > 0) {
                    labels.push(key.charAt(0).toUpperCase() + key.slice(1));
                    data.push(complexityCount[key]);
                    backgroundColors.push(colors[key]);
                }
            });

            const ctx = document.getElementById('complexity-chart');
            new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: labels,
                    datasets: [{
                        data: data,
                        backgroundColor: backgroundColors,
                        borderWidth: 2,
                        borderColor: '#fff'
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: true,
                    plugins: {
                        legend: {
                            position: 'bottom',
                            labels: {
                                padding: 10,
                                font: { size: 11 }
                            }
                        },
                        tooltip: {
                            callbacks: {
                                label: function(context) {
                                    const label = context.label || '';
                                    const value = context.parsed || 0;
                                    const total = context.dataset.data.reduce((a, b) => a + b, 0);
                                    const percentage = Math.round((value / total) * 100);
                                    return label + ': ' + value + ' (' + percentage + '%)';
                                }
                            }
                        }
                    }
                }
            });
        }

        function renderCategoryChart() {
            // Aggregate violations by category
            const categoryCount = {};

            // Build category array from template
            const categories = [
                {{range $pIndex, $phase := .Plan.Phases}}
                    {{range $vIndex, $violation := $phase.Violations}}
                        '{{$violation.Category}}',
                    {{end}}
                {{end}}
            ];

            categories.forEach(category => {
                categoryCount[category] = (categoryCount[category] || 0) + 1;
            });

            const ctx = document.getElementById('category-chart');
            new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: Object.keys(categoryCount).map(k => k.charAt(0).toUpperCase() + k.slice(1)),
                    datasets: [{
                        label: 'Violations',
                        data: Object.values(categoryCount),
                        backgroundColor: '#3498db',
                        borderColor: '#2980b9',
                        borderWidth: 1
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: true,
                    plugins: {
                        legend: { display: false },
                        tooltip: {
                            callbacks: {
                                label: function(context) {
                                    return 'Violations: ' + context.parsed.y;
                                }
                            }
                        }
                    },
                    scales: {
                        y: {
                            beginAtZero: true,
                            ticks: { stepSize: 1 }
                        }
                    }
                }
            });
        }

        function renderRiskChart() {
            // Aggregate phases by risk level
            const riskCount = { low: 0, medium: 0, high: 0 };

            // Build risk array from template
            const risks = [
                {{range $index, $phase := .Plan.Phases}}
                    '{{$phase.Risk}}',
                {{end}}
            ];

            risks.forEach(risk => {
                const riskLower = risk.toLowerCase();
                if (riskCount.hasOwnProperty(riskLower)) {
                    riskCount[riskLower]++;
                }
            });

            const ctx = document.getElementById('risk-chart');
            new Chart(ctx, {
                type: 'pie',
                data: {
                    labels: ['Low Risk', 'Medium Risk', 'High Risk'],
                    datasets: [{
                        data: [riskCount.low, riskCount.medium, riskCount.high],
                        backgroundColor: ['#27ae60', '#f39c12', '#e74c3c'],
                        borderWidth: 2,
                        borderColor: '#fff'
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: true,
                    plugins: {
                        legend: {
                            position: 'bottom',
                            labels: {
                                padding: 10,
                                font: { size: 11 }
                            }
                        },
                        tooltip: {
                            callbacks: {
                                label: function(context) {
                                    const label = context.label || '';
                                    const value = context.parsed || 0;
                                    return label + ': ' + value + ' phase' + (value !== 1 ? 's' : '');
                                }
                            }
                        }
                    }
                }
            });
        }
    </script>
</body>
</html>
`
