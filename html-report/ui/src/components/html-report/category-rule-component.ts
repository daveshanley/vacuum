import { BaseComponent } from '../../ts/base-component';
import { html, css, TemplateResult } from 'lit';
import { property, query } from 'lit/decorators.js';
import {
  RuleSelected,
  RuleSelectedEvent,
  ViolationSelectedEvent,
} from '../../model/events';
import { CategoryRuleResultComponent } from './category-rule-result-component';

export class CategoryRuleComponent extends BaseComponent {
  static get styles() {
    const iconCss = css`
      .rule-icon {
        font-family: 'Arial';
        font-size: var(--sl-font-size-small);
        width: 20px;
        display: inline-block;
      }

      li {
        margin: 0;
        padding-left: 0;
      }
      li::after {
        content: '';
      }

      .details {
        margin-bottom: calc(var(--global-margin) / 2);
      }

      .details > .summary {
        background-color: var(--card-bgcolor);
        border: 1px solid var(--card-bordercolor);
        padding: 5px;
        border-radius: 3px;
      }

      .rule-violation-count {
        font-size: var(--sl-font-size-x-small);
        border: 1px solid var(--card-bordercolor);
        color: var(--tertiary-color);
        padding: 2px;
        border-radius: 2px;
      }

      .details.open .summary {
        background-color: var(--primary-color);
        color: var(--invert-font-color);
      }

      .details.open .rule-violation-count {
        background-color: var(--primary-color);
        color: var(--invert-font-color);
        border: 1px solid var(--invert-font-color);
      }

      .details.open .expand-state {
        color: var(--invert-font-color);
      }

      .details > div.violations {
        font-size: var(--sl-font-size-x-small);
        overflow-y: auto;
        overflow-x: hidden;
        border: 1px solid var(--card-bordercolor);
      }

      @media only screen and (max-width: 1200px) {
        .details > div.violations {
          max-height: 230px;
        }
      }

      .details > .summary::marker {
        color: var(--secondary-color);
      }

      .rule-description {
        font-size: var(--rule-font-size);
      }

      .details[open] .summary span.rule-description {
        background-color: var(--primary-color);
        color: var(--invert-font-color);
      }

      .summary span.rule-description:hover {
        cursor: pointer;
        background-color: var(--primary-color);
        color: var(--invert-font-color);
      }

      .rule-details::part(header) {
        margin: 0;
        padding: 3px;
      }

      .rule-details::part(summary) {
        font-family: Arial;
        padding: 2px;
      }
      .rule-details::part(base) {
        max-height: 500px;
        overflow-y: auto;
      }
      .rule-details::part(content) {
        margin: 0;
        padding: 0;
      }

      .violations {
        display: none;
        scrollbar-width: thin;
      }

      .violations::-webkit-scrollbar {
        width: 10px;
      }

      .violations::-webkit-scrollbar-track {
        background-color: var(--card-bgcolor);
      }

      .violations::-webkit-scrollbar-thumb {
        box-shadow: inset 0 0 6px rgba(0, 0, 0, 0.3);
        background: var(--primary-color);
      }

      .expand-state {
        color: var(--font-color);
        vertical-align: sub;
        height: 20px;
        width: 20px;
        display: inline-block;
      }
      .expand-state:hover {
        cursor: pointer;
        color: var(--primary-color);
      }
    `;

    return [iconCss];
  }

  @property()
  totalRulesViolated: number;

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

  @query('.violations')
  _violations: HTMLElement;

  private _expandState: boolean;

  otherRuleSelected() {
    this.open = false;
    this._violations.style.display = 'none';
    this._expandState = false;
    this.requestUpdate();
  }

  render() {
    let truncatedAlert: TemplateResult;
    if (this.truncated) {
      // todo: make this into something good.
      truncatedAlert = html`
        <div style="background-color: red; color: white">
          Too many results...
        </div>
      `;
    }

    const expandIcon = html`
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="20"
        height="20"
        fill="currentColor"
        class="bi bi-plus-square"
        viewBox="0 0 16 16"
      >
        <path
          d="M14 1a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1h12zM2 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V2a2 2 0 0 0-2-2H2z"
        />
        <path
          d="M8 4a.5.5 0 0 1 .5.5v3h3a.5.5 0 0 1 0 1h-3v3a.5.5 0 0 1-1 0v-3h-3a.5.5 0 0 1 0-1h3v-3A.5.5 0 0 1 8 4z"
        />
      </svg>
    `;

    const contractIcon = html`
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="20"
        height="20"
        fill="currentColor"
        class="bi bi-dash-square"
        viewBox="0 0 16 16"
      >
        <path
          d="M14 1a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1h12zM2 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V2a2 2 0 0 0-2-2H2z"
        />
        <path d="M4 8a.5.5 0 0 1 .5-.5h7a.5.5 0 0 1 0 1h-7A.5.5 0 0 1 4 8z" />
      </svg>
    `;

    const expanded = this._expandState ? contractIcon : expandIcon;

    return html`
      <div class="details ${this._expandState ? 'open' : ''}">
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
      </div>
    `;
  }

  private _ruleSelected() {
    if (!this.open) {
      this._violations.style.display = 'block';
      // use some intelligence to resize this in a responsive way.
      const heightCalc =
        this.parentElement.parentElement.offsetHeight -
        this.totalRulesViolated * 60;
      this._violations.style.maxHeight = heightCalc + 'px';
      this._expandState = true;
    } else {
      this._violations.style.display = 'none';
      this._expandState = false;
    }

    this.open = !this.open;

    this.dispatchEvent(
      new CustomEvent<RuleSelectedEvent>(RuleSelected, {
        bubbles: true,
        composed: true,
        detail: { id: this.ruleId },
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
