class PlanApp {
    constructor() {
        this.plan = null;
        this.ws = null;
        this.charts = {
            complexity: null,
            category: null,
            risk: null
        };
        this.init();
    }

    async init() {
        try {
            await this.loadPlan();
            this.render();
            this.attachEventListeners();
            this.connectWebSocket();
        } catch (error) {
            console.error('Failed to initialize app:', error);
            this.showError('Failed to load plan. Please refresh the page.');
        }
    }

    async loadPlan() {
        const response = await fetch('/api/plan');
        if (!response.ok) {
            throw new Error('Failed to load plan');
        }
        this.plan = await response.json();
        console.log('Loaded plan:', this.plan);

        // Validate plan structure
        if (!this.plan.Phases) {
            throw new Error('Plan is missing Phases array');
        }
    }

    render() {
        try {
            console.log('Rendering dashboard...');
            this.renderDashboard();
            console.log('Rendering charts...');
            this.renderCharts();
            console.log('Rendering phases...');
            this.renderPhases();
            console.log('Updating progress...');
            this.updateProgress();
            console.log('Render complete!');
        } catch (error) {
            console.error('Error in render:', error);
            throw error;
        }
    }

    renderDashboard() {
        console.log('Plan Phases:', this.plan.Phases);
        const totalViolations = this.plan.Phases.reduce((sum, p) => sum + (p.Violations ? p.Violations.length : 0), 0);
        const totalIncidents = this.plan.Phases.reduce((sum, p) =>
            sum + (p.Violations ? p.Violations.reduce((vSum, v) => vSum + (v.IncidentCount || 0), 0) : 0), 0
        );
        const totalCost = this.plan.Phases.reduce((sum, p) => sum + (p.EstimatedCost || 0), 0);

        document.getElementById('total-phases').textContent = this.plan.Phases.length;
        document.getElementById('total-violations').textContent = totalViolations;
        document.getElementById('total-incidents').textContent = totalIncidents;
        document.getElementById('estimated-cost').textContent = this.formatCost(totalCost);
    }

    renderPhases() {
        const container = document.getElementById('phases-container');
        container.innerHTML = this.plan.Phases.map(phase => this.renderPhase(phase)).join('');
    }

    renderCharts() {
        this.renderComplexityChart();
        this.renderCategoryChart();
        this.renderRiskChart();
    }

