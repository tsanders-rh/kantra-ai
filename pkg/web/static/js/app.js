class PlanApp {
    constructor() {
        this.plan = null;
        this.ws = null;
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
                            ✓ Approve
                        </button>
                        <button class="btn btn-warning" onclick="app.deferPhase('${phase.ID}')">
                            ↷ Defer
                        </button>
                        <button class="btn btn-info" onclick="app.toggleDetails('${phase.ID}')">
                            <span class="toggle-icon" id="toggle-icon-${phase.ID}">▼</span> View Details
                        </button>
                    </div>
                </div>
            </div>
        `;
    }

    renderViolationDetails(violation) {
        const sampleIncidents = violation.Incidents.slice(0, 5);
        const hasMore = violation.Incidents.length > 5;

        return `
            <div class="violation-detail">
                <div class="violation-header">
                    <strong>${this.escapeHtml(violation.ViolationID)}</strong>
                    <span class="violation-meta">
                        ${violation.IncidentCount} incident${violation.IncidentCount !== 1 ? 's' : ''} •
                        Effort: ${violation.Effort}
                    </span>
                </div>
                <p class="violation-description">${this.escapeHtml(violation.Description)}</p>

                <div class="incidents-list">
                    <h5>Incidents (showing ${sampleIncidents.length} of ${violation.Incidents.length}):</h5>
                    ${sampleIncidents.map((incident, idx) => `
                        <div class="incident-item">
                            <div class="incident-location">
                                <span class="incident-number">${idx + 1}.</span>
                                <code>${this.escapeHtml(incident.URI.replace('file://', '').replace('/opt/input/source/', ''))}</code>
                                <span class="line-number">:${incident.LineNumber}</span>
                            </div>
                            ${incident.Message ? `
                                <div class="incident-message">${this.formatIncidentMessage(incident.Message)}</div>
                            ` : ''}
                        </div>
                    `).join('')}
                    ${hasMore ? `
                        <div class="more-incidents">
                            ... and ${violation.Incidents.length - 5} more incidents
                        </div>
                    ` : ''}
                </div>
            </div>
        `;
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
            iconEl.textContent = '▲';
        } else {
            detailsEl.classList.add('hidden');
            iconEl.textContent = '▼';
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

    formatIncidentMessage(message) {
        // Escape the message first
        let formatted = this.escapeHtml(message);

        // Replace Before: code blocks with red highlighting
        formatted = formatted.replace(
            /Before:\n```\n([\s\S]*?)\n```/g,
            '<div class="code-block before-block"><div class="code-label">Before:</div><pre>$1</pre></div>'
        );

        // Replace After: code blocks with green highlighting
        formatted = formatted.replace(
            /After:\n```\n([\s\S]*?)\n```/g,
            '<div class="code-block after-block"><div class="code-label">After:</div><pre>$1</pre></div>'
        );

        return formatted;
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
