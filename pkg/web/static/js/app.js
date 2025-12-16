class PlanApp {
    constructor() {
        this.plan = null;
        this.ws = null;
        this.charts = {
            complexity: null,
            category: null,
            risk: null
        };
        this.executionStartTime = null;
        this.executionTimer = null;
        this.init();
    }

    async init() {
        try {
            await this.loadPlan();
            this.render();
            this.attachEventListeners();
            this.initializeSortable();
            this.connectWebSocket();
            this.setupKeyboardShortcuts();
        } catch (error) {
            console.error('Failed to initialize app:', error);
            this.showError('Failed to load plan. Please refresh the page.');
        }
    }

    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            // Don't trigger shortcuts when typing in inputs, textareas, or with modifiers
            if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.tagName === 'SELECT') {
                return;
            }

            // Cmd/Ctrl + key shortcuts
            if (e.metaKey || e.ctrlKey) {
                switch(e.key.toLowerCase()) {
                    case 's':
                        e.preventDefault();
                        this.savePlan();
                        break;
                    case 'e':
                        e.preventDefault();
                        if (!document.getElementById('execution-view').classList.contains('hidden')) {
                            return; // Don't start execution if already executing
                        }
                        this.executePhases();
                        break;
                }
                return;
            }

            // Single key shortcuts (no modifiers)
            switch(e.key.toLowerCase()) {
                case '?':
                    e.preventDefault();
                    this.showKeyboardHelp();
                    break;
                case 'escape':
                    // Close any open modals
                    this.closeSettings();
                    this.closeConfirmExecution();
                    break;
            }
        });
    }

    showKeyboardHelp() {
        const helpText = `
Keyboard Shortcuts:
────────────────────────────────────
Ctrl/Cmd + S    Save plan
Ctrl/Cmd + E    Execute approved phases
?               Show this help
Esc             Close modals
        `.trim();

        alert(helpText);
    }

    showLoading(message = 'Loading...') {
        const overlay = document.getElementById('loading-overlay');
        const text = overlay.querySelector('.loading-text');
        if (text) text.textContent = message;
        overlay.classList.remove('hidden');
    }

    hideLoading() {
        document.getElementById('loading-overlay').classList.add('hidden');
    }

    async loadPlan() {
        this.showLoading('Loading plan...');
        try {
            const response = await fetch('/api/plan');
            if (!response.ok) {
                throw new Error(`Server returned ${response.status}: ${response.statusText}`);
            }
            this.plan = await response.json();
            console.log('Loaded plan:', this.plan);

            // Validate plan structure
            if (!this.plan.Phases) {
                throw new Error('Invalid plan format: Missing Phases array');
            }
        } finally {
            this.hideLoading();
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
        const riskWarning = this.renderRiskWarning(phase);

        return `
            <div class="phase ${status}" data-phase-id="${phase.ID}">
                <div class="phase-header">
                    <h3>Phase ${phase.Order}: ${this.escapeHtml(phase.Name)}</h3>
                    <span class="status-badge ${status}">${statusText}</span>
                </div>
                <div class="phase-content">
                    ${riskWarning}
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
                            <div class="violation-filter-controls">
                                <h4>Detailed Violations:</h4>
                                <div class="filter-actions">
                                    <button class="btn btn-info btn-sm" onclick="app.selectAllViolations('${phase.ID}')">
                                        <i class="fas fa-check-square"></i> Select All
                                    </button>
                                    <button class="btn btn-info btn-sm" onclick="app.deselectAllViolations('${phase.ID}')">
                                        <i class="fas fa-square"></i> Deselect All
                                    </button>
                                    <button class="btn btn-info btn-sm" onclick="app.filterByEffort('${phase.ID}', 5)">
                                        Select Low/Medium Effort
                                    </button>
                                </div>
                                <div class="selection-summary" id="selection-summary-${phase.ID}">
                                    <span class="selected-count">All ${phase.Violations.length} violations selected</span>
                                </div>
                            </div>
                            ${phase.Violations.map(v => this.renderViolationDetails(v, phase.ID)).join('')}
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

    renderViolationDetails(violation, phaseID) {
        const violationId = violation.ViolationID.replace(/[^a-zA-Z0-9]/g, '-');

        return `
            <div class="violation-detail" data-violation-id="${violationId}">
                <div class="violation-header">
                    <label class="violation-checkbox">
                        <input type="checkbox"
                               checked
                               onchange="app.toggleViolation('${phaseID}', '${violationId}')"
                               data-phase-id="${phaseID}"
                               data-violation-id="${violationId}">
                        <strong>${this.escapeHtml(violation.ViolationID)}</strong>
                    </label>
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
        // Show confirmation dialog with estimates
        const approvedPhases = this.plan.Phases.filter(p => !p.Deferred);

        if (approvedPhases.length === 0) {
            this.showWarning('No phases approved for execution');
            return;
        }

        // Calculate estimates
        const totalViolations = approvedPhases.reduce((sum, p) => sum + p.Violations.length, 0);
        const totalIncidents = approvedPhases.reduce((sum, p) =>
            sum + p.Violations.reduce((vSum, v) => vSum + v.IncidentCount, 0), 0
        );
        const totalCost = approvedPhases.reduce((sum, p) => sum + p.EstimatedCost, 0);
        const totalDuration = approvedPhases.reduce((sum, p) => sum + p.EstimatedDurationMinutes, 0);

        // Update modal with estimates
        document.getElementById('estimate-phases').textContent = approvedPhases.length;
        document.getElementById('estimate-violations').textContent = totalViolations;
        document.getElementById('estimate-incidents').textContent = totalIncidents;
        document.getElementById('estimate-cost').textContent = this.formatCost(totalCost);
        document.getElementById('estimate-duration').textContent = `~${totalDuration} min`;

        // Show confirmation modal
        document.getElementById('confirm-execution-modal').classList.remove('hidden');
    }

    closeConfirmExecution() {
        document.getElementById('confirm-execution-modal').classList.add('hidden');
    }

    async confirmExecution() {
        // Close confirmation modal
        this.closeConfirmExecution();

        try {
            const response = await fetch('/api/execute/start', { method: 'POST' });

            if (!response.ok) {
                throw new Error('Failed to start execution');
            }

            // Show execution view
            document.getElementById('execution-view').classList.remove('hidden');
            document.getElementById('phases-container').classList.add('hidden');

            // Start execution timer
            this.startExecutionTimer();
        } catch (error) {
            console.error('Error executing phases:', error);
            this.showError('Failed to start execution');
        }
    }

    async cancelExecution() {
        if (!confirm('Are you sure you want to cancel execution?')) {
            return;
        }

        try {
            const response = await fetch('/api/execute/cancel', { method: 'POST' });

            if (!response.ok) {
                throw new Error('Failed to cancel execution');
            }

            this.stopExecutionTimer();
            this.addActivityMessage('Execution cancelled by user', 'error');
        } catch (error) {
            console.error('Error cancelling execution:', error);
            this.showError('Failed to cancel execution');
        }
    }

    startExecutionTimer() {
        this.executionStartTime = Date.now();
        this.updateExecutionTimer();
        this.executionTimer = setInterval(() => this.updateExecutionTimer(), 1000);
    }

    stopExecutionTimer() {
        if (this.executionTimer) {
            clearInterval(this.executionTimer);
            this.executionTimer = null;
        }
    }

    updateExecutionTimer() {
        if (!this.executionStartTime) return;

        const elapsed = Math.floor((Date.now() - this.executionStartTime) / 1000);
        const minutes = Math.floor(elapsed / 60);
        const seconds = elapsed % 60;

        const timerEl = document.getElementById('execution-timer');
        if (timerEl) {
            timerEl.textContent = `${minutes}:${seconds.toString().padStart(2, '0')}`;
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

    toggleViolation(phaseId, violationId) {
        this.updateSelectionSummary(phaseId);
    }

    selectAllViolations(phaseId) {
        const checkboxes = document.querySelectorAll(`input[data-phase-id="${phaseId}"]`);
        checkboxes.forEach(cb => cb.checked = true);
        this.updateSelectionSummary(phaseId);
    }

    deselectAllViolations(phaseId) {
        const checkboxes = document.querySelectorAll(`input[data-phase-id="${phaseId}"]`);
        checkboxes.forEach(cb => cb.checked = false);
        this.updateSelectionSummary(phaseId);
    }

    filterByEffort(phaseId, maxEffort) {
        const phase = this.plan.Phases.find(p => p.ID === phaseId);
        if (!phase) return;

        const checkboxes = document.querySelectorAll(`input[data-phase-id="${phaseId}"]`);
        checkboxes.forEach(cb => {
            const violationId = cb.dataset.violationId;
            const originalViolationId = violationId.replace(/-/g, match => {
                // Try to find the original violation to check its effort
                const violation = phase.Violations.find(v =>
                    v.ViolationID.replace(/[^a-zA-Z0-9]/g, '-') === violationId
                );
                return violation ? '' : match;
            });

            const violation = phase.Violations.find(v =>
                v.ViolationID.replace(/[^a-zA-Z0-9]/g, '-') === violationId
            );

            if (violation) {
                cb.checked = (violation.Effort || 0) <= maxEffort;
            }
        });
        this.updateSelectionSummary(phaseId);
    }

    updateSelectionSummary(phaseId) {
        const summaryEl = document.getElementById(`selection-summary-${phaseId}`);
        if (!summaryEl) return;

        const checkboxes = document.querySelectorAll(`input[data-phase-id="${phaseId}"]`);
        const totalCount = checkboxes.length;
        const selectedCount = Array.from(checkboxes).filter(cb => cb.checked).length;

        if (selectedCount === totalCount) {
            summaryEl.innerHTML = `<span class="selected-count">All ${totalCount} violations selected</span>`;
        } else if (selectedCount === 0) {
            summaryEl.innerHTML = `<span class="selected-count warning">No violations selected</span>`;
        } else {
            summaryEl.innerHTML = `
                <span class="selected-count">${selectedCount} of ${totalCount} violations selected</span>
                <span class="excluded-count">${totalCount - selectedCount} excluded</span>
            `;
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
            case 'phase_start':
                this.handlePhaseStart(update.data);
                break;
            case 'phase_end':
                this.handlePhaseEnd(update.data);
                break;
            case 'info':
                this.addActivityMessage(update.data.message, 'info');
                break;
            case 'error':
                this.addActivityMessage(update.data.message, 'error');
                break;
            case 'cancelled':
                this.handleExecutionCancelled(update.data);
                break;
            case 'complete':
                this.showExecutionSummary(update.data);
                break;
            default:
                console.log('Unknown update type:', update.type);
        }
    }

    handleExecutionCancelled(data) {
        this.stopExecutionTimer();
        this.addActivityMessage(data.message || 'Execution cancelled', 'error');

        // Hide cancel button
        const cancelBtn = document.getElementById('cancel-execution-btn');
        if (cancelBtn) {
            cancelBtn.disabled = true;
            cancelBtn.innerHTML = '<i class="fas fa-check"></i> Cancelled';
        }
    }

    handlePhaseStart(data) {
        this.addActivityMessage(`Starting phase: ${data.phase_name}`, 'phase');
        const percentage = data.total > 0 ? (data.phase_index / data.total) * 100 : 0;
        this.updateExecutionProgress({
            phase: data.phase_index,
            total: data.total,
            percentage: percentage,
            message: `Processing phase ${data.phase_index} of ${data.total}: ${data.phase_name}`
        });
    }

    handlePhaseEnd(data) {
        this.addActivityMessage(`Completed phase: ${data.phase_name}`, 'success');
    }

    updateExecutionProgress(data) {
        const progressBar = document.getElementById('execution-progress-bar');
        const progressText = document.getElementById('execution-progress-text');
        const currentPhase = document.getElementById('execution-current-phase');

        if (progressBar) {
            progressBar.style.width = `${data.percentage}%`;
        }

        if (progressText) {
            progressText.textContent = `${Math.round(data.percentage)}% (${data.phase || 0} of ${data.total || 0} phases)`;
        }

        if (currentPhase && data.message) {
            currentPhase.textContent = data.message;
        }
    }

    addActivityMessage(message, type = 'info') {
        const activityFeed = document.getElementById('execution-activity');
        if (!activityFeed) return;

        const timestamp = new Date().toLocaleTimeString();
        const icon = {
            'info': '•',
            'success': '✓',
            'error': '✗',
            'phase': '▶'
        }[type] || '•';

        const className = {
            'info': 'activity-info',
            'success': 'activity-success',
            'error': 'activity-error',
            'phase': 'activity-phase'
        }[type] || 'activity-info';

        const entry = document.createElement('div');
        entry.className = `activity-entry ${className}`;
        entry.innerHTML = `
            <span class="activity-time">${timestamp}</span>
            <span class="activity-icon">${icon}</span>
            <span class="activity-message">${this.escapeHtml(message)}</span>
        `;

        activityFeed.insertBefore(entry, activityFeed.firstChild);

        // Limit to last 100 entries
        while (activityFeed.children.length > 100) {
            activityFeed.removeChild(activityFeed.lastChild);
        }
    }

    showExecutionSummary(data) {
        // Stop execution timer
        this.stopExecutionTimer();

        // Calculate final duration
        const duration = this.executionStartTime
            ? Math.floor((Date.now() - this.executionStartTime) / 1000)
            : 0;
        const durationMin = Math.floor(duration / 60);
        const durationSec = duration % 60;
        const durationText = `${durationMin}:${durationSec.toString().padStart(2, '0')}`;

        const summaryEl = document.getElementById('execution-summary');
        if (!summaryEl) return;

        summaryEl.classList.remove('hidden');
        summaryEl.innerHTML = `
            <div class="execution-complete">
                <h3>✓ Execution Complete</h3>
                <div class="summary-stats">
                    <div class="stat-card">
                        <div class="stat-value">${data.completed_phases || 0}</div>
                        <div class="stat-label">Phases Completed</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value success">${data.successful_fixes || 0}</div>
                        <div class="stat-label">Successful Fixes</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value ${data.failed_fixes > 0 ? 'error' : ''}">${data.failed_fixes || 0}</div>
                        <div class="stat-label">Failed Fixes</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value">$${(data.total_cost || 0).toFixed(4)}</div>
                        <div class="stat-label">Total Cost</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value">${durationText}</div>
                        <div class="stat-label">Duration</div>
                    </div>
                </div>
                <div class="summary-actions">
                    <button class="btn btn-primary" onclick="app.closeExecution()">
                        <i class="fas fa-check"></i> Done
                    </button>
                </div>
            </div>
        `;

        this.addActivityMessage('Execution completed successfully', 'success');

        // Hide cancel button
        const cancelBtn = document.getElementById('cancel-execution-btn');
        if (cancelBtn) {
            cancelBtn.style.display = 'none';
        }
    }

    closeExecution() {
        document.getElementById('execution-view').classList.add('hidden');
        document.getElementById('phases-container').classList.remove('hidden');

        // Reload plan to get updated state
        this.loadPlan().then(() => this.render());
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
        const exportBtn = document.getElementById('export-btn');
        const importBtn = document.getElementById('import-btn');
        const settingsBtn = document.getElementById('settings-btn');
        const importFileInput = document.getElementById('import-file-input');

        if (saveBtn) {
            saveBtn.addEventListener('click', () => this.savePlan());
        }

        if (executeBtn) {
            executeBtn.addEventListener('click', () => this.executePhases());
        }

        if (exportBtn) {
            exportBtn.addEventListener('click', () => this.exportPlan());
        }

        if (importBtn) {
            importBtn.addEventListener('click', () => {
                importFileInput.click();
            });
        }

        if (settingsBtn) {
            settingsBtn.addEventListener('click', () => this.openSettings());
        }

        if (importFileInput) {
            importFileInput.addEventListener('change', (e) => this.importPlan(e));
        }

        // Settings slider update
        const confidenceSlider = document.getElementById('setting-confidence-threshold');
        if (confidenceSlider) {
            confidenceSlider.addEventListener('input', (e) => {
                document.getElementById('confidence-value').textContent = e.target.value + '%';
            });
        }
    }

    initializeSortable() {
        const container = document.getElementById('phases-container');
        if (!container || !window.Sortable) return;

        Sortable.create(container, {
            animation: 150,
            handle: '.phase-header',
            ghostClass: 'phase-ghost',
            chosenClass: 'phase-chosen',
            dragClass: 'phase-drag',
            onEnd: (evt) => {
                // Reorder phases in the plan
                const oldIndex = evt.oldIndex;
                const newIndex = evt.newIndex;

                if (oldIndex !== newIndex) {
                    const movedPhase = this.plan.Phases.splice(oldIndex, 1)[0];
                    this.plan.Phases.splice(newIndex, 0, movedPhase);

                    // Update order numbers
                    this.plan.Phases.forEach((phase, index) => {
                        phase.Order = index + 1;
                    });

                    this.render();
                    this.initializeSortable(); // Reinitialize after render
                }
            }
        });
    }

    exportPlan() {
        const dataStr = JSON.stringify(this.plan, null, 2);
        const dataBlob = new Blob([dataStr], { type: 'application/json' });
        const url = URL.createObjectURL(dataBlob);

        const link = document.createElement('a');
        link.href = url;
        link.download = 'kantra-ai-plan-export.json';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);

        this.showSuccess('Plan exported successfully');
    }

    async importPlan(event) {
        const file = event.target.files[0];
        if (!file) return;

        try {
            const text = await file.text();
            const importedPlan = JSON.parse(text);

            // Validate basic structure
            if (!importedPlan.Phases || !Array.isArray(importedPlan.Phases)) {
                throw new Error('Invalid plan format: missing Phases array');
            }

            this.plan = importedPlan;
            this.render();
            this.showSuccess('Plan imported successfully');

            // Clear the file input
            event.target.value = '';
        } catch (error) {
            console.error('Import error:', error);
            this.showError(`Failed to import plan: ${error.message}`);
            event.target.value = '';
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

    renderRiskWarning(phase) {
        if (phase.Risk.toLowerCase() !== 'high') {
            return '';
        }

        // Count high/expert effort violations
        const highEffortViolations = phase.Violations.filter(v => (v.Effort || 0) >= 6);
        const expertViolations = phase.Violations.filter(v => (v.Effort || 0) >= 8);

        const recommendations = [];
        if (highEffortViolations.length > 0) {
            recommendations.push('Review code diffs carefully before approving');
        }
        if (phase.Category === 'mandatory') {
            recommendations.push('These changes are required for migration to succeed');
        }
        if (phase.EstimatedDurationMinutes > 30) {
            recommendations.push('Plan for extended execution time');
        }
        recommendations.push('Have a backup/branch before executing');
        recommendations.push('Plan for additional testing time');

        const riskFactors = [];
        if (highEffortViolations.length > 0) {
            riskFactors.push(`Contains ${highEffortViolations.length} high-effort violation${highEffortViolations.length !== 1 ? 's' : ''}`);
        }
        if (expertViolations.length > 0) {
            riskFactors.push(`Contains ${expertViolations.length} expert-level violation${expertViolations.length !== 1 ? 's' : ''}`);
        }
        if (phase.Violations.length > 20) {
            riskFactors.push(`Large number of violations (${phase.Violations.length} total)`);
        }
        if (phase.EstimatedCost > 1.0) {
            riskFactors.push(`High estimated cost (${this.formatCost(phase.EstimatedCost)})`);
        }

        return `
            <div class="risk-warning">
                <div class="risk-warning-header">
                    <i class="fas fa-exclamation-triangle"></i>
                    <strong>WARNING: HIGH RISK PHASE</strong>
                </div>
                <div class="risk-warning-content">
                    <p>This phase includes complex changes that may require careful review or manual intervention.</p>

                    ${recommendations.length > 0 ? `
                        <div class="risk-section">
                            <strong>Recommendations:</strong>
                            <ul>
                                ${recommendations.map(r => `<li>${r}</li>`).join('')}
                            </ul>
                        </div>
                    ` : ''}

                    ${riskFactors.length > 0 ? `
                        <div class="risk-section">
                            <strong>Risk Factors:</strong>
                            <ul>
                                ${riskFactors.map(f => `<li>${f}</li>`).join('')}
                            </ul>
                        </div>
                    ` : ''}
                </div>
            </div>
        `;
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
        // Create toast notification
        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;

        const icon = {
            'success': '✓',
            'warning': '⚠',
            'info': 'ℹ',
            'error': '✗'
        }[type] || '';

        toast.innerHTML = `
            <span class="toast-icon">${icon}</span>
            <span class="toast-message">${this.escapeHtml(message)}</span>
            <button class="toast-close" onclick="this.parentElement.remove()">×</button>
        `;

        // Add to page
        let container = document.getElementById('toast-container');
        if (!container) {
            container = document.createElement('div');
            container.id = 'toast-container';
            document.body.appendChild(container);
        }
        container.appendChild(toast);

        // Auto-remove after 5 seconds
        setTimeout(() => {
            if (toast.parentElement) {
                toast.classList.add('toast-fadeout');
                setTimeout(() => toast.remove(), 300);
            }
        }, 5000);
    }

    openSettings() {
        const modal = document.getElementById('settings-modal');
        if (modal) {
            // Load current settings
            this.loadSettings();
            modal.classList.remove('hidden');
        }
    }

    closeSettings() {
        const modal = document.getElementById('settings-modal');
        if (modal) {
            modal.classList.add('hidden');
        }
    }

    loadSettings() {
        // Load from localStorage or use defaults
        const settings = JSON.parse(localStorage.getItem('kantra-ai-settings') || '{}');

        document.getElementById('setting-confidence-enabled').checked = settings.confidenceEnabled !== false;
        document.getElementById('setting-confidence-threshold').value = settings.confidenceThreshold || 70;
        document.getElementById('confidence-value').textContent = (settings.confidenceThreshold || 70) + '%';
        document.getElementById('setting-low-confidence-action').value = settings.lowConfidenceAction || 'skip';

        document.getElementById('setting-run-build').checked = settings.runBuild !== false;
        document.getElementById('setting-build-strategy').value = settings.buildStrategy || 'at-end';
        document.getElementById('setting-fail-fast').checked = settings.failFast !== false;

        document.getElementById('setting-create-commits').checked = settings.createCommits || false;
        document.getElementById('setting-commit-strategy').value = settings.commitStrategy || 'single';
        document.getElementById('setting-create-pr').checked = settings.createPR || false;

        document.getElementById('setting-batch-enabled').checked = settings.batchEnabled !== false;
        document.getElementById('setting-batch-size').value = settings.batchSize || 10;
        document.getElementById('setting-parallelism').value = settings.parallelism || 4;
    }

    saveSettings() {
        const settings = {
            confidenceEnabled: document.getElementById('setting-confidence-enabled').checked,
            confidenceThreshold: parseInt(document.getElementById('setting-confidence-threshold').value),
            lowConfidenceAction: document.getElementById('setting-low-confidence-action').value,

            runBuild: document.getElementById('setting-run-build').checked,
            buildStrategy: document.getElementById('setting-build-strategy').value,
            failFast: document.getElementById('setting-fail-fast').checked,

            createCommits: document.getElementById('setting-create-commits').checked,
            commitStrategy: document.getElementById('setting-commit-strategy').value,
            createPR: document.getElementById('setting-create-pr').checked,

            batchEnabled: document.getElementById('setting-batch-enabled').checked,
            batchSize: parseInt(document.getElementById('setting-batch-size').value),
            parallelism: parseInt(document.getElementById('setting-parallelism').value),
        };

        localStorage.setItem('kantra-ai-settings', JSON.stringify(settings));
        this.showSuccess('Settings saved successfully');
        this.closeSettings();
    }

    resetSettings() {
        localStorage.removeItem('kantra-ai-settings');
        this.loadSettings();
        this.showInfo('Settings reset to defaults');
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