    renderComplexityChart() {
        // Destroy existing chart if it exists
        if (this.charts.complexity) {
            this.charts.complexity.destroy();
        }

        // Map effort to complexity levels
        const effortToComplexity = (effort) => {
            if (effort <= 1) return 'trivial';
            if (effort <= 3) return 'low';
            if (effort <= 5) return 'medium';
            if (effort <= 7) return 'high';
            return 'expert';
        };

        // Aggregate violations by complexity (mapped from effort)
        const complexityCount = {
            'trivial': 0,
            'low': 0,
            'medium': 0,
            'high': 0,
            'expert': 0
        };

        this.plan.Phases.forEach(phase => {
            phase.Violations.forEach(v => {
                const effort = v.Effort || 0;
                const complexity = effortToComplexity(effort);
                complexityCount[complexity]++;
            });
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
        this.charts.complexity = new Chart(ctx, {
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
                            font: {
                                size: 11
                            }
                        }
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const label = context.label || '';
                                const value = context.parsed || 0;
                                const total = context.dataset.data.reduce((a, b) => a + b, 0);
                                const percentage = Math.round((value / total) * 100);
                                return `${label}: ${value} (${percentage}%)`;
                            }
                        }
                    }
                }
            }
        });
    }

    renderCategoryChart() {
        // Destroy existing chart if it exists
        if (this.charts.category) {
            this.charts.category.destroy();
        }

        // Aggregate violations by category
        const categoryCount = {};
        this.plan.Phases.forEach(phase => {
            phase.Violations.forEach(v => {
                const category = v.Category || 'unknown';
                categoryCount[category] = (categoryCount[category] || 0) + 1;
            });
        });

        const ctx = document.getElementById('category-chart');
        this.charts.category = new Chart(ctx, {
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
                    legend: {
                        display: false
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                return `Violations: ${context.parsed.y}`;
                            }
                        }
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        ticks: {
                            stepSize: 1
                        }
                    }
                }
            }
        });
    }

    renderRiskChart() {
        // Destroy existing chart if it exists
        if (this.charts.risk) {
            this.charts.risk.destroy();
        }

        // Aggregate phases by risk level
        const riskCount = {
            low: 0,
            medium: 0,
            high: 0
        };

        this.plan.Phases.forEach(phase => {
            const risk = phase.Risk.toLowerCase();
            if (riskCount.hasOwnProperty(risk)) {
                riskCount[risk]++;
            }
        });

        const ctx = document.getElementById('risk-chart');
        this.charts.risk = new Chart(ctx, {
            type: 'pie',
            data: {
                labels: ['Low Risk', 'Medium Risk', 'High Risk'],
                datasets: [{
                    data: [riskCount.low, riskCount.medium, riskCount.high],
                    backgroundColor: [
                        '#27ae60', // Low - green
                        '#f39c12', // Medium - orange
                        '#e74c3c'  // High - red
                    ],
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
                            font: {
                                size: 11
                            }
                        }
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const label = context.label || '';
                                const value = context.parsed || 0;
                                return `${label}: ${value} phase${value !== 1 ? 's' : ''}`;
                            }
                        }
                    }
                }
            }
        });
    }

    renderPhase(phase) {
        const status = phase.Deferred ? 'deferred' : (this.isPhaseApproved(phase) ? 'approved' : 'pending');
        const statusText = status.charAt(0).toUpperCase() + status.slice(1);
        const riskIcon = this.getRiskIcon(phase.Risk);

        return `
            <div class="phase ${status}" data-phase-id="${phase.ID}">
                <div class="phase-header">
                    <h3>Phase ${phase.Order}: ${this.escapeHtml(phase.Name)}</h3>
                    <span class="status-badge ${status}">${statusText}</span>
                </div>
                <div class="phase-content">
                    <div class="phase-meta">
                        <span class="risk ${phase.Risk}">${riskIcon} ${phase.Risk.toUpperCase()} RISK</span>
                        <span>${this.escapeHtml(phase.Category)}</span>
                        <span>Effort: ${phase.EffortRange[0]}-${phase.EffortRange[1]}</span>
                    </div>
                    <div class="phase-stats">
                        <span>${phase.Violations.length} violations</span>
                        <span>${phase.Violations.reduce((sum, v) => sum + v.IncidentCount, 0)} incidents</span>
                        <span>${this.formatCost(phase.EstimatedCost)}</span>
                        <span>~${phase.EstimatedDurationMinutes} minutes</span>
                    </div>
                    <p class="phase-explanation">${this.escapeHtml(phase.Explanation)}</p>
                    <div class="phase-violations">
                        <h4>Violations Summary:</h4>
                        <ul class="violations-list">
                            ${phase.Violations.map(v => `
                                <li>${this.escapeHtml(v.ViolationID)} (${v.IncidentCount} incident${v.IncidentCount !== 1 ? 's' : ''})</li>
                            `).join('')}
                        </ul>
                    </div>

                    <!-- Expandable details section -->
                    <div class="phase-details hidden" id="details-${phase.ID}">
                        <div class="details-content">
                            <h4>Detailed Violations:</h4>
                            ${phase.Violations.map(v => this.renderViolationDetails(v)).join('')}
                        </div>
                    </div>

                    <div class="phase-actions">
                        <button class="btn btn-success" onclick="app.approvePhase('${phase.ID}')">
                            <i class="fas fa-check"></i> Approve
                        </button>
                        <button class="btn btn-warning" onclick="app.deferPhase('${phase.ID}')">
                            <i class="fas fa-clock"></i> Defer
                        </button>
                        <button class="btn btn-info" onclick="app.toggleDetails('${phase.ID}')">
                            <i class="fas fa-chevron-down toggle-icon" id="toggle-icon-${phase.ID}"></i> Details
                        </button>
                    </div>
                </div>
            </div>
        `;
    }

    renderViolationDetails(violation) {
        const violationId = violation.ViolationID.replace(/[^a-zA-Z0-9]/g, '-');

        return `
            <div class="violation-detail" data-violation-id="${violationId}">
                <div class="violation-header">
                    <strong>${this.escapeHtml(violation.ViolationID)}</strong>
                    <span class="violation-meta">
                        ${violation.IncidentCount} incident${violation.IncidentCount !== 1 ? 's' : ''} •
                        Effort: ${violation.Effort}
                    </span>
                </div>
                <p class="violation-description">${this.escapeHtml(violation.Description)}</p>

                <div class="incidents-viewer">
                    <div class="incidents-nav">
                        <button class="btn-nav" onclick="app.navigateIncident('${violationId}', -1)" ${violation.Incidents.length <= 1 ? 'disabled' : ''}>
                            <i class="fas fa-chevron-left"></i> Previous
                        </button>
                        <span class="incident-counter" id="counter-${violationId}">
                            1 of ${violation.Incidents.length}
                        </span>
                        <button class="btn-nav" onclick="app.navigateIncident('${violationId}', 1)" ${violation.Incidents.length <= 1 ? 'disabled' : ''}>
                            Next <i class="fas fa-chevron-right"></i>
                        </button>
                    </div>

                    <div class="incident-display" id="incident-${violationId}">
                        ${this.renderIncident(violation.Incidents[0], 0, violation)}
                    </div>
                </div>
            </div>
        `;
    }

    renderIncident(incident, index, violation) {
        const filePath = incident.URI.replace('file://', '').replace('/opt/input/source/', '').replace('/Users/tsanders/Workspace/tackle2-ui/', '');

        let contentHtml = '';

        // Show diff view if there's a message with Before/After blocks
        if (incident.Message && (incident.Message.includes('Before:') || incident.Message.includes('After:'))) {
            contentHtml = `
                <div class="incident-diff">
                    ${this.renderDiffView(incident.Message)}
                </div>
            `;
        }
        // Show message as description if it exists but has no Before/After
        else if (incident.Message && incident.Message.trim()) {
            contentHtml = `
                <div class="incident-message">
                    <p>${this.escapeHtml(incident.Message)}</p>
                </div>
            `;
        }

        // Show code snippet if available
        if (incident.CodeSnip && incident.CodeSnip.trim()) {
            contentHtml += `
                <div class="incident-code-context">
                    <h5><i class="fas fa-code"></i> Code Context</h5>
                    <pre class="line-numbers"><code class="language-tsx">${this.escapeHtml(incident.CodeSnip)}</code></pre>
                </div>
            `;
        }

        return `
            <div class="incident-card">
                <div class="incident-header">
                    <div class="incident-location">
                        <i class="fas fa-file-code"></i>
                        <code class="file-path">${this.escapeHtml(filePath)}</code>
                        <span class="line-number">Line ${incident.LineNumber}</span>
                    </div>
                </div>
                ${contentHtml || '<div class="incident-message"><p class="no-details">See file at specified line number for details.</p></div>'}
            </div>
        `;
    }

    navigateIncident(violationId, direction) {
        // Find the violation in the plan
        let violation = null;
        for (const phase of this.plan.Phases) {
            violation = phase.Violations.find(v => v.ViolationID.replace(/[^a-zA-Z0-9]/g, '-') === violationId);
            if (violation) break;
        }

        if (!violation) return;

        // Get current index from data attribute or initialize to 0
        const displayEl = document.getElementById(`incident-${violationId}`);
        if (!displayEl) return;

        let currentIndex = parseInt(displayEl.dataset.currentIndex || '0');
        currentIndex = (currentIndex + direction + violation.Incidents.length) % violation.Incidents.length;

        // Update display
        displayEl.dataset.currentIndex = currentIndex;
        displayEl.innerHTML = this.renderIncident(violation.Incidents[currentIndex], currentIndex, violation);

        // Update counter
        const counterEl = document.getElementById(`counter-${violationId}`);
        if (counterEl) {
            counterEl.textContent = `${currentIndex + 1} of ${violation.Incidents.length}`;
        }

        // Apply syntax highlighting
        if (window.Prism) {
            Prism.highlightAllUnder(displayEl);
        }
    }

    isPhaseApproved(phase) {
        // A phase is considered approved if it's explicitly not deferred
        return !phase.Deferred;
    }

    async approvePhase(phaseId) {
        try {
            const response = await fetch('/api/phase/approve', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ phase_id: phaseId })
            });

            if (!response.ok) {
                throw new Error('Failed to approve phase');
            }

            // Update local state
            const phase = this.plan.Phases.find(p => p.ID === phaseId);
            if (phase) {
                phase.Deferred = false;
                this.render();
            }
        } catch (error) {
            console.error('Error approving phase:', error);
            this.showError('Failed to approve phase');
        }
    }

    async deferPhase(phaseId) {
        try {
            const response = await fetch('/api/phase/defer', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ phase_id: phaseId })
            });

            if (!response.ok) {
                throw new Error('Failed to defer phase');
            }

            // Update local state
            const phase = this.plan.Phases.find(p => p.ID === phaseId);
            if (phase) {
                phase.Deferred = true;
                this.render();
            }
        } catch (error) {
            console.error('Error deferring phase:', error);
            this.showError('Failed to defer phase');
        }
    }

    async savePlan() {
        try {
            const response = await fetch('/api/plan/save', { method: 'POST' });

            if (!response.ok) {
                throw new Error('Failed to save plan');
            }

            const result = await response.json();
            this.showSuccess(`Plan saved to ${result.path}`);
        } catch (error) {
            console.error('Error saving plan:', error);
            this.showError('Failed to save plan');
        }
    }

    async executePhases() {
        try {
            const response = await fetch('/api/execute/start', { method: 'POST' });

            if (!response.ok) {
                throw new Error('Failed to start execution');
            }

            const result = await response.json();

            // Show execution view
            document.getElementById('execution-view').classList.remove('hidden');
            document.getElementById('phases-container').classList.add('hidden');

            this.showInfo(result.message);
        } catch (error) {
            console.error('Error executing phases:', error);
            this.showError('Failed to start execution');
        }
    }

    toggleDetails(phaseId) {
        const detailsEl = document.getElementById(`details-${phaseId}`);
        const iconEl = document.getElementById(`toggle-icon-${phaseId}`);

        if (!detailsEl || !iconEl) return;

        if (detailsEl.classList.contains('hidden')) {
            detailsEl.classList.remove('hidden');
            iconEl.classList.remove('fa-chevron-down');
            iconEl.classList.add('fa-chevron-up');

            // Apply syntax highlighting after expanding
            if (window.Prism) {
                setTimeout(() => {
                    Prism.highlightAllUnder(detailsEl);
                }, 100);
            }
        } else {
            detailsEl.classList.add('hidden');
            iconEl.classList.remove('fa-chevron-up');
            iconEl.classList.add('fa-chevron-down');
        }
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;

        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
        };

        this.ws.onmessage = (event) => {
            try {
                const update = JSON.parse(event.data);
                this.handleExecutionUpdate(update);
            } catch (error) {
                console.error('Failed to parse WebSocket message:', error);
            }
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            // Attempt to reconnect after 5 seconds
            setTimeout(() => this.connectWebSocket(), 5000);
        };
    }

    handleExecutionUpdate(update) {
        switch (update.type) {
            case 'progress':
                this.updateExecutionProgress(update.data);
                break;
            case 'incident':
                this.addIncidentUpdate(update.data);
                break;
            case 'complete':
                this.showExecutionSummary(update.data);
                break;
            case 'error':
                this.showExecutionError(update.data);
                break;
            default:
                console.log('Unknown update type:', update.type);
        }
    }

    updateExecutionProgress(data) {
        // TODO: Update progress in execution view
        console.log('Progress update:', data);
    }

    addIncidentUpdate(data) {
        // TODO: Add incident to activity stream
        console.log('Incident update:', data);
    }

    showExecutionSummary(data) {
        // TODO: Show completion summary
        console.log('Execution complete:', data);
    }

    showExecutionError(data) {
        // TODO: Show error in execution view
        console.error('Execution error:', data);
    }

    updateProgress() {
        const approved = this.plan.Phases.filter(p => !p.Deferred).length;
        const total = this.plan.Phases.length;
        const percentage = total > 0 ? (approved / total) * 100 : 0;

        const progressFill = document.getElementById('progress-fill');
        const progressText = document.getElementById('progress-text');
        const progressContainer = document.getElementById('progress-container');

        if (progressFill && progressText) {
            progressFill.style.width = `${percentage}%`;
            progressText.textContent = `${Math.round(percentage)}% (${approved} of ${total} approved)`;

            if (progressContainer) {
                progressContainer.classList.remove('hidden');
            }
        }
    }

    attachEventListeners() {
        const saveBtn = document.getElementById('save-btn');
        const executeBtn = document.getElementById('execute-btn');

        if (saveBtn) {
            saveBtn.addEventListener('click', () => this.savePlan());
        }

        if (executeBtn) {
            executeBtn.addEventListener('click', () => this.executePhases());
        }
    }

    formatCost(cost) {
        return `$${cost.toFixed(2)}`;
    }

    getRiskIcon(risk) {
        const icons = {
            'high': '🔴',
            'medium': '🟡',
            'low': '🟢'
        };
        return icons[risk] || '⚪';
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    renderDiffView(message) {
        // Extract description and code blocks
        const parts = message.split(/(?=Before:|After:)/);
        let description = '';
        let beforeCode = '';
        let afterCode = '';

        parts.forEach(part => {
            if (part.includes('Before:')) {
                const match = part.match(/Before:\s*\n```\s*\n([\s\S]*?)\n```/);
                if (match) beforeCode = match[1];
            } else if (part.includes('After:')) {
                const match = part.match(/After:\s*\n```\s*\n([\s\S]*?)\n```/);
                if (match) afterCode = match[1];
            } else if (part.trim()) {
                description += part;
            }
        });

        let html = '';

        if (description.trim()) {
            html += `<div class="diff-description">${this.escapeHtml(description.trim())}</div>`;
        }

        if (beforeCode || afterCode) {
            html += '<div class="diff-container">';

            if (beforeCode) {
                html += `
                    <div class="diff-pane before-pane">
                        <div class="diff-header">
                            <i class="fas fa-minus-circle"></i> Before
                        </div>
                        <pre class="line-numbers"><code class="language-tsx">${this.escapeHtml(beforeCode)}</code></pre>
                    </div>
                `;
            }

            if (afterCode) {
                html += `
                    <div class="diff-pane after-pane">
                        <div class="diff-header">
                            <i class="fas fa-plus-circle"></i> After
                        </div>
                        <pre class="line-numbers"><code class="language-tsx">${this.escapeHtml(afterCode)}</code></pre>
                    </div>
                `;
            }

            html += '</div>';
        }

        return html || `<div class="diff-description">${this.escapeHtml(message)}</div>`;
    }

    showSuccess(message) {
        this.showNotification(message, 'success');
    }

    showWarning(message) {
        this.showNotification(message, 'warning');
    }

    showInfo(message) {
        this.showNotification(message, 'info');
    }

    showError(message) {
        this.showNotification(message, 'error');
    }

    showNotification(message, type) {
        // Simple alert for MVP - could be enhanced with toast notifications
        const emoji = {
            'success': '✓',
            'warning': '⚠',
            'info': 'ℹ',
            'error': '✗'
        }[type] || '';

        alert(`${emoji} ${message}`);
    }
}

// Initialize app when DOM is ready
let app;
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        app = new PlanApp();
    });
} else {
    app = new PlanApp();
}
