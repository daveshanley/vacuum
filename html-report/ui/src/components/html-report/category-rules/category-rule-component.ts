import {BaseComponent} from '../../../ts/base-component';
import {html, TemplateResult} from 'lit';
import {customElement, property} from 'lit/decorators.js';
import {RuleSelected, RuleSelectedEvent, ViolationSelectedEvent,} from '../../../model/events';
import {CategoryRuleResultComponent} from './category-rule-result-component';
import categoryRuleStyles from './category-rule.styles';
import expandIcon from './svg/expand.icon';
import contractIcon from './svg/contract.icon';

@customElement('category-rule')
export class CategoryRuleComponent extends BaseComponent {
    static styles = categoryRuleStyles;

    @property()
    totalRulesViolated: number;

    @property()
    maxViolations: number;

    @property()
    truncated: boolean;

    @property()
    ruleId: string;

    @property()
    description: string;

    @property()
    numResults: number;

    @property()
    ruleIcon: number;

    @property()
    open: boolean;

    // @query('.violations')
    private violations: HTMLElement;

    private _expandState: boolean;

    otherRuleSelected() {
        this.open = false;
        this.violations = this.renderRoot.querySelector('.violations');
        if (this.violations) {
            this.violations.style.display = 'none';
        }
        this._expandState = false;
        this._slottedChildren.forEach((result: CategoryRuleResultComponent) => {
            result.selected = false;
        });
        this.requestUpdate();
    }

    render() {
        this.violations = this.renderRoot.querySelector('.violations');
        let truncatedAlert: TemplateResult;
        if (this.truncated) {
            truncatedAlert = html`
                <div class="truncated">
                    <strong>${this.numResults - this.maxViolations}</strong> more
                    violations not rendered, There are just too many!
                </div>
            `;
        }

        const expanded = this._expandState ? contractIcon : expandIcon;

        return html`
            <nav
                    aria-label="Rules and Violations"
                    class="details ${this._expandState ? 'open' : ''}"
            >
                <div class="summary" @click=${this._ruleSelected}>
                    <span class="expand-state">${expanded}</span>
                    <span class="rule-icon">${this.ruleIcon}</span>
                    <span class="rule-description">${this.description}</span>
                    <span class="rule-violation-count">${this.numResults}</span>
                </div>
                <div class="violations" @violationSelected=${this._violationSelected}>
                    <slot name="results"></slot>
                    ${truncatedAlert}
                </div>
            </nav>
        `;
    }

    private _ruleSelected() {
        if (!this.open) {
            if (this.violations) {
                this.violations.style.display = 'block';
            }
            // use some intelligence to resize this in a responsive way.
            const heightCalc =
                this.parentElement.parentElement.offsetHeight -
                this.totalRulesViolated * 60;
            if (this.violations) {
                this.violations.style.maxHeight = heightCalc + 'px';
            }
            this._expandState = true;
        } else {
            if (this.violations) {
                this.violations.style.display = 'none';
            }
            this._expandState = false;
        }

        this.open = !this.open;

        this.dispatchEvent(
            new CustomEvent<RuleSelectedEvent>(RuleSelected, {
                bubbles: true,
                composed: true,
                detail: {id: this.ruleId},
            })
        );
        this.requestUpdate();
    }

    private _violationSelected(evt: CustomEvent<ViolationSelectedEvent>) {
        this._slottedChildren.forEach((result: CategoryRuleResultComponent) => {
            result.selected = evt.detail.violationId == result.violationId;
        });
    }
}
