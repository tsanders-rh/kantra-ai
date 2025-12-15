package report

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Migration Plan Report</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #151515;
            background: #f5f5f5;
            padding: 20px;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            border-radius: 8px;
        }

        header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px;
            border-radius: 8px 8px 0 0;
        }

        header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
        }

        header p {
            font-size: 1.1em;
            opacity: 0.9;
        }

        .content {
            padding: 40px;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 40px;
        }

        .stat-card {
            background: #f8f9fa;
            padding: 24px;
            border-radius: 8px;
            border-left: 4px solid #667eea;
            cursor: help;
        }

        .stat-card h3 {
            font-size: 0.9em;
            color: #6a6e73;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 8px;
        }

        .stat-card .value {
            font-size: 2.5em;
            font-weight: bold;
            color: #151515;
        }

        .stat-card .subtext {
            font-size: 0.9em;
            color: #6a6e73;
            margin-top: 4px;
        }

        .stat-card .subtext[title] {
            cursor: help;
        }

        .section {
            margin-bottom: 40px;
        }

        .section-title {
            font-size: 1.8em;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 2px solid #e0e0e0;
        }

        .phase {
            background: #fff;
            border: 1px solid #e0e0e0;
            border-radius: 8px;
            margin-bottom: 20px;
            overflow: hidden;
        }

        .phase-header {
            padding: 20px;
            background: #f8f9fa;
            cursor: pointer;
            display: flex;
            justify-content: space-between;
            align-items: center;
            transition: background 0.2s;
        }

        .phase-header:hover {
            background: #e9ecef;
        }

        .phase-header h3 {
            font-size: 1.3em;
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .phase-order {
            background: #667eea;
            color: white;
            width: 32px;
            height: 32px;
            border-radius: 50%;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            font-weight: bold;
        }

        .badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 0.85em;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .badge-risk-low {
            background: #e8f5e9;
            color: #2e7d32;
        }

        .badge-risk-medium {
            background: #fff3e0;
            color: #e65100;
        }

        .badge-risk-high {
            background: #ffebee;
            color: #c62828;
        }

        .badge-category-mandatory {
            background: #ffebee;
            color: #c62828;
        }

        .badge-category-optional {
            background: #fff3e0;
            color: #e65100;
        }

        .badge-category-potential {
            background: #e3f2fd;
            color: #1565c0;
        }

        .phase-meta {
            display: flex;
            gap: 12px;
            align-items: center;
        }

        .phase-content {
            padding: 20px;
            display: none;
        }

        .phase.expanded .phase-content {
            display: block;
        }

        .phase-stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 16px;
            margin-bottom: 20px;
            padding: 16px;
            background: #f8f9fa;
            border-radius: 6px;
        }

        .phase-stat {
            display: flex;
            flex-direction: column;
        }

        .phase-stat[title] {
            cursor: help;
        }

        .phase-stat label {
            font-size: 0.85em;
            color: #6a6e73;
            margin-bottom: 4px;
        }

        .phase-stat value {
            font-size: 1.2em;
            font-weight: 600;
            color: #151515;
        }

        .explanation {
            padding: 16px;
            background: #e3f2fd;
            border-left: 4px solid #2196f3;
            border-radius: 4px;
            margin-bottom: 20px;
        }

        .violations-table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }

        .violations-table th {
            background: #f8f9fa;
            padding: 12px;
            text-align: left;
            font-weight: 600;
            border-bottom: 2px solid #e0e0e0;
            font-size: 0.9em;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            color: #6a6e73;
        }

        .violations-table td {
            padding: 12px;
            border-bottom: 1px solid #e0e0e0;
        }

        .violations-table tr:hover {
            background: #f8f9fa;
        }

        .violation-expandable {
            cursor: pointer;
        }

        .violation-details {
            display: none;
            padding: 16px;
            background: #f8f9fa;
            margin: 8px 0;
            border-radius: 4px;
        }

        .violation-details.expanded {
            display: block;
        }

        .incident {
            background: white;
            border: 1px solid #e0e0e0;
            border-radius: 4px;
            padding: 12px;
            margin-bottom: 12px;
        }

        .incident-header {
            font-weight: 600;
            margin-bottom: 8px;
            color: #667eea;
        }

        .incident-message {
            color: #6a6e73;
            font-size: 0.9em;
            margin-bottom: 12px;
            white-space: pre-wrap;
        }

        .code-snippet {
            background: #1e1e1e;
            color: #d4d4d4;
            padding: 16px;
            border-radius: 4px;
            overflow-x: auto;
            font-family: 'Courier New', Courier, monospace;
            font-size: 0.85em;
            line-height: 1.5;
        }

        .diff-container {
            font-family: 'Courier New', Courier, monospace;
            font-size: 0.85em;
            line-height: 1.5;
            margin: 12px 0;
        }

        .diff-section {
            margin: 8px 0;
        }

        .diff-label {
            font-weight: bold;
            margin-bottom: 4px;
            font-size: 0.9em;
        }

        .diff-removal {
            background: #2d1517;
            border-left: 3px solid #f85149;
            padding: 12px;
            border-radius: 4px;
        }

        .diff-removal .diff-label {
            color: #f85149;
        }

        .diff-removal code {
            color: #ffa198;
            background: transparent;
        }

        .diff-addition {
            background: #162d1a;
            border-left: 3px solid #3fb950;
            padding: 12px;
            border-radius: 4px;
        }

        .diff-addition .diff-label {
            color: #3fb950;
        }

        .diff-addition code {
            color: #7ee787;
            background: transparent;
        }

        .diff-code {
            white-space: pre-wrap;
            word-break: break-word;
            margin: 0;
        }

        .diff-code code {
            padding: 0;
        }

        .expand-toggle {
            color: #667eea;
            cursor: pointer;
            font-size: 0.9em;
            margin-top: 12px;
            display: inline-block;
        }

        .expand-toggle:hover {
            text-decoration: underline;
        }

        .arrow {
            display: inline-block;
            transition: transform 0.2s;
        }

        .phase.expanded .phase-header .arrow {
            transform: rotate(90deg);
        }

        footer {
            padding: 20px 40px;
            background: #f8f9fa;
            border-top: 1px solid #e0e0e0;
            text-align: center;
            color: #6a6e73;
            font-size: 0.9em;
        }

        .show-first-3 .incident:nth-child(n+4) {
            display: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>Migration Plan Report</h1>
            <p>Generated: {{.Plan.Metadata.CreatedAt.Format "January 2, 2006 at 3:04 PM"}}</p>
        </header>

        <div class="content">
            <!-- Summary Statistics -->
            <div class="stats-grid">
                <div class="stat-card" title="Number of sequential phases in the migration plan">
                    <h3>Total Phases</h3>
                    <div class="value">{{len .Plan.Phases}}</div>
                    <div class="subtext">Migration phases</div>
                </div>

                <div class="stat-card" title="Unique violation types identified by Kantra analysis">
                    <h3>Total Violations</h3>
                    <div class="value">{{.Plan.Metadata.TotalViolations}}</div>
                    <div class="subtext">Across all phases</div>
                </div>

                <div class="stat-card" title="Total number of code locations that need to be fixed">
                    <h3>Total Incidents</h3>
                    <div class="value">{{.TotalIncidents}}</div>
                    <div class="subtext">Code locations to fix</div>
                </div>

                <div class="stat-card" title="Estimated AI API cost to execute all fixes">
                    <h3>Estimated Cost</h3>
                    <div class="value">${{printf "%.2f" .TotalCost}}</div>
                    <div class="subtext" title="Estimated time for AI to analyze and apply all fixes">{{.TotalDuration}} minutes</div>
                </div>
            </div>

            <!-- Phases -->
            <div class="section">
                <h2 class="section-title">Migration Phases</h2>

                {{range $index, $phase := .Plan.Phases}}
                <div class="phase" id="phase-{{$phase.ID}}">
                    <div class="phase-header" onclick="togglePhase('{{$phase.ID}}')">
                        <h3>
                            <span class="phase-order">{{$phase.Order}}</span>
                            {{$phase.Name}}
                        </h3>
                        <div class="phase-meta">
                            <span class="badge badge-risk-{{$phase.Risk}}">{{$phase.Risk}} risk</span>
                            <span class="badge badge-category-{{$phase.Category}}">{{$phase.Category}}</span>
                            <span class="arrow">▶</span>
                        </div>
                    </div>

                    <div class="phase-content">
                        <div class="phase-stats">
                            <div class="phase-stat" title="Number of unique violation types in this phase">
                                <label>Violations</label>
                                <value>{{len $phase.Violations}}</value>
                            </div>
                            <div class="phase-stat" title="Effort level range: 1 (trivial) to 13+ (very complex)">
                                <label>Effort Range</label>
                                <value>{{index $phase.EffortRange 0}} - {{index $phase.EffortRange 1}}</value>
                            </div>
                            <div class="phase-stat" title="Estimated AI API cost for this phase">
                                <label>Estimated Cost</label>
                                <value>${{printf "%.2f" $phase.EstimatedCost}}</value>
                            </div>
                            <div class="phase-stat" title="Estimated execution time for this phase">
                                <label>Duration</label>
                                <value>{{$phase.EstimatedDurationMinutes}} min</value>
                            </div>
                        </div>

                        <div class="explanation">
                            <strong>Rationale:</strong> {{$phase.Explanation}}
                        </div>

                        <h4>Violations ({{len $phase.Violations}})</h4>
                        <table class="violations-table">
                            <thead>
                                <tr>
                                    <th>Violation ID</th>
                                    <th>Description</th>
                                    <th>Category</th>
                                    <th>Effort</th>
                                    <th>Incidents</th>
                                    <th></th>
                                </tr>
                            </thead>
                            <tbody>
                                {{range $vIndex, $violation := $phase.Violations}}
                                <tr>
                                    <td><code>{{$violation.ViolationID}}</code></td>
                                    <td>{{truncate $violation.Description 80}}</td>
                                    <td><span class="badge badge-category-{{$violation.Category}}">{{$violation.Category}}</span></td>
                                    <td>{{$violation.Effort}}</td>
                                    <td>{{$violation.IncidentCount}}</td>
                                    <td>
                                        <span class="expand-toggle" onclick="toggleViolation('{{$phase.ID}}-{{$vIndex}}')">
                                            View Details
                                        </span>
                                    </td>
                                </tr>
                                <tr>
                                    <td colspan="6">
                                        <div class="violation-details" id="violation-{{$phase.ID}}-{{$vIndex}}">
                                            <h5>{{$violation.Description}}</h5>
                                            <p><strong>Full Description:</strong> {{$violation.Description}}</p>
                                            <p><strong>Migration Complexity:</strong> {{if $violation.MigrationComplexity}}{{$violation.MigrationComplexity}}{{else}}Not specified{{end}}</p>

                                            <h5 style="margin-top: 16px;">Incidents (showing first 3 of {{$violation.IncidentCount}})</h5>
                                            <div class="incidents show-first-3" id="incidents-{{$phase.ID}}-{{$vIndex}}">
                                                {{range $incident := $violation.Incidents}}
                                                <div class="incident">
                                                    <div class="incident-header">{{$incident.URI}} : {{$incident.LineNumber}}</div>
                                                    {{if $incident.Message}}
                                                    {{formatDiff $incident.Message}}
                                                    {{end}}
                                                    {{if $incident.CodeSnip}}
                                                    <pre class="code-snippet">{{$incident.CodeSnip}}</pre>
                                                    {{end}}
                                                </div>
                                                {{end}}
                                            </div>
                                            {{if gt $violation.IncidentCount 3}}
                                            <span class="expand-toggle" onclick="toggleAllIncidents('{{$phase.ID}}-{{$vIndex}}')">
                                                Show all {{$violation.IncidentCount}} incidents
                                            </span>
                                            {{end}}
                                        </div>
                                    </td>
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
                {{end}}
            </div>
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

        function toggleViolation(violationId) {
            const details = document.getElementById('violation-' + violationId);
            details.classList.toggle('expanded');
        }

        function toggleAllIncidents(violationId) {
            const incidents = document.getElementById('incidents-' + violationId);
            incidents.classList.toggle('show-first-3');

            const toggle = event.target;
            if (incidents.classList.contains('show-first-3')) {
                toggle.textContent = 'Show all incidents';
            } else {
                toggle.textContent = 'Show less';
            }
        }

        // Expand first phase by default
        document.addEventListener('DOMContentLoaded', function() {
            const firstPhase = document.querySelector('.phase');
            if (firstPhase) {
                firstPhase.classList.add('expanded');
            }
        });
    </script>
</body>
</html>
`
